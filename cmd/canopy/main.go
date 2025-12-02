package main

import (
	"context"
	"fmt"
	"os"

	"github.com/oleg-kozlyuk/canopy/internal/local"
	"github.com/spf13/cobra"
)

var (
	// Version information (set via ldflags during build)
	version = "dev"
	commit  = "unknown"
	date    = "unknown"

	// CLI flags
	coveragePath string
	format       string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "canopy",
	Short: "Canopy - Local code coverage analysis",
	Long: `Canopy is a tool that analyzes local code coverage against the current git diff
to find uncovered lines in your changes.

Usage:
  canopy [flags]

This command analyzes coverage files in the specified directory and compares them
against the current git diff to highlight uncovered lines in your changes.`,
	RunE: run,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Canopy %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(versionCmd)

	// Define flags
	rootCmd.Flags().StringVar(&coveragePath, "coverage", ".coverage", "Directory containing coverage files")
	rootCmd.Flags().StringVar(&format, "format", "Text", "Output format (Text, Markdown, GitHubAnnotations)")
}

func run(cmd *cobra.Command, args []string) error {
	runner := local.NewRunner(local.Config{
		CoveragePath: coveragePath,
		Format:       format,
	})
	return runner.Run(context.Background())
}
