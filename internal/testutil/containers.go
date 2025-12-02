package testutil

import (
	"os"
	"os/exec"
	"strings"

	"github.com/testcontainers/testcontainers-go"
)

// isPodman checks if the current container engine is Podman.
func isPodman() bool {
	// Check 1: DOCKER_HOST environment variable contains "podman"
	// This is common on Fedora/RHEL systems where DOCKER_HOST is explicitly set
	if dockerHost := os.Getenv("DOCKER_HOST"); strings.Contains(dockerHost, "podman") {
		return true
	}

	// Check 2: Run "docker info" and check if it's actually Podman
	// Podman's docker-compat layer reports "podman" in various fields
	// such as socket paths, package names, and version information
	cmd := exec.Command("docker", "info")
	output, err := cmd.CombinedOutput()
	if err == nil {
		outputLower := strings.ToLower(string(output))
		if strings.Contains(outputLower, "podman") {
			return true
		}
	}

	return false
}

// DetectContainerProvider detects whether Docker or Podman is being used
// and returns the appropriate testcontainers provider type.
//
// This function performs the following checks in order:
//  1. Checks if DOCKER_HOST environment variable contains "podman"
//  2. Runs "docker info" and checks if the output contains "podman"
//
// If either check indicates Podman is being used, it returns ProviderPodman.
// Otherwise, it returns ProviderDocker as the default.
//
// This auto-detection ensures that testcontainers can work seamlessly with
// both Docker and Podman without requiring manual configuration.
func DetectContainerProvider() testcontainers.ProviderType {
	if isPodman() {
		return testcontainers.ProviderPodman
	}
	return testcontainers.ProviderDocker
}

// ConfigureRyuk sets the TESTCONTAINERS_RYUK_DISABLED environment variable
// if Podman is detected, as Ryuk often has permission issues with Podman.
//
// This function should be called once in TestMain or at the beginning of
// integration tests to ensure proper configuration.
//
// Returns true if Ryuk was disabled, false otherwise.
func ConfigureRyuk() bool {
	if isPodman() && os.Getenv("TESTCONTAINERS_RYUK_DISABLED") == "" {
		os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
		return true
	}
	return false
}
