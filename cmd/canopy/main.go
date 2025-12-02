package main

import (
	"context"
	"fmt"
	"os"

	"github.com/oleg-kozlyuk-grafana/go-canopy/internal/diff"
	"github.com/oleg-kozlyuk-grafana/go-canopy/internal/local"
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
	baseRef      string
	commitSHA    string
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
	rootCmd.Flags().StringVar(&baseRef, "base", "", "Analyze diff against base ref (uses git diff <base>..HEAD)")
	rootCmd.Flags().StringVar(&commitSHA, "commit", "", "Analyze diff for a specific commit (uses git diff-tree)")
}

func run(cmd *cobra.Command, args []string) error {
	// Create appropriate DiffSource based on flags (hierarchy-based selection)
	var diffSource diff.DiffSource

	if baseRef != "" {
		// --base flag takes highest priority (for PR context)
		diffSource = diff.NewGitBaseDiffSource(baseRef, "")
	} else if commitSHA != "" {
		// --commit flag for single commit analysis
		diffSource = diff.NewGitCommitDiffSource(commitSHA, "")
	} else {
		// Default: use local git diff
		diffSource = diff.NewLocalDiffSource("")
	}

	runner := local.NewRunner(local.Config{
		CoveragePath: coveragePath,
		Format:       format,
	}, local.WithDiffSource(diffSource))

	return runner.Run(context.Background())
}
