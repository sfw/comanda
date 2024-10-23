package processor

import (
	"testing"
)

func TestValidateModel(t *testing.T) {
	processor := NewProcessor(&DSLConfig{}, createTestEnvConfig(), false)

	tests := []struct {
		name      string
		models    []string
		expectErr bool
	}{
		{
			name:      "valid OpenAI model",
			models:    []string{"gpt-3.5-turbo"},
			expectErr: false,
		},
		{
			name:      "valid Anthropic model",
			models:    []string{"claude-2"},
			expectErr: false,
		},
		{
			name:      "multiple valid models",
			models:    []string{"gpt-3.5-turbo", "claude-2"},
			expectErr: false,
		},
		{
			name:      "invalid model",
			models:    []string{"invalid-model"},
			expectErr: true,
		},
		{
			name:      "empty model list",
			models:    []string{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.validateModel(tt.models)
			if (err != nil) != tt.expectErr {
				t.Errorf("validateModel() error = %v, expectErr %v", err, tt.expectErr)
			}

			if !tt.expectErr {
				for _, model := range tt.models {
					if provider := processor.GetModelProvider(model); provider == nil && len(tt.models) > 0 {
						t.Errorf("validateModel() did not store provider for model %s", model)
					}
				}
			}
		})
	}
}

func TestConfigureProviders(t *testing.T) {
	tests := []struct {
		name      string
		models    []string
		expectErr bool
	}{
		{
			name:      "configure OpenAI provider",
			models:    []string{"gpt-3.5-turbo"},
			expectErr: false,
		},
		{
			name:      "configure Anthropic provider",
			models:    []string{"claude-2"},
			expectErr: false,
		},
		{
			name:      "configure multiple providers",
			models:    []string{"gpt-3.5-turbo", "claude-2"},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewProcessor(&DSLConfig{}, createTestEnvConfig(), false)

			// First validate the models
			err := processor.validateModel(tt.models)
			if err != nil {
				t.Fatalf("Failed to validate models: %v", err)
			}

			// Then configure the providers
			err = processor.configureProviders()
			if (err != nil) != tt.expectErr {
				t.Errorf("configureProviders() error = %v, expectErr %v", err, tt.expectErr)
			}

			if !tt.expectErr {
				for _, model := range tt.models {
					provider := processor.GetModelProvider(model)
					if provider == nil {
						t.Errorf("Provider not found for model %s", model)
						continue
					}

					// Check if provider was properly configured
					if mp, ok := provider.(*MockProvider); ok && !mp.configured {
						t.Errorf("Provider %s was not configured", provider.Name())
					}
				}
			}
		})
	}
}
