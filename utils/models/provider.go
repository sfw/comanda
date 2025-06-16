package models

import (
	"strings"

	"github.com/kris-hansen/comanda/utils/config"
	"github.com/kris-hansen/comanda/utils/discovery"
)

// ModelConfig represents configuration options for model calls
type ModelConfig struct {
	Temperature         float64
	MaxTokens           int
	MaxCompletionTokens int
	TopP                float64
}

// FileInput represents a file to be processed by the model
type FileInput struct {
	Path     string
	MimeType string
}

// ResponsesConfig represents configuration for OpenAI Responses API
type ResponsesConfig struct {
	Model              string
	Input              string
	Instructions       string
	PreviousResponseID string
	MaxOutputTokens    int
	Temperature        float64
	TopP               float64
	Stream             bool
	Tools              []map[string]interface{}
	ResponseFormat     map[string]interface{}
}

// Provider represents a model provider (e.g., Anthropic, OpenAI)
type Provider interface {
	Name() string
	SupportsModel(modelName string) bool
	SendPrompt(modelName string, prompt string) (string, error)
	SendPromptWithFile(modelName string, prompt string, file FileInput) (string, error)
	Configure(apiKey string) error
	SetVerbose(verbose bool)
	ListModels() ([]string, error) // Dynamic model listing when supported
}

// ResponsesStreamHandler defines callbacks for streaming responses
type ResponsesStreamHandler interface {
	OnResponseCreated(response map[string]interface{})
	OnResponseInProgress(response map[string]interface{})
	OnOutputItemAdded(index int, item map[string]interface{})
	OnOutputTextDelta(itemID string, index int, contentIndex int, delta string)
	OnResponseCompleted(response map[string]interface{})
	OnError(err error)
}

// ResponsesProvider extends Provider with Responses API capabilities
type ResponsesProvider interface {
	Provider
	SendPromptWithResponses(config ResponsesConfig) (string, error)
	SendPromptWithResponsesStream(config ResponsesConfig, handler ResponsesStreamHandler) error
}

// ListModelsForProvider is a generic function that all providers can use to list their models
// It delegates to the discovery module which handles caching and API calls
func ListModelsForProvider(providerName string, apiKey string) ([]string, error) {
	// Convert provider name to lowercase to match discovery module expectations
	normalizedName := strings.ToLower(providerName)
	return discovery.GetAvailableModels(normalizedName, apiKey)
}

// DetectProviderFunc is the type for the provider detection function
type DetectProviderFunc func(modelName string) Provider

// DetectProvider determines the appropriate provider based on the model name
var DetectProvider DetectProviderFunc = defaultDetectProvider

// defaultDetectProvider is the default implementation of DetectProvider
func defaultDetectProvider(modelName string) Provider {
	config.DebugLog("[Provider] Attempting to detect provider for model: %s", modelName)

	provider := registry.FindProvider(modelName)
	if provider != nil {
		config.DebugLog("[Provider] Found provider %s for model %s", provider.Name(), modelName)
		return provider
	}

	config.DebugLog("[Provider] No provider found for model %s", modelName)
	return nil
}
