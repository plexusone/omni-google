// Package drive provides a Google Drive backend for omnistorage.
//
// Google Drive differs from traditional filesystems in that it uses file IDs
// rather than paths. This backend handles path-to-ID resolution transparently,
// creating folders as needed.
//
// Authentication:
//
// The backend supports two authentication methods:
//
//  1. Service Account: Use a service account JSON key file. The service account
//     must have access to the target folder/drive.
//
//  2. OAuth2 User Credentials: Use OAuth2 client credentials with a token.
//     Useful for accessing a user's personal Drive.
//
// Basic usage with service account:
//
//	cfg := drive.Config{
//	    CredentialsFile: "/path/to/service-account.json",
//	    RootFolderID:    "folder-id-here",
//	}
//	backend, err := drive.New(cfg)
package drive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/grokify/omnistorage"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

const (
	// mimeTypeFolder is the MIME type for Google Drive folders.
	mimeTypeFolder = "application/vnd.google-apps.folder"

	// backendName is the registered name for this backend.
	backendName = "gdrive"
)

func init() {
	omnistorage.Register(backendName, func(config map[string]string) (omnistorage.Backend, error) {
		cfg := ConfigFromMap(config)
		return New(cfg)
	})
}

// Backend implements omnistorage.Backend and omnistorage.ExtendedBackend
// for Google Drive.
type Backend struct {
	service *drive.Service
	config  Config
	closed  bool
	mu      sync.RWMutex

	// pathCache caches path-to-ID mappings for performance.
	pathCache   map[string]string
	pathCacheMu sync.RWMutex
}

// New creates a new Google Drive backend.
func New(cfg Config) (*Backend, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx := context.Background()
	service, err := cfg.createDriveService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	// Use "root" if no root folder specified
	if cfg.RootFolderID == "" {
		cfg.RootFolderID = "root"
	}

	return &Backend{
		service:   service,
		config:    cfg,
		pathCache: make(map[string]string),
	}, nil
}

// NewReader returns a reader for the file at the given path.
func (b *Backend) NewReader(ctx context.Context, filePath string, opts ...omnistorage.ReaderOption) (io.ReadCloser, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	options := omnistorage.ApplyReaderOptions(opts...)

	fileID, err := b.resolvePathToID(ctx, filePath)
	if err != nil {
		return nil, err
	}

	call := b.service.Files.Get(fileID).Context(ctx)

	resp, err := call.Download()
	if err != nil {
		return nil, b.wrapError(err, filePath)
	}

	// Handle offset and limit by wrapping the reader
	if options.Offset > 0 || options.Limit > 0 {
		return &offsetReader{
			r:      resp.Body,
			offset: options.Offset,
			limit:  options.Limit,
		}, nil
	}

	return resp.Body, nil
}

// NewWriter returns a writer for the file at the given path.
func (b *Backend) NewWriter(ctx context.Context, filePath string, opts ...omnistorage.WriterOption) (io.WriteCloser, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	options := omnistorage.ApplyWriterOptions(opts...)

	return &driveWriter{
		backend:     b,
		ctx:         ctx,
		path:        filePath,
		contentType: options.ContentType,
		buf:         &bytes.Buffer{},
	}, nil
}

// Exists checks if a file exists at the given path.
func (b *Backend) Exists(ctx context.Context, filePath string) (bool, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return false, omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	_, err := b.resolvePathToID(ctx, filePath)
	if err != nil {
		if omnistorage.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Delete removes the file at the given path.
func (b *Backend) Delete(ctx context.Context, filePath string) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	fileID, err := b.resolvePathToID(ctx, filePath)
	if err != nil {
		if omnistorage.IsNotFound(err) {
			return nil // Idempotent delete
		}
		return err
	}

	err = b.service.Files.Delete(fileID).Context(ctx).SupportsAllDrives(b.config.SharedDrive).Do()
	if err != nil {
		return b.wrapError(err, filePath)
	}

	// Invalidate cache
	b.invalidateCache(filePath)

	return nil
}

// List returns all file paths with the given prefix.
func (b *Backend) List(ctx context.Context, prefix string) ([]string, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	// Find the folder to list
	var folderID string
	var basePath string

	if prefix == "" {
		folderID = b.config.RootFolderID
		basePath = ""
	} else {
		// Check if prefix is a folder
		id, err := b.resolvePathToID(ctx, prefix)
		if err != nil {
			if omnistorage.IsNotFound(err) {
				// Prefix might be a partial path, list parent
				dir := path.Dir(prefix)
				if dir == "." {
					folderID = b.config.RootFolderID
					basePath = ""
				} else {
					folderID, err = b.resolvePathToID(ctx, dir)
					if err != nil {
						return nil, err
					}
					basePath = dir
				}
			} else {
				return nil, err
			}
		} else {
			folderID = id
			basePath = prefix
		}
	}

	return b.listRecursive(ctx, folderID, basePath, prefix)
}

// listRecursive lists files recursively from a folder.
func (b *Backend) listRecursive(ctx context.Context, folderID, basePath, prefix string) ([]string, error) {
	var paths []string

	query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	pageToken := ""

	for {
		call := b.service.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name, mimeType)").
			Context(ctx).
			SupportsAllDrives(b.config.SharedDrive).
			IncludeItemsFromAllDrives(b.config.SharedDrive)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		result, err := call.Do()
		if err != nil {
			return nil, b.wrapError(err, basePath)
		}

		for _, file := range result.Files {
			var filePath string
			if basePath == "" {
				filePath = file.Name
			} else {
				filePath = path.Join(basePath, file.Name)
			}

			// Filter by prefix if specified
			if prefix != "" && !strings.HasPrefix(filePath, prefix) {
				continue
			}

			if file.MimeType == mimeTypeFolder {
				// Recurse into folder
				subPaths, err := b.listRecursive(ctx, file.Id, filePath, prefix)
				if err != nil {
					return nil, err
				}
				paths = append(paths, subPaths...)
			} else {
				paths = append(paths, filePath)
			}
		}

		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return paths, nil
}

// Close closes the backend.
func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	return nil
}

// Stat returns metadata about the file at the given path.
func (b *Backend) Stat(ctx context.Context, filePath string) (omnistorage.ObjectInfo, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	fileID, err := b.resolvePathToID(ctx, filePath)
	if err != nil {
		return nil, err
	}

	file, err := b.service.Files.Get(fileID).
		Fields("id, name, size, mimeType, modifiedTime, md5Checksum").
		Context(ctx).
		SupportsAllDrives(b.config.SharedDrive).
		Do()
	if err != nil {
		return nil, b.wrapError(err, filePath)
	}

	modTime, _ := time.Parse(time.RFC3339, file.ModifiedTime)

	return &objectInfo{
		path:        filePath,
		size:        file.Size,
		modTime:     modTime,
		contentType: file.MimeType,
		isDir:       file.MimeType == mimeTypeFolder,
		md5:         file.Md5Checksum,
	}, nil
}

// Mkdir creates a directory at the given path.
func (b *Backend) Mkdir(ctx context.Context, dirPath string) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	_, err := b.ensureFolderPath(ctx, dirPath)
	return err
}

// Rmdir removes an empty directory at the given path.
func (b *Backend) Rmdir(ctx context.Context, dirPath string) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	folderID, err := b.resolvePathToID(ctx, dirPath)
	if err != nil {
		if omnistorage.IsNotFound(err) {
			return nil
		}
		return err
	}

	// Check if folder is empty
	query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	result, err := b.service.Files.List().
		Q(query).
		Fields("files(id)").
		PageSize(1).
		Context(ctx).
		SupportsAllDrives(b.config.SharedDrive).
		Do()
	if err != nil {
		return b.wrapError(err, dirPath)
	}

	if len(result.Files) > 0 {
		return fmt.Errorf("directory not empty: %s", dirPath)
	}

	err = b.service.Files.Delete(folderID).Context(ctx).SupportsAllDrives(b.config.SharedDrive).Do()
	if err != nil {
		return b.wrapError(err, dirPath)
	}

	b.invalidateCache(dirPath)
	return nil
}

// Copy copies a file from src to dst.
func (b *Backend) Copy(ctx context.Context, src, dst string) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	srcID, err := b.resolvePathToID(ctx, src)
	if err != nil {
		return err
	}

	// Ensure destination folder exists
	dstDir := path.Dir(dst)
	dstName := path.Base(dst)

	var parentID string
	if dstDir == "." || dstDir == "" {
		parentID = b.config.RootFolderID
	} else {
		parentID, err = b.ensureFolderPath(ctx, dstDir)
		if err != nil {
			return err
		}
	}

	// Copy file
	copyFile := &drive.File{
		Name:    dstName,
		Parents: []string{parentID},
	}

	_, err = b.service.Files.Copy(srcID, copyFile).
		Context(ctx).
		SupportsAllDrives(b.config.SharedDrive).
		Do()
	if err != nil {
		return b.wrapError(err, dst)
	}

	return nil
}

// Move moves a file from src to dst.
func (b *Backend) Move(ctx context.Context, src, dst string) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return omnistorage.ErrBackendClosed
	}
	b.mu.RUnlock()

	srcID, err := b.resolvePathToID(ctx, src)
	if err != nil {
		return err
	}

	// Get current parent
	file, err := b.service.Files.Get(srcID).
		Fields("parents").
		Context(ctx).
		SupportsAllDrives(b.config.SharedDrive).
		Do()
	if err != nil {
		return b.wrapError(err, src)
	}

	// Ensure destination folder exists
	dstDir := path.Dir(dst)
	dstName := path.Base(dst)

	var newParentID string
	if dstDir == "." || dstDir == "" {
		newParentID = b.config.RootFolderID
	} else {
		newParentID, err = b.ensureFolderPath(ctx, dstDir)
		if err != nil {
			return err
		}
	}

	// Move and rename file
	updateFile := &drive.File{
		Name: dstName,
	}

	previousParents := strings.Join(file.Parents, ",")

	_, err = b.service.Files.Update(srcID, updateFile).
		AddParents(newParentID).
		RemoveParents(previousParents).
		Context(ctx).
		SupportsAllDrives(b.config.SharedDrive).
		Do()
	if err != nil {
		return b.wrapError(err, dst)
	}

	// Update cache
	b.invalidateCache(src)

	return nil
}

// Features returns the capabilities of this backend.
func (b *Backend) Features() omnistorage.Features {
	return omnistorage.Features{
		Copy:       true,
		Move:       true,
		Mkdir:      true,
		Rmdir:      true,
		Stat:       true,
		RangeRead:  true,
		ListPrefix: true,
		Hashes:     []omnistorage.HashType{omnistorage.HashMD5},
	}
}

// resolvePathToID converts a path like "folder/subfolder/file.txt" to a Drive file ID.
func (b *Backend) resolvePathToID(ctx context.Context, filePath string) (string, error) {
	if filePath == "" || filePath == "/" {
		return b.config.RootFolderID, nil
	}

	// Check cache first
	b.pathCacheMu.RLock()
	if id, ok := b.pathCache[filePath]; ok {
		b.pathCacheMu.RUnlock()
		return id, nil
	}
	b.pathCacheMu.RUnlock()

	// Split path into components
	cleanPath := path.Clean(filePath)
	parts := strings.Split(cleanPath, "/")

	currentID := b.config.RootFolderID

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		// Search for this part in the current folder
		query := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false",
			escapeQuery(part), currentID)

		result, err := b.service.Files.List().
			Q(query).
			Fields("files(id, mimeType)").
			PageSize(1).
			Context(ctx).
			SupportsAllDrives(b.config.SharedDrive).
			IncludeItemsFromAllDrives(b.config.SharedDrive).
			Do()
		if err != nil {
			return "", b.wrapError(err, filePath)
		}

		if len(result.Files) == 0 {
			return "", omnistorage.ErrNotFound
		}

		currentID = result.Files[0].Id
	}

	// Cache the result
	b.pathCacheMu.Lock()
	b.pathCache[filePath] = currentID
	b.pathCacheMu.Unlock()

	return currentID, nil
}

// ensureFolderPath creates all folders in the path and returns the final folder ID.
func (b *Backend) ensureFolderPath(ctx context.Context, folderPath string) (string, error) {
	if folderPath == "" || folderPath == "/" {
		return b.config.RootFolderID, nil
	}

	cleanPath := path.Clean(folderPath)
	parts := strings.Split(cleanPath, "/")

	currentID := b.config.RootFolderID
	currentPath := ""

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		// Check cache
		b.pathCacheMu.RLock()
		cachedID, cached := b.pathCache[currentPath]
		b.pathCacheMu.RUnlock()

		if cached {
			currentID = cachedID
			continue
		}

		// Search for existing folder
		query := fmt.Sprintf("name = '%s' and '%s' in parents and mimeType = '%s' and trashed = false",
			escapeQuery(part), currentID, mimeTypeFolder)

		result, err := b.service.Files.List().
			Q(query).
			Fields("files(id)").
			PageSize(1).
			Context(ctx).
			SupportsAllDrives(b.config.SharedDrive).
			IncludeItemsFromAllDrives(b.config.SharedDrive).
			Do()
		if err != nil {
			return "", b.wrapError(err, folderPath)
		}

		if len(result.Files) > 0 {
			currentID = result.Files[0].Id
		} else {
			// Create folder
			folder := &drive.File{
				Name:     part,
				MimeType: mimeTypeFolder,
				Parents:  []string{currentID},
			}

			created, err := b.service.Files.Create(folder).
				Context(ctx).
				SupportsAllDrives(b.config.SharedDrive).
				Do()
			if err != nil {
				return "", b.wrapError(err, folderPath)
			}
			currentID = created.Id
		}

		// Cache result
		b.pathCacheMu.Lock()
		b.pathCache[currentPath] = currentID
		b.pathCacheMu.Unlock()
	}

	return currentID, nil
}

// invalidateCache removes a path and its children from the cache.
func (b *Backend) invalidateCache(filePath string) {
	b.pathCacheMu.Lock()
	defer b.pathCacheMu.Unlock()

	delete(b.pathCache, filePath)

	// Also invalidate any children
	prefix := filePath + "/"
	for p := range b.pathCache {
		if strings.HasPrefix(p, prefix) {
			delete(b.pathCache, p)
		}
	}
}

// wrapError converts Google API errors to omnistorage errors.
func (b *Backend) wrapError(err error, path string) error {
	if err == nil {
		return nil
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 404:
			return omnistorage.ErrNotFound
		case 403:
			return fmt.Errorf("access denied: %s: %w", path, err)
		case 401:
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	return err
}

// escapeQuery escapes a string for use in Drive query syntax.
func escapeQuery(s string) string {
	// Escape single quotes
	return strings.ReplaceAll(s, "'", "\\'")
}

// offsetReader wraps a reader to skip offset bytes and limit reading.
type offsetReader struct {
	r        io.ReadCloser
	offset   int64
	limit    int64
	skipped  int64
	read     int64
	skipDone bool
}

func (r *offsetReader) Read(p []byte) (n int, err error) {
	// Skip offset bytes first
	if !r.skipDone {
		for r.skipped < r.offset {
			toSkip := r.offset - r.skipped
			if toSkip > int64(len(p)) {
				toSkip = int64(len(p))
			}
			n, err := r.r.Read(p[:toSkip])
			r.skipped += int64(n)
			if err != nil {
				return 0, err
			}
		}
		r.skipDone = true
	}

	// Apply limit
	if r.limit > 0 {
		remaining := r.limit - r.read
		if remaining <= 0 {
			return 0, io.EOF
		}
		if int64(len(p)) > remaining {
			p = p[:remaining]
		}
	}

	n, err = r.r.Read(p)
	r.read += int64(n)
	return n, err
}

func (r *offsetReader) Close() error {
	return r.r.Close()
}

// driveWriter buffers writes and uploads on Close.
type driveWriter struct {
	backend     *Backend
	ctx         context.Context
	path        string
	contentType string
	buf         *bytes.Buffer
	closed      bool
}

func (w *driveWriter) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	return w.buf.Write(p)
}

func (w *driveWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	// Ensure parent folder exists
	dir := path.Dir(w.path)
	name := path.Base(w.path)

	var parentID string
	var err error

	if dir == "." || dir == "" {
		parentID = w.backend.config.RootFolderID
	} else {
		parentID, err = w.backend.ensureFolderPath(w.ctx, dir)
		if err != nil {
			return err
		}
	}

	// Check if file already exists
	existingID, err := w.backend.resolvePathToID(w.ctx, w.path)
	if err != nil && !omnistorage.IsNotFound(err) {
		return err
	}

	// Determine content type
	contentType := w.contentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if existingID != "" {
		// Update existing file
		file := &drive.File{}
		_, err = w.backend.service.Files.Update(existingID, file).
			Media(bytes.NewReader(w.buf.Bytes()), googleapi.ContentType(contentType)).
			Context(w.ctx).
			SupportsAllDrives(w.backend.config.SharedDrive).
			Do()
	} else {
		// Create new file
		file := &drive.File{
			Name:    name,
			Parents: []string{parentID},
		}
		_, err = w.backend.service.Files.Create(file).
			Media(bytes.NewReader(w.buf.Bytes()), googleapi.ContentType(contentType)).
			Context(w.ctx).
			SupportsAllDrives(w.backend.config.SharedDrive).
			Do()
	}

	if err != nil {
		return w.backend.wrapError(err, w.path)
	}

	// Invalidate cache for this path
	w.backend.invalidateCache(w.path)

	return nil
}

// objectInfo implements omnistorage.ObjectInfo.
type objectInfo struct {
	path        string
	size        int64
	modTime     time.Time
	contentType string
	isDir       bool
	md5         string
}

func (o *objectInfo) Path() string        { return o.path }
func (o *objectInfo) Size() int64         { return o.size }
func (o *objectInfo) ModTime() time.Time  { return o.modTime }
func (o *objectInfo) IsDir() bool         { return o.isDir }
func (o *objectInfo) ContentType() string { return o.contentType }
func (o *objectInfo) Hash(t omnistorage.HashType) string {
	if t == omnistorage.HashMD5 {
		return o.md5
	}
	return ""
}
func (o *objectInfo) Metadata() map[string]string { return nil }

// Ensure interfaces are implemented
var (
	_ omnistorage.Backend         = (*Backend)(nil)
	_ omnistorage.ExtendedBackend = (*Backend)(nil)
)
