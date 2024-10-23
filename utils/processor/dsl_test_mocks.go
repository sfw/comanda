package processor

import (
	"comanda/utils/config"
	"fmt"
)

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
	switch m.name {
	case "OpenAI":
		return modelName == "gpt-4" || modelName == "gpt-3.5-turbo"
	case "Anthropic":
		return modelName == "claude-2" || modelName == "claude-instant"
	default:
		return false
	}
}

func (m *MockProvider) Configure(apiKey string) error {
	m.configured = true
	m.apiKey = apiKey
	return nil
}

func (m *MockProvider) SendPrompt(model, prompt string) (string, error) {
	if !m.configured {
		return "", fmt.Errorf("provider not configured")
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
					{Name: "gpt-4", Type: "chat"},
					{Name: "gpt-3.5-turbo", Type: "chat"},
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
