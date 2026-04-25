# Omni-Google

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omni-google/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omni-google/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omni-google/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omni-google/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omni-google/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omni-google/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omni-google
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omni-google
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omni-google
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omni-google
 [docs-mkdoc-svg]: https://img.shields.io/badge/Go-dev%20guide-blue.svg
 [docs-mkdoc-url]: https://plexusone.dev/omni-google
 [viz-svg]: https://img.shields.io/badge/Go-visualizaton-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomni-google
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omni-google
 [repo-url]: https://github.com/plexusone/omni-google
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omni-google/blob/main/LICENSE

Google Cloud providers for the OmniStorage and OmniLLM ecosystems.

## Modules

| Module | Package | Description |
|--------|---------|-------------|
| **omnillm** | `github.com/plexusone/omni-google/omnillm` | Gemini LLM provider for [omnillm-core](https://github.com/plexusone/omnillm-core) |
| **omnistorage** | `github.com/plexusone/omni-google/omnistorage` | GCS and Drive backends for [omnistorage-core](https://github.com/plexusone/omnistorage-core) |

## Installation

```bash
# For Gemini LLM provider
go get github.com/plexusone/omni-google/omnillm

# For storage backends (GCS, Drive)
go get github.com/plexusone/omni-google/omnistorage
```

---

## OmniLLM - Gemini Provider

The `omnillm` module provides a Gemini provider for [omnillm-core](https://github.com/plexusone/omnillm-core).

### Quick Start

```go
import (
    "context"
    "os"

    "github.com/plexusone/omni-google/omnillm"
)

func main() {
    ctx := context.Background()

    // Create provider with API key
    provider := gemini.NewProvider(os.Getenv("GEMINI_API_KEY"))

    // Create a chat completion
    resp, err := provider.CreateChatCompletion(ctx, &provider.ChatCompletionRequest{
        Model: "gemini-2.0-flash",
        Messages: []provider.Message{
            {Role: "user", Content: "Hello, Gemini!"},
        },
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

### Using the Registry

```go
import (
    omnillm "github.com/plexusone/omnillm-core"
    _ "github.com/plexusone/omni-google/omnillm" // Register Gemini provider
)

provider, err := omnillm.NewProvider(omnillm.ProviderConfig{
    Provider: omnillm.ProviderNameGemini,
    APIKey:   os.Getenv("GEMINI_API_KEY"),
})
```

### Supported Features

- ✅ Chat completions
- ✅ Streaming responses
- ✅ System prompts
- ✅ Multi-turn conversations

---

## OmniStorage - GCS and Drive Backends

The `omnistorage` module provides Google Cloud Storage and Google Drive backends.

### Storage Backends

| Backend | Package | Description |
|---------|---------|-------------|
| Google Drive | `omnistorage/backend/drive` | Google Drive API with OAuth2 and service account auth |
| Google Cloud Storage | `omnistorage/backend/gcs` | GCS with Application Default Credentials |

Both backends implement `omnistorage.ExtendedBackend` with full support for:

- 📄 Read/Write operations
- ℹ️ Stat, Copy, Move
- 📂 Mkdir, Rmdir
- ⚡ Server-side copy
- 🔐 Hash support (MD5, CRC32C for GCS)

### Google Drive Backend

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

### Google Cloud Storage Backend

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
}
```

### Using the Storage Registry

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

### Features Comparison

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

---

## Authentication

### Gemini API

Set your API key:

```bash
export GEMINI_API_KEY="your-api-key"
```

### Google Drive

1. **Service Account** (recommended for server-to-server):
   - Create a service account in Google Cloud Console
   - Download the JSON key file
   - Share the target folder with the service account email

2. **OAuth2 User Credentials** (for user-facing apps):
   - Create OAuth2 client credentials
   - Implement OAuth2 flow to get user token

### Google Cloud Storage

1. **Application Default Credentials** (recommended):
   - Run `gcloud auth application-default login` locally
   - Use Workload Identity in GKE
   - Automatic on Compute Engine, Cloud Run, etc.

2. **Service Account**:
   - Create a service account with Storage permissions
   - Download the JSON key file

## Related Projects

- [omnillm-core](https://github.com/plexusone/omnillm-core) - Core LLM abstraction library
- [omnistorage-core](https://github.com/plexusone/omnistorage-core) - Core storage abstraction library
- [omni-aws](https://github.com/plexusone/omni-aws) - AWS Bedrock and S3 providers
- [omni-github](https://github.com/plexusone/omni-github) - GitHub repository backend

## License

MIT License - see [LICENSE](LICENSE) for details.
