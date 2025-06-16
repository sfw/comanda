package models

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kris-hansen/comanda/utils/fileutil"
)

// AnthropicProvider handles Anthropic family of models
type AnthropicProvider struct {
	apiKey  string
	config  ModelConfig
	verbose bool
}

// NewAnthropicProvider creates a new Anthropic provider instance
func NewAnthropicProvider() *AnthropicProvider {
	return &AnthropicProvider{
		config: ModelConfig{
			Temperature: 0.7,
			MaxTokens:   2000,
			TopP:        1.0,
		},
	}
}

// debugf prints debug information if verbose mode is enabled
func (a *AnthropicProvider) debugf(format string, args ...interface{}) {
	if a.verbose {
		fmt.Printf("[DEBUG][Anthropic] "+format+"\n", args...)
	}
}

// Name returns the provider name
func (a *AnthropicProvider) Name() string {
	return "anthropic"
}

// SupportsModel checks if the given model name is supported by Anthropic
func (a *AnthropicProvider) SupportsModel(modelName string) bool {
	a.debugf("Checking if model is supported: %s", modelName)
	modelName = strings.ToLower(modelName)
	isSupported := strings.HasPrefix(modelName, "claude-")
	a.debugf("Model %s support result: %v", modelName, isSupported)
	return isSupported
}

// Configure sets up the provider with necessary credentials
func (a *AnthropicProvider) Configure(apiKey string) error {
	a.debugf("Configuring Anthropic provider")
	if apiKey == "" {
		return fmt.Errorf("API key is required for Anthropic provider")
	}
	a.apiKey = apiKey
	a.debugf("API key configured successfully")
	return nil
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type   string           `json:"type"`
	Text   string           `json:"text,omitempty"`
	Source *anthropicSource `json:"source,omitempty"`
}

type anthropicSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
	TopP        float64            `json:"top_p"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// SendPrompt sends a prompt to the specified model and returns the response
func (a *AnthropicProvider) SendPrompt(modelName string, prompt string) (string, error) {
	a.debugf("Preparing to send prompt to model: %s", modelName)
	a.debugf("Prompt length: %d characters", len(prompt))

	if a.apiKey == "" {
		return "", fmt.Errorf("Anthropic provider not configured: missing API key")
	}

	if !a.ValidateModel(modelName) {
		return "", fmt.Errorf("invalid Anthropic model: %s", modelName)
	}

	a.debugf("Model validation passed, preparing API call")
	a.debugf("Using configuration: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		a.config.Temperature, a.config.MaxTokens, a.config.TopP)

	reqBody := anthropicRequest{
		Model: modelName,
		Messages: []anthropicMessage{
			{
				Role: "user",
				Content: []anthropicContent{
					{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		TopP:        a.config.TopP,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response anthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no response content returned from Anthropic")
	}

	result := response.Content[0].Text
	a.debugf("API call completed, response length: %d characters", len(result))

	return result, nil
}

// SendPromptWithFile sends a prompt along with a file to the specified model and returns the response
func (a *AnthropicProvider) SendPromptWithFile(modelName string, prompt string, file FileInput) (string, error) {
	a.debugf("Preparing to send prompt with file to model: %s", modelName)
	a.debugf("File path: %s", file.Path)

	if a.apiKey == "" {
		return "", fmt.Errorf("Anthropic provider not configured: missing API key")
	}

	if !a.ValidateModel(modelName) {
		return "", fmt.Errorf("invalid Anthropic model: %s", modelName)
	}

	// Read the file content with size check
	fileData, err := fileutil.SafeReadFile(file.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	var content []anthropicContent

	// Handle different file types
	switch {
	case strings.HasPrefix(file.MimeType, "image/"):
		// For images, use base64 encoding
		base64Data := base64.StdEncoding.EncodeToString(fileData)
		content = []anthropicContent{
			{
				Type: "text",
				Text: prompt,
			},
			{
				Type: "image",
				Source: &anthropicSource{
					Type:      "base64",
					MediaType: file.MimeType,
					Data:      base64Data,
				},
			},
		}
	case file.MimeType == "application/pdf":
		// For PDFs, use base64 encoding with beta header
		base64Data := base64.StdEncoding.EncodeToString(fileData)
		content = []anthropicContent{
			{
				Type: "text",
				Text: prompt,
			},
			{
				Type: "document",
				Source: &anthropicSource{
					Type:      "base64",
					MediaType: file.MimeType,
					Data:      base64Data,
				},
			},
		}
	default:
		// For text files, include content in the prompt
		fileContent := string(fileData)
		content = []anthropicContent{
			{
				Type: "text",
				Text: fmt.Sprintf("File content:\n%s\n\nUser prompt: %s", fileContent, prompt),
			},
		}
	}

	reqBody := anthropicRequest{
		Model: modelName,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: content,
			},
		},
		MaxTokens:   a.config.MaxTokens,
		Temperature: a.config.Temperature,
		TopP:        a.config.TopP,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Add beta header for PDF support when sending PDF files
	if file.MimeType == "application/pdf" {
		req.Header.Set("anthropic-beta", "pdfs-2024-09-25")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response anthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no response content returned from Anthropic")
	}

	result := response.Content[0].Text
	a.debugf("API call completed, response length: %d characters", len(result))

	return result, nil
}

// ValidateModel checks if the specific Anthropic model variant is valid
func (a *AnthropicProvider) ValidateModel(modelName string) bool {
	a.debugf("Validating model: %s", modelName)
	validModels := []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-latest",
		"claude-3-5-haiku-latest",
		"claude-3-7-sonnet-20250219",
		"claude-3-7-sonnet-latest",
		"claude-3-5-haiku-20241022",
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
	}

	// Trim whitespace and convert to lowercase
	modelName = strings.TrimSpace(strings.ToLower(modelName))

	// Add extra debug logging
	a.debugf("Checking model '%s' against valid models: %v", modelName, validModels)
	// Check exact matches
	for _, valid := range validModels {
		if modelName == valid {
			a.debugf("Found exact model match: %s", modelName)
			return true
		}
	}

	// Check model families for flexibility
	modelFamilies := []string{
		"claude-3-5-sonnet",
		"claude-3-5-haiku",
		"claude-3-7-sonnet",
		"claude-opus-4",
		"claude-sonnet-4",
	}

	for _, family := range modelFamilies {
		if strings.HasPrefix(modelName, family) {
			a.debugf("Model %s matches family: %s", modelName, family)
			return true
		}
	}

	a.debugf("Model %s validation failed - no matches found", modelName)
	return false
}

// SetConfig updates the provider configuration
func (a *AnthropicProvider) SetConfig(config ModelConfig) {
	a.debugf("Updating provider configuration")
	a.debugf("Old config: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		a.config.Temperature, a.config.MaxTokens, a.config.TopP)
	a.config = config
	a.debugf("New config: Temperature=%.2f, MaxTokens=%d, TopP=%.2f",
		a.config.Temperature, a.config.MaxTokens, a.config.TopP)
}

// GetConfig returns the current provider configuration
func (a *AnthropicProvider) GetConfig() ModelConfig {
	return a.config
}

// SetVerbose enables or disables verbose mode
func (a *AnthropicProvider) SetVerbose(verbose bool) {
	a.verbose = verbose
}

// ListModels returns the list of available Anthropic models
func (a *AnthropicProvider) ListModels() ([]string, error) {
	return ListModelsForProvider(a.Name(), a.apiKey)
}

// Register the Anthropic provider on package initialization
func init() {
	factory := NewProviderFactory(
		func() Provider { return NewAnthropicProvider() },
		ProviderMetadata{
			Name:          "anthropic",
			Description:   "Anthropic Claude models (claude-3.5-sonnet, claude-3-haiku, etc.)",
			Version:       "1.0.0",
			ModelPrefixes: []string{"claude-"},
			Priority:      90, // High priority for claude- models
		},
	)
	RegisterProvider("anthropic", factory)
}
