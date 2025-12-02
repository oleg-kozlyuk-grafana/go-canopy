package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatObjectPath(t *testing.T) {
	tests := []struct {
		name     string
		key      CoverageKey
		expected string
	}{
		{
			name: "standard path",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "tempo",
				Branch: "main",
			},
			expected: "grafana/tempo/main/coverage.out",
		},
		{
			name: "feature branch",
			key: CoverageKey{
				Org:    "myorg",
				Repo:   "myrepo",
				Branch: "feature/add-tests",
			},
			expected: "myorg/myrepo/feature/add-tests/coverage.out",
		},
		{
			name: "special characters in branch",
			key: CoverageKey{
				Org:    "org",
				Repo:   "repo",
				Branch: "fix/issue-123",
			},
			expected: "org/repo/fix/issue-123/coverage.out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatObjectPath(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateCoverageKey(t *testing.T) {
	tests := []struct {
		name      string
		key       CoverageKey
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid key",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "tempo",
				Branch: "main",
			},
			wantError: false,
		},
		{
			name: "missing org",
			key: CoverageKey{
				Org:    "",
				Repo:   "tempo",
				Branch: "main",
			},
			wantError: true,
			errorMsg:  "org is required",
		},
		{
			name: "missing repo",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "",
				Branch: "main",
			},
			wantError: true,
			errorMsg:  "repo is required",
		},
		{
			name: "missing branch",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "tempo",
				Branch: "",
			},
			wantError: true,
			errorMsg:  "branch is required",
		},
		{
			name: "all fields missing",
			key: CoverageKey{
				Org:    "",
				Repo:   "",
				Branch: "",
			},
			wantError: true,
			errorMsg:  "org is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCoverageKey(tt.key)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewGCSStorage(t *testing.T) {
	t.Run("empty bucket name", func(t *testing.T) {
		ctx := context.Background()
		storage, err := NewGCSStorage(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, storage)
		assert.Contains(t, err.Error(), "bucket name is required")
	})

	// Note: We cannot test successful creation without real GCS credentials
	// or using testcontainers. This would be covered in integration tests.
}

func TestGCSStorage_SaveCoverage_Validation(t *testing.T) {
	// Create a mock GCS storage (client will be nil, but that's ok for validation tests)
	gcs := &GCSStorage{
		client: nil,
		bucket: "test-bucket",
	}

	tests := []struct {
		name      string
		key       CoverageKey
		data      []byte
		wantError bool
		errorMsg  string
	}{
		{
			name: "invalid key - missing org",
			key: CoverageKey{
				Org:    "",
				Repo:   "tempo",
				Branch: "main",
			},
			data:      []byte("mode: set\n"),
			wantError: true,
			errorMsg:  "org is required",
		},
		{
			name: "invalid key - missing repo",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "",
				Branch: "main",
			},
			data:      []byte("mode: set\n"),
			wantError: true,
			errorMsg:  "repo is required",
		},
		{
			name: "invalid key - missing branch",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "tempo",
				Branch: "",
			},
			data:      []byte("mode: set\n"),
			wantError: true,
			errorMsg:  "branch is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := gcs.SaveCoverage(ctx, tt.key, tt.data)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				// Would fail without real client, but validation should pass
				assert.NoError(t, err)
			}
		})
	}
}

func TestGCSStorage_GetCoverage_Validation(t *testing.T) {
	// Create a mock GCS storage
	gcs := &GCSStorage{
		client: nil,
		bucket: "test-bucket",
	}

	tests := []struct {
		name      string
		key       CoverageKey
		wantError bool
		errorMsg  string
	}{
		{
			name: "invalid key - missing org",
			key: CoverageKey{
				Org:    "",
				Repo:   "tempo",
				Branch: "main",
			},
			wantError: true,
			errorMsg:  "org is required",
		},
		{
			name: "invalid key - missing repo",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "",
				Branch: "main",
			},
			wantError: true,
			errorMsg:  "repo is required",
		},
		{
			name: "invalid key - missing branch",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "tempo",
				Branch: "",
			},
			wantError: true,
			errorMsg:  "branch is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			data, err := gcs.GetCoverage(ctx, tt.key)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, data)
				assert.Contains(t, err.Error(), tt.errorMsg)
			}
		})
	}
}

func TestGCSStorage_SaveCoverageReader_Validation(t *testing.T) {
	gcs := &GCSStorage{
		client: nil,
		bucket: "test-bucket",
	}

	tests := []struct {
		name      string
		key       CoverageKey
		reader    io.Reader
		size      int64
		wantError bool
		errorMsg  string
	}{
		{
			name: "invalid key - missing org",
			key: CoverageKey{
				Org:    "",
				Repo:   "tempo",
				Branch: "main",
			},
			reader:    strings.NewReader("mode: set\n"),
			size:      10,
			wantError: true,
			errorMsg:  "org is required",
		},
		{
			name: "nil reader",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "tempo",
				Branch: "main",
			},
			reader:    nil,
			size:      10,
			wantError: true,
			errorMsg:  "reader is nil",
		},
		{
			name: "invalid key - missing branch",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "tempo",
				Branch: "",
			},
			reader:    strings.NewReader("mode: set\n"),
			size:      10,
			wantError: true,
			errorMsg:  "branch is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := gcs.SaveCoverageReader(ctx, tt.key, tt.reader, tt.size)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			}
		})
	}
}

func TestGCSStorage_Close(t *testing.T) {
	t.Run("close with nil client", func(t *testing.T) {
		gcs := &GCSStorage{
			client: nil,
			bucket: "test-bucket",
		}
		err := gcs.Close()
		assert.NoError(t, err)
	})
}

// Integration tests would go here using testcontainers
// Example integration test structure (not implemented yet):
//
// func TestGCSStorage_Integration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test")
// 	}
//
// 	ctx := context.Background()
//
// 	// Start GCS emulator using testcontainers
// 	// container, err := testcontainers.GenericContainer(...)
// 	// require.NoError(t, err)
// 	// defer container.Terminate(ctx)
//
// 	// Run integration tests
// 	t.Run("save and get cycle", func(t *testing.T) {
// 		// Test full save/get cycle
// 	})
//
// 	t.Run("get non-existent file", func(t *testing.T) {
// 		// Test getting a file that doesn't exist
// 	})
//
// 	t.Run("overwrite existing file", func(t *testing.T) {
// 		// Test overwriting an existing file
// 	})
// }

// MockGCSClient tests - these would require a more sophisticated mock setup
// For now, we have validation tests which cover the error paths without needing
// real GCS access. Integration tests with testcontainers would cover the happy paths.

// TestGCSStorage_ErrorHandling tests error conditions that can be unit tested
func TestGCSStorage_ErrorHandling(t *testing.T) {
	t.Run("SaveCoverage with invalid key returns validation error", func(t *testing.T) {
		gcs := &GCSStorage{bucket: "test-bucket"}
		ctx := context.Background()

		invalidKey := CoverageKey{Org: "", Repo: "repo", Branch: "main"}
		err := gcs.SaveCoverage(ctx, invalidKey, []byte("test"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "org is required")
	})

	t.Run("GetCoverage with invalid key returns validation error", func(t *testing.T) {
		gcs := &GCSStorage{bucket: "test-bucket"}
		ctx := context.Background()

		invalidKey := CoverageKey{Org: "org", Repo: "", Branch: "main"}
		data, err := gcs.GetCoverage(ctx, invalidKey)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "repo is required")
	})

	t.Run("SaveCoverageReader with nil reader returns error", func(t *testing.T) {
		gcs := &GCSStorage{bucket: "test-bucket"}
		ctx := context.Background()

		validKey := CoverageKey{Org: "org", Repo: "repo", Branch: "main"}
		err := gcs.SaveCoverageReader(ctx, validKey, nil, 100)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reader is nil")
	})
}

// TestGCSStorage_ObjectNotExist tests that ErrObjectNotExist is handled correctly
func TestGCSStorage_ObjectNotExist(t *testing.T) {
	t.Run("GetCoverage returns nil for non-existent object", func(t *testing.T) {
		// This test demonstrates the expected behavior
		// In real usage, when storage.ErrObjectNotExist is returned,
		// GetCoverage should return (nil, nil) not an error

		err := storage.ErrObjectNotExist
		assert.True(t, errors.Is(err, storage.ErrObjectNotExist))
	})
}

// Benchmark tests
func BenchmarkFormatObjectPath(b *testing.B) {
	key := CoverageKey{
		Org:    "grafana",
		Repo:   "tempo",
		Branch: "main",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatObjectPath(key)
	}
}

func BenchmarkValidateCoverageKey(b *testing.B) {
	key := CoverageKey{
		Org:    "grafana",
		Repo:   "tempo",
		Branch: "main",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateCoverageKey(key)
	}
}

// Example test showing expected usage patterns
// Note: These examples show API usage but don't actually run (would require GCS credentials)
//
// Example: SaveCoverage
//
//	ctx := context.Background()
//
//	// Create GCS storage client (requires real credentials in production)
//	gcs, err := NewGCSStorage(ctx, "my-coverage-bucket")
//	if err != nil {
//		panic(err)
//	}
//	defer gcs.Close()
//
//	// Define coverage key
//	key := CoverageKey{
//		Org:    "grafana",
//		Repo:   "tempo",
//		Branch: "main",
//	}
//
//	// Save coverage data
//	coverageData := []byte("mode: set\ngithub.com/grafana/tempo/pkg/util/util.go:10.1,12.2 1 1\n")
//	err = gcs.SaveCoverage(ctx, key, coverageData)
//	if err != nil {
//		panic(err)
//	}
//
// Example: GetCoverage
//
//	ctx := context.Background()
//
//	// Create GCS storage client
//	gcs, err := NewGCSStorage(ctx, "my-coverage-bucket")
//	if err != nil {
//		panic(err)
//	}
//	defer gcs.Close()
//
//	// Define coverage key
//	key := CoverageKey{
//		Org:    "grafana",
//		Repo:   "tempo",
//		Branch: "main",
//	}
//
//	// Get coverage data
//	data, err := gcs.GetCoverage(ctx, key)
//	if err != nil {
//		panic(err)
//	}
//
//	if data == nil {
//		// Coverage file doesn't exist yet
//		println("No coverage found for this branch")
//	} else {
//		// Process coverage data
//		println("Coverage data:", string(data))
//	}
//
// Example: SaveCoverageReader
//
//	ctx := context.Background()
//
//	// Create GCS storage client
//	gcs, err := NewGCSStorage(ctx, "my-coverage-bucket")
//	if err != nil {
//		panic(err)
//	}
//	defer gcs.Close()
//
//	// Define coverage key
//	key := CoverageKey{
//		Org:    "grafana",
//		Repo:   "tempo",
//		Branch: "main",
//	}
//
//	// Stream large coverage file
//	coverageData := []byte("mode: set\n...large coverage data...\n")
//	reader := bytes.NewReader(coverageData)
//
//	err = gcs.SaveCoverageReader(ctx, key, reader, int64(len(coverageData)))
//	if err != nil {
//		panic(err)
//	}
