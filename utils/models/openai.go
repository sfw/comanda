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
	validModels := []string{
		"gpt-4o",
		"gpt-4o-mini",
		"o1-preview",
		"o1-mini",
	}
	for _, valid := range validModels {
		if modelName == valid {
			o.debugf("Model %s is supported", modelName)
			return true
		}
	}
	o.debugf("Model %s is not supported", modelName)
	return false
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

	if !o.SupportsModel(modelName) {
		return "", fmt.Errorf("invalid OpenAI model: %s", modelName)
	}

	o.debugf("Model validation passed, preparing API call")
	o.debugf("Using configuration: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		o.config.Temperature, o.config.MaxTokens, o.config.TopP)

	client := openai.NewClient(o.apiKey)

	// Check if this is a vision model request
	if modelName == "gpt-4o" || modelName == "gpt-4o-mini" {
		return o.handleVisionPrompt(client, prompt, modelName)
	}

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

// handleVisionPrompt processes a vision model request with image data
func (o *OpenAIProvider) handleVisionPrompt(client *openai.Client, prompt string, modelName string) (string, error) {
	// Split the prompt into text and base64 image data
	parts := strings.SplitN(prompt, "\nInput:\n", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid vision prompt format")
	}

	action := parts[0]
	imageData := strings.TrimSpace(parts[1])

	// Create the message content with text and image parts
	content := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: action,
		},
		{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: fmt.Sprintf("data:image/png;base64,%s", imageData),
			},
		},
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:         openai.ChatMessageRoleUser,
					MultiContent: content,
				},
			},
			MaxTokens: o.config.MaxTokens,
		},
	)

	if err != nil {
		return "", fmt.Errorf("OpenAI Vision API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from OpenAI Vision")
	}

	return resp.Choices[0].Message.Content, nil
}

// ValidateModel checks if the specific OpenAI model variant is valid
func (o *OpenAIProvider) ValidateModel(modelName string) bool {
	return o.SupportsModel(modelName)
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
