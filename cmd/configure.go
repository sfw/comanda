package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Provider struct {
	APIKey string  `yaml:"api_key"`
	Models []Model `yaml:"models"`
}

type Model struct {
	Name        string  `yaml:"name"`
	MaxTokens   int     `yaml:"max_tokens"`
	SystemMsg   string  `yaml:"system_msg"`
	Temperature float32 `yaml:"temperature"`
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure API keys and model settings",
	Long: `Configure your API keys and model settings for different providers.
This will create a configuration file in your home directory.`,
	Run: runConfigure,
}

func init() {
	rootCmd.AddCommand(configureCmd)
}

func runConfigure(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}
	configDir := filepath.Join(homeDir, ".comanda")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		return
	}

	providers := []string{"openai", "anthropic", "ollama"}
	for _, provider := range providers {
		fmt.Printf("\nConfiguring %s:\n", provider)
		configureProvider(reader, configDir, provider)
	}
}

func configureProvider(reader *bufio.Reader, configDir, provider string) {
	configFile := filepath.Join(configDir, provider+".yaml")

	var apiKey string
	if provider != "ollama" {
		fmt.Printf("Enter your %s API key (press Enter to skip): ", provider)
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
	}

	if apiKey == "" && provider != "ollama" {
		fmt.Printf("Skipping %s configuration\n", provider)
		return
	}

	// Default model configurations
	models := getDefaultModels(provider)

	config := Provider{
		APIKey: apiKey,
		Models: models,
	}

	// Marshal the configuration
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		fmt.Printf("Error marshaling configuration: %v\n", err)
		return
	}

	// Write the configuration file
	if err := os.WriteFile(configFile, yamlData, 0600); err != nil {
		fmt.Printf("Error writing configuration file: %v\n", err)
		return
	}

	fmt.Printf("%s configuration saved to %s\n", provider, configFile)
}

func getDefaultModels(provider string) []Model {
	switch provider {
	case "openai":
		return []Model{
			{
				Name:        "gpt-4",
				MaxTokens:   8000,
				SystemMsg:   "You are a helpful assistant.",
				Temperature: 0.7,
			},
			{
				Name:        "gpt-3.5-turbo",
				MaxTokens:   4000,
				SystemMsg:   "You are a helpful assistant.",
				Temperature: 0.7,
			},
		}
	case "anthropic":
		return []Model{
			{
				Name:        "claude-2",
				MaxTokens:   100000,
				SystemMsg:   "You are Claude, a helpful AI assistant.",
				Temperature: 0.7,
			},
		}
	case "ollama":
		return []Model{
			{
				Name:        "llama2",
				MaxTokens:   4096,
				SystemMsg:   "You are a helpful AI assistant.",
				Temperature: 0.7,
			},
		}
	default:
		return []Model{}
	}
}
