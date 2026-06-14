package realtime

import (
	"context"
	"encoding/base64"
	"sync"

	corereal "github.com/plexusone/omnivoice-core/realtime"
)

// RealtimeProvider provides native voice-to-voice processing via Gemini Live API.
// It implements the [corereal.Provider] interface.
type RealtimeProvider struct {
	apiKey string
	config Config
}

// Ensure RealtimeProvider implements corereal.Provider.
var _ corereal.Provider = (*RealtimeProvider)(nil)

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
// Implements [corereal.Provider].
func (p *RealtimeProvider) ProcessAudioStream(
	ctx context.Context,
	audioIn <-chan []byte,
	config corereal.ProcessConfig,
) (<-chan corereal.AudioChunk, <-chan corereal.Transcript, error) {
	// Apply config overrides
	opts := []Option{}
	if config.Instructions != "" {
		opts = append(opts, WithInstructions(config.Instructions))
	}
	if config.Voice != "" {
		opts = append(opts, WithVoice(config.Voice))
	}
	if len(config.Functions) > 0 {
		// Convert corereal.FunctionDeclaration to local FunctionDeclaration
		funcs := make([]FunctionDeclaration, len(config.Functions))
		for i, f := range config.Functions {
			funcs[i] = FunctionDeclaration{
				Name:        f.Name,
				Description: f.Description,
				Parameters:  f.Parameters,
			}
		}
		opts = append(opts, WithFunctions(funcs...))
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

	audioCh := make(chan corereal.AudioChunk, 100)
	transcriptCh := make(chan corereal.Transcript, 100)

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
	audioCh chan<- corereal.AudioChunk,
	transcriptCh chan<- corereal.Transcript,
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
				case audioCh <- corereal.AudioChunk{Audio: audio}:
				case <-ctx.Done():
					return
				}
			}
		}

		// Handle text
		if part.Text != "" {
			select {
			case transcriptCh <- corereal.Transcript{Text: part.Text, IsInput: false}:
			case <-ctx.Done():
				return
			}
		}
	}

	// Send final marker if turn is complete
	if content.TurnComplete {
		select {
		case audioCh <- corereal.AudioChunk{IsFinal: true}:
		case <-ctx.Done():
			return
		}
		select {
		case transcriptCh <- corereal.Transcript{IsFinal: true}:
		case <-ctx.Done():
			return
		}
	}
}

// Name returns the provider name.
// Implements [corereal.Provider].
func (p *RealtimeProvider) Name() string {
	return "gemini-live"
}

// Close releases any resources held by the provider.
// Implements [corereal.Provider].
func (p *RealtimeProvider) Close() error {
	// No persistent resources to clean up
	return nil
}
