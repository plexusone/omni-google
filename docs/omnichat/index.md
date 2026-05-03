# OmniChat Provider for Gmail

[OmniChat](https://github.com/plexusone/omnichat) Provider for Gmail enables sending emails through the Gmail API.

## Installation

```bash
go get github.com/plexusone/omni-google
```

## Usage

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "os"

    "github.com/plexusone/omni-google/omnichat/gmail"
)

func main() {
    ctx := context.Background()

    // Load credentials from file
    creds, err := os.ReadFile("client_secret.json")
    if err != nil {
        log.Fatal(err)
    }

    // Create Gmail provider
    provider, err := gmail.New(
        gmail.WithCredentialsJSON(creds),
        gmail.WithFromAddress("me"),
        gmail.WithLogger(slog.Default()),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Connect (will prompt OAuth flow on first run)
    if err := provider.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer provider.Disconnect(ctx)

    // Send an email
    err = provider.SendEmail(ctx,
        "recipient@example.com",
        "Hello from OmniChat",
        "This email was sent via Gmail API!",
        false, // plain text
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## Using with OmniChat Router

```go
import (
    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omni-google/omnichat/gmail"
)

router := provider.NewRouter(logger)

gmailProvider, _ := gmail.New(
    gmail.WithCredentialsJSON(creds),
    gmail.WithFromAddress("me"),
    gmail.WithLogger(logger),
)

router.Register(gmailProvider)
router.ConnectAll(ctx)

// Send email via router
router.Send(ctx, "gmail", "recipient@example.com", provider.OutgoingMessage{
    Content: "Hello from OmniChat!",
    Format:  provider.MessageFormatPlain,
    Metadata: map[string]any{
        "subject": "Test Email",
    },
})
```

## Configuration Options

| Option | Description |
|--------|-------------|
| `WithCredentialsJSON([]byte)` | OAuth credentials JSON (required) |
| `WithTokenFile(string)` | Path to store OAuth token (default: `~/.omnichat/gmail_token.json`) |
| `WithFromAddress(string)` | Sender address, use "me" for authenticated user |
| `WithScopes([]string)` | OAuth scopes (default: `gmail.GmailSendScope`) |
| `WithLogger(*slog.Logger)` | Logger instance |
| `WithForceNewToken(bool)` | Force re-authentication |

## Authentication

### Setup Steps

1. **Create OAuth2 credentials** in Google Cloud Console
2. **Enable Gmail API** for your project
3. **Download** `client_secret.json`
4. **First run** opens browser for authorization
5. **Token cached** to `~/.omnichat/gmail_token.json`

### OAuth Scopes

| Scope | Description |
|-------|-------------|
| `gmail.GmailSendScope` | Send emails only (default) |
| `gmail.GmailReadonlyScope` | Read emails (for future Watch API) |
| `gmail.GmailModifyScope` | Read and modify emails |

## Message Formats

The provider supports multiple message formats:

```go
// Plain text
router.Send(ctx, "gmail", to, provider.OutgoingMessage{
    Content: "Plain text message",
    Format:  provider.MessageFormatPlain,
})

// HTML
router.Send(ctx, "gmail", to, provider.OutgoingMessage{
    Content: "<h1>HTML Message</h1><p>With formatting!</p>",
    Format:  provider.MessageFormatHTML,
})

// Markdown (converted to HTML)
router.Send(ctx, "gmail", to, provider.OutgoingMessage{
    Content: "# Markdown\n\nWith *formatting*!",
    Format:  provider.MessageFormatMarkdown,
})
```

## Setting Email Subject

Use the `subject` metadata key:

```go
router.Send(ctx, "gmail", to, provider.OutgoingMessage{
    Content: "Email body here",
    Metadata: map[string]any{
        "subject": "My Email Subject",
    },
})
```

If no subject is provided, the first line of content is used.

## Feature Support

| Feature | Supported |
|---------|-----------|
| Send Email | Yes |
| HTML Format | Yes |
| Plain Text | Yes |
| Markdown | Yes (basic) |
| Custom Subject | Yes |
| Reply-To | Yes |
| Attachments | Planned |
| Receive Email | Planned (Watch API) |

## Dependencies

This provider uses:

- [goauth](https://github.com/grokify/goauth) - OAuth2 authentication
- [gogoogle](https://github.com/grokify/gogoogle) - Gmail API utilities
