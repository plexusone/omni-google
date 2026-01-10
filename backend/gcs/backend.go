// Package gcs provides a Google Cloud Storage backend for omnistorage.
//
// Basic usage with Application Default Credentials:
//
//	backend, err := gcs.New(gcs.Config{
//	    Bucket: "my-bucket",
//	})
//
// With service account:
//
//	backend, err := gcs.New(gcs.Config{
//	    Bucket:          "my-bucket",
//	    CredentialsFile: "/path/to/service-account.json",
//	})
package gcs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/grokify/omnistorage"
)

func init() {
	omnistorage.Register("gcs", NewFromConfig)
}

// Backend implements omnistorage.ExtendedBackend for Google Cloud Storage.
type Backend struct {
	client *storage.Client
	bucket *storage.BucketHandle
	config Config
	closed bool
	mu     sync.RWMutex
}

// New creates a new GCS backend with the given configuration.
func New(cfg Config) (*Backend, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.ChunkSize == 0 {
		cfg.ChunkSize = 16 * 1024 * 1024 // 16MB
	}

	ctx := context.Background()

	// Build client options
	var opts []option.ClientOption

	// Credentials - use google.CredentialsFromJSON for explicit credentials
	// to avoid deprecated WithCredentialsJSON/WithCredentialsFile functions
	if len(cfg.CredentialsJSON) > 0 {
		creds, err := google.CredentialsFromJSON(ctx, cfg.CredentialsJSON, storage.ScopeFullControl)
		if err != nil {
			return nil, fmt.Errorf("gcs: parsing credentials JSON: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	} else if cfg.CredentialsFile != "" {
		data, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, fmt.Errorf("gcs: reading credentials file: %w", err)
		}
		creds, err := google.CredentialsFromJSON(ctx, data, storage.ScopeFullControl)
		if err != nil {
			return nil, fmt.Errorf("gcs: parsing credentials file: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}
	// If no credentials specified, the client will use Application Default Credentials

	// Create client
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("gcs: creating client: %w", err)
	}

	bucket := client.Bucket(cfg.Bucket)

	return &Backend{
		client: client,
		bucket: bucket,
		config: cfg,
	}, nil
}

// NewFromConfig creates a new GCS backend from a config map.
// This is used by the omnistorage registry.
func NewFromConfig(configMap map[string]string) (omnistorage.Backend, error) {
	cfg := ConfigFromMap(configMap)
	return New(cfg)
}

// NewWriter creates a writer for the given path.
func (b *Backend) NewWriter(ctx context.Context, p string, opts ...omnistorage.WriterOption) (io.WriteCloser, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	objPath := b.fullPath(p)
	cfg := omnistorage.ApplyWriterOptions(opts...)

	obj := b.bucket.Object(objPath)
	w := obj.NewWriter(ctx)

	// Set content type if provided
	if cfg.ContentType != "" {
		w.ContentType = cfg.ContentType
	}

	// Set metadata if provided
	if len(cfg.Metadata) > 0 {
		w.Metadata = cfg.Metadata
	}

	// Set chunk size
	w.ChunkSize = b.config.ChunkSize

	return w, nil
}

// NewReader creates a reader for the given path.
func (b *Backend) NewReader(ctx context.Context, p string, opts ...omnistorage.ReaderOption) (io.ReadCloser, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	objPath := b.fullPath(p)
	cfg := omnistorage.ApplyReaderOptions(opts...)

	obj := b.bucket.Object(objPath)

	// Handle range requests
	if cfg.Offset > 0 || cfg.Limit > 0 {
		var length int64 = -1 // Read to end
		if cfg.Limit > 0 {
			length = cfg.Limit
		}
		r, err := obj.NewRangeReader(ctx, cfg.Offset, length)
		if err != nil {
			return nil, b.translateError(err)
		}
		return r, nil
	}

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, b.translateError(err)
	}
	return r, nil
}

// Exists checks if a path exists.
func (b *Backend) Exists(ctx context.Context, p string) (bool, error) {
	if err := b.checkClosed(); err != nil {
		return false, err
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	objPath := b.fullPath(p)
	obj := b.bucket.Object(objPath)

	_, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return false, nil
		}
		return false, b.translateError(err)
	}

	return true, nil
}

// Delete removes a path.
func (b *Backend) Delete(ctx context.Context, p string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	objPath := b.fullPath(p)
	obj := b.bucket.Object(objPath)

	err := obj.Delete(ctx)
	if err != nil {
		// Delete is idempotent
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil
		}
		return b.translateError(err)
	}

	return nil
}

// List lists paths with the given prefix.
func (b *Backend) List(ctx context.Context, prefix string) ([]string, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	fullPrefix := b.fullPath(prefix)

	var paths []string
	it := b.bucket.Objects(ctx, &storage.Query{Prefix: fullPrefix})

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gcs: listing objects: %w", err)
		}

		// Remove prefix to get relative path
		relPath := strings.TrimPrefix(attrs.Name, b.config.Prefix)
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath != "" {
			paths = append(paths, relPath)
		}
	}

	return paths, nil
}

// Close releases any resources held by the backend.
func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	return b.client.Close()
}

// Stat returns metadata about an object.
func (b *Backend) Stat(ctx context.Context, p string) (omnistorage.ObjectInfo, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	objPath := b.fullPath(p)
	obj := b.bucket.Object(objPath)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, b.translateError(err)
	}

	// Build hash map
	hashes := make(map[omnistorage.HashType]string)
	if attrs.MD5 != nil {
		hashes[omnistorage.HashMD5] = fmt.Sprintf("%x", attrs.MD5)
	}
	if attrs.CRC32C != 0 {
		hashes[omnistorage.HashCRC32C] = fmt.Sprintf("%08x", attrs.CRC32C)
	}

	return &omnistorage.BasicObjectInfo{
		ObjectPath:        p,
		ObjectSize:        attrs.Size,
		ObjectModTime:     attrs.Updated,
		ObjectIsDir:       strings.HasSuffix(attrs.Name, "/") && attrs.Size == 0,
		ObjectContentType: attrs.ContentType,
		ObjectHashes:      hashes,
		ObjectMetadata:    attrs.Metadata,
	}, nil
}

// Mkdir creates a directory marker.
func (b *Backend) Mkdir(ctx context.Context, p string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// GCS doesn't need directories - they're implicit from object paths
	// Create a zero-byte object with trailing slash as directory marker
	objPath := b.fullPath(p)
	if !strings.HasSuffix(objPath, "/") {
		objPath += "/"
	}

	obj := b.bucket.Object(objPath)
	w := obj.NewWriter(ctx)
	w.ContentType = "application/x-directory"

	if err := w.Close(); err != nil {
		return fmt.Errorf("gcs: creating directory marker: %w", err)
	}

	return nil
}

// Rmdir removes a directory marker.
func (b *Backend) Rmdir(ctx context.Context, p string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	objPath := b.fullPath(p)
	if !strings.HasSuffix(objPath, "/") {
		objPath += "/"
	}

	// Check if directory is empty
	it := b.bucket.Objects(ctx, &storage.Query{
		Prefix: objPath,
	})

	count := 0
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("gcs: checking directory: %w", err)
		}
		// Count non-marker objects
		if attrs.Name != objPath {
			count++
			if count > 0 {
				break // No need to continue
			}
		}
	}

	if count > 0 {
		return fmt.Errorf("gcs: directory not empty: %s", p)
	}

	// Delete the directory marker
	obj := b.bucket.Object(objPath)
	if err := obj.Delete(ctx); err != nil {
		if !errors.Is(err, storage.ErrObjectNotExist) {
			return b.translateError(err)
		}
	}

	return nil
}

// Copy copies an object using server-side copy.
func (b *Backend) Copy(ctx context.Context, src, dst string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	srcPath := b.fullPath(src)
	dstPath := b.fullPath(dst)

	srcObj := b.bucket.Object(srcPath)
	dstObj := b.bucket.Object(dstPath)

	// Perform server-side copy
	copier := dstObj.CopierFrom(srcObj)
	_, err := copier.Run(ctx)
	if err != nil {
		return b.translateError(err)
	}

	return nil
}

// Move moves an object by copying then deleting.
func (b *Backend) Move(ctx context.Context, src, dst string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Copy first
	if err := b.Copy(ctx, src, dst); err != nil {
		return err
	}

	// Delete source
	return b.Delete(ctx, src)
}

// Features returns the capabilities of the GCS backend.
func (b *Backend) Features() omnistorage.Features {
	return omnistorage.Features{
		Copy:                 true,
		Move:                 true, // Implemented as copy+delete
		Mkdir:                true, // Creates marker objects
		Rmdir:                true, // Deletes marker objects
		Stat:                 true,
		Hashes:               []omnistorage.HashType{omnistorage.HashMD5, omnistorage.HashCRC32C},
		CanStream:            true,
		ServerSideEncryption: true,
		Versioning:           true, // Depends on bucket config
		RangeRead:            true,
		ListPrefix:           true,
	}
}

// fullPath returns the full GCS object path.
func (b *Backend) fullPath(p string) string {
	if b.config.Prefix == "" {
		return p
	}
	return path.Join(b.config.Prefix, p)
}

// checkClosed returns an error if the backend is closed.
func (b *Backend) checkClosed() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return omnistorage.ErrBackendClosed
	}
	return nil
}

// translateError converts GCS errors to omnistorage errors.
func (b *Backend) translateError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, storage.ErrObjectNotExist) {
		return omnistorage.ErrNotFound
	}

	if errors.Is(err, storage.ErrBucketNotExist) {
		return fmt.Errorf("gcs: bucket not found: %s", b.config.Bucket)
	}

	// Check for permission errors
	errStr := err.Error()
	if strings.Contains(errStr, "403") || strings.Contains(errStr, "Permission denied") {
		return omnistorage.ErrPermissionDenied
	}

	return fmt.Errorf("gcs: %w", err)
}

// WriteAll writes all data to the given path.
// This is a convenience method for small files.
func (b *Backend) WriteAll(ctx context.Context, p string, data []byte, opts ...omnistorage.WriterOption) error {
	w, err := b.NewWriter(ctx, p, opts...)
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		_ = w.Close()
		return err
	}

	return w.Close()
}

// ReadAll reads all data from the given path.
// This is a convenience method for small files.
func (b *Backend) ReadAll(ctx context.Context, p string) ([]byte, error) {
	r, err := b.NewReader(ctx, p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()

	return io.ReadAll(r)
}

// Ensure Backend implements omnistorage.ExtendedBackend
var _ omnistorage.ExtendedBackend = (*Backend)(nil)
