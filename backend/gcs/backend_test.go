package gcs

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/grokify/omnistorage"
)

// Integration tests require these environment variables:
//   - OMNISTORAGE_GCS_BUCKET or GCS_BUCKET
//   - GOOGLE_APPLICATION_CREDENTIALS (optional, for service account)

func getTestBackend(t *testing.T) *Backend {
	cfg := ConfigFromEnv()

	if cfg.Bucket == "" {
		t.Skip("OMNISTORAGE_GCS_BUCKET or GCS_BUCKET not set, skipping integration test")
	}

	backend, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create GCS backend: %v", err)
	}

	return backend
}

func TestFeatures(t *testing.T) {
	backend := &Backend{
		config: Config{Bucket: "test"},
	}

	features := backend.Features()

	if !features.Copy {
		t.Error("Features.Copy = false, want true")
	}
	if !features.Move {
		t.Error("Features.Move = false, want true")
	}
	if !features.Mkdir {
		t.Error("Features.Mkdir = false, want true")
	}
	if !features.Rmdir {
		t.Error("Features.Rmdir = false, want true")
	}
	if !features.Stat {
		t.Error("Features.Stat = false, want true")
	}
	if !features.RangeRead {
		t.Error("Features.RangeRead = false, want true")
	}
	if !features.ListPrefix {
		t.Error("Features.ListPrefix = false, want true")
	}
	if !features.CanStream {
		t.Error("Features.CanStream = false, want true")
	}
	if !features.ServerSideEncryption {
		t.Error("Features.ServerSideEncryption = false, want true")
	}
	if !features.Versioning {
		t.Error("Features.Versioning = false, want true")
	}
	if !features.SupportsHash(omnistorage.HashMD5) {
		t.Error("Features should support MD5 hash")
	}
	if !features.SupportsHash(omnistorage.HashCRC32C) {
		t.Error("Features should support CRC32C hash")
	}
}

func TestExtendedBackendInterface(t *testing.T) {
	backend := &Backend{config: Config{Bucket: "test"}}

	// Verify backend implements ExtendedBackend
	var _ omnistorage.ExtendedBackend = backend

	// Test AsExtended helper
	ext, ok := omnistorage.AsExtended(backend)
	if !ok {
		t.Error("AsExtended returned false for GCS backend")
	}
	if ext == nil {
		t.Error("AsExtended returned nil for GCS backend")
	}
}

func TestFullPath(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		path     string
		expected string
	}{
		{"no prefix", "", "file.txt", "file.txt"},
		{"with prefix", "data", "file.txt", "data/file.txt"},
		{"prefix with slash", "data/", "file.txt", "data/file.txt"},
		{"nested path", "prefix", "a/b/c.txt", "prefix/a/b/c.txt"},
		{"empty path", "prefix", "", "prefix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &Backend{config: Config{Bucket: "test", Prefix: tt.prefix}}
			result := backend.fullPath(tt.path)
			if result != tt.expected {
				t.Errorf("fullPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestCheckClosed(t *testing.T) {
	backend := &Backend{config: Config{Bucket: "test"}, closed: false}

	if err := backend.checkClosed(); err != nil {
		t.Errorf("checkClosed() = %v, want nil", err)
	}

	backend.closed = true
	if err := backend.checkClosed(); err != omnistorage.ErrBackendClosed {
		t.Errorf("checkClosed() = %v, want ErrBackendClosed", err)
	}
}

func TestBackendClosed(t *testing.T) {
	backend := &Backend{
		config: Config{Bucket: "test"},
		closed: true,
	}

	ctx := context.Background()

	_, err := backend.NewReader(ctx, "test.txt")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("NewReader error = %v, want ErrBackendClosed", err)
	}

	_, err = backend.NewWriter(ctx, "test.txt")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("NewWriter error = %v, want ErrBackendClosed", err)
	}

	_, err = backend.Exists(ctx, "test.txt")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Exists error = %v, want ErrBackendClosed", err)
	}

	err = backend.Delete(ctx, "test.txt")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Delete error = %v, want ErrBackendClosed", err)
	}

	_, err = backend.List(ctx, "")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("List error = %v, want ErrBackendClosed", err)
	}

	_, err = backend.Stat(ctx, "test.txt")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Stat error = %v, want ErrBackendClosed", err)
	}

	err = backend.Mkdir(ctx, "dir")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Mkdir error = %v, want ErrBackendClosed", err)
	}

	err = backend.Rmdir(ctx, "dir")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Rmdir error = %v, want ErrBackendClosed", err)
	}

	err = backend.Copy(ctx, "src", "dst")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Copy error = %v, want ErrBackendClosed", err)
	}

	err = backend.Move(ctx, "src", "dst")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Move error = %v, want ErrBackendClosed", err)
	}
}

func TestContextCancellation(t *testing.T) {
	backend := &Backend{
		config: Config{Bucket: "test"},
		closed: false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := backend.NewReader(ctx, "test.txt")
	if err == nil {
		t.Error("NewReader expected error for cancelled context")
	}

	_, err = backend.NewWriter(ctx, "test.txt")
	if err == nil {
		t.Error("NewWriter expected error for cancelled context")
	}

	_, err = backend.Exists(ctx, "test.txt")
	if err == nil {
		t.Error("Exists expected error for cancelled context")
	}

	err = backend.Delete(ctx, "test.txt")
	if err == nil {
		t.Error("Delete expected error for cancelled context")
	}

	_, err = backend.List(ctx, "")
	if err == nil {
		t.Error("List expected error for cancelled context")
	}

	_, err = backend.Stat(ctx, "test.txt")
	if err == nil {
		t.Error("Stat expected error for cancelled context")
	}

	err = backend.Mkdir(ctx, "dir")
	if err == nil {
		t.Error("Mkdir expected error for cancelled context")
	}

	err = backend.Rmdir(ctx, "dir")
	if err == nil {
		t.Error("Rmdir expected error for cancelled context")
	}

	err = backend.Copy(ctx, "src", "dst")
	if err == nil {
		t.Error("Copy expected error for cancelled context")
	}

	err = backend.Move(ctx, "src", "dst")
	if err == nil {
		t.Error("Move expected error for cancelled context")
	}
}

func TestNewWithInvalidConfig(t *testing.T) {
	cfg := Config{} // Missing bucket

	_, err := New(cfg)
	if err != ErrBucketRequired {
		t.Errorf("New() error = %v, want ErrBucketRequired", err)
	}
}

func TestNewFromConfig(t *testing.T) {
	configMap := map[string]string{
		"bucket": "test-bucket",
	}

	// This will fail because we don't have actual GCS credentials,
	// but it tests the config parsing
	_, err := NewFromConfig(configMap)
	// The error will be about credentials or connection, not config validation
	if err == ErrBucketRequired {
		t.Error("NewFromConfig should parse bucket correctly")
	}
}

func TestRegistration(t *testing.T) {
	if !omnistorage.IsRegistered("gcs") {
		t.Error("gcs backend should be registered")
	}
}

func TestCloseIdempotent(t *testing.T) {
	backend := &Backend{
		config: Config{Bucket: "test"},
		closed: false,
	}

	// Close should be idempotent
	backend.closed = true
	if err := backend.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil for already closed backend", err)
	}
}

// Integration tests - only run when credentials are available

func TestIntegrationWriteRead(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := "integration-test-" + time.Now().Format("20060102-150405") + "/test.txt"

	// Write
	w, err := backend.NewWriter(ctx, testPath, omnistorage.WithContentType("text/plain"))
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	data := []byte("hello GCS world")
	if _, err := w.Write(data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Read
	r, err := backend.NewReader(ctx, testPath)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	readData, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	_ = r.Close()

	if string(readData) != string(data) {
		t.Errorf("Read data = %q, want %q", readData, data)
	}

	// Cleanup
	_ = backend.Delete(ctx, testPath)
}

func TestIntegrationExists(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := "integration-test-" + time.Now().Format("20060102-150405") + "/exists-test.txt"

	// Should not exist
	exists, err := backend.Exists(ctx, testPath)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("File should not exist")
	}

	// Create file
	w, _ := backend.NewWriter(ctx, testPath)
	_, _ = w.Write([]byte("test"))
	_ = w.Close()

	// Should exist
	exists, err = backend.Exists(ctx, testPath)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("File should exist")
	}

	// Cleanup
	_ = backend.Delete(ctx, testPath)
}

func TestIntegrationDelete(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := "integration-test-" + time.Now().Format("20060102-150405") + "/delete-test.txt"

	// Create file
	w, _ := backend.NewWriter(ctx, testPath)
	_, _ = w.Write([]byte("test"))
	_ = w.Close()

	// Delete
	if err := backend.Delete(ctx, testPath); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Delete again should not error (idempotent)
	if err := backend.Delete(ctx, testPath); err != nil {
		t.Errorf("Second Delete failed: %v, want nil (idempotent)", err)
	}

	// Should not exist
	exists, _ := backend.Exists(ctx, testPath)
	if exists {
		t.Error("File should not exist after delete")
	}
}

func TestIntegrationStat(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := "integration-test-" + time.Now().Format("20060102-150405") + "/stat-test.txt"

	// Create file
	data := []byte("stat test data")
	w, _ := backend.NewWriter(ctx, testPath, omnistorage.WithContentType("text/plain"))
	_, _ = w.Write(data)
	_ = w.Close()

	// Stat
	info, err := backend.Stat(ctx, testPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Size() != int64(len(data)) {
		t.Errorf("Size = %d, want %d", info.Size(), len(data))
	}
	if info.ModTime().IsZero() {
		t.Error("ModTime is zero")
	}
	if info.IsDir() {
		t.Error("IsDir = true, want false")
	}

	// Cleanup
	_ = backend.Delete(ctx, testPath)
}

func TestIntegrationWriteAllReadAll(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := "integration-test-" + time.Now().Format("20060102-150405") + "/writeall-test.txt"

	// WriteAll
	data := []byte("writeall test data")
	if err := backend.WriteAll(ctx, testPath, data, omnistorage.WithContentType("text/plain")); err != nil {
		t.Fatalf("WriteAll failed: %v", err)
	}

	// ReadAll
	readData, err := backend.ReadAll(ctx, testPath)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if string(readData) != string(data) {
		t.Errorf("ReadAll data = %q, want %q", readData, data)
	}

	// Cleanup
	_ = backend.Delete(ctx, testPath)
}

func TestIntegrationCopy(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	basePath := "integration-test-" + time.Now().Format("20060102-150405")
	srcPath := basePath + "/copy-src.txt"
	dstPath := basePath + "/copy-dst.txt"

	// Create source
	data := []byte("copy test data")
	if err := backend.WriteAll(ctx, srcPath, data); err != nil {
		t.Fatalf("WriteAll failed: %v", err)
	}

	// Copy
	if err := backend.Copy(ctx, srcPath, dstPath); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// Verify destination
	readData, err := backend.ReadAll(ctx, dstPath)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if string(readData) != string(data) {
		t.Errorf("Copied data = %q, want %q", readData, data)
	}

	// Verify source still exists
	exists, _ := backend.Exists(ctx, srcPath)
	if !exists {
		t.Error("Source should still exist after copy")
	}

	// Cleanup
	_ = backend.Delete(ctx, srcPath)
	_ = backend.Delete(ctx, dstPath)
}

func TestIntegrationMove(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	basePath := "integration-test-" + time.Now().Format("20060102-150405")
	srcPath := basePath + "/move-src.txt"
	dstPath := basePath + "/move-dst.txt"

	// Create source
	data := []byte("move test data")
	if err := backend.WriteAll(ctx, srcPath, data); err != nil {
		t.Fatalf("WriteAll failed: %v", err)
	}

	// Move
	if err := backend.Move(ctx, srcPath, dstPath); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	// Verify destination
	readData, err := backend.ReadAll(ctx, dstPath)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if string(readData) != string(data) {
		t.Errorf("Moved data = %q, want %q", readData, data)
	}

	// Verify source is gone
	exists, _ := backend.Exists(ctx, srcPath)
	if exists {
		t.Error("Source should not exist after move")
	}

	// Cleanup
	_ = backend.Delete(ctx, dstPath)
}

func TestIntegrationList(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	basePath := "integration-test-" + time.Now().Format("20060102-150405")

	// Create some files
	files := []string{
		basePath + "/list-a.txt",
		basePath + "/list-b.txt",
		basePath + "/subdir/list-c.txt",
	}

	for _, f := range files {
		if err := backend.WriteAll(ctx, f, []byte("test")); err != nil {
			t.Fatalf("WriteAll failed: %v", err)
		}
	}

	// List
	paths, err := backend.List(ctx, basePath)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should have all test files
	if len(paths) < len(files) {
		t.Errorf("Found %d files, want at least %d", len(paths), len(files))
	}

	// Cleanup
	for _, f := range files {
		_ = backend.Delete(ctx, f)
	}
}

func TestIntegrationMkdirRmdir(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	dirPath := "integration-test-" + time.Now().Format("20060102-150405") + "/mkdir-test"

	// Create directory
	if err := backend.Mkdir(ctx, dirPath); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Check it exists (as directory marker)
	exists, err := backend.Exists(ctx, dirPath+"/")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Directory marker should exist after Mkdir")
	}

	// Remove directory
	if err := backend.Rmdir(ctx, dirPath); err != nil {
		t.Fatalf("Rmdir failed: %v", err)
	}

	// Should be gone
	exists, _ = backend.Exists(ctx, dirPath+"/")
	if exists {
		t.Error("Directory marker should not exist after Rmdir")
	}
}

func TestIntegrationRangeRead(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := "integration-test-" + time.Now().Format("20060102-150405") + "/range-test.txt"

	// Create file with known content
	data := []byte("0123456789abcdef")
	if err := backend.WriteAll(ctx, testPath, data); err != nil {
		t.Fatalf("WriteAll failed: %v", err)
	}

	// Read range (offset 5, length 5)
	r, err := backend.NewReader(ctx, testPath, omnistorage.WithOffset(5), omnistorage.WithLimit(5))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	readData, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	_ = r.Close()

	if string(readData) != "56789" {
		t.Errorf("Range read data = %q, want %q", readData, "56789")
	}

	// Cleanup
	_ = backend.Delete(ctx, testPath)
}

func TestDefaultChunkSize(t *testing.T) {
	cfg := Config{Bucket: "test"}

	// Chunk size should default to 16MB
	if cfg.ChunkSize != 0 {
		t.Errorf("Default ChunkSize = %d, want 0 (will be set during New)", cfg.ChunkSize)
	}

	defCfg := DefaultConfig()
	if defCfg.ChunkSize != 16*1024*1024 {
		t.Errorf("DefaultConfig().ChunkSize = %d, want %d", defCfg.ChunkSize, 16*1024*1024)
	}
}
