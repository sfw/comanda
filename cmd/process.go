package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"comanda/utils/config"
	"comanda/utils/processor"
)

var processCmd = &cobra.Command{
	Use:   "process [files...]",
	Short: "Process the DSL configuration",
	Long:  `Process one or more DSL configuration files and execute the specified actions.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Load environment configuration
		if verbose {
			fmt.Println("[DEBUG] Loading environment configuration from .env")
		}
		envConfig, err := config.LoadEnvConfig(".env")
		if err != nil {
			log.Fatalf("Error loading environment configuration: %v", err)
		}
		if verbose {
			fmt.Println("[DEBUG] Environment configuration loaded successfully")
		}

		for _, file := range args {
			fmt.Printf("\nProcessing DSL file: %s\n", file)

			// Read YAML file
			if verbose {
				fmt.Printf("[DEBUG] Reading YAML file: %s\n", file)
			}
			yamlFile, err := os.ReadFile(file)
			if err != nil {
				log.Printf("Error reading YAML file %s: %v\n", file, err)
				continue
			}

			// Parse YAML into DSL config
			if verbose {
				fmt.Printf("[DEBUG] Parsing YAML content for %s\n", file)
			}
			var dslConfig processor.DSLConfig
			err = yaml.Unmarshal(yamlFile, &dslConfig)
			if err != nil {
				log.Printf("Error parsing YAML file %s: %v\n", file, err)
				continue
			}

			// Create and run processor
			if verbose {
				fmt.Printf("[DEBUG] Creating processor for %s\n", file)
			}
			proc := processor.NewProcessor(&dslConfig, envConfig, verbose)
			if err := proc.Process(); err != nil {
				log.Printf("Error processing DSL file %s: %v\n", file, err)
				continue
			}

			// Print configuration summary
			fmt.Println("\nConfiguration:")
			fmt.Printf("- Model: %v\n", dslConfig.Model)
			fmt.Printf("- Action: %v\n", dslConfig.Action)
			fmt.Printf("- Output: %v\n", dslConfig.Output)
			nextActions := proc.NormalizeStringSlice(dslConfig.NextAction)
			if len(nextActions) > 0 {
				fmt.Printf("- Next Action: %v\n", nextActions)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(processCmd)
}
