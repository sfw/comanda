package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaProvider handles Ollama family of models
type OllamaProvider struct {
	verbose bool
}

// OllamaRequest represents the request structure for Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents the response structure from Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// NewOllamaProvider creates a new Ollama provider instance
func NewOllamaProvider() *OllamaProvider {
	return &OllamaProvider{}
}

// Name returns the provider name
func (o *OllamaProvider) Name() string {
	return "Ollama"
}

// debugf prints debug information if verbose mode is enabled
func (o *OllamaProvider) debugf(format string, args ...interface{}) {
	if o.verbose {
		fmt.Printf("[DEBUG][Ollama] "+format+"\n", args...)
	}
}

// SupportsModel checks if the given model name is supported by Ollama
func (o *OllamaProvider) SupportsModel(modelName string) bool {
	o.debugf("Checking if model is supported: %s", modelName)
	// Ollama supports any model that has been pulled locally
	// We'll return true here and let the actual API call validate the model
	return true
}

// Configure sets up the provider (no API key needed for Ollama)
func (o *OllamaProvider) Configure(apiKey string) error {
	o.debugf("Configuring Ollama provider")
	return nil
}

// SendPrompt sends a prompt to the specified model and returns the response
func (o *OllamaProvider) SendPrompt(modelName string, prompt string) (string, error) {
	o.debugf("Preparing to send prompt to model: %s", modelName)
	o.debugf("Prompt length: %d characters", len(prompt))

	reqBody := OllamaRequest{
		Model:  modelName,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error calling Ollama API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Read and accumulate all responses
	var fullResponse strings.Builder
	decoder := json.NewDecoder(resp.Body)
	for {
		var ollamaResp OllamaResponse
		if err := decoder.Decode(&ollamaResp); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("error decoding response: %v", err)
		}
		fullResponse.WriteString(ollamaResp.Response)
		if ollamaResp.Done {
			break
		}
	}

	result := fullResponse.String()
	o.debugf("API call completed, response length: %d characters", len(result))
	return result, nil
}

// SetVerbose enables or disables verbose mode
func (o *OllamaProvider) SetVerbose(verbose bool) {
	o.verbose = verbose
}
