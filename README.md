# OmniStorage Google

[![Go Reference](https://pkg.go.dev/badge/github.com/grokify/omnistorage-google.svg)](https://pkg.go.dev/github.com/grokify/omnistorage-google)
[![Go Report Card](https://goreportcard.com/badge/github.com/grokify/omnistorage-google)](https://goreportcard.com/report/github.com/grokify/omnistorage-google)
[![CI](https://github.com/grokify/omnistorage-google/actions/workflows/ci.yml/badge.svg)](https://github.com/grokify/omnistorage-google/actions/workflows/ci.yml)

Google Cloud backends for [OmniStorage](https://github.com/grokify/omnistorage) - Google Drive and Google Cloud Storage (GCS).

This package is separate from the core OmniStorage module to keep dependencies minimal. Import only what you need.

## Installation

```bash
go get github.com/grokify/omnistorage-google
```

## Backends

| Backend | Package | Description |
|---------|---------|-------------|
| Google Drive | `backend/drive` | Google Drive API with OAuth2 and service account auth |
| Google Cloud Storage | `backend/gcs` | GCS with Application Default Credentials |

Both backends implement `omnistorage.ExtendedBackend` with full support for:

- Read/Write operations
- Stat, Copy, Move
- Mkdir, Rmdir
- Server-side copy
- Hash support (MD5, CRC32C for GCS)

## Google Drive Backend

### With Service Account

```go
import (
    "context"
    "github.com/grokify/omnistorage-google/backend/drive"
)

func main() {
    ctx := context.Background()

    backend, err := drive.New(drive.Config{
        CredentialsFile: "/path/to/service-account.json",
        RootFolderID:    "your-folder-id",
    })
    if err != nil {
        panic(err)
    }
    defer backend.Close()

    // Write a file
    w, _ := backend.NewWriter(ctx, "docs/hello.txt")
    w.Write([]byte("Hello from Drive!"))
    w.Close()

    // Read it back
    r, _ := backend.NewReader(ctx, "docs/hello.txt")
    data, _ := io.ReadAll(r)
    r.Close()
}
```

### With OAuth2 User Credentials

```go
backend, err := drive.New(drive.Config{
    CredentialsFile: "/path/to/oauth-client.json",
    TokenFile:       "/path/to/token.json",
    RootFolderID:    "your-folder-id",
})
```

### Configuration Options

| Option | Description |
|--------|-------------|
| `RootFolderID` | Folder ID to use as root (default: "root" for My Drive) |
| `CredentialsFile` | Path to service account or OAuth2 client JSON |
| `CredentialsJSON` | Raw JSON credentials (alternative to file) |
| `TokenFile` | Path to OAuth2 token file (for user credentials) |
| `Token` | Existing OAuth2 token |
| `SharedDrive` | Enable Shared Drive (Team Drive) support |
| `ChunkSize` | Resumable upload chunk size (default: 8MB) |

### Environment Variables

```bash
export OMNISTORAGE_GDRIVE_ROOT_FOLDER_ID="folder-id"
export OMNISTORAGE_GDRIVE_CREDENTIALS_FILE="/path/to/creds.json"
export OMNISTORAGE_GDRIVE_TOKEN_FILE="/path/to/token.json"
export OMNISTORAGE_GDRIVE_SHARED_DRIVE="true"
```

```go
cfg := drive.ConfigFromEnv()
backend, err := drive.New(cfg)
```

## Google Cloud Storage Backend

### With Application Default Credentials

```go
import (
    "context"
    "github.com/grokify/omnistorage-google/backend/gcs"
)

func main() {
    ctx := context.Background()

    backend, err := gcs.New(gcs.Config{
        Bucket: "my-bucket",
    })
    if err != nil {
        panic(err)
    }
    defer backend.Close()

    // Write a file
    w, _ := backend.NewWriter(ctx, "data/hello.txt")
    w.Write([]byte("Hello from GCS!"))
    w.Close()

    // Read it back
    r, _ := backend.NewReader(ctx, "data/hello.txt")
    data, _ := io.ReadAll(r)
    r.Close()
}
```

### With Service Account

```go
backend, err := gcs.New(gcs.Config{
    Bucket:          "my-bucket",
    CredentialsFile: "/path/to/service-account.json",
})
```

### Configuration Options

| Option | Description |
|--------|-------------|
| `Bucket` | GCS bucket name (required) |
| `Project` | Google Cloud project ID |
| `Prefix` | Object path prefix |
| `CredentialsFile` | Path to service account JSON |
| `CredentialsJSON` | Raw JSON credentials |
| `ChunkSize` | Resumable upload chunk size (default: 16MB) |
| `Concurrency` | Concurrent operations (default: 5) |

### Environment Variables

```bash
export OMNISTORAGE_GCS_BUCKET="my-bucket"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/creds.json"
export OMNISTORAGE_GCS_PREFIX="data/"
```

```go
cfg := gcs.ConfigFromEnv()
backend, err := gcs.New(cfg)
```

## Using the Registry

Backends register themselves automatically when imported:

```go
import (
    "github.com/grokify/omnistorage"
    _ "github.com/grokify/omnistorage-google/backend/drive"
    _ "github.com/grokify/omnistorage-google/backend/gcs"
)

// Open Google Drive
driveBackend, _ := omnistorage.Open("gdrive", map[string]string{
    "credentials_file": "/path/to/creds.json",
    "root_folder_id":   "folder-id",
})

// Open GCS
gcsBackend, _ := omnistorage.Open("gcs", map[string]string{
    "bucket": "my-bucket",
})
```

## Features Comparison

| Feature | Google Drive | GCS |
|---------|--------------|-----|
| Read/Write | Yes | Yes |
| Stat | Yes | Yes |
| Copy | Yes (server-side) | Yes (server-side) |
| Move | Yes (server-side) | Yes (copy+delete) |
| Mkdir | Yes | Yes (marker objects) |
| Range Read | Yes | Yes |
| MD5 Hash | Yes | Yes |
| CRC32C Hash | No | Yes |
| Versioning | No | Yes (bucket config) |
| Shared Drives | Yes | N/A |

## Authentication

### Google Drive

1. **Service Account** (recommended for server-to-server):
   - Create a service account in Google Cloud Console
   - Download the JSON key file
   - Share the target folder with the service account email

2. **OAuth2 User Credentials** (for user-facing apps):
   - Create OAuth2 client credentials
   - Implement OAuth2 flow to get user token
   - Store token for reuse

### Google Cloud Storage

1. **Application Default Credentials** (recommended):
   - Run `gcloud auth application-default login` locally
   - Use Workload Identity in GKE
   - Automatic on Compute Engine, Cloud Run, etc.

2. **Service Account**:
   - Create a service account with Storage permissions
   - Download the JSON key file

## Related

- [OmniStorage](https://github.com/grokify/omnistorage) - Core library
- [Google Drive API](https://developers.google.com/drive/api/v3/about-sdk)
- [Google Cloud Storage](https://cloud.google.com/storage/docs)

## License

MIT License - see [LICENSE](LICENSE) for details.
