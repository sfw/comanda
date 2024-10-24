package processor

import (
	"fmt"

	"github.com/kris-hansen/comanda/utils/config"
	"github.com/kris-hansen/comanda/utils/models"
)

var originalDetectProvider models.DetectProviderFunc

func init() {
	// Store the original function
	originalDetectProvider = models.DetectProvider

	// Override with test version
	models.DetectProvider = func(modelName string) models.Provider {
		providers := []models.Provider{
			NewMockProvider("OpenAI"),
			NewMockProvider("Anthropic"),
		}

		for _, provider := range providers {
			if provider.SupportsModel(modelName) {
				return provider
			}
		}
		return nil
	}
}

// Restore the original DetectProvider function
func restoreDetectProvider() {
	models.DetectProvider = originalDetectProvider
}

// MockProvider implements the models.Provider interface for testing
type MockProvider struct {
	name       string
	configured bool
	verbose    bool
	apiKey     string
}

func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		name: name,
	}
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) SupportsModel(modelName string) bool {
	validModels := map[string][]string{
		"OpenAI": {
			"gpt-4o",
			"gpt-4o-mini",
			"o1-preview",
			"o1-mini",
		},
		"Anthropic": {
			"claude-2",
			"claude-instant",
		},
	}

	if models, ok := validModels[m.name]; ok {
		for _, validModel := range models {
			if modelName == validModel {
				return true
			}
		}
	}
	return false
}

func (m *MockProvider) Configure(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	m.configured = true
	m.apiKey = apiKey
	return nil
}

func (m *MockProvider) SendPrompt(model, prompt string) (string, error) {
	if !m.configured {
		return "", fmt.Errorf("provider not configured")
	}
	if !m.SupportsModel(model) {
		return "", fmt.Errorf("unsupported model: %s", model)
	}
	return "mock response", nil
}

func (m *MockProvider) SetVerbose(verbose bool) {
	m.verbose = verbose
}

// Helper function to create a test environment config
func createTestEnvConfig() *config.EnvConfig {
	return &config.EnvConfig{
		Providers: map[string]config.Provider{
			"openai": {
				APIKey: "test-openai-key",
				Models: []config.Model{
					{Name: "gpt-4o", Type: "chat"},
					{Name: "gpt-4o-mini", Type: "chat"},
					{Name: "o1-preview", Type: "chat"},
					{Name: "o1-mini", Type: "chat"},
				},
			},
			"anthropic": {
				APIKey: "test-anthropic-key",
				Models: []config.Model{
					{Name: "claude-2", Type: "chat"},
					{Name: "claude-instant", Type: "chat"},
				},
			},
		},
	}
}
