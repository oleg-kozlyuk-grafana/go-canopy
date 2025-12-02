package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/oleg-kozlyuk/canopy/internal/config"
)

var (
	// Version information (set via ldflags during build)
	version = "dev"
	commit  = "unknown"
	date    = "unknown"

	// CLI flags
	port        int
	disableHMAC bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "canopy MODE",
	Short: "Canopy - Code coverage annotations for GitHub PRs",
	Long: `Canopy is a Go service that provides code coverage annotations on GitHub pull requests.
It receives GitHub workflow_run webhooks, processes Go coverage files, and posts check runs
with annotations highlighting uncovered lines.

MODE must be one of: all-in-one, initiator, or worker`,
	Args: cobra.ExactArgs(1),
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
	rootCmd.Flags().IntVar(&port, "port", 8080, "HTTP server port")
	rootCmd.Flags().BoolVar(&disableHMAC, "disable-hmac", false, "Disable HMAC signature validation (for local development only)")
}

func run(cmd *cobra.Command, args []string) error {
	// Get mode from positional argument
	mode := config.Mode(args[0])

	// Override environment variables with CLI flags if they were explicitly set
	if cmd.Flags().Changed("port") {
		os.Setenv("CANOPY_PORT", fmt.Sprintf("%d", port))
	}
	if cmd.Flags().Changed("disable-hmac") {
		os.Setenv("CANOPY_DISABLE_HMAC", fmt.Sprintf("%t", disableHMAC))
	}

	// Load configuration
	cfg, err := config.Load(mode)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Print startup information
	fmt.Printf("Starting Canopy in %s mode on port %d\n", mode, cfg.Port)
	if cfg.DisableHMAC {
		fmt.Println("WARNING: HMAC validation is disabled. This should only be used for local development.")
	}

	// TODO: Start the appropriate service based on mode
	switch mode {
	case config.ModeAllInOne:
		return runAllInOne(cfg)
	case config.ModeInitiator:
		return runInitiator(cfg)
	case config.ModeWorker:
		return runWorker(cfg)
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

func runAllInOne(cfg *config.Config) error {
	// TODO: Implement all-in-one mode
	fmt.Println("All-in-one mode not yet implemented")
	return nil
}

func runInitiator(cfg *config.Config) error {
	// TODO: Implement initiator mode
	fmt.Println("Initiator mode not yet implemented")
	return nil
}

func runWorker(cfg *config.Config) error {
	// TODO: Implement worker mode
	fmt.Println("Worker mode not yet implemented")
	return nil
}
