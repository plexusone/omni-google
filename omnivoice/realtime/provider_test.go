package realtime

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNewLiveClient(t *testing.T) {
	client, err := NewLiveClient("test-api-key",
		WithModel("gemini-2.0-flash-live"),
		WithVoice(VoicePuck),
		WithInstructions("You are a helpful assistant."),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.config.APIKey != "test-api-key" {
		t.Errorf("expected API key 'test-api-key', got %q", client.config.APIKey)
	}
	if client.config.Model != "gemini-2.0-flash-live" {
		t.Errorf("expected model 'gemini-2.0-flash-live', got %q", client.config.Model)
	}
	if client.config.Voice != VoicePuck {
		t.Errorf("expected voice 'Puck', got %q", client.config.Voice)
	}
	if client.config.Instructions != "You are a helpful assistant." {
		t.Error("expected instructions to be set")
	}
}

func TestNewLiveClient_NoAPIKey(t *testing.T) {
	_, err := NewLiveClient("")
	if err == nil {
		t.Error("expected error when API key is empty")
	}
}

func TestNewLiveClientFromEnv(t *testing.T) {
	// Test with no env var
	os.Unsetenv("GOOGLE_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	_, err := NewLiveClientFromEnv()
	if err == nil {
		t.Error("expected error when env vars are not set")
	}

	// Test with GOOGLE_API_KEY
	os.Setenv("GOOGLE_API_KEY", "google-key")
	defer os.Unsetenv("GOOGLE_API_KEY")

	client, err := NewLiveClientFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.config.APIKey != "google-key" {
		t.Errorf("expected API key 'google-key', got %q", client.config.APIKey)
	}

	// Test with GEMINI_API_KEY (fallback)
	os.Unsetenv("GOOGLE_API_KEY")
	os.Setenv("GEMINI_API_KEY", "gemini-key")
	defer os.Unsetenv("GEMINI_API_KEY")

	client, err = NewLiveClientFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.config.APIKey != "gemini-key" {
		t.Errorf("expected API key 'gemini-key', got %q", client.config.APIKey)
	}
}

func TestConfigDefaults(t *testing.T) {
	client, _ := NewLiveClient("test-key")

	if client.config.Model != DefaultModel {
		t.Errorf("expected default model %q, got %q", DefaultModel, client.config.Model)
	}
	if client.config.Voice != DefaultVoice {
		t.Errorf("expected default voice %q, got %q", DefaultVoice, client.config.Voice)
	}
	if client.config.Temperature != DefaultTemperature {
		t.Errorf("expected default temperature %f, got %f", DefaultTemperature, client.config.Temperature)
	}
	if len(client.config.ResponseModalities) != 2 {
		t.Errorf("expected 2 modalities, got %d", len(client.config.ResponseModalities))
	}
}

func TestWithFunctions(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]string{"type": "string"},
		},
	})

	fn := FunctionDeclaration{
		Name:        "search",
		Description: "Search for information",
		Parameters:  params,
	}

	client, _ := NewLiveClient("test-key", WithFunctions(fn))

	if len(client.config.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(client.config.Tools))
	}
	if len(client.config.Tools[0].FunctionDeclarations) != 1 {
		t.Errorf("expected 1 function declaration, got %d", len(client.config.Tools[0].FunctionDeclarations))
	}
	if client.config.Tools[0].FunctionDeclarations[0].Name != "search" {
		t.Errorf("expected function name 'search', got %q", client.config.Tools[0].FunctionDeclarations[0].Name)
	}
}

func TestWithGoogleSearch(t *testing.T) {
	client, _ := NewLiveClient("test-key", WithGoogleSearch())

	if !client.config.EnableGoogleSearch {
		t.Error("expected EnableGoogleSearch to be true")
	}
}

func TestWithCodeExecution(t *testing.T) {
	client, _ := NewLiveClient("test-key", WithCodeExecution())

	if !client.config.EnableCodeExecution {
		t.Error("expected EnableCodeExecution to be true")
	}
}

func TestBuildSetupMessage(t *testing.T) {
	client, _ := NewLiveClient("test-key",
		WithModel("gemini-2.0-flash-live"),
		WithVoice(VoiceCharon),
		WithInstructions("Be helpful"),
		WithTemperature(0.7),
		WithGoogleSearch(),
	)

	msg := client.buildSetupMessage()

	if msg.Setup == nil {
		t.Fatal("expected Setup to be set")
	}
	if msg.Setup.Model != "gemini-2.0-flash-live" {
		t.Errorf("expected model 'gemini-2.0-flash-live', got %q", msg.Setup.Model)
	}
	if msg.Setup.GenerationConfig == nil {
		t.Fatal("expected GenerationConfig to be set")
	}
	if msg.Setup.GenerationConfig.SpeechConfig == nil {
		t.Fatal("expected SpeechConfig to be set")
	}
	if msg.Setup.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName != VoiceCharon {
		t.Errorf("expected voice Charon, got %q",
			msg.Setup.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName)
	}
	if msg.Setup.SystemInstruction == nil {
		t.Fatal("expected SystemInstruction to be set")
	}
	if len(msg.Setup.SystemInstruction.Parts) != 1 {
		t.Errorf("expected 1 part in system instruction, got %d", len(msg.Setup.SystemInstruction.Parts))
	}
	if msg.Setup.SystemInstruction.Parts[0].Text != "Be helpful" {
		t.Errorf("expected instruction text 'Be helpful', got %q", msg.Setup.SystemInstruction.Parts[0].Text)
	}
	if msg.Setup.GenerationConfig.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", msg.Setup.GenerationConfig.Temperature)
	}

	// Check tools include Google Search
	hasGoogleSearch := false
	for _, tool := range msg.Setup.Tools {
		if tool.GoogleSearch != nil {
			hasGoogleSearch = true
			break
		}
	}
	if !hasGoogleSearch {
		t.Error("expected Google Search tool to be included")
	}
}

func TestParseServerMessage(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantError bool
	}{
		{
			name:     "setupComplete",
			input:    `{"setupComplete":{}}`,
			wantType: TypeSetupComplete,
		},
		{
			name:     "serverContent with text",
			input:    `{"serverContent":{"modelTurn":{"parts":[{"text":"Hello"}]}}}`,
			wantType: TypeServerContent,
		},
		{
			name:     "toolCall",
			input:    `{"toolCall":{"functionCalls":[{"id":"fc1","name":"search","args":{"q":"test"}}]}}`,
			wantType: TypeToolCall,
		},
		{
			name:     "toolCallCancellation",
			input:    `{"toolCallCancellation":{"ids":["fc1"]}}`,
			wantType: TypeToolCallCancel,
		},
		{
			name:      "invalid json",
			input:     `{invalid}`,
			wantError: true,
		},
		{
			name:     "empty message",
			input:    `{}`,
			wantType: "", // nil event
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := parseServerMessage([]byte(tt.input))
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantType == "" {
				if event != nil {
					t.Errorf("expected nil event, got %T", event)
				}
				return
			}

			if event.GetType() != tt.wantType {
				t.Errorf("expected type %q, got %q", tt.wantType, event.GetType())
			}
		})
	}
}

func TestServerContentParsing(t *testing.T) {
	input := `{
		"serverContent": {
			"modelTurn": {
				"parts": [
					{"text": "Hello, how can I help?"},
					{"inlineData": {"mimeType": "audio/pcm;rate=24000", "data": "SGVsbG8="}}
				]
			},
			"turnComplete": true
		}
	}`

	event, err := parseServerMessage([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, ok := event.(*ServerContent)
	if !ok {
		t.Fatalf("expected *ServerContent, got %T", event)
	}

	if content.ModelTurn == nil {
		t.Fatal("expected ModelTurn to be set")
	}
	if len(content.ModelTurn.Parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(content.ModelTurn.Parts))
	}
	if content.ModelTurn.Parts[0].Text != "Hello, how can I help?" {
		t.Errorf("expected text 'Hello, how can I help?', got %q", content.ModelTurn.Parts[0].Text)
	}
	if content.ModelTurn.Parts[1].InlineData == nil {
		t.Fatal("expected InlineData in second part")
	}
	if content.ModelTurn.Parts[1].InlineData.MimeType != AudioMimeTypePCM24k {
		t.Errorf("expected mime type %q, got %q", AudioMimeTypePCM24k, content.ModelTurn.Parts[1].InlineData.MimeType)
	}
	if !content.TurnComplete {
		t.Error("expected TurnComplete to be true")
	}
}

func TestToolCallParsing(t *testing.T) {
	input := `{
		"toolCall": {
			"functionCalls": [
				{
					"id": "call_123",
					"name": "get_weather",
					"args": {"location": "San Francisco"}
				}
			]
		}
	}`

	event, err := parseServerMessage([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	toolCall, ok := event.(*ToolCall)
	if !ok {
		t.Fatalf("expected *ToolCall, got %T", event)
	}

	if len(toolCall.FunctionCalls) != 1 {
		t.Errorf("expected 1 function call, got %d", len(toolCall.FunctionCalls))
	}
	if toolCall.FunctionCalls[0].ID != "call_123" {
		t.Errorf("expected id 'call_123', got %q", toolCall.FunctionCalls[0].ID)
	}
	if toolCall.FunctionCalls[0].Name != "get_weather" {
		t.Errorf("expected name 'get_weather', got %q", toolCall.FunctionCalls[0].Name)
	}
}

func TestNewRealtimeProvider(t *testing.T) {
	provider := NewRealtimeProvider("test-key",
		WithVoice(VoiceKore),
		WithInstructions("Test instructions"),
	)

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.Name() != "gemini-live" {
		t.Errorf("expected name 'gemini-live', got %q", provider.Name())
	}
}

// Integration test - only runs when GOOGLE_API_KEY or GEMINI_API_KEY is set
func TestIntegration_Connect(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		t.Skip("GOOGLE_API_KEY or GEMINI_API_KEY not set, skipping integration test")
	}

	// This test would actually connect to Google
	// Skipped by default to avoid API charges
	t.Skip("Integration test skipped to avoid API charges")
}
