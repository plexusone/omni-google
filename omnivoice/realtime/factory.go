package realtime

import (
	"github.com/plexusone/omnivoice-core/gateway"
	corereal "github.com/plexusone/omnivoice-core/realtime"
)

// Factory creates Gemini Live realtime providers from gateway configuration.
// It implements [gateway.RealtimeProviderFactory].
type Factory struct{}

// NewFactory creates a new Gemini realtime provider factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Ensure Factory implements gateway.RealtimeProviderFactory.
var _ gateway.RealtimeProviderFactory = (*Factory)(nil)

// Create creates a Gemini RealtimeProvider from the given configuration.
func (f *Factory) Create(config *gateway.RealtimeConfig) (corereal.Provider, error) {
	if config.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}

	opts := []Option{}

	if config.Model != "" {
		opts = append(opts, WithModel(config.Model))
	}
	if config.Voice != "" {
		opts = append(opts, WithVoice(config.Voice))
	}
	if config.Instructions != "" {
		opts = append(opts, WithInstructions(config.Instructions))
	}
	if config.Temperature > 0 {
		opts = append(opts, WithTemperature(config.Temperature))
	}

	// Convert gateway functions to local function declarations
	if len(config.Functions) > 0 {
		funcs := make([]FunctionDeclaration, len(config.Functions))
		for i, fn := range config.Functions {
			funcs[i] = FunctionDeclaration{
				Name:        fn.Name,
				Description: fn.Description,
				Parameters:  fn.Parameters,
			}
		}
		opts = append(opts, WithFunctions(funcs...))
	}

	return NewRealtimeProvider(config.APIKey, opts...), nil
}

// Name returns the provider name.
func (f *Factory) Name() string {
	return "gemini"
}

// ProviderName is the name used to identify Gemini realtime provider.
const ProviderName = "gemini"
