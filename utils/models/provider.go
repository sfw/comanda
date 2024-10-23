package models

// ModelConfig represents configuration options for model calls
type ModelConfig struct {
	Temperature float64
	MaxTokens   int
	TopP        float64
}

// Provider represents a model provider (e.g., Anthropic, OpenAI)
type Provider interface {
	Name() string
	SupportsModel(modelName string) bool
	SendPrompt(modelName string, prompt string) (string, error)
	Configure(apiKey string) error
	SetVerbose(verbose bool)
}

// DetectProvider determines the appropriate provider based on the model name
func DetectProvider(modelName string) Provider {
	providers := []Provider{
		NewAnthropicProvider(),
		NewOpenAIProvider(),
		// Add other providers here as needed
	}

	for _, provider := range providers {
		if provider.SupportsModel(modelName) {
			return provider
		}
	}
	return nil
}
