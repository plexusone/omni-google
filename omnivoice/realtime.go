package omnivoice

import (
	"context"
	"encoding/base64"
	"sync"
)

// AudioChunk represents a chunk of audio data.
type AudioChunk struct {
	// Audio is the raw audio data (PCM16 24kHz mono).
	Audio []byte

	// IsFinal indicates this is the last chunk for a turn.
	IsFinal bool
}

// Transcript represents a transcript update.
type Transcript struct {
	// Text is the transcript text.
	Text string

	// IsFinal indicates this is a final transcript.
	IsFinal bool

	// IsInput indicates this is input (user) transcription.
	IsInput bool
}

// ProcessConfig configures audio processing.
type ProcessConfig struct {
	// Instructions is the system prompt.
	Instructions string

	// Voice is the voice for output.
	Voice string

	// Functions are functions the model can call.
	Functions []FunctionDeclaration

	// OnFunctionCall is called when the model calls a function.
	// Return the function output as any JSON-serializable value.
	OnFunctionCall func(id, name string, args string) (any, error)
}

// RealtimeProvider provides native voice-to-voice processing.
type RealtimeProvider struct {
	apiKey string
	config Config
}

// NewRealtimeProvider creates a new RealtimeProvider.
func NewRealtimeProvider(apiKey string, opts ...Option) *RealtimeProvider {
	cfg := Config{APIKey: apiKey}
	for _, opt := range opts {
		opt(&cfg)
	}
	applyDefaults(&cfg)

	return &RealtimeProvider{
		apiKey: apiKey,
		config: cfg,
	}
}

// ProcessAudioStream processes audio input and returns audio output.
func (p *RealtimeProvider) ProcessAudioStream(
	ctx context.Context,
	audioIn <-chan []byte,
	config ProcessConfig,
) (<-chan AudioChunk, <-chan Transcript, error) {
	// Apply config overrides
	opts := []Option{}
	if config.Instructions != "" {
		opts = append(opts, WithInstructions(config.Instructions))
	}
	if config.Voice != "" {
		opts = append(opts, WithVoice(config.Voice))
	}
	if len(config.Functions) > 0 {
		opts = append(opts, WithFunctions(config.Functions...))
	}

	// Create client with overrides
	client, err := NewLiveClient(p.apiKey, opts...)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.Connect(ctx)
	if err != nil {
		return nil, nil, err
	}

	audioCh := make(chan AudioChunk, 100)
	transcriptCh := make(chan Transcript, 100)

	var wg sync.WaitGroup
	wg.Add(2)

	// Send audio to session
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-audioIn:
				if !ok {
					return
				}
				_ = session.SendAudio(ctx, audio)
			}
		}
	}()

	// Process events from session
	go func() {
		defer wg.Done()
		defer close(audioCh)
		defer close(transcriptCh)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-session.Events():
				if !ok {
					return
				}

				switch e := event.(type) {
				case *ServerContent:
					p.handleServerContent(ctx, e, audioCh, transcriptCh)

				case *ToolCall:
					if config.OnFunctionCall != nil {
						for _, fc := range e.FunctionCalls {
							result, err := config.OnFunctionCall(fc.ID, fc.Name, string(fc.Args))
							if err != nil {
								_ = session.SendFunctionResponse(fc.ID, fc.Name, map[string]string{"error": err.Error()})
							} else {
								_ = session.SendFunctionResponse(fc.ID, fc.Name, result)
							}
						}
					}
				}
			}
		}
	}()

	// Close session when done
	go func() {
		wg.Wait()
		session.Close()
	}()

	return audioCh, transcriptCh, nil
}

// handleServerContent processes server content events.
func (p *RealtimeProvider) handleServerContent(
	ctx context.Context,
	content *ServerContent,
	audioCh chan<- AudioChunk,
	transcriptCh chan<- Transcript,
) {
	if content.ModelTurn == nil {
		return
	}

	for _, part := range content.ModelTurn.Parts {
		// Handle audio
		if part.InlineData != nil && part.InlineData.MimeType == AudioMimeTypePCM24k {
			audio, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
			if err == nil && len(audio) > 0 {
				select {
				case audioCh <- AudioChunk{Audio: audio}:
				case <-ctx.Done():
					return
				}
			}
		}

		// Handle text
		if part.Text != "" {
			select {
			case transcriptCh <- Transcript{Text: part.Text, IsInput: false}:
			case <-ctx.Done():
				return
			}
		}
	}

	// Send final marker if turn is complete
	if content.TurnComplete {
		select {
		case audioCh <- AudioChunk{IsFinal: true}:
		case <-ctx.Done():
			return
		}
		select {
		case transcriptCh <- Transcript{IsFinal: true}:
		case <-ctx.Done():
			return
		}
	}
}

// Name returns the provider name.
func (p *RealtimeProvider) Name() string {
	return "gemini-live"
}
