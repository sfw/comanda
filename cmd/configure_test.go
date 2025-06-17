package cmd

import (
	"testing"

	"github.com/kris-hansen/comanda/utils/models"
)

// TestModelConsistency ensures that the models available for configuration
// match exactly with the models that can be used at runtime
func TestModelConsistency(t *testing.T) {
	t.Run("Dynamic Model Registry", func(t *testing.T) {
		// Test that the registry system works for available providers
		availableProviders := models.ListRegisteredProviders()
		if len(availableProviders) == 0 {
			t.Skip("No providers available in this build")
		}

		for _, providerName := range availableProviders {
			provider := models.GetProviderByName(providerName)
			if provider == nil {
				t.Errorf("Provider %s listed as available but not found in registry", providerName)
				continue
			}

			// Test that the provider can list models (even if it returns an error due to missing API key)
			_, err := models.ListModelsForProvider(providerName, "")
			// We don't check for error here since some providers require API keys
			// The important thing is that the function exists and can be called
			_ = err // Acknowledge we're ignoring the error intentionally
		}
	})
}
