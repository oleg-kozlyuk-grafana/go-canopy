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
	Use:   "canopy-initiator",
	Short: "Canopy Initiator - GitHub webhook handler",
	Long: `Canopy Initiator receives GitHub workflow_run webhooks, validates them,
and publishes work requests to a message queue for processing by workers.

The initiator does not have GitHub credentials and operates on the principle
of least privilege - it only validates webhooks and publishes messages.`,
	RunE: run,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Canopy Initiator %s\n", version)
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
	// Override environment variables with CLI flags if they were explicitly set
	if cmd.Flags().Changed("port") {
		os.Setenv("CANOPY_PORT", fmt.Sprintf("%d", port))
	}
	if cmd.Flags().Changed("disable-hmac") {
		os.Setenv("CANOPY_DISABLE_HMAC", fmt.Sprintf("%t", disableHMAC))
	}

	// Load configuration
	cfg, err := config.Load(config.ModeInitiator)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Print startup information
	fmt.Printf("Starting Canopy Initiator on port %d\n", cfg.Port)
	if cfg.DisableHMAC {
		fmt.Println("WARNING: HMAC validation is disabled. This should only be used for local development.")
	}
	fmt.Printf("Queue type: %s\n", cfg.Queue.Type)
	fmt.Printf("Allowed orgs: %v\n", cfg.Initiator.AllowedOrgs)
	if len(cfg.Initiator.AllowedWorkflows) > 0 {
		fmt.Printf("Allowed workflows: %v\n", cfg.Initiator.AllowedWorkflows)
	}

	// TODO: Start the initiator service
	fmt.Println("Initiator service not yet implemented")
	return nil
}
