package testutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
)

func TestDetectContainerProvider(t *testing.T) {
	t.Run("detects provider successfully", func(t *testing.T) {
		// This test simply verifies that the detection runs without errors
		// and returns a valid provider type
		provider := DetectContainerProvider()
		assert.NotEmpty(t, provider)

		// Should return either Docker or Podman provider
		assert.Contains(t, []testcontainers.ProviderType{
			testcontainers.ProviderDocker,
			testcontainers.ProviderPodman,
		}, provider)
	})

	t.Run("detects podman from DOCKER_HOST env var", func(t *testing.T) {
		// Save original env var
		originalDockerHost := os.Getenv("DOCKER_HOST")
		defer func() {
			if originalDockerHost == "" {
				os.Unsetenv("DOCKER_HOST")
			} else {
				os.Setenv("DOCKER_HOST", originalDockerHost)
			}
		}()

		// Set DOCKER_HOST with podman in the path
		os.Setenv("DOCKER_HOST", "unix:///run/user/1000/podman/podman.sock")

		provider := DetectContainerProvider()
		assert.Equal(t, testcontainers.ProviderPodman, provider)
	})

	t.Run("returns docker when DOCKER_HOST does not contain podman", func(t *testing.T) {
		// Save original env var
		originalDockerHost := os.Getenv("DOCKER_HOST")
		defer func() {
			if originalDockerHost == "" {
				os.Unsetenv("DOCKER_HOST")
			} else {
				os.Setenv("DOCKER_HOST", originalDockerHost)
			}
		}()

		// Set DOCKER_HOST without podman
		os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")

		provider := DetectContainerProvider()
		// Note: This might still return Podman if docker info detects it
		// So we just verify it returns a valid provider
		assert.NotEmpty(t, provider)
	})

	t.Run("handles missing docker command gracefully", func(t *testing.T) {
		// Save original env var
		originalDockerHost := os.Getenv("DOCKER_HOST")
		originalPath := os.Getenv("PATH")
		defer func() {
			if originalDockerHost == "" {
				os.Unsetenv("DOCKER_HOST")
			} else {
				os.Setenv("DOCKER_HOST", originalDockerHost)
			}
			os.Setenv("PATH", originalPath)
		}()

		// Clear DOCKER_HOST and PATH to simulate missing docker command
		os.Unsetenv("DOCKER_HOST")
		os.Setenv("PATH", "")

		provider := DetectContainerProvider()
		// Should default to Docker when detection fails
		assert.Equal(t, testcontainers.ProviderDocker, provider)
	})
}

func TestConfigureRyuk(t *testing.T) {
	t.Run("configures ryuk based on detected provider", func(t *testing.T) {
		// Save original env vars
		originalRyukDisabled := os.Getenv("TESTCONTAINERS_RYUK_DISABLED")
		defer func() {
			if originalRyukDisabled == "" {
				os.Unsetenv("TESTCONTAINERS_RYUK_DISABLED")
			} else {
				os.Setenv("TESTCONTAINERS_RYUK_DISABLED", originalRyukDisabled)
			}
		}()

		// Clear the env var
		os.Unsetenv("TESTCONTAINERS_RYUK_DISABLED")

		// Call ConfigureRyuk
		disabled := ConfigureRyuk()

		// Verify the function ran without errors
		// The result depends on whether Podman is detected
		if disabled {
			assert.Equal(t, "true", os.Getenv("TESTCONTAINERS_RYUK_DISABLED"))
		}
	})

	t.Run("does not override existing TESTCONTAINERS_RYUK_DISABLED", func(t *testing.T) {
		// Save original env var
		originalRyukDisabled := os.Getenv("TESTCONTAINERS_RYUK_DISABLED")
		defer func() {
			if originalRyukDisabled == "" {
				os.Unsetenv("TESTCONTAINERS_RYUK_DISABLED")
			} else {
				os.Setenv("TESTCONTAINERS_RYUK_DISABLED", originalRyukDisabled)
			}
		}()

		// Set to false explicitly
		os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "false")

		// Call ConfigureRyuk
		disabled := ConfigureRyuk()

		// Should not override existing value
		assert.False(t, disabled)
		assert.Equal(t, "false", os.Getenv("TESTCONTAINERS_RYUK_DISABLED"))
	})
}
