package realtime

import (
	"fmt"

	omnivoice "github.com/plexusone/omnivoice-core"
	"github.com/plexusone/omnivoice-core/registry"
)

func init() {
	omnivoice.RegisterRealtimeProvider("gemini", NewRealtimeProviderFromConfig, omnivoice.PriorityThick)
}

// NewRealtimeProviderFromConfig creates a Gemini realtime provider from registry config.
func NewRealtimeProviderFromConfig(cfg registry.ProviderConfig) (registry.RealtimeProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini realtime: apiKey is required")
	}

	opts := []Option{}

	if v := getExtString(cfg.Extensions, "model"); v != "" {
		opts = append(opts, WithModel(v))
	}
	if v := getExtString(cfg.Extensions, "voice"); v != "" {
		opts = append(opts, WithVoice(v))
	}
	if v := getExtString(cfg.Extensions, "instructions"); v != "" {
		opts = append(opts, WithInstructions(v))
	}

	// Type-safe extensions
	if v, ok := cfg.Extensions["tools"].([]ToolDef); ok {
		opts = append(opts, WithTools(v...))
	}
	if v, ok := cfg.Extensions["functions"].([]FunctionDeclaration); ok {
		opts = append(opts, WithFunctions(v...))
	}
	if v, ok := cfg.Extensions["responseModalities"].([]string); ok {
		opts = append(opts, WithResponseModalities(v...))
	}
	if v, ok := cfg.Extensions["temperature"].(float64); ok {
		opts = append(opts, WithTemperature(v))
	}
	if v, ok := cfg.Extensions["topP"].(float64); ok {
		opts = append(opts, WithTopP(v))
	}
	if v, ok := cfg.Extensions["topK"].(int); ok {
		opts = append(opts, WithTopK(v))
	}
	if v, ok := cfg.Extensions["maxOutputTokens"].(int); ok {
		opts = append(opts, WithMaxOutputTokens(v))
	}
	if v, ok := cfg.Extensions["enableGoogleSearch"].(bool); ok && v {
		opts = append(opts, WithGoogleSearch())
	}
	if v, ok := cfg.Extensions["enableCodeExecution"].(bool); ok && v {
		opts = append(opts, WithCodeExecution())
	}

	provider := NewRealtimeProvider(cfg.APIKey, opts...)
	return &realtimeWrapper{provider}, nil
}

// realtimeWrapper wraps RealtimeProvider to implement registry.RealtimeProvider.
type realtimeWrapper struct {
	p *RealtimeProvider
}

func (w *realtimeWrapper) Name() string {
	return ProviderName
}

func (w *realtimeWrapper) Close() error {
	// RealtimeProvider doesn't have a Close method, but individual sessions do.
	// This is a no-op at the provider level.
	return nil
}

// Provider returns the underlying RealtimeProvider for full API access.
func (w *realtimeWrapper) Provider() *RealtimeProvider {
	return w.p
}

func getExtString(ext map[string]any, key string) string {
	if ext == nil {
		return ""
	}
	if v, ok := ext[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
