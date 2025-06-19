package processor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kris-hansen/comanda/utils/config"
	"github.com/kris-hansen/comanda/utils/models"
)

// --- Ollama specific types (copied from ollama.go for local check) ---

// OllamaTagsResponse represents the top-level structure of Ollama's /api/tags response
type OllamaTagsResponse struct {
	Models []OllamaModelTag `json:"models"`
}

// OllamaModelTag represents the details of a single model tag from /api/tags
type OllamaModelTag struct {
	Name string `json:"name"`
}

// --- Helper function to check local Ollama models ---

// checkOllamaModelExists queries the local Ollama instance to see if a model tag exists.
func checkOllamaModelExists(modelName string) (bool, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return false, fmt.Errorf("failed to connect to Ollama at http://localhost:11434 to verify model '%s'. Is Ollama running?", modelName)
		}
		return false, fmt.Errorf("error calling Ollama /api/tags to verify model '%s': %v", modelName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("Ollama /api/tags returned non-OK status %d while verifying model '%s': %s", resp.StatusCode, modelName, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading Ollama /api/tags response body while verifying model '%s': %v", modelName, err)
	}

	var tagsResponse OllamaTagsResponse
	if err := json.Unmarshal(bodyBytes, &tagsResponse); err != nil {
		return false, fmt.Errorf("error unmarshaling Ollama /api/tags response while verifying model '%s': %v. Body: %s", modelName, err, string(bodyBytes))
	}

	modelNameLower := strings.ToLower(modelName)
	for _, model := range tagsResponse.Models {
		if strings.ToLower(model.Name) == modelNameLower {
			return true, nil // Model found
		}
	}

	// Model not found in the list
	// Construct a helpful error message suggesting how to pull the model
	availableModels := make([]string, len(tagsResponse.Models))
	for i, m := range tagsResponse.Models {
		availableModels[i] = m.Name
	}
	errMsg := fmt.Sprintf("model tag '%s' not found in local Ollama instance. Available models: %v. Try running 'ollama pull %s'", modelName, availableModels, modelName)
	return false, fmt.Errorf("%s", errMsg)
}

// validateModel checks if the specified model is supported and has the required capabilities
func (p *Processor) validateModel(modelNames []string, inputs []string) error {
	if len(modelNames) == 0 {
		return fmt.Errorf("no model specified")
	}

	// Special case: if the only model is "NA", skip validation
	if len(modelNames) == 1 && modelNames[0] == "NA" {
		p.debugf("Model is NA, skipping provider validation")
		return nil
	}

	p.debugf("Validating %d model(s)", len(modelNames))
	for _, modelName := range modelNames {
		p.debugf("Starting validation for model: %s", modelName)
		p.debugf("Attempting provider detection for model: %s", modelName)
		provider := models.DetectProvider(modelName)
		p.debugf("Provider detection result for %s: found=%v", modelName, provider != nil)
		if provider == nil {
			errMsg := fmt.Sprintf("unsupported model: %s (no provider found)", modelName)
			p.debugf("Validation failed: %s", errMsg)
			return fmt.Errorf("%s", errMsg)
		}

		// Check if the provider actually supports this model
		p.debugf("Checking if provider %s supports model %s", provider.Name(), modelName)
		if !provider.SupportsModel(modelName) {
			errMsg := fmt.Sprintf("unsupported model: %s (provider %s does not support it)", modelName, provider.Name())
			p.debugf("Validation failed: %s", errMsg)
			return fmt.Errorf("%s", errMsg)
		}
		p.debugf("Provider %s confirmed support for model %s", provider.Name(), modelName)

		// Get provider name
		providerName := provider.Name()

		// --- Add Ollama specific local check ---
		if providerName == "ollama" {
			p.debugf("Performing local check for Ollama model tag: %s", modelName)
			exists, err := checkOllamaModelExists(modelName)
			if err != nil {
				// Error occurred during check (e.g., connection refused, API error)
				p.debugf("Ollama local check failed for %s: %v", modelName, err)
				return fmt.Errorf("Ollama check failed: %w", err) // Wrap the specific error
			}
			if !exists {
				// Model tag specifically not found in the list from /api/tags
				// The error from checkOllamaModelExists already contains the helpful message
				p.debugf("Ollama model tag %s not found locally.", modelName)
				// The error from checkOllamaModelExists includes the suggestion to pull
				return fmt.Errorf("model tag '%s' not found locally via Ollama API", modelName)
			}
			p.debugf("Ollama model tag %s confirmed to exist locally.", modelName)
		}
		// --- End Ollama specific check ---

		// Get model configuration from environment
		p.debugf("Getting model configuration for %s from provider %s", modelName, providerName)
		modelConfig, err := p.envConfig.GetModelConfig(providerName, modelName)
		if err != nil {
			// Check if the error is specifically "model not found" after provider support was confirmed
			if strings.Contains(err.Error(), fmt.Sprintf("model %s not found for provider %s", modelName, providerName)) {
				errMsg := fmt.Sprintf("model %s is supported by provider %s but is not enabled in your configuration. Use 'comanda configure' to add it.", modelName, providerName)
				p.debugf("Configuration error: %s", errMsg)
				return fmt.Errorf("%s", errMsg)
			}
			// Otherwise, return the original configuration error
			errMsg := fmt.Sprintf("failed to get model configuration for %s: %v", modelName, err)
			p.debugf("Configuration error: %s", errMsg)
			return fmt.Errorf("%s", errMsg)
		}
		p.debugf("Successfully retrieved model configuration for %s", modelName)

		// Check if model has required capabilities based on input types
		for _, input := range inputs {
			if input == "NA" || input == "STDIN" {
				continue
			}

			// Check for file mode support if input is a document file
			if p.validator.IsDocumentFile(input) && !modelConfig.HasMode(config.FileMode) {
				return fmt.Errorf("model %s does not support file processing", modelName)
			}

			// Check for vision mode support if input is an image file
			if p.validator.IsImageFile(input) && !modelConfig.HasMode(config.VisionMode) {
				return fmt.Errorf("model %s does not support image processing", modelName)
			}

			// For text files, ensure model supports text mode
			if !p.validator.IsDocumentFile(input) && !p.validator.IsImageFile(input) && !modelConfig.HasMode(config.TextMode) {
				return fmt.Errorf("model %s does not support text processing", modelName)
			}
		}

		provider.SetVerbose(p.verbose)
		// Store provider by provider name instead of model name (thread-safe)
		p.providerMutex.Lock()
		p.providers[provider.Name()] = provider
		p.providerMutex.Unlock()
		p.debugf("Model %s is supported by provider %s", modelName, provider.Name())
	}
	return nil
}

// configureProviders sets up all detected providers with API keys
func (p *Processor) configureProviders() error {
	p.debugf("Configuring providers")

	// Create a copy of providers map for safe iteration
	p.providerMutex.RLock()
	providersCopy := make(map[string]models.Provider)
	for k, v := range p.providers {
		providersCopy[k] = v
	}
	p.providerMutex.RUnlock()

	for providerName, provider := range providersCopy {
		p.debugf("Configuring provider %s", providerName)

		// Handle Ollama provider separately since it doesn't need an API key, but expects "LOCAL"
		if providerName == "ollama" {
			if err := provider.Configure("LOCAL"); err != nil { // Pass "LOCAL" as expected by OllamaProvider.Configure
				return fmt.Errorf("failed to configure provider %s: %w", providerName, err)
			}
			p.debugf("Successfully configured local provider %s", providerName)
			continue
		}

		var providerConfig *config.Provider
		var err error

		// Get provider configuration directly by name instead of using hardcoded switch
		providerConfig, err = p.envConfig.GetProviderConfig(providerName)

		if err != nil {
			return fmt.Errorf("failed to get config for provider %s: %w", providerName, err)
		}

		if providerConfig.APIKey == "" {
			return fmt.Errorf("missing API key for provider %s", providerName)
		}

		p.debugf("Found API key for provider %s", providerName)

		if err := provider.Configure(providerConfig.APIKey); err != nil {
			return fmt.Errorf("failed to configure provider %s: %w", providerName, err)
		}

		p.debugf("Successfully configured provider %s", providerName)
	}
	return nil
}

// GetModelProvider returns the provider for the specified model
func (p *Processor) GetModelProvider(modelName string) models.Provider {
	// Special case: if model is "NA", return nil since no provider is needed
	if modelName == "NA" {
		return nil
	}

	provider := models.DetectProvider(modelName)
	if provider == nil {
		return nil
	}

	// Thread-safe read access to providers map
	p.providerMutex.RLock()
	defer p.providerMutex.RUnlock()
	return p.providers[provider.Name()]
}
