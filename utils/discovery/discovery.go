package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// ModelCache stores cached model lists with TTL
type ModelCache struct {
	models    []string
	timestamp time.Time
	ttl       time.Duration
}

var (
	cache      = make(map[string]*ModelCache)
	cacheMutex sync.RWMutex
	defaultTTL = 1 * time.Hour // Cache models for 1 hour
)

// OllamaModel represents the structure of a model returned by the Ollama API
type OllamaModel struct {
	Name    string `json:"name"`
	ModTime string `json:"modified_at"`
	Size    int64  `json:"size"`
}

// GoogleModel represents a model returned by Google's models API
type GoogleModel struct {
	Name string `json:"name"`
}

// getCachedModels returns cached models if still valid
func getCachedModels(cacheKey string) ([]string, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	cache, exists := cache[cacheKey]
	if !exists {
		return nil, false
	}

	// Check if cache is still valid
	if time.Since(cache.timestamp) < cache.ttl {
		return cache.models, true
	}

	return nil, false
}

// setCachedModels stores models in cache
func setCachedModels(cacheKey string, models []string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	cache[cacheKey] = &ModelCache{
		models:    models,
		timestamp: time.Now(),
		ttl:       defaultTTL,
	}
}

// GetOpenAIModels fetches the list of available models from the OpenAI API.
func GetOpenAIModels(apiKey string) ([]string, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("openai_%s", apiKey[:min(8, len(apiKey))]) // Use first 8 chars for cache key
	if cached, found := getCachedModels(cacheKey); found {
		return cached, nil
	}

	client := openai.NewClient(apiKey)
	models, err := client.ListModels(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error fetching OpenAI models: %v", err)
	}

	var allModels []string
	for _, model := range models.Models {
		allModels = append(allModels, model.ID)
	}

	// Cache the results
	setCachedModels(cacheKey, allModels)

	return allModels, nil
}

// GetGoogleModels fetches models from Google's generative AI API
func GetGoogleModels(apiKey string) ([]string, error) {
	// If no API key provided, return static list
	if apiKey == "" {
		return GetGoogleModelsStatic(), nil
	}

	// Check cache first
	cacheKey := fmt.Sprintf("google_%s", apiKey[:min(8, len(apiKey))])
	if cached, found := getCachedModels(cacheKey); found {
		return cached, nil
	}

	// Try to fetch from Google's API
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		// Fallback to static list on error
		return GetGoogleModelsStatic(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback to static list on API error
		return GetGoogleModelsStatic(), nil
	}

	var response struct {
		Models []GoogleModel `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		// Fallback to static list on decode error
		return GetGoogleModelsStatic(), nil
	}

	var modelNames []string
	for _, model := range response.Models {
		// Extract model name from full path (e.g., "models/gemini-pro" -> "gemini-pro")
		name := model.Name
		if idx := strings.LastIndex(name, "/"); idx != -1 {
			name = name[idx+1:]
		}
		// Only include generative models, not embedding models for now
		if strings.Contains(name, "gemini") || strings.Contains(name, "aqa") {
			modelNames = append(modelNames, name)
		}
	}

	// Cache the results
	setCachedModels(cacheKey, modelNames)

	return modelNames, nil
}

// GetGoogleModelsStatic returns a hardcoded list of known Google models as fallback.
func GetGoogleModelsStatic() []string {
	// This list should be updated periodically based on Google's offerings.
	// List based on user input and existing models, matching utils/models/google.go
	return []string{
		// From user input
		"gemini-2.5-pro-exp-03-25",
		"gemini-2.0-flash",
		"gemini-2.0-flash-lite",
		"gemini-1.5-flash",
		"gemini-1.5-flash-8b",
		"gemini-1.5-pro",
		"gemini-2.5-pro-preview-03-25",
		"gemini-2.5-pro-preview-05-06",
		"gemini-embedding-exp",

		// Existing models not explicitly in user list but kept for compatibility/completeness
		"gemini-1.0-pro",
		"gemini-2.0-flash-exp",                // Experimental version
		"gemini-2.0-flash-001",                // Specific version
		"gemini-2.0-pro-exp-02-05",            // Experimental version
		"gemini-2.0-flash-lite-preview-02-05", // Preview version
		"gemini-2.0-flash-thinking-exp-01-21", // Experimental version
		"aqa",                                 // Attributed Question Answering model
	}
}

// GetAnthropicModels returns a hardcoded list of known Anthropic models.
func GetAnthropicModels() []string {
	// Check cache first
	cacheKey := "anthropic_static"
	if cached, found := getCachedModels(cacheKey); found {
		return cached
	}

	// This list should be updated periodically based on Anthropic's offerings.
	models := []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-latest",
		"claude-3-5-haiku-latest",
		"claude-3-7-sonnet-20250219",
		"claude-3-7-sonnet-latest",
		"claude-3-5-haiku-20241022",
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
	}

	// Cache static models too (with longer TTL for static lists)
	setCachedModels(cacheKey, models)

	return models
}

// GetXAIModels returns a hardcoded list of known X.AI models.
func GetXAIModels() []string {
	// Check cache first
	cacheKey := "xai_static"
	if cached, found := getCachedModels(cacheKey); found {
		return cached
	}

	// This list should be updated periodically based on X.AI's offerings.
	models := []string{
		"grok-beta",
		"grok-vision-beta",
	}

	setCachedModels(cacheKey, models)
	return models
}

// GetDeepseekModels returns a hardcoded list of known Deepseek models.
func GetDeepseekModels() []string {
	// Check cache first
	cacheKey := "deepseek_static"
	if cached, found := getCachedModels(cacheKey); found {
		return cached
	}

	// This list should be updated periodically based on Deepseek's offerings.
	models := []string{
		"deepseek-chat",
		"deepseek-coder",
		"deepseek-vision",
		"deepseek-reasoner",
	}

	setCachedModels(cacheKey, models)
	return models
}

// CheckOllamaInstalled checks if the Ollama CLI is installed and runnable.
func CheckOllamaInstalled() bool {
	cmd := exec.Command("ollama", "list")
	if err := cmd.Run(); err != nil {
		// Consider logging the error if verbose/debug is enabled
		return false
	}
	return true
}

// GetOllamaModels fetches the list of locally available models from the Ollama API.
func GetOllamaModels() ([]OllamaModel, error) {
	// Check if Ollama is running first
	if !CheckOllamaInstalled() {
		return nil, fmt.Errorf("Ollama is not installed or not running")
	}

	// Check cache first
	cacheKey := "ollama_local"
	if cached, found := getCachedModels(cacheKey); found {
		// Convert back to OllamaModel structs
		var models []OllamaModel
		for _, name := range cached {
			models = append(models, OllamaModel{Name: name})
		}
		return models, nil
	}

	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil, fmt.Errorf("error connecting to Ollama API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var response struct {
		Models []OllamaModel `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding Ollama response: %v", err)
	}

	// Cache the model names
	var modelNames []string
	for _, model := range response.Models {
		modelNames = append(modelNames, model.Name)
	}
	setCachedModels(cacheKey, modelNames)

	return response.Models, nil
}

// GetAvailableModels retrieves the list of available models for a given provider.
// For providers like OpenAI and Ollama, it requires the API key or connection.
// For others, it returns a hardcoded list.
func GetAvailableModels(providerName string, apiKey string) ([]string, error) {
	switch providerName {
	case "openai":
		return GetOpenAIModels(apiKey)
	case "anthropic":
		return GetAnthropicModels(), nil
	case "google":
		return GetGoogleModels(apiKey) // Now supports dynamic listing
	case "xai":
		return GetXAIModels(), nil
	case "deepseek":
		return GetDeepseekModels(), nil
	case "ollama":
		ollamaModels, err := GetOllamaModels()
		if err != nil {
			return nil, err
		}
		modelNames := make([]string, len(ollamaModels))
		for i, m := range ollamaModels {
			modelNames[i] = m.Name
		}
		return modelNames, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}

// ClearCache clears all cached model lists (useful for testing or manual refresh)
func ClearCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	cache = make(map[string]*ModelCache)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
