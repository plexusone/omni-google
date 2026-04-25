# OmniLLM Provider for Google Gemini

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omni-google/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omni-google/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omni-google/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omni-google/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omni-google/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omni-google/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omni-google
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omni-google
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omni-google/omnillm
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omni-google/omnillm
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omni-google/blob/main/LICENSE

Thick provider for [OmniLLM](https://github.com/plexusone/omnillm-core) using the official [google.golang.org/genai](https://pkg.go.dev/google.golang.org/genai) SDK.

## Installation

```bash
go get github.com/plexusone/omni-google/omnillm
```

## Quick Start

```go
import (
    omnillm "github.com/plexusone/omnillm-core"
    _ "github.com/plexusone/omni-google/omnillm" // Auto-registers thick provider
)

client, _ := omnillm.NewClient(omnillm.ClientConfig{
    Provider: omnillm.ProviderNameGemini,
    APIKey:   os.Getenv("GEMINI_API_KEY"),
})
```

## Feature Support

| Feature | Supported |
|---------|-----------|
| Chat Completion | Yes |
| Streaming | Yes |
| Tool Calling | No |
| System Messages | Yes |
| JSON Mode | No |

## Configuration

| Field | Required | Description |
|-------|----------|-------------|
| `APIKey` | Yes | Google AI API key |

## Documentation

- [OmniLLM Core](https://github.com/plexusone/omnillm-core) - Full API documentation
- [omni-google](https://github.com/plexusone/omni-google) - Parent repository

## License

MIT
