// Package omnivoice provides a client for the Gemini Live API.
//
// The Gemini Live API enables real-time, multimodal conversational experiences
// with Gemini models. It supports bidirectional streaming of audio and text
// with built-in voice activity detection and interruption handling.
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
//	client, err := omnivoice.NewLiveClient(apiKey,
//	    omnivoice.WithModel("gemini-2.0-flash-live"),
//	    omnivoice.WithVoice("Puck"),
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
//	    case *omnivoice.AudioData:
//	        // Handle audio output
//	    case *omnivoice.TextData:
//	        // Handle text
//	    }
//	}
//
// # References
//
//   - Gemini Live API: https://ai.google.dev/gemini-api/docs/live
//   - API Reference: https://ai.google.dev/api/multimodal-live
package omnivoice
