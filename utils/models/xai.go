//go:build xai || all

package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kris-hansen/comanda/utils/fileutil"
	openai "github.com/sashabaranov/go-openai"
)

// XAIProvider handles X.AI family of models
type XAIProvider struct {
	apiKey  string
	config  ModelConfig
	verbose bool
}

// Default configuration values
const (
	defaultTimeout  = 30 * time.Second
	maxPromptTokens = 4000
	// Rough approximation of tokens to characters ratio (1 token ≈ 4 characters)
	tokensToCharsRatio = 4
)

// NewXAIProvider creates a new X.AI provider instance
func NewXAIProvider() *XAIProvider {
	return &XAIProvider{
		config: ModelConfig{
			Temperature: 0.7,
			MaxTokens:   2000,
			TopP:        1.0,
		},
	}
}

// Name returns the provider name
func (x *XAIProvider) Name() string {
	return "xai"
}

// debugf prints debug information if verbose mode is enabled
func (x *XAIProvider) debugf(format string, args ...interface{}) {
	if x.verbose {
		fmt.Printf("[DEBUG][XAI] "+format+"\n", args...)
	}
}

// SupportsModel checks if the given model name is supported by X.AI
func (x *XAIProvider) SupportsModel(modelName string) bool {
	x.debugf("Checking if model is supported: %s", modelName)
	modelName = strings.ToLower(modelName)

	// Accept any model name that starts with our known prefixes
	validPrefixes := []string{
		"grok-",
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(modelName, prefix) {
			x.debugf("Model %s is supported (matches prefix %s)", modelName, prefix)
			return true
		}
	}

	x.debugf("Model %s is not supported (no matching prefix)", modelName)
	return false
}

// Configure sets up the provider with necessary credentials
func (x *XAIProvider) Configure(apiKey string) error {
	x.debugf("Configuring X.AI provider")
	if apiKey == "" {
		return fmt.Errorf("API key is required for X.AI provider")
	}
	x.apiKey = apiKey
	x.debugf("API key configured successfully")
	return nil
}

// estimateTokenCount provides a rough estimate of token count from character count
func (x *XAIProvider) estimateTokenCount(text string) int {
	return len(text) / tokensToCharsRatio
}

// SendPrompt sends a prompt to the specified model and returns the response
func (x *XAIProvider) SendPrompt(modelName string, prompt string) (string, error) {
	x.debugf("Preparing to send prompt to model: %s", modelName)
	x.debugf("Prompt length: %d characters", len(prompt))

	if x.apiKey == "" {
		return "", fmt.Errorf("X.AI provider not configured: missing API key")
	}

	if !x.SupportsModel(modelName) {
		return "", fmt.Errorf("invalid X.AI model: %s", modelName)
	}

	// Check estimated token count
	estimatedTokens := x.estimateTokenCount(prompt)
	if estimatedTokens > maxPromptTokens {
		return "", fmt.Errorf("prompt likely exceeds maximum token limit of %d (estimated tokens: %d)", maxPromptTokens, estimatedTokens)
	}

	x.debugf("Model validation passed, preparing API call")
	x.debugf("Using configuration: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		x.config.Temperature, x.config.MaxTokens, x.config.TopP)

	config := openai.DefaultConfig(x.apiKey)
	config.BaseURL = "https://api.x.ai/v1"
	client := openai.NewClientWithConfig(config)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: float32(x.config.Temperature),
			MaxTokens:   x.config.MaxTokens,
			TopP:        float32(x.config.TopP),
		},
	)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("request timed out after %v", defaultTimeout)
		}
		return "", fmt.Errorf("X.AI API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from X.AI")
	}

	response := resp.Choices[0].Message.Content
	x.debugf("API call completed, response length: %d characters", len(response))

	return response, nil
}

// SendPromptWithFile sends a prompt along with a file to the specified model and returns the response
func (x *XAIProvider) SendPromptWithFile(modelName string, prompt string, file FileInput) (string, error) {
	x.debugf("Preparing to send prompt with file to model: %s", modelName)
	x.debugf("File path: %s", file.Path)

	if x.apiKey == "" {
		return "", fmt.Errorf("X.AI provider not configured: missing API key")
	}

	if !x.SupportsModel(modelName) {
		return "", fmt.Errorf("invalid X.AI model: %s", modelName)
	}

	// Read the file content with size check
	fileData, err := fileutil.SafeReadFile(file.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	config := openai.DefaultConfig(x.apiKey)
	config.BaseURL = "https://api.x.ai/v1"
	client := openai.NewClientWithConfig(config)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// For image files, use MultiContent approach similar to OpenAI
	if strings.HasPrefix(file.MimeType, "image/") {
		base64Data := fmt.Sprintf("data:%s;base64,%s", file.MimeType, string(fileData))
		content := []openai.ChatMessagePart{
			{
				Type: openai.ChatMessagePartTypeText,
				Text: prompt,
			},
			{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL: base64Data,
				},
			},
		}

		resp, err := client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model: modelName,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:         openai.ChatMessageRoleUser,
						MultiContent: content,
					},
				},
				MaxTokens: x.config.MaxTokens,
			},
		)

		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return "", fmt.Errorf("request timed out after %v", defaultTimeout)
			}
			return "", fmt.Errorf("X.AI API error: %v", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no response choices returned from X.AI")
		}

		response := resp.Choices[0].Message.Content
		x.debugf("API call completed, response length: %d characters", len(response))

		return response, nil
	}

	// For non-image files, combine content with prompt
	fileContent := string(fileData)
	combinedPrompt := fmt.Sprintf("File content:\n%s\n\nUser prompt: %s", fileContent, prompt)

	// Check estimated token count for combined prompt
	estimatedTokens := x.estimateTokenCount(combinedPrompt)
	if estimatedTokens > maxPromptTokens {
		return "", fmt.Errorf("combined prompt likely exceeds maximum token limit of %d (estimated tokens: %d)", maxPromptTokens, estimatedTokens)
	}

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: combinedPrompt,
				},
			},
			Temperature: float32(x.config.Temperature),
			MaxTokens:   x.config.MaxTokens,
			TopP:        float32(x.config.TopP),
		},
	)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("request timed out after %v", defaultTimeout)
		}
		return "", fmt.Errorf("X.AI API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from X.AI")
	}

	response := resp.Choices[0].Message.Content
	x.debugf("API call completed, response length: %d characters", len(response))

	return response, nil
}

// ValidateModel checks if the specific X.AI model variant is valid
func (x *XAIProvider) ValidateModel(modelName string) bool {
	return x.SupportsModel(modelName)
}

// SetConfig updates the provider configuration
func (x *XAIProvider) SetConfig(config ModelConfig) {
	x.debugf("Updating provider configuration")
	x.debugf("Old config: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		x.config.Temperature, x.config.MaxTokens, x.config.TopP)
	x.config = config
	x.debugf("New config: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		x.config.Temperature, x.config.MaxTokens, x.config.TopP)
}

// GetConfig returns the current provider configuration
func (x *XAIProvider) GetConfig() ModelConfig {
	return x.config
}

// SetVerbose enables or disables verbose mode
func (x *XAIProvider) SetVerbose(verbose bool) {
	x.verbose = verbose
}

// ListModels returns the list of available XAI models
func (x *XAIProvider) ListModels() ([]string, error) {
	return ListModelsForProvider(x.Name(), x.apiKey)
}

// Register the XAI provider on package initialization
func init() {
	factory := NewProviderFactory(
		func() Provider { return NewXAIProvider() },
		ProviderMetadata{
			Name:          "xai",
			Description:   "X.AI Grok models (grok-beta, grok-vision-beta, etc.)",
			Version:       "1.0.0",
			ModelPrefixes: []string{"grok-"},
			Priority:      75, // Medium-high priority for Grok models
		},
	)
	RegisterProvider("xai", factory)
}
