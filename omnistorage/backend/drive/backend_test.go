package drive

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	omnistorage "github.com/plexusone/omnistorage-core/object"
)

// Integration tests require these environment variables:
//   - OMNISTORAGE_GDRIVE_CREDENTIALS_FILE or OMNISTORAGE_GDRIVE_CREDENTIALS_JSON
//   - OMNISTORAGE_GDRIVE_ROOT_FOLDER_ID (optional, defaults to root)
//   - OMNISTORAGE_GDRIVE_TOKEN_FILE (for OAuth2 user credentials)

func getTestBackend(t *testing.T) *Backend {
	cfg := ConfigFromEnv()

	// Check if credentials are available
	if cfg.CredentialsFile == "" && len(cfg.CredentialsJSON) == 0 {
		t.Skip("OMNISTORAGE_GDRIVE_CREDENTIALS_FILE or OMNISTORAGE_GDRIVE_CREDENTIALS_JSON not set, skipping integration test")
	}

	// Use a test subfolder to avoid polluting the root
	if cfg.RootFolderID == "" || cfg.RootFolderID == "root" {
		t.Skip("OMNISTORAGE_GDRIVE_ROOT_FOLDER_ID not set to a test folder, skipping integration test")
	}

	backend, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create Google Drive backend: %v", err)
	}

	return backend
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "empty credentials",
			config:  Config{},
			wantErr: true,
		},
		{
			name: "valid with credentials file",
			//nolint:gosec // G101: test fixture, not real credentials
			config: Config{
				CredentialsFile: "/path/to/creds.json",
			},
			wantErr: false,
		},
		{
			name: "valid with credentials json",
			config: Config{
				CredentialsJSON: []byte(`{"type":"service_account"}`),
			},
			wantErr: false,
		},
		{
			name: "invalid chunk size",
			//nolint:gosec // G101: test fixture, not real credentials
			config: Config{
				CredentialsFile: "/path/to/creds.json",
				ChunkSize:       1000, // Not a multiple of 256KB
			},
			wantErr: true,
		},
		{
			name: "valid chunk size",
			//nolint:gosec // G101: test fixture, not real credentials
			config: Config{
				CredentialsFile: "/path/to/creds.json",
				ChunkSize:       256 * 1024, // 256KB
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.RootFolderID != "root" {
		t.Errorf("RootFolderID = %q, want %q", cfg.RootFolderID, "root")
	}
	if cfg.ChunkSize != 8*1024*1024 {
		t.Errorf("ChunkSize = %d, want %d", cfg.ChunkSize, 8*1024*1024)
	}
	if len(cfg.Scopes) != 1 {
		t.Errorf("Scopes length = %d, want 1", len(cfg.Scopes))
	}
}

func TestConfigFromMap(t *testing.T) {
	//nolint:gosec // G101: test fixture, not real credentials
	m := map[string]string{
		"root_folder_id":   "folder123",
		"credentials_file": "/path/to/creds.json",
		"token_file":       "/path/to/token.json",
		"shared_drive":     "true",
		"chunk_size":       "16777216",
	}

	cfg := ConfigFromMap(m)

	if cfg.RootFolderID != "folder123" {
		t.Errorf("RootFolderID = %q, want %q", cfg.RootFolderID, "folder123")
	}
	if cfg.CredentialsFile != "/path/to/creds.json" {
		t.Errorf("CredentialsFile = %q, want %q", cfg.CredentialsFile, "/path/to/creds.json")
	}
	if cfg.TokenFile != "/path/to/token.json" {
		t.Errorf("TokenFile = %q, want %q", cfg.TokenFile, "/path/to/token.json")
	}
	if !cfg.SharedDrive {
		t.Error("SharedDrive = false, want true")
	}
	if cfg.ChunkSize != 16777216 {
		t.Errorf("ChunkSize = %d, want %d", cfg.ChunkSize, 16777216)
	}
}

func TestFeatures(t *testing.T) {
	// Create a mock backend for testing features
	backend := &Backend{
		config: Config{RootFolderID: "root"},
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
	if !features.SupportsHash(omnistorage.HashMD5) {
		t.Error("Features should support MD5 hash")
	}
}

func TestExtendedBackendInterface(t *testing.T) {
	backend := &Backend{config: Config{RootFolderID: "root"}}

	// Verify backend implements ExtendedBackend
	var _ omnistorage.ExtendedBackend = backend

	// Test AsExtended helper
	ext, ok := omnistorage.AsExtended(backend)
	if !ok {
		t.Error("AsExtended returned false for Google Drive backend")
	}
	if ext == nil {
		t.Error("AsExtended returned nil for Google Drive backend")
	}
}

func TestEscapeQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"file.txt", "file.txt"},
		{"it's a test", "it\\'s a test"},
		{"don't", "don\\'t"},
		{"no'quotes'here", "no\\'quotes\\'here"},
	}

	for _, tt := range tests {
		result := escapeQuery(tt.input)
		if result != tt.expected {
			t.Errorf("escapeQuery(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestObjectInfo(t *testing.T) {
	now := time.Now()
	info := &objectInfo{
		path:        "test/file.txt",
		size:        1024,
		modTime:     now,
		contentType: "text/plain",
		isDir:       false,
		md5:         "abc123",
	}

	if info.Path() != "test/file.txt" {
		t.Errorf("Path() = %q, want %q", info.Path(), "test/file.txt")
	}
	if info.Size() != 1024 {
		t.Errorf("Size() = %d, want %d", info.Size(), 1024)
	}
	if !info.ModTime().Equal(now) {
		t.Error("ModTime() doesn't match")
	}
	if info.ContentType() != "text/plain" {
		t.Errorf("ContentType() = %q, want %q", info.ContentType(), "text/plain")
	}
	if info.IsDir() {
		t.Error("IsDir() = true, want false")
	}
	if info.Hash(omnistorage.HashMD5) != "abc123" {
		t.Errorf("Hash(MD5) = %q, want %q", info.Hash(omnistorage.HashMD5), "abc123")
	}
	if info.Hash(omnistorage.HashSHA256) != "" {
		t.Errorf("Hash(SHA256) = %q, want empty", info.Hash(omnistorage.HashSHA256))
	}
}

func TestRegistration(t *testing.T) {
	if !omnistorage.IsRegistered("gdrive") {
		t.Error("gdrive backend should be registered")
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

	data := []byte("hello Google Drive world")
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

func TestIntegrationCopy(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	basePath := "integration-test-" + time.Now().Format("20060102-150405")
	srcPath := basePath + "/copy-src.txt"
	dstPath := basePath + "/copy-dst.txt"

	// Create source
	data := []byte("copy test data")
	w, _ := backend.NewWriter(ctx, srcPath)
	_, _ = w.Write(data)
	_ = w.Close()

	// Copy
	if err := backend.Copy(ctx, srcPath, dstPath); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// Verify destination
	r, err := backend.NewReader(ctx, dstPath)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	readData, _ := io.ReadAll(r)
	_ = r.Close()

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
	w, _ := backend.NewWriter(ctx, srcPath)
	_, _ = w.Write(data)
	_ = w.Close()

	// Move
	if err := backend.Move(ctx, srcPath, dstPath); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	// Verify destination
	r, err := backend.NewReader(ctx, dstPath)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	readData, _ := io.ReadAll(r)
	_ = r.Close()

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
		w, _ := backend.NewWriter(ctx, f)
		_, _ = w.Write([]byte("test"))
		_ = w.Close()
	}

	// List
	paths, err := backend.List(ctx, basePath)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should have all test files
	found := 0
	for _, p := range paths {
		for _, f := range files {
			if p == f {
				found++
				break
			}
		}
	}

	if found != len(files) {
		t.Errorf("Found %d of %d test files in list (got %v)", found, len(files), paths)
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

	// Check it exists
	exists, err := backend.Exists(ctx, dirPath)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Directory should exist after Mkdir")
	}

	// Remove directory
	if err := backend.Rmdir(ctx, dirPath); err != nil {
		t.Fatalf("Rmdir failed: %v", err)
	}

	// Should be gone
	exists, _ = backend.Exists(ctx, dirPath)
	if exists {
		t.Error("Directory should not exist after Rmdir")
	}
}

func TestIntegrationOverwrite(t *testing.T) {
	backend := getTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := "integration-test-" + time.Now().Format("20060102-150405") + "/overwrite-test.txt"

	// Write initial content
	w, _ := backend.NewWriter(ctx, testPath)
	_, _ = w.Write([]byte("initial content"))
	_ = w.Close()

	// Overwrite with new content
	w, _ = backend.NewWriter(ctx, testPath)
	_, _ = w.Write([]byte("new content"))
	_ = w.Close()

	// Read and verify
	r, _ := backend.NewReader(ctx, testPath)
	data, _ := io.ReadAll(r)
	_ = r.Close()

	if string(data) != "new content" {
		t.Errorf("Overwritten data = %q, want %q", data, "new content")
	}

	// Cleanup
	_ = backend.Delete(ctx, testPath)
}

func TestBackendClosed(t *testing.T) {
	cfg := ConfigFromEnv()
	if cfg.CredentialsFile == "" && len(cfg.CredentialsJSON) == 0 {
		// Create a mock backend for closed tests
		backend := &Backend{
			config:    Config{RootFolderID: "root"},
			closed:    true,
			pathCache: make(map[string]string),
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

		return
	}

	backend := getTestBackend(t)
	_ = backend.Close()

	ctx := context.Background()

	_, err := backend.NewReader(ctx, "test.txt")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("NewReader after close error = %v, want ErrBackendClosed", err)
	}
}

func TestConfigFromEnv(t *testing.T) {
	// Save original values
	origCredFile := os.Getenv("OMNISTORAGE_GDRIVE_CREDENTIALS_FILE")
	origRootFolder := os.Getenv("OMNISTORAGE_GDRIVE_ROOT_FOLDER_ID")

	// Set test values
	_ = os.Setenv("OMNISTORAGE_GDRIVE_CREDENTIALS_FILE", "/test/creds.json")
	_ = os.Setenv("OMNISTORAGE_GDRIVE_ROOT_FOLDER_ID", "test-folder-id")

	cfg := ConfigFromEnv()

	if cfg.CredentialsFile != "/test/creds.json" {
		t.Errorf("CredentialsFile = %q, want %q", cfg.CredentialsFile, "/test/creds.json")
	}
	if cfg.RootFolderID != "test-folder-id" {
		t.Errorf("RootFolderID = %q, want %q", cfg.RootFolderID, "test-folder-id")
	}

	// Restore original values
	if origCredFile != "" {
		_ = os.Setenv("OMNISTORAGE_GDRIVE_CREDENTIALS_FILE", origCredFile)
	} else {
		_ = os.Unsetenv("OMNISTORAGE_GDRIVE_CREDENTIALS_FILE")
	}
	if origRootFolder != "" {
		_ = os.Setenv("OMNISTORAGE_GDRIVE_ROOT_FOLDER_ID", origRootFolder)
	} else {
		_ = os.Unsetenv("OMNISTORAGE_GDRIVE_ROOT_FOLDER_ID")
	}
}

func TestOffsetReader(t *testing.T) {
	// Create a test reader with known content
	data := []byte("0123456789abcdef")
	baseReader := io.NopCloser(bytes.NewReader(data))

	// Test reading with offset and limit
	r := &offsetReader{
		r:      baseReader,
		offset: 5,
		limit:  5,
	}

	buf := make([]byte, 100)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}

	if string(buf[:n]) != "56789" {
		t.Errorf("Read data = %q, want %q", buf[:n], "56789")
	}

	// Continue reading should return EOF
	n, err = r.Read(buf)
	if err != io.EOF {
		t.Errorf("Second read error = %v, want EOF", err)
	}
	if n != 0 {
		t.Errorf("Second read n = %d, want 0", n)
	}

	// Test close
	if err := r.Close(); err != nil {
		t.Errorf("Close error = %v", err)
	}
}

func TestOffsetReaderNoLimit(t *testing.T) {
	data := []byte("0123456789abcdef")
	baseReader := io.NopCloser(bytes.NewReader(data))

	// Test reading with offset only (no limit)
	r := &offsetReader{
		r:      baseReader,
		offset: 10,
		limit:  0, // no limit
	}

	buf := make([]byte, 100)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}

	if string(buf[:n]) != "abcdef" {
		t.Errorf("Read data = %q, want %q", buf[:n], "abcdef")
	}
}

func TestDriveWriter(t *testing.T) {
	backend := &Backend{
		config:    Config{RootFolderID: "root"},
		pathCache: make(map[string]string),
	}

	ctx := context.Background()
	w := &driveWriter{
		backend:     backend,
		ctx:         ctx,
		path:        "test.txt",
		contentType: "text/plain",
		buf:         &bytes.Buffer{},
	}

	// Write some data
	data := []byte("test data")
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write n = %d, want %d", n, len(data))
	}

	// Writing after close should fail
	w.closed = true
	_, err = w.Write([]byte("more data"))
	if err != io.ErrClosedPipe {
		t.Errorf("Write after close error = %v, want ErrClosedPipe", err)
	}
}

func TestCloseIdempotent(t *testing.T) {
	backend := &Backend{
		config:    Config{RootFolderID: "root"},
		pathCache: make(map[string]string),
	}

	// Close should be idempotent
	if err := backend.Close(); err != nil {
		t.Errorf("First Close error = %v", err)
	}

	if err := backend.Close(); err != nil {
		t.Errorf("Second Close error = %v", err)
	}
}

func TestInvalidateCache(t *testing.T) {
	backend := &Backend{
		config: Config{RootFolderID: "root"},
		pathCache: map[string]string{
			"file.txt":            "id1",
			"dir/file.txt":        "id2",
			"dir/subdir/file.txt": "id3",
			"other/file.txt":      "id4",
		},
	}

	// Invalidate "dir" should remove dir and its children
	backend.invalidateCache("dir")

	if _, ok := backend.pathCache["dir"]; ok {
		t.Error("dir should be removed from cache")
	}
	if _, ok := backend.pathCache["dir/file.txt"]; ok {
		t.Error("dir/file.txt should be removed from cache")
	}
	if _, ok := backend.pathCache["dir/subdir/file.txt"]; ok {
		t.Error("dir/subdir/file.txt should be removed from cache")
	}
	if _, ok := backend.pathCache["file.txt"]; !ok {
		t.Error("file.txt should not be removed from cache")
	}
	if _, ok := backend.pathCache["other/file.txt"]; !ok {
		t.Error("other/file.txt should not be removed from cache")
	}
}

func TestWrapError(t *testing.T) {
	backend := &Backend{}

	// nil error should return nil
	if err := backend.wrapError(nil, "test"); err != nil {
		t.Errorf("wrapError(nil) = %v, want nil", err)
	}

	// Generic error should pass through
	genericErr := errors.New("generic error")
	if err := backend.wrapError(genericErr, "test"); err != genericErr {
		t.Errorf("wrapError(generic) = %v, want %v", err, genericErr)
	}
}
