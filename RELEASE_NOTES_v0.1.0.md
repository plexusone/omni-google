# v0.1.0 Release Notes

**Release Date:** 2026-01-10

OmniStorage Google v0.1.0 is the initial release providing Google Drive and Google Cloud Storage backends for [OmniStorage](https://github.com/grokify/omnistorage).

## Highlights

- **Google Drive Backend** - Full ExtendedBackend implementation with OAuth2 and service account auth
- **Google Cloud Storage Backend** - Full ExtendedBackend implementation with Application Default Credentials
- **Server-Side Operations** - Copy and move operations leverage server-side APIs for efficiency
- **Flexible Authentication** - Service accounts, OAuth2 tokens, and Application Default Credentials

## Installation

```bash
go get github.com/grokify/omnistorage-google@v0.1.0
```

## What's Included

### Google Drive Backend

| Feature | Description |
|---------|-------------|
| Package | `backend/drive` |
| Registry Name | `gdrive` |
| Interface | `ExtendedBackend` |
| Auth | Service account, OAuth2 user credentials |
| Server-side Copy | Yes |
| Server-side Move | Yes |
| Hash | MD5 |
| Shared Drives | Yes (Team Drives) |

**Key Features:**

- Path-to-ID resolution with intelligent caching
- Automatic folder creation when writing files
- Shared Drive (Team Drive) support
- Resumable uploads with configurable chunk size

### Google Cloud Storage Backend

| Feature | Description |
|---------|-------------|
| Package | `backend/gcs` |
| Registry Name | `gcs` |
| Interface | `ExtendedBackend` |
| Auth | Service account, Application Default Credentials |
| Server-side Copy | Yes |
| Server-side Move | Copy + Delete |
| Hashes | MD5, CRC32C |
| Versioning | Depends on bucket config |

**Key Features:**

- Application Default Credentials for seamless GCP integration
- Server-side copy for efficient file duplication
- Range reads for partial file access
- Object prefix support for organizing data

## Quick Start

### Google Drive

```go
import "github.com/grokify/omnistorage-google/backend/drive"

backend, err := drive.New(drive.Config{
    CredentialsFile: "/path/to/service-account.json",
    RootFolderID:    "folder-id",
})
defer backend.Close()

w, _ := backend.NewWriter(ctx, "hello.txt")
w.Write([]byte("Hello, Drive!"))
w.Close()
```

### Google Cloud Storage

```go
import "github.com/grokify/omnistorage-google/backend/gcs"

backend, err := gcs.New(gcs.Config{
    Bucket: "my-bucket",
})
defer backend.Close()

w, _ := backend.NewWriter(ctx, "hello.txt")
w.Write([]byte("Hello, GCS!"))
w.Close()
```

### Using the Registry

```go
import (
    "github.com/grokify/omnistorage"
    _ "github.com/grokify/omnistorage-google/backend/drive"
    _ "github.com/grokify/omnistorage-google/backend/gcs"
)

// Open by name
backend, _ := omnistorage.Open("gcs", map[string]string{
    "bucket": "my-bucket",
})
```

## Dependencies

This package requires:

- Go 1.21 or later
- [OmniStorage](https://github.com/grokify/omnistorage) v0.1.0 or later
- Google Cloud SDK libraries

## Authentication Setup

### Google Drive

1. **Service Account** (recommended for servers):
   - Create a service account in [Google Cloud Console](https://console.cloud.google.com/)
   - Download the JSON key file
   - Share the target Drive folder with the service account email

2. **OAuth2 User Credentials** (for user apps):
   - Create OAuth2 credentials in Cloud Console
   - Implement OAuth2 flow to obtain user token
   - Pass token via `Token` or `TokenFile` config options

### Google Cloud Storage

1. **Application Default Credentials** (recommended):
   - Run `gcloud auth application-default login` for local development
   - Use Workload Identity for GKE
   - Automatic on Compute Engine, Cloud Run, Cloud Functions

2. **Service Account**:
   - Create service account with Storage permissions
   - Set `GOOGLE_APPLICATION_CREDENTIALS` or use `CredentialsFile`

## Known Limitations

- Google Drive uploads are buffered in memory before upload (no streaming upload)
- GCS Move is implemented as Copy + Delete (no atomic move in GCS API)
- Google Drive API has quota limits that may affect high-volume operations

## Documentation

- [README](https://github.com/grokify/omnistorage-google)
- [API Reference](https://pkg.go.dev/github.com/grokify/omnistorage-google)
- [OmniStorage Documentation](https://grokify.github.io/omnistorage/)

## What's Next

Planned improvements:

- Streaming uploads for Google Drive
- Batch operations for GCS
- Progress callbacks for large uploads
- Additional GCS features (lifecycle, ACLs)

## Contributing

Contributions welcome! Areas of interest:

1. Streaming upload support for Drive
2. Integration tests with emulators
3. Documentation improvements
4. Performance optimizations

## License

MIT License - see [LICENSE](https://github.com/grokify/omnistorage-google/blob/main/LICENSE)
