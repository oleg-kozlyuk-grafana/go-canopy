package minio

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storagepkg "github.com/oleg-kozlyuk-grafana/go-canopy/internal/storage"
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
		key       storagepkg.CoverageKey
		data      []byte
		wantError bool
		errorMsg  string
	}{
		{
			name: "invalid key - missing org",
			key: storagepkg.CoverageKey{
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
			key: storagepkg.CoverageKey{
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
			key: storagepkg.CoverageKey{
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
		key       storagepkg.CoverageKey
		wantError bool
		errorMsg  string
	}{
		{
			name: "invalid key - missing org",
			key: storagepkg.CoverageKey{
				Org:    "",
				Repo:   "tempo",
				Branch: "main",
			},
			wantError: true,
			errorMsg:  "org is required",
		},
		{
			name: "invalid key - missing repo",
			key: storagepkg.CoverageKey{
				Org:    "grafana",
				Repo:   "",
				Branch: "main",
			},
			wantError: true,
			errorMsg:  "repo is required",
		},
		{
			name: "invalid key - missing branch",
			key: storagepkg.CoverageKey{
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
		key       storagepkg.CoverageKey
		reader    io.Reader
		size      int64
		wantError bool
		errorMsg  string
	}{
		{
			name: "invalid key - missing org",
			key: storagepkg.CoverageKey{
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
			key: storagepkg.CoverageKey{
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
			key: storagepkg.CoverageKey{
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

func TestMinIOStorage_ErrorHandling(t *testing.T) {
	t.Run("SaveCoverage with invalid key returns validation error", func(t *testing.T) {
		minioStorage := &MinIOStorage{bucket: "test-bucket"}
		ctx := context.Background()

		invalidKey := storagepkg.CoverageKey{Org: "", Repo: "repo", Branch: "main"}
		err := minioStorage.SaveCoverage(ctx, invalidKey, []byte("test"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "org is required")
	})

	t.Run("GetCoverage with invalid key returns validation error", func(t *testing.T) {
		minioStorage := &MinIOStorage{bucket: "test-bucket"}
		ctx := context.Background()

		invalidKey := storagepkg.CoverageKey{Org: "org", Repo: "", Branch: "main"}
		data, err := minioStorage.GetCoverage(ctx, invalidKey)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "repo is required")
	})

	t.Run("SaveCoverageReader with nil reader returns error", func(t *testing.T) {
		minioStorage := &MinIOStorage{bucket: "test-bucket"}
		ctx := context.Background()

		validKey := storagepkg.CoverageKey{Org: "org", Repo: "repo", Branch: "main"}
		err := minioStorage.SaveCoverageReader(ctx, validKey, nil, 100)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reader is nil")
	})
}
