package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Model represents a single model configuration
type Model struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// Provider represents a provider's configuration
type Provider struct {
	APIKey string   `yaml:"api_key"`
	Models []Model  `yaml:"models"`
}

// EnvConfig represents the complete environment configuration
type EnvConfig struct {
	Providers map[string]Provider `yaml:"providers"`
}

// LoadEnvConfig loads the environment configuration from .env file
func LoadEnvConfig(path string) (*EnvConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading env file: %w", err)
	}

	var config EnvConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing env file: %w", err)
	}

	return &config, nil
}

// SaveEnvConfig saves the environment configuration to .env file
func SaveEnvConfig(path string, config *EnvConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling env config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("error writing env file: %w", err)
	}

	return nil
}

// GetProviderConfig retrieves configuration for a specific provider
func (c *EnvConfig) GetProviderConfig(providerName string) (*Provider, error) {
	provider, exists := c.Providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found in configuration", providerName)
	}
	return &provider, nil
}

// AddProvider adds or updates a provider configuration
func (c *EnvConfig) AddProvider(name string, provider Provider) {
	if c.Providers == nil {
		c.Providers = make(map[string]Provider)
	}
	c.Providers[name] = provider
}

// AddModelToProvider adds a model to a specific provider
func (c *EnvConfig) AddModelToProvider(providerName string, model Model) error {
	provider, exists := c.Providers[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	// Check if model already exists
	for _, m := range provider.Models {
		if m.Name == model.Name {
			return fmt.Errorf("model %s already exists for provider %s", model.Name, providerName)
		}
	}

	provider.Models = append(provider.Models, model)
	c.Providers[providerName] = provider
	return nil
}

// GetModelConfig retrieves configuration for a specific model
func (c *EnvConfig) GetModelConfig(providerName, modelName string) (*Model, error) {
	provider, err := c.GetProviderConfig(providerName)
	if err != nil {
		return nil, err
	}

	for _, model := range provider.Models {
		if model.Name == modelName {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model %s not found for provider %s", modelName, providerName)
}

// UpdateAPIKey updates the API key for a specific provider
func (c *EnvConfig) UpdateAPIKey(providerName, apiKey string) error {
	provider, exists := c.Providers[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	provider.APIKey = apiKey
	c.Providers[providerName] = provider
	return nil
}
