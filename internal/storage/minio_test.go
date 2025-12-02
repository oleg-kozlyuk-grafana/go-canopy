package storage

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMinIOStorage(t *testing.T) {
	tests := []struct {
		name      string
		config    MinIOConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "empty endpoint",
			config: MinIOConfig{
				Endpoint:        "",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UseSSL:          false,
				Bucket:          "test-bucket",
			},
			wantError: true,
			errorMsg:  "endpoint is required",
		},
		{
			name: "empty access key ID",
			config: MinIOConfig{
				Endpoint:        "localhost:9000",
				AccessKeyID:     "",
				SecretAccessKey: "minioadmin",
				UseSSL:          false,
				Bucket:          "test-bucket",
			},
			wantError: true,
			errorMsg:  "access key ID is required",
		},
		{
			name: "empty secret access key",
			config: MinIOConfig{
				Endpoint:        "localhost:9000",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "",
				UseSSL:          false,
				Bucket:          "test-bucket",
			},
			wantError: true,
			errorMsg:  "secret access key is required",
		},
		{
			name: "empty bucket name",
			config: MinIOConfig{
				Endpoint:        "localhost:9000",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UseSSL:          false,
				Bucket:          "",
			},
			wantError: true,
			errorMsg:  "bucket name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage, err := NewMinIOStorage(ctx, tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, storage)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, storage)
				if storage != nil {
					storage.Close()
				}
			}
		})
	}
}

func TestMinIOStorage_SaveCoverage_Validation(t *testing.T) {
	// Create a mock MinIO storage (client will be nil, but that's ok for validation tests)
	minioStorage := &MinIOStorage{
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
			err := minioStorage.SaveCoverage(ctx, tt.key, tt.data)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			}
		})
	}
}

func TestMinIOStorage_GetCoverage_Validation(t *testing.T) {
	// Create a mock MinIO storage
	minioStorage := &MinIOStorage{
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
			data, err := minioStorage.GetCoverage(ctx, tt.key)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, data)
				assert.Contains(t, err.Error(), tt.errorMsg)
			}
		})
	}
}

func TestMinIOStorage_SaveCoverageReader_Validation(t *testing.T) {
	minioStorage := &MinIOStorage{
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
			err := minioStorage.SaveCoverageReader(ctx, tt.key, tt.reader, tt.size)
			assert.Error(t, err)
			if tt.wantError {
				assert.Contains(t, err.Error(), tt.errorMsg)
			}
		})
	}
}

func TestMinIOStorage_Close(t *testing.T) {
	t.Run("close with nil client", func(t *testing.T) {
		minioStorage := &MinIOStorage{
			client: nil,
			bucket: "test-bucket",
		}
		err := minioStorage.Close()
		assert.NoError(t, err)
	})

	t.Run("close with valid config returns no error", func(t *testing.T) {
		// MinIO client doesn't require explicit cleanup
		minioStorage := &MinIOStorage{
			client: &minio.Client{},
			bucket: "test-bucket",
		}
		err := minioStorage.Close()
		assert.NoError(t, err)
	})
}

// TestMinIOStorage_ErrorHandling tests error conditions that can be unit tested
func TestMinIOStorage_ErrorHandling(t *testing.T) {
	t.Run("SaveCoverage with invalid key returns validation error", func(t *testing.T) {
		minioStorage := &MinIOStorage{bucket: "test-bucket"}
		ctx := context.Background()

		invalidKey := CoverageKey{Org: "", Repo: "repo", Branch: "main"}
		err := minioStorage.SaveCoverage(ctx, invalidKey, []byte("test"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "org is required")
	})

	t.Run("GetCoverage with invalid key returns validation error", func(t *testing.T) {
		minioStorage := &MinIOStorage{bucket: "test-bucket"}
		ctx := context.Background()

		invalidKey := CoverageKey{Org: "org", Repo: "", Branch: "main"}
		data, err := minioStorage.GetCoverage(ctx, invalidKey)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "repo is required")
	})

	t.Run("SaveCoverageReader with nil reader returns error", func(t *testing.T) {
		minioStorage := &MinIOStorage{bucket: "test-bucket"}
		ctx := context.Background()

		validKey := CoverageKey{Org: "org", Repo: "repo", Branch: "main"}
		err := minioStorage.SaveCoverageReader(ctx, validKey, nil, 100)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reader is nil")
	})
}

// TestMinIOStorage_ObjectPath tests that object paths are formatted correctly
func TestMinIOStorage_ObjectPath(t *testing.T) {
	t.Run("object path format is consistent", func(t *testing.T) {
		key := CoverageKey{
			Org:    "grafana",
			Repo:   "tempo",
			Branch: "main",
		}

		expected := "grafana/tempo/main/coverage.out"
		result := formatObjectPath(key)

		assert.Equal(t, expected, result)
	})
}

// Benchmark tests
func BenchmarkMinIOStorage_SaveCoverage_Validation(b *testing.B) {
	minioStorage := &MinIOStorage{bucket: "test-bucket"}
	ctx := context.Background()
	key := CoverageKey{
		Org:    "grafana",
		Repo:   "tempo",
		Branch: "main",
	}
	data := []byte("mode: set\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will fail at the client level, but we're benchmarking validation
		_ = minioStorage.SaveCoverage(ctx, key, data)
	}
}

// Integration tests would go here using testcontainers
// Example integration test structure (not implemented yet):
//
// func TestMinIOStorage_Integration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test")
// 	}
//
// 	ctx := context.Background()
//
// 	// Start MinIO using testcontainers
// 	// container, err := testcontainers.GenericContainer(...)
// 	// require.NoError(t, err)
// 	// defer container.Terminate(ctx)
//
// 	// Get MinIO endpoint and credentials
// 	// endpoint, _ := container.Endpoint(ctx, "")
//
// 	// Create MinIO storage client
// 	// config := MinIOConfig{
// 	// 	Endpoint:        endpoint,
// 	// 	AccessKeyID:     "minioadmin",
// 	// 	SecretAccessKey: "minioadmin",
// 	// 	UseSSL:          false,
// 	// 	Bucket:          "test-bucket",
// 	// }
// 	// storage, err := NewMinIOStorage(ctx, config)
// 	// require.NoError(t, err)
// 	// defer storage.Close()
//
// 	// Run integration tests
// 	t.Run("save and get cycle", func(t *testing.T) {
// 		// Test full save/get cycle
// 		// key := CoverageKey{Org: "test", Repo: "repo", Branch: "main"}
// 		// data := []byte("mode: set\ntest coverage data\n")
// 		//
// 		// err := storage.SaveCoverage(ctx, key, data)
// 		// require.NoError(t, err)
// 		//
// 		// retrieved, err := storage.GetCoverage(ctx, key)
// 		// require.NoError(t, err)
// 		// assert.Equal(t, data, retrieved)
// 	})
//
// 	t.Run("get non-existent file", func(t *testing.T) {
// 		// Test getting a file that doesn't exist
// 		// key := CoverageKey{Org: "test", Repo: "repo", Branch: "nonexistent"}
// 		//
// 		// retrieved, err := storage.GetCoverage(ctx, key)
// 		// require.NoError(t, err)
// 		// assert.Nil(t, retrieved)
// 	})
//
// 	t.Run("overwrite existing file", func(t *testing.T) {
// 		// Test overwriting an existing file
// 		// key := CoverageKey{Org: "test", Repo: "repo", Branch: "main"}
// 		// data1 := []byte("first version")
// 		// data2 := []byte("second version")
// 		//
// 		// err := storage.SaveCoverage(ctx, key, data1)
// 		// require.NoError(t, err)
// 		//
// 		// err = storage.SaveCoverage(ctx, key, data2)
// 		// require.NoError(t, err)
// 		//
// 		// retrieved, err := storage.GetCoverage(ctx, key)
// 		// require.NoError(t, err)
// 		// assert.Equal(t, data2, retrieved)
// 	})
//
// 	t.Run("save and get with reader", func(t *testing.T) {
// 		// Test SaveCoverageReader and GetCoverage
// 		// key := CoverageKey{Org: "test", Repo: "repo", Branch: "feature"}
// 		// data := []byte("mode: set\nstreaming coverage data\n")
// 		// reader := bytes.NewReader(data)
// 		//
// 		// err := storage.SaveCoverageReader(ctx, key, reader, int64(len(data)))
// 		// require.NoError(t, err)
// 		//
// 		// retrieved, err := storage.GetCoverage(ctx, key)
// 		// require.NoError(t, err)
// 		// assert.Equal(t, data, retrieved)
// 	})
//
// 	t.Run("list coverage files", func(t *testing.T) {
// 		// Test ListCoverageFiles
// 		// key1 := CoverageKey{Org: "test", Repo: "repo1", Branch: "main"}
// 		// key2 := CoverageKey{Org: "test", Repo: "repo2", Branch: "main"}
// 		//
// 		// err := storage.SaveCoverage(ctx, key1, []byte("data1"))
// 		// require.NoError(t, err)
// 		//
// 		// err = storage.SaveCoverage(ctx, key2, []byte("data2"))
// 		// require.NoError(t, err)
// 		//
// 		// files, err := storage.ListCoverageFiles(ctx, "test/")
// 		// require.NoError(t, err)
// 		// assert.Len(t, files, 2)
// 	})
// }

// Example test showing expected usage patterns
// Note: These examples show API usage but don't actually run (would require MinIO credentials)
//
// Example: SaveCoverage
//
//	ctx := context.Background()
//
//	// Create MinIO storage client
//	config := MinIOConfig{
//		Endpoint:        "localhost:9000",
//		AccessKeyID:     "minioadmin",
//		SecretAccessKey: "minioadmin",
//		UseSSL:          false,
//		Bucket:          "coverage-bucket",
//	}
//	minioStorage, err := NewMinIOStorage(ctx, config)
//	if err != nil {
//		panic(err)
//	}
//	defer minioStorage.Close()
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
//	err = minioStorage.SaveCoverage(ctx, key, coverageData)
//	if err != nil {
//		panic(err)
//	}
//
// Example: GetCoverage
//
//	ctx := context.Background()
//
//	// Create MinIO storage client
//	config := MinIOConfig{
//		Endpoint:        "localhost:9000",
//		AccessKeyID:     "minioadmin",
//		SecretAccessKey: "minioadmin",
//		UseSSL:          false,
//		Bucket:          "coverage-bucket",
//	}
//	minioStorage, err := NewMinIOStorage(ctx, config)
//	if err != nil {
//		panic(err)
//	}
//	defer minioStorage.Close()
//
//	// Define coverage key
//	key := CoverageKey{
//		Org:    "grafana",
//		Repo:   "tempo",
//		Branch: "main",
//	}
//
//	// Get coverage data
//	data, err := minioStorage.GetCoverage(ctx, key)
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
//	// Create MinIO storage client
//	config := MinIOConfig{
//		Endpoint:        "localhost:9000",
//		AccessKeyID:     "minioadmin",
//		SecretAccessKey: "minioadmin",
//		UseSSL:          false,
//		Bucket:          "coverage-bucket",
//	}
//	minioStorage, err := NewMinIOStorage(ctx, config)
//	if err != nil {
//		panic(err)
//	}
//	defer minioStorage.Close()
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
//	err = minioStorage.SaveCoverageReader(ctx, key, reader, int64(len(coverageData)))
//	if err != nil {
//		panic(err)
//	}
