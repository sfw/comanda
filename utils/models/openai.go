package models

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider handles OpenAI family of models
type OpenAIProvider struct {
	apiKey  string
	config  ModelConfig
	verbose bool
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		config: ModelConfig{
			Temperature: 0.7,
			MaxTokens:   2000,
			TopP:        1.0,
		},
	}
}

// Name returns the provider name
func (o *OpenAIProvider) Name() string {
	return "OpenAI"
}

// debugf prints debug information if verbose mode is enabled
func (o *OpenAIProvider) debugf(format string, args ...interface{}) {
	if o.verbose {
		fmt.Printf("[DEBUG][OpenAI] "+format+"\n", args...)
	}
}

// SupportsModel checks if the given model name is supported by OpenAI
func (o *OpenAIProvider) SupportsModel(modelName string) bool {
	o.debugf("Checking if model is supported: %s", modelName)
	modelName = strings.ToLower(modelName)
	isSupported := strings.HasPrefix(modelName, "gpt-") || modelName == "davinci" || modelName == "curie" || modelName == "babbage" || modelName == "ada"
	o.debugf("Model %s support result: %v", modelName, isSupported)
	return isSupported
}

// Configure sets up the provider with necessary credentials
func (o *OpenAIProvider) Configure(apiKey string) error {
	o.debugf("Configuring OpenAI provider")
	if apiKey == "" {
		return fmt.Errorf("API key is required for OpenAI provider")
	}
	o.apiKey = apiKey
	o.debugf("API key configured successfully")
	return nil
}

// SendPrompt sends a prompt to the specified model and returns the response
func (o *OpenAIProvider) SendPrompt(modelName string, prompt string) (string, error) {
	o.debugf("Preparing to send prompt to model: %s", modelName)
	o.debugf("Prompt length: %d characters", len(prompt))

	if o.apiKey == "" {
		return "", fmt.Errorf("OpenAI provider not configured: missing API key")
	}

	if !o.ValidateModel(modelName) {
		return "", fmt.Errorf("invalid OpenAI model: %s", modelName)
	}

	o.debugf("Model validation passed, preparing API call")
	o.debugf("Using configuration: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		o.config.Temperature, o.config.MaxTokens, o.config.TopP)

	client := openai.NewClient(o.apiKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: float32(o.config.Temperature),
			MaxTokens:   o.config.MaxTokens,
			TopP:        float32(o.config.TopP),
		},
	)

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from OpenAI")
	}

	response := resp.Choices[0].Message.Content
	o.debugf("API call completed, response length: %d characters", len(response))

	return response, nil
}

// ValidateModel checks if the specific OpenAI model variant is valid
func (o *OpenAIProvider) ValidateModel(modelName string) bool {
	o.debugf("Validating model: %s", modelName)
	validModels := []string{
		"gpt-4",
		"gpt-4-turbo",
		"gpt-4-32k",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
		"davinci",
		"curie",
		"babbage",
		"ada",
		"gpt-4o-mini", // Custom model
	}

	modelName = strings.ToLower(modelName)
	// Check exact matches
	for _, valid := range validModels {
		if modelName == valid {
			o.debugf("Found exact model match: %s", modelName)
			return true
		}
	}

	// Check model families with version numbers
	modelFamilies := []string{
		"gpt-4-",
		"gpt-3.5-turbo-",
	}

	for _, family := range modelFamilies {
		if strings.HasPrefix(modelName, family) {
			o.debugf("Model %s matches family: %s", modelName, family)
			return true
		}
	}

	o.debugf("Model %s validation failed - no matches found", modelName)
	return false
}

// SetConfig updates the provider configuration
func (o *OpenAIProvider) SetConfig(config ModelConfig) {
	o.debugf("Updating provider configuration")
	o.debugf("Old config: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		o.config.Temperature, o.config.MaxTokens, o.config.TopP)
	o.config = config
	o.debugf("New config: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		o.config.Temperature, o.config.MaxTokens, o.config.TopP)
}

// GetConfig returns the current provider configuration
func (o *OpenAIProvider) GetConfig() ModelConfig {
	return o.config
}

// SetVerbose enables or disables verbose mode
func (o *OpenAIProvider) SetVerbose(verbose bool) {
	o.verbose = verbose
}
