package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

const (
	// GeminiLiveEndpoint is the WebSocket endpoint for Gemini Live API.
	GeminiLiveEndpoint = "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"
)

// LiveClient is the Gemini Live API client.
type LiveClient struct {
	config Config
}

// NewLiveClient creates a new Gemini Live API client.
func NewLiveClient(apiKey string, opts ...Option) (*LiveClient, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	c := &LiveClient{
		config: Config{
			APIKey: apiKey,
		},
	}

	for _, opt := range opts {
		opt(&c.config)
	}

	applyDefaults(&c.config)

	return c, nil
}

// NewLiveClientFromEnv creates a client using the GOOGLE_API_KEY or GEMINI_API_KEY environment variable.
func NewLiveClientFromEnv(opts ...Option) (*LiveClient, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("GOOGLE_API_KEY or GEMINI_API_KEY environment variable not set")
	}
	return NewLiveClient(apiKey, opts...)
}

// Connect establishes a WebSocket connection and returns a LiveSession.
func (c *LiveClient) Connect(ctx context.Context) (*LiveSession, error) {
	url := fmt.Sprintf("%s?key=%s", GeminiLiveEndpoint, c.config.APIKey)

	header := http.Header{}
	header.Set("Content-Type", "application/json")

	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.ConnectTimeout,
	}

	conn, resp, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket dial failed with status %d: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	session := &LiveSession{
		conn:     conn,
		config:   c.config,
		eventsCh: make(chan ServerMessage, 100),
		sendCh:   make(chan any, 100),
		closeCh:  make(chan struct{}),
	}

	// Send setup message
	setupMsg := c.buildSetupMessage()
	data, err := json.Marshal(setupMsg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to marshal setup message: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send setup message: %w", err)
	}

	// Start read/write goroutines
	session.wg.Add(2)
	go session.readLoop()
	go session.writeLoop()

	// Wait for setup complete
	select {
	case event := <-session.eventsCh:
		if _, ok := event.(*SetupComplete); !ok {
			session.Close()
			return nil, fmt.Errorf("expected SetupComplete, got %T", event)
		}
	case <-ctx.Done():
		session.Close()
		return nil, ctx.Err()
	}

	return session, nil
}

// buildSetupMessage creates the setup message from config.
func (c *LiveClient) buildSetupMessage() SetupMessage {
	setup := SetupConfig{
		Model: c.config.Model,
		GenerationConfig: &GenerationConfig{
			ResponseModalities: c.config.ResponseModalities,
			Temperature:        c.config.Temperature,
		},
	}

	// Add speech config if voice is set
	if c.config.Voice != "" {
		setup.GenerationConfig.SpeechConfig = &SpeechConfig{
			VoiceConfig: &VoiceConfig{
				PrebuiltVoiceConfig: &PrebuiltVoiceConfig{
					VoiceName: c.config.Voice,
				},
			},
		}
	}

	// Add optional parameters
	if c.config.TopP > 0 {
		setup.GenerationConfig.TopP = c.config.TopP
	}
	if c.config.TopK > 0 {
		setup.GenerationConfig.TopK = c.config.TopK
	}
	if c.config.MaxOutputTokens > 0 {
		setup.GenerationConfig.MaxOutputTokens = c.config.MaxOutputTokens
	}

	// Add system instruction
	if c.config.Instructions != "" {
		setup.SystemInstruction = &Content{
			Parts: []Part{{Text: c.config.Instructions}},
		}
	}

	// Add tools
	setup.Tools = c.config.Tools

	// Add Google Search if enabled
	if c.config.EnableGoogleSearch {
		setup.Tools = append(setup.Tools, ToolDef{GoogleSearch: &GoogleSearchConfig{}})
	}

	// Add code execution if enabled
	if c.config.EnableCodeExecution {
		setup.Tools = append(setup.Tools, ToolDef{CodeExecution: &CodeExecutionConfig{}})
	}

	return SetupMessage{Setup: &setup}
}

// LiveSession represents an active Gemini Live session.
type LiveSession struct {
	conn   *websocket.Conn
	config Config

	eventsCh chan ServerMessage
	sendCh   chan any
	closeCh  chan struct{}
	closed   bool
	closeMu  sync.Mutex
	wg       sync.WaitGroup
}

// Events returns a channel of server events.
func (s *LiveSession) Events() <-chan ServerMessage {
	return s.eventsCh
}

// SendAudio sends audio data to the session.
// Audio should be PCM16 16kHz mono.
func (s *LiveSession) SendAudio(ctx context.Context, audio []byte) error {
	encoded := base64.StdEncoding.EncodeToString(audio)
	return s.send(RealtimeInput{
		RealtimeInput: &RealtimeInputData{
			MediaChunks: []MediaChunk{
				{
					MimeType: AudioMimeTypePCM16k,
					Data:     encoded,
				},
			},
		},
	})
}

// SendText sends text content to the session.
func (s *LiveSession) SendText(ctx context.Context, text string) error {
	return s.send(ClientContent{
		ClientContent: &ClientContentData{
			Turns: []Content{
				{
					Parts: []Part{{Text: text}},
					Role:  "user",
				},
			},
			TurnComplete: true,
		},
	})
}

// SendFunctionResponse sends a function call response.
func (s *LiveSession) SendFunctionResponse(id, name string, response any) error {
	respData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	return s.send(ToolResponse{
		ToolResponse: &ToolResponseData{
			FunctionResponses: []FunctionResponse{
				{
					ID:       id,
					Name:     name,
					Response: respData,
				},
			},
		},
	})
}

// Interrupt sends an interrupt signal.
// This can be used to implement barge-in functionality.
func (s *LiveSession) Interrupt() error {
	// Send empty realtime input to interrupt
	return s.send(RealtimeInput{
		RealtimeInput: &RealtimeInputData{
			MediaChunks: []MediaChunk{},
		},
	})
}

// Close closes the session.
func (s *LiveSession) Close() error {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return nil
	}
	s.closed = true
	close(s.closeCh)
	s.closeMu.Unlock()

	err := s.conn.Close()
	s.wg.Wait()
	close(s.eventsCh)

	return err
}

// send sends a message to the server.
func (s *LiveSession) send(msg any) error {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return errors.New("session closed")
	}
	s.closeMu.Unlock()

	select {
	case s.sendCh <- msg:
		return nil
	default:
		return errors.New("send channel full")
	}
}

// readLoop reads messages from the WebSocket.
func (s *LiveSession) readLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closeCh:
			return
		default:
		}

		_, data, err := s.conn.ReadMessage()
		if err != nil {
			// Error reading from connection - exit loop
			return
		}

		event, err := parseServerMessage(data)
		if err != nil || event == nil {
			continue // Skip unparseable events
		}

		select {
		case s.eventsCh <- event:
		case <-s.closeCh:
			return
		default:
			// Drop event if channel is full
		}
	}
}

// writeLoop writes messages to the WebSocket.
func (s *LiveSession) writeLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closeCh:
			return
		case msg := <-s.sendCh:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}
