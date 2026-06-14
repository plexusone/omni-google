# Omni-Google

Google Cloud providers for the PlexusOne ecosystem.

## Packages

| Package | Description | Import Path |
|---------|-------------|-------------|
| **omnillm** | Gemini LLM provider for OmniLLM | `github.com/plexusone/omni-google/omnillm` |
| **omnistorage** | GCS and Drive backends for OmniStorage | `github.com/plexusone/omni-google/omnistorage` |
| **omnichat** | Gmail provider for OmniChat | `github.com/plexusone/omni-google/omnichat/gmail` |
| **omnivoice** | Gemini Live API for real-time voice | `github.com/plexusone/omni-google/omnivoice/realtime` |

## Installation

```bash
go get github.com/plexusone/omni-google
```

## Quick Start

### OmniLLM (Gemini)

```go
import "github.com/plexusone/omni-google/omnillm"

provider := gemini.NewProvider(os.Getenv("GEMINI_API_KEY"))
```

### OmniStorage (GCS)

```go
import "github.com/plexusone/omni-google/omnistorage/backend/gcs"

backend, err := gcs.New(gcs.Config{
    Bucket: "my-bucket",
})
```

### OmniStorage (Drive)

```go
import "github.com/plexusone/omni-google/omnistorage/backend/drive"

backend, err := drive.New(drive.Config{
    CredentialsFile: "/path/to/service-account.json",
    RootFolderID:    "folder-id",
})
```

### OmniChat (Gmail)

```go
import "github.com/plexusone/omni-google/omnichat/gmail"

provider, err := gmail.New(
    gmail.WithCredentialsJSON(creds),
    gmail.WithFromAddress("me"),
)
```

### OmniVoice (Gemini Live)

```go
import "github.com/plexusone/omni-google/omnivoice/realtime"

provider := realtime.NewRealtimeProvider(os.Getenv("GEMINI_API_KEY"),
    realtime.WithVoice("Puck"),
    realtime.WithInstructions("You are a helpful assistant."),
)

audioIn := make(chan []byte, 100)
audioCh, transcriptCh, err := provider.ProcessAudioStream(ctx, audioIn, config)
```

## Links

- [GitHub Repository](https://github.com/plexusone/omni-google)
- [Go Package Documentation](https://pkg.go.dev/github.com/plexusone/omni-google)
- [Release Notes](releases/index.md)
- [Changelog](https://github.com/plexusone/omni-google/blob/main/CHANGELOG.md)
