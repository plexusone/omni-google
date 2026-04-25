package gemini

import (
	"os"
	"testing"

	"github.com/plexusone/omnillm-core/provider/providertest"
)

func TestConformance(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")

	// Always create a real provider - the client handles empty API key gracefully
	p := NewProvider(apiKey).(*Provider)

	providertest.RunAll(t, providertest.Config{
		Provider:        p,
		SkipIntegration: apiKey == "",
		TestModel:       "gemini-2.0-flash",
	})
}
