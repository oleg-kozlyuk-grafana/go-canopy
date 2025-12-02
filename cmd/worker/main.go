package main

import (
	"fmt"
	"os"

	"github.com/oleg-kozlyuk/canopy/internal/config"
	"github.com/spf13/cobra"
)

var (
	// Version information (set via ldflags during build)
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "canopy-worker",
	Short: "Canopy Worker - Coverage processing service",
	Long: `Canopy Worker processes coverage requests from the message queue.

It fetches workflow artifacts, processes Go coverage files, merges them,
analyzes coverage changes, and posts check runs with annotations to GitHub PRs.

The worker has GitHub credentials and handles all interactions with GitHub API
and storage backends.`,
	RunE: run,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Canopy Worker %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(versionCmd)

	// Worker doesn't need any CLI flags
	// All configuration is loaded from environment variables
}

func run(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(config.ModeWorker)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Print startup information
	fmt.Println("Starting Canopy Worker")
	fmt.Printf("Queue type: %s\n", cfg.Queue.Type)
	fmt.Printf("Storage type: %s\n", cfg.Storage.Type)
	fmt.Printf("GitHub App ID: %d\n", cfg.GitHub.AppID)
	fmt.Printf("GitHub Installation ID: %d\n", cfg.GitHub.InstallationID)

	// TODO: Start the worker service
	fmt.Println("Worker service not yet implemented")
	return nil
}
