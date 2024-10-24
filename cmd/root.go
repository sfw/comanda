package cmd

import (
	"fmt"
	"os"

	"github.com/kris-hansen/comanda/utils/config"
	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "comanda",
	Short: "A DSL processor for handling model interactions",
	Long: `comanda is a command line tool that processes DSL configurations 
for model interactions and executes the specified actions.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Verbose = verbose
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
