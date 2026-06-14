// Package realtime provides a client for the Gemini Live API.
//
// The Gemini Live API enables real-time, multimodal conversational experiences
// with Gemini models. It supports bidirectional streaming of audio and text
// with built-in voice activity detection and interruption handling.
//
// This package implements the [github.com/plexusone/omnivoice-core/realtime.Provider]
// interface for unified voice-to-voice processing.
//
// # Features
//
//   - Real-time voice conversations with Gemini models
//   - WebSocket-based bidirectional streaming
//   - Voice activity detection
//   - Function calling support
//   - Multiple voice options
//
// # Audio Format
//
// Input:  PCM16 16kHz mono
// Output: PCM16 24kHz mono
//
// # Usage
//
//	import "github.com/plexusone/omni-google/omnivoice/realtime"
//
//	client, err := realtime.NewLiveClient(apiKey,
//	    realtime.WithModel("gemini-2.0-flash-live"),
//	    realtime.WithVoice("Puck"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	session, err := client.Connect(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer session.Close()
//
//	// Send audio
//	session.SendAudio(ctx, audioData)
//
//	// Receive events
//	for event := range session.Events() {
//	    switch e := event.(type) {
//	    case *realtime.ServerContent:
//	        // Handle audio/text output
//	    case *realtime.ToolCall:
//	        // Handle function calls
//	    }
//	}
//
// # References
//
//   - Gemini Live API: https://ai.google.dev/gemini-api/docs/live
//   - API Reference: https://ai.google.dev/api/multimodal-live
package realtime
