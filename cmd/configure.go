package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/kris-hansen/comanda/utils/config"
	"github.com/spf13/cobra"
)

type Provider struct {
	APIKey string  `yaml:"api_key"`
	Models []Model `yaml:"models"`
}

type Model struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type Config struct {
	Providers map[string]Provider `yaml:"providers"`
}

var listFlag bool

func checkOllamaInstalled() bool {
	cmd := exec.Command("ollama", "list")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func isValidOllamaModel(modelName string) bool {
	cmd := exec.Command("ollama", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Convert output to string and check if model exists
	outputStr := string(output)
	return strings.Contains(outputStr, modelName)
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure model settings",
	Long:  `Configure model settings including provider, model name, and API key`,
	Run: func(cmd *cobra.Command, args []string) {
		if listFlag {
			listConfiguration()
			return
		}

		reader := bufio.NewReader(os.Stdin)

		// Get config path from environment or default
		configPath := config.GetEnvPath()

		// Read existing configuration
		var cfg Config
		configData, err := os.ReadFile(configPath)
		if err == nil {
			err = yaml.Unmarshal(configData, &cfg)
			if err != nil {
				cfg.Providers = make(map[string]Provider)
			}
		} else {
			cfg.Providers = make(map[string]Provider)
		}

		// Prompt for provider
		var provider string
		for {
			fmt.Print("Enter provider (openai/anthropic/ollama): ")
			provider, _ = reader.ReadString('\n')
			provider = strings.TrimSpace(provider)
			if provider == "openai" || provider == "anthropic" || provider == "ollama" {
				break
			}
			fmt.Println("Invalid provider. Please enter 'openai', 'anthropic', or 'ollama'")
		}

		// Special handling for ollama provider
		if provider == "ollama" {
			if !checkOllamaInstalled() {
				fmt.Println("Error: Ollama is not installed or not running. Please install ollama and try again.")
				return
			}
		}

		// Check if provider exists
		existingProvider, exists := cfg.Providers[provider]
		var apiKey string
		if !exists {
			if provider != "ollama" {
				// Only prompt for API key if not ollama
				fmt.Print("Enter API key: ")
				apiKey, _ = reader.ReadString('\n')
				apiKey = strings.TrimSpace(apiKey)
			}
			existingProvider = Provider{
				APIKey: apiKey,
				Models: []Model{},
			}
		} else {
			apiKey = existingProvider.APIKey
		}

		// Prompt for model name
		var modelName string
		for {
			if provider == "ollama" {
				fmt.Print("Enter model name (must be pulled in ollama): ")
			} else {
				fmt.Print("Enter model name: ")
			}
			modelName, _ = reader.ReadString('\n')
			modelName = strings.TrimSpace(modelName)

			if provider == "ollama" {
				if !isValidOllamaModel(modelName) {
					fmt.Printf("Model '%s' is not available in ollama. Please pull it first using 'ollama pull %s'\n", modelName, modelName)
					continue
				}
			}
			break
		}

		// Add new model to provider
		modelType := "external"
		if provider == "ollama" {
			modelType = "local"
		}

		newModel := Model{
			Name: modelName,
			Type: modelType,
		}
		existingProvider.Models = append(existingProvider.Models, newModel)
		cfg.Providers[provider] = existingProvider

		// Ensure the directory exists
		if dir := strings.TrimSuffix(configPath, "/"+filepath.Base(configPath)); dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("Error creating directory: %v\n", err)
				return
			}
		}

		// Write updated configuration
		yamlData, err := yaml.Marshal(&cfg)
		if err != nil {
			fmt.Printf("Error marshaling configuration: %v\n", err)
			return
		}

		err = os.WriteFile(configPath, yamlData, 0644)
		if err != nil {
			fmt.Printf("Error writing configuration: %v\n", err)
			return
		}

		fmt.Printf("Configuration saved successfully to %s!\n", configPath)
	},
}

func listConfiguration() {
	configPath := config.GetEnvPath()
	configData, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("No configuration found at %s\n", configPath)
		return
	}

	var cfg Config
	err = yaml.Unmarshal(configData, &cfg)
	if err != nil {
		fmt.Printf("Error reading configuration: %v\n", err)
		return
	}

	if len(cfg.Providers) == 0 {
		fmt.Println("No providers configured.")
		return
	}

	fmt.Printf("Configuration from %s:\n\n", configPath)
	fmt.Println("Configured Providers:")
	for provider, data := range cfg.Providers {
		fmt.Printf("\n%s:\n", provider)
		if len(data.Models) == 0 {
			fmt.Println("  No models configured")
			continue
		}
		for _, model := range data.Models {
			fmt.Printf("  - %s (%s)\n", model.Name, model.Type)
		}
	}
}

func init() {
	configureCmd.Flags().BoolVar(&listFlag, "list", false, "List all configured providers and models")
	rootCmd.AddCommand(configureCmd)
}
