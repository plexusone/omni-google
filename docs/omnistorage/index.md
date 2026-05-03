# OmniStorage Backends for Google Cloud

[OmniStorage](https://github.com/plexusone/omnistorage-core) backends for Google Cloud Storage (GCS) and Google Drive.

## Installation

```bash
go get github.com/plexusone/omni-google
```

## Available Backends

| Backend | Package | Description |
|---------|---------|-------------|
| Google Cloud Storage | `omnistorage/backend/gcs` | GCS with Application Default Credentials |
| Google Drive | `omnistorage/backend/drive` | Google Drive API with OAuth2 and service account auth |

Both backends implement `omnistorage.ExtendedBackend` with full support for:

- Read/Write operations
- Stat, Copy, Move
- Mkdir, Rmdir
- Server-side copy
- Hash support (MD5, CRC32C for GCS)

## Google Cloud Storage

### Quick Start

```go
import (
    "context"
    "github.com/plexusone/omni-google/omnistorage/backend/gcs"
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

    // Read a file
    r, _ := backend.NewReader(ctx, "data/hello.txt")
    data, _ := io.ReadAll(r)
    r.Close()
}
```

### Authentication

1. **Application Default Credentials** (recommended):
   - Run `gcloud auth application-default login` locally
   - Use Workload Identity in GKE
   - Automatic on Compute Engine, Cloud Run, etc.

2. **Service Account**:
   - Create a service account with Storage permissions
   - Download the JSON key file
   - Set `GOOGLE_APPLICATION_CREDENTIALS` environment variable

### Features

| Feature | Supported |
|---------|-----------|
| Read/Write | Yes |
| Stat | Yes |
| Copy | Yes (server-side) |
| Move | Yes (copy+delete) |
| Mkdir | Yes (marker objects) |
| Range Read | Yes |
| MD5 Hash | Yes |
| CRC32C Hash | Yes |
| Versioning | Yes (bucket config) |

## Google Drive

### Quick Start

```go
import (
    "context"
    "github.com/plexusone/omni-google/omnistorage/backend/drive"
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
}
```

### Authentication

1. **Service Account** (recommended for server-to-server):
   - Create a service account in Google Cloud Console
   - Download the JSON key file
   - Share the target folder with the service account email

2. **OAuth2 User Credentials** (for user-facing apps):
   - Create OAuth2 client credentials
   - Implement OAuth2 flow to get user token

### Features

| Feature | Supported |
|---------|-----------|
| Read/Write | Yes |
| Stat | Yes |
| Copy | Yes (server-side) |
| Move | Yes (server-side) |
| Mkdir | Yes |
| Range Read | Yes |
| MD5 Hash | Yes |
| Shared Drives | Yes |

## Using the Registry

```go
import (
    omnistorage "github.com/plexusone/omnistorage-core/object"
    _ "github.com/plexusone/omni-google/omnistorage/backend/drive"
    _ "github.com/plexusone/omni-google/omnistorage/backend/gcs"
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
