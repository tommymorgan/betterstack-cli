package betterstack

import (
	"github.com/spf13/cobra"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:           "betterstack",
	Short:         "CLI for interacting with the BetterStack API",
	Long:          "A command-line tool for querying BetterStack incidents, monitors, and more.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "Output results as JSON")
}
