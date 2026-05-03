# Release Notes: v0.4.0

**Release Date:** 2026-05-03

## Summary

This release adds Gmail as a messaging provider for omnichat, enabling email integration through the Gmail API.

## Highlights

- **Gmail provider** for omnichat messaging integration

## New Features

### Gmail Provider (omnichat)

Email messaging support for omnichat using Gmail API:

```go
import "github.com/plexusone/omni-google/omnichat/gmail"

p, err := gmail.New(
    gmail.WithCredentialsJSON(credentialsJSON),
    gmail.WithFromAddress("me"),
    gmail.WithLogger(slog.Default()),
)

// Send email
router.Send(ctx, "gmail", "recipient@example.com", provider.OutgoingMessage{
    Content: "Hello from OmniChat!",
    Format:  provider.MessageFormatPlain,
    Metadata: map[string]any{
        "subject": "Test Email",
    },
})
```

Features:

- **Send emails** via Gmail REST API
- **OAuth authentication** with CLI token flow
- **HTML and plain text** message formats
- **Token persistence** for seamless re-authentication

## Supported Modules

omni-google now provides adapters for three core libraries:

| Module | Package | Description |
|--------|---------|-------------|
| omnillm | `omni-google/omnillm` | Gemini LLM provider |
| omnistorage | `omni-google/omnistorage` | GCS and Google Drive backends |
| omnichat | `omni-google/omnichat/gmail` | Gmail email provider |

## Dependencies

- Add `github.com/grokify/goauth` v0.23.29
- Add `github.com/grokify/gogoogle` v0.9.0
- Add `github.com/plexusone/omnichat` v0.6.0 (provider interface)

## Upgrade Guide

This release is backwards compatible. To add Gmail support:

```bash
go get github.com/plexusone/omni-google@v0.4.0
```

### Gmail Setup

1. Create OAuth credentials in Google Cloud Console
2. Enable Gmail API for your project
3. Download `client_secret.json`
4. Configure the provider with credentials
5. Run once to complete OAuth flow (opens browser)

See the Gmail Provider documentation for detailed setup instructions.

## Contributors

- Claude Opus 4.5 (AI pair programming)
