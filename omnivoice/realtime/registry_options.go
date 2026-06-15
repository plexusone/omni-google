package realtime

import (
	"github.com/plexusone/omnivoice-core/registry"
)

// Provider-specific option functions for type-safe configuration via the registry.
// These functions return registry.ProviderOption and can be used with
// omnivoice.GetRealtimeProvider("gemini", opts...).
//
// Note: These are named with "Registry" prefix to avoid conflicts with the
// provider-internal Option functions (which configure the local Config struct).

// WithRegistryTools sets the tools available to the Gemini model via registry.
func WithRegistryTools(tools []ToolDef) registry.ProviderOption {
	return registry.WithExtension("tools", tools)
}

// WithRegistryFunctions sets the function declarations via registry.
func WithRegistryFunctions(functions []FunctionDeclaration) registry.ProviderOption {
	return registry.WithExtension("functions", functions)
}

// WithRegistryResponseModalities sets the response modalities (e.g., ["AUDIO", "TEXT"]) via registry.
func WithRegistryResponseModalities(modalities []string) registry.ProviderOption {
	return registry.WithExtension("responseModalities", modalities)
}

// WithRegistryTemperature sets the temperature for response generation via registry.
func WithRegistryTemperature(temp float64) registry.ProviderOption {
	return registry.WithExtension("temperature", temp)
}

// WithRegistryTopP sets the top-p value via registry.
func WithRegistryTopP(topP float64) registry.ProviderOption {
	return registry.WithExtension("topP", topP)
}

// WithRegistryTopK sets the top-k value via registry.
func WithRegistryTopK(topK int) registry.ProviderOption {
	return registry.WithExtension("topK", topK)
}

// WithRegistryMaxOutputTokens sets the maximum output tokens via registry.
func WithRegistryMaxOutputTokens(max int) registry.ProviderOption {
	return registry.WithExtension("maxOutputTokens", max)
}

// WithRegistryGoogleSearch enables grounding with Google Search via registry.
func WithRegistryGoogleSearch() registry.ProviderOption {
	return registry.WithExtension("enableGoogleSearch", true)
}

// WithRegistryCodeExecution enables code execution via registry.
func WithRegistryCodeExecution() registry.ProviderOption {
	return registry.WithExtension("enableCodeExecution", true)
}
