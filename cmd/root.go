package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "git-scoper",
	Short: "A CLI tool for git automation",
	Long:  `git-scoper is a CLI tool that automates common git workflows and operations.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
