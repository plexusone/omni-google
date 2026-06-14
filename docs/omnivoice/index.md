# OmniVoice - Gemini Live API

The `omnivoice` package provides real-time voice-to-voice capabilities via the Gemini Live API.

## Overview

Gemini Live API enables:

- **Real-time voice-to-voice** - ~200ms response latency
- **Native audio processing** - Model handles audio directly
- **Function calling** - Execute tools during conversation
- **Multimodal input** - Audio + text + video simultaneously
- **Google Search grounding** - Ground responses in search results
- **Code execution** - Run Python code during conversation

## Quick Start

```go
import (
    "context"
    "os"

    "github.com/plexusone/omni-google/omnivoice"
)

func main() {
    ctx := context.Background()

    // Create client
    client, err := omnivoice.NewLiveClient(os.Getenv("GOOGLE_API_KEY"),
        omnivoice.WithVoice("Puck"),
        omnivoice.WithInstructions("You are a helpful assistant."),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Connect to session
    session, err := client.Connect(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer session.Close()

    // Send audio (PCM16 16kHz mono)
    go func() {
        for chunk := range microphoneAudio {
            session.SendAudio(ctx, chunk)
        }
    }()

    // Receive events
    for event := range session.Events() {
        switch e := event.(type) {
        case *omnivoice.ServerContent:
            for _, part := range e.ModelTurn.Parts {
                if part.InlineData != nil {
                    // Decode and play audio (PCM16 24kHz)
                    audio, _ := base64.StdEncoding.DecodeString(part.InlineData.Data)
                    playAudio(audio)
                }
                if part.Text != "" {
                    log.Printf("Assistant: %s", part.Text)
                }
            }
        case *omnivoice.ToolCall:
            // Handle function calls
            handleToolCall(session, e)
        }
    }
}
```

## Using RealtimeProvider

For a higher-level interface compatible with OmniVoice patterns:

```go
import "github.com/plexusone/omni-google/omnivoice"

// Create provider
provider := omnivoice.NewRealtimeProvider(os.Getenv("GOOGLE_API_KEY"),
    omnivoice.WithVoice("Puck"),
    omnivoice.WithInstructions("You are a helpful assistant."),
)

// Stream audio
audioIn := make(chan []byte, 100)
audioCh, transcriptCh, err := provider.ProcessAudioStream(ctx, audioIn, omnivoice.ProcessConfig{
    Functions: []omnivoice.FunctionDeclaration{
        {
            Name:        "get_weather",
            Description: "Get current weather",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "location": map[string]any{"type": "string"},
                },
            },
        },
    },
    OnFunctionCall: func(id, name, args string) (any, error) {
        return map[string]any{"temp": 72, "condition": "sunny"}, nil
    },
})
```

## Configuration

### Client Options

```go
client, err := omnivoice.NewLiveClient(apiKey,
    // Voice selection
    omnivoice.WithVoice("Puck"),

    // System instructions
    omnivoice.WithInstructions("You are a customer service agent."),

    // Model selection (default: gemini-2.0-flash-live)
    omnivoice.WithModel("gemini-2.0-flash-live"),

    // Response modalities
    omnivoice.WithResponseModalities("TEXT", "AUDIO"),

    // Temperature
    omnivoice.WithTemperature(0.7),

    // Enable Google Search
    omnivoice.WithGoogleSearch(true),

    // Enable code execution
    omnivoice.WithCodeExecution(true),

    // Custom functions
    omnivoice.WithFunctions(
        omnivoice.FunctionDeclaration{
            Name:        "lookup_order",
            Description: "Look up an order by ID",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "order_id": map[string]any{"type": "string"},
                },
            },
        },
    ),
)
```

### Available Voices

| Voice | Description |
|-------|-------------|
| Puck | Upbeat, lively |
| Charon | Informative, direct |
| Kore | Firm, authoritative |
| Fenrir | Enthusiastic, positive |
| Aoede | Bright, clear |

## Audio Format

### Input Audio

- **Format**: PCM16 (signed 16-bit little-endian)
- **Sample Rate**: 16kHz
- **Channels**: Mono
- **MIME Type**: `audio/pcm;rate=16000`

### Output Audio

- **Format**: PCM16 (signed 16-bit little-endian)
- **Sample Rate**: 24kHz
- **Channels**: Mono
- **MIME Type**: `audio/pcm;rate=24000`

## Function Calling

Handle function calls during the conversation:

```go
for event := range session.Events() {
    switch e := event.(type) {
    case *omnivoice.ToolCall:
        for _, fc := range e.FunctionCalls {
            // Process the function call
            var result any
            switch fc.Name {
            case "get_weather":
                var args struct {
                    Location string `json:"location"`
                }
                json.Unmarshal(fc.Args, &args)
                result = getWeather(args.Location)
            }

            // Send response back
            session.SendFunctionResponse(fc.ID, fc.Name, result)
        }
    }
}
```

## Message Types

### Client Messages

| Message | Description |
|---------|-------------|
| `setup` | Initialize session with config |
| `realtimeInput` | Send audio/video chunks |
| `clientContent` | Send text content |
| `toolResponse` | Send function call response |

### Server Messages

| Message | Description |
|---------|-------------|
| `setupComplete` | Session initialized |
| `serverContent` | Audio/text response |
| `toolCall` | Function call request |
| `interrupted` | Turn was interrupted |

## Interruptions

Handle user interruptions (barge-in):

```go
// Send interrupt signal
session.Interrupt()

// Handle interruption events
for event := range session.Events() {
    switch e := event.(type) {
    case *omnivoice.ServerContent:
        if e.Interrupted {
            log.Println("Turn was interrupted")
        }
    }
}
```

## Integration with Call Systems

### Twilio Media Streams

```go
// Twilio sends mulaw 8kHz
twilioAudio := make(chan []byte)

// Convert to PCM16 16kHz for Gemini
audioIn := make(chan []byte)
go func() {
    for chunk := range twilioAudio {
        pcm := convertMulawToPCM16(chunk)
        upsampled := resample8kTo16k(pcm)
        audioIn <- upsampled
    }
}()

// Gemini output is 24kHz
audioCh, _, _ := provider.ProcessAudioStream(ctx, audioIn, config)
for audio := range audioCh {
    // Convert to Twilio format
    downsampled := resample24kTo8k(audio.Audio)
    mulaw := convertPCM16ToMulaw(downsampled)
    sendToTwilio(mulaw)
}
```

## Comparison with OpenAI Realtime

| Feature | Gemini Live | OpenAI Realtime |
|---------|-------------|-----------------|
| Latency | ~200ms | ~100ms |
| Input audio | 16kHz | 24kHz |
| Output audio | 24kHz | 24kHz |
| Google Search | Yes | No |
| Code execution | Yes | No |
| Video input | Yes | No |
| Voices | 5 | 11 |

## Environment Variables

```bash
# Option 1: Google API key
export GOOGLE_API_KEY="your-api-key"

# Option 2: Gemini API key
export GEMINI_API_KEY="your-api-key"
```

## vs Traditional Pipeline (STT+LLM+TTS)

Gemini Live provides native voice-to-voice, eliminating the need for separate STT and TTS providers.

| Aspect | Traditional | Gemini Live |
|--------|-------------|-------------|
| **Latency** | 500-1500ms | ~200ms |
| **API Calls** | 3 (STT + LLM + TTS) | 1 WebSocket |
| **Barge-in** | Complex coordination | Native support |
| **Voice options** | 1000s (ElevenLabs, etc.) | 5 preset voices |
| **Google Search** | No | Yes (grounding) |
| **Code execution** | No | Yes |
| **Video input** | No | Yes |

### When to Use Gemini Live

- Low latency is critical
- Need Google Search grounding for factual responses
- Need code execution during conversation
- Need video/multimodal input
- Simpler architecture preferred

### When to Use Traditional Pipeline

- Custom/cloned voices required
- Domain-specific STT accuracy needed
- Specific language support
- Lowest possible latency (~100ms with OpenAI Realtime)

See [Voice Architecture Guide](https://plexusone.dev/omnivoice-core/voice-architecture) for detailed comparison.

## Best Practices

1. **Buffer audio input** - Use buffered channel (100+ capacity)
2. **Handle disconnects** - Implement reconnection logic
3. **Use appropriate sample rate** - 16kHz input, 24kHz output
4. **Enable Google Search** - For factual queries
5. **Test with real audio** - Synthetic tests miss edge cases
