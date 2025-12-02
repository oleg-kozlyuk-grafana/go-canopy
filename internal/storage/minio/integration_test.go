package minio

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	storagepkg "github.com/oleg-kozlyuk-grafana/go-canopy/internal/storage"
	"github.com/oleg-kozlyuk-grafana/go-canopy/internal/testutil"
)

// TestMinIOStorage_Integration runs integration tests against a real MinIO instance.
func TestMinIOStorage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Configure Ryuk for Podman compatibility
	testutil.ConfigureRyuk()

	ctx := context.Background()

	// Start MinIO container
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/ready").WithPort("9000").WithStartupTimeout(60 * time.Second),
	}

	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		ProviderType:     testutil.DetectContainerProvider(),
	})
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, minioContainer.Terminate(ctx))
	}()

	// Get MinIO endpoint
	endpoint, err := minioContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	// Create MinIO storage client
	config := MinIOConfig{
		Endpoint:        endpoint,
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UseSSL:          false,
		Bucket:          "test-coverage-bucket",
	}
	storage, err := NewMinIOStorage(ctx, config)
	require.NoError(t, err)
	defer storage.Close()

	t.Run("save and get cycle", func(t *testing.T) {
		key := storagepkg.CoverageKey{Org: "grafana", Repo: "tempo", Branch: "main"}
		data := []byte("mode: set\ngithub.com/grafana/tempo/pkg/util/util.go:10.1,12.2 1 1\n")

		// Save coverage
		err := storage.SaveCoverage(ctx, key, data)
		require.NoError(t, err)

		// Get coverage
		retrieved, err := storage.GetCoverage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, data, retrieved)
	})

	t.Run("get non-existent file returns nil", func(t *testing.T) {
		key := storagepkg.CoverageKey{Org: "grafana", Repo: "nonexistent", Branch: "main"}

		// Get coverage that doesn't exist
		retrieved, err := storage.GetCoverage(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		key := storagepkg.CoverageKey{Org: "grafana", Repo: "mimir", Branch: "main"}
		data1 := []byte("first version\n")
		data2 := []byte("second version with more content\n")

		// Save first version
		err := storage.SaveCoverage(ctx, key, data1)
		require.NoError(t, err)

		// Verify first version
		retrieved, err := storage.GetCoverage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, data1, retrieved)

		// Overwrite with second version
		err = storage.SaveCoverage(ctx, key, data2)
		require.NoError(t, err)

		// Verify second version
		retrieved, err = storage.GetCoverage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, data2, retrieved)
	})

	t.Run("save and get with reader", func(t *testing.T) {
		key := storagepkg.CoverageKey{Org: "grafana", Repo: "loki", Branch: "feature"}
		data := []byte("mode: set\ngithub.com/grafana/loki/pkg/logql/parser.go:50.1,55.2 3 1\n")
		reader := bytes.NewReader(data)

		// Save coverage from reader
		err := storage.SaveCoverageReader(ctx, key, reader, int64(len(data)))
		require.NoError(t, err)

		// Get coverage
		retrieved, err := storage.GetCoverage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, data, retrieved)
	})

	t.Run("save with reader without size", func(t *testing.T) {
		key := storagepkg.CoverageKey{Org: "grafana", Repo: "pyroscope", Branch: "main"}
		data := []byte("mode: atomic\ngithub.com/grafana/pyroscope/pkg/server/server.go:10.1,15.2 2 1\n")
		reader := bytes.NewReader(data)

		// Save coverage from reader with size -1 (unknown size)
		err := storage.SaveCoverageReader(ctx, key, reader, -1)
		require.NoError(t, err)

		// Get coverage
		retrieved, err := storage.GetCoverage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, data, retrieved)
	})

	t.Run("multiple keys with different data", func(t *testing.T) {
		keys := []storagepkg.CoverageKey{
			{Org: "grafana", Repo: "repo1", Branch: "main"},
			{Org: "grafana", Repo: "repo2", Branch: "main"},
			{Org: "grafana", Repo: "repo1", Branch: "develop"},
		}

		// Save different data for each key
		for i, key := range keys {
			data := []byte("mode: set\ndata for key " + string(rune('A'+i)) + "\n")
			err := storage.SaveCoverage(ctx, key, data)
			require.NoError(t, err)
		}

		// Verify each key returns its own data
		for i, key := range keys {
			expected := []byte("mode: set\ndata for key " + string(rune('A'+i)) + "\n")
			retrieved, err := storage.GetCoverage(ctx, key)
			require.NoError(t, err)
			assert.Equal(t, expected, retrieved, "data mismatch for key %v", key)
		}
	})

	t.Run("list coverage files", func(t *testing.T) {
		// Save coverage for multiple repos
		key1 := storagepkg.CoverageKey{Org: "test-org", Repo: "list-repo1", Branch: "main"}
		key2 := storagepkg.CoverageKey{Org: "test-org", Repo: "list-repo2", Branch: "main"}
		key3 := storagepkg.CoverageKey{Org: "test-org", Repo: "list-repo1", Branch: "feature"}

		err := storage.SaveCoverage(ctx, key1, []byte("data1"))
		require.NoError(t, err)
		err = storage.SaveCoverage(ctx, key2, []byte("data2"))
		require.NoError(t, err)
		err = storage.SaveCoverage(ctx, key3, []byte("data3"))
		require.NoError(t, err)

		// List all files under test-org/
		files, err := storage.ListCoverageFiles(ctx, "test-org/")
		require.NoError(t, err)
		assert.Len(t, files, 3)
		assert.Contains(t, files, "test-org/list-repo1/main/coverage.out")
		assert.Contains(t, files, "test-org/list-repo2/main/coverage.out")
		assert.Contains(t, files, "test-org/list-repo1/feature/coverage.out")

		// List files for specific repo
		files, err = storage.ListCoverageFiles(ctx, "test-org/list-repo1/")
		require.NoError(t, err)
		assert.Len(t, files, 2)
		assert.Contains(t, files, "test-org/list-repo1/main/coverage.out")
		assert.Contains(t, files, "test-org/list-repo1/feature/coverage.out")
	})

	t.Run("empty coverage data", func(t *testing.T) {
		key := storagepkg.CoverageKey{Org: "grafana", Repo: "empty", Branch: "main"}
		data := []byte{}

		// Save empty coverage
		err := storage.SaveCoverage(ctx, key, data)
		require.NoError(t, err)

		// Get coverage
		retrieved, err := storage.GetCoverage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, data, retrieved)
	})

	t.Run("large coverage file", func(t *testing.T) {
		key := storagepkg.CoverageKey{Org: "grafana", Repo: "large", Branch: "main"}

		// Generate large coverage data (1MB)
		var buf strings.Builder
		buf.WriteString("mode: set\n")
		for i := 0; i < 10000; i++ {
			buf.WriteString("github.com/grafana/test/pkg/util/util.go:10.1,12.2 1 1\n")
		}
		data := []byte(buf.String())

		// Save large coverage
		err := storage.SaveCoverage(ctx, key, data)
		require.NoError(t, err)

		// Get coverage
		retrieved, err := storage.GetCoverage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, len(data), len(retrieved))
		assert.Equal(t, data, retrieved)
	})

	t.Run("branch names with special characters", func(t *testing.T) {
		specialBranches := []string{
			"feature/add-tests",
			"fix/issue-123",
			"release-v1.0.0",
			"user/john/experiment",
		}

		for _, branch := range specialBranches {
			key := storagepkg.CoverageKey{Org: "grafana", Repo: "special", Branch: branch}
			data := []byte("mode: set\ndata for " + branch + "\n")

			// Save coverage
			err := storage.SaveCoverage(ctx, key, data)
			require.NoError(t, err)

			// Get coverage
			retrieved, err := storage.GetCoverage(ctx, key)
			require.NoError(t, err)
			assert.Equal(t, data, retrieved, "branch: %s", branch)
		}
	})

	t.Run("concurrent save and get operations", func(t *testing.T) {
		// Test concurrent access to storage
		done := make(chan bool)
		errChan := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func(idx int) {
				key := storagepkg.CoverageKey{
					Org:    "concurrent",
					Repo:   "test",
					Branch: "branch-" + string(rune('A'+idx)),
				}
				data := []byte("concurrent data " + string(rune('A'+idx)))

				// Save
				if err := storage.SaveCoverage(ctx, key, data); err != nil {
					errChan <- err
					done <- true
					return
				}

				// Get
				retrieved, err := storage.GetCoverage(ctx, key)
				if err != nil {
					errChan <- err
					done <- true
					return
				}

				if !bytes.Equal(data, retrieved) {
					errChan <- assert.AnError
					done <- true
					return
				}

				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		close(errChan)
		for err := range errChan {
			assert.NoError(t, err)
		}
	})
}

// TestMinIOStorage_ErrorScenarios tests error handling in integration scenarios.
func TestMinIOStorage_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Configure Ryuk for Podman compatibility
	testutil.ConfigureRyuk()

	ctx := context.Background()

	// Start MinIO container
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/ready").WithPort("9000").WithStartupTimeout(60 * time.Second),
	}

	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		ProviderType:     testutil.DetectContainerProvider(),
	})
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, minioContainer.Terminate(ctx))
	}()

	// Get MinIO endpoint
	endpoint, err := minioContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	t.Run("save with context timeout", func(t *testing.T) {
		config := MinIOConfig{
			Endpoint:        endpoint,
			AccessKeyID:     "minioadmin",
			SecretAccessKey: "minioadmin",
			UseSSL:          false,
			Bucket:          "timeout-test-bucket",
		}
		storage, err := NewMinIOStorage(ctx, config)
		require.NoError(t, err)
		defer storage.Close()

		key := storagepkg.CoverageKey{Org: "grafana", Repo: "timeout", Branch: "main"}
		data := []byte("test data")

		// Create context with very short timeout
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		// Wait for context to timeout
		time.Sleep(10 * time.Millisecond)

		// This should fail due to context timeout
		err = storage.SaveCoverage(ctxWithTimeout, key, data)
		assert.Error(t, err)
	})

	t.Run("get with context timeout", func(t *testing.T) {
		config := MinIOConfig{
			Endpoint:        endpoint,
			AccessKeyID:     "minioadmin",
			SecretAccessKey: "minioadmin",
			UseSSL:          false,
			Bucket:          "timeout-test-bucket-2",
		}
		storage, err := NewMinIOStorage(ctx, config)
		require.NoError(t, err)
		defer storage.Close()

		key := storagepkg.CoverageKey{Org: "grafana", Repo: "timeout", Branch: "main"}

		// Create context with very short timeout
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		// Wait for context to timeout
		time.Sleep(10 * time.Millisecond)

		// This should fail due to context timeout
		data, err := storage.GetCoverage(ctxWithTimeout, key)
		assert.Error(t, err)
		assert.Nil(t, data)
	})
}

// TestMinIOStorage_BucketCreation tests that the bucket is created if it doesn't exist.
func TestMinIOStorage_BucketCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Configure Ryuk for Podman compatibility
	testutil.ConfigureRyuk()

	ctx := context.Background()

	// Start MinIO container
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/ready").WithPort("9000").WithStartupTimeout(60 * time.Second),
	}

	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		ProviderType:     testutil.DetectContainerProvider(),
	})
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, minioContainer.Terminate(ctx))
	}()

	// Get MinIO endpoint
	endpoint, err := minioContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	// Create storage with a new bucket that doesn't exist yet
	config := MinIOConfig{
		Endpoint:        endpoint,
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UseSSL:          false,
		Bucket:          "auto-created-bucket",
	}
	storage, err := NewMinIOStorage(ctx, config)
	require.NoError(t, err)
	defer storage.Close()

	// Verify we can use the bucket immediately
	key := storagepkg.CoverageKey{Org: "grafana", Repo: "test", Branch: "main"}
	data := []byte("test data")

	err = storage.SaveCoverage(ctx, key, data)
	require.NoError(t, err)

	retrieved, err := storage.GetCoverage(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}
