# OmniLLM Provider for Google Gemini

[OmniLLM](https://github.com/plexusone/omnillm-core) Provider for Google Gemini enables using Gemini models with OmniLLM.

## Installation

```bash
go get github.com/plexusone/omni-google
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/plexusone/omnillm-core"
    "github.com/plexusone/omni-google/omnillm"
    "github.com/plexusone/omnillm-core/provider"
)

func main() {
    ctx := context.Background()

    // Create the Gemini provider
    geminiProvider := gemini.NewProvider(os.Getenv("GEMINI_API_KEY"))

    // Use it with omnillm via CustomProvider
    client, err := omnillm.NewClient(omnillm.ClientConfig{
        CustomProvider: geminiProvider,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Make requests as usual
    resp, err := client.CreateChatCompletion(ctx, &provider.ChatCompletionRequest{
        Model: "gemini-2.0-flash",
        Messages: []provider.Message{
            {Role: provider.RoleUser, Content: "Hello!"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

## Using the Registry

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

## Authentication

Set your Gemini API key:

```bash
export GEMINI_API_KEY="your-api-key"
```

Get an API key from [Google AI Studio](https://aistudio.google.com/app/apikey).

## Streaming

Stream responses token-by-token:

```go
stream, err := provider.CreateChatCompletionStream(ctx, &provider.ChatCompletionRequest{
    Model: "gemini-2.0-flash",
    Messages: []provider.Message{
        {Role: provider.RoleUser, Content: "Tell me a story"},
    },
})
defer stream.Close()

for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    fmt.Print(chunk.Choices[0].Delta.Content)
}
```

## Feature Support

| Feature | Supported |
|---------|-----------|
| Chat Completion | Yes |
| Streaming | Yes |
| System Messages | Yes |
| Multi-turn Conversations | Yes |
| Tool Calling | Planned |
| JSON Mode | Planned |

## Available Models

| Model | Description |
|-------|-------------|
| `gemini-2.0-flash` | Fast, efficient model for most tasks |
| `gemini-2.0-flash-lite` | Fastest model, lower cost |
| `gemini-1.5-pro` | Most capable model |
| `gemini-1.5-flash` | Previous generation fast model |

See [Google AI Models](https://ai.google.dev/models) for the complete list.
