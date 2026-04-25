// Package gemini provides a Google Gemini provider for OmniLLM.
//
// This is a thick provider that uses the official Google GenAI SDK
// (google.golang.org/genai). Import this package to enable Gemini
// support in omnillm-core.
//
// # Usage
//
// Import this package with a blank identifier to register the Gemini provider:
//
//	import (
//	    "github.com/plexusone/omnillm-core"
//	    _ "github.com/plexusone/omnillm-gemini"
//	)
//
//	func main() {
//	    client, _ := omnillm.NewClient(omnillm.ClientConfig{
//	        Providers: []omnillm.ProviderConfig{
//	            {Provider: omnillm.ProviderNameGemini, APIKey: "your-api-key"},
//	        },
//	    })
//	    // Use client...
//	}
//
// # Direct Usage
//
// You can also use the provider directly without the registry:
//
//	provider := gemini.NewProvider("your-api-key")
//	resp, err := provider.CreateChatCompletion(ctx, req)
package gemini
