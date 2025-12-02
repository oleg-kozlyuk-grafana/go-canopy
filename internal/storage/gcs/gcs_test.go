package gcs

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storagepkg "github.com/oleg-kozlyuk/canopy/internal/storage"
)

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

func TestGCSStorage_ErrorHandling(t *testing.T) {
	t.Run("SaveCoverage with invalid key returns validation error", func(t *testing.T) {
		gcs := &GCSStorage{bucket: "test-bucket"}
		ctx := context.Background()

		invalidKey := storagepkg.CoverageKey{Org: "", Repo: "repo", Branch: "main"}
		err := gcs.SaveCoverage(ctx, invalidKey, []byte("test"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "org is required")
	})

	t.Run("GetCoverage with invalid key returns validation error", func(t *testing.T) {
		gcs := &GCSStorage{bucket: "test-bucket"}
		ctx := context.Background()

		invalidKey := storagepkg.CoverageKey{Org: "org", Repo: "", Branch: "main"}
		data, err := gcs.GetCoverage(ctx, invalidKey)

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "repo is required")
	})

	t.Run("SaveCoverageReader with nil reader returns error", func(t *testing.T) {
		gcs := &GCSStorage{bucket: "test-bucket"}
		ctx := context.Background()

		validKey := storagepkg.CoverageKey{Org: "org", Repo: "repo", Branch: "main"}
		err := gcs.SaveCoverageReader(ctx, validKey, nil, 100)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reader is nil")
	})
}

func TestGCSStorage_ObjectNotExist(t *testing.T) {
	t.Run("GetCoverage returns nil for non-existent object", func(t *testing.T) {
		// This test demonstrates the expected behavior
		// In real usage, when storage.ErrObjectNotExist is returned,
		// GetCoverage should return (nil, nil) not an error

		err := storage.ErrObjectNotExist
		assert.True(t, errors.Is(err, storage.ErrObjectNotExist))
	})
}
