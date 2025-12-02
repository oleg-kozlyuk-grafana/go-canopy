package storage

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStorage is a mock implementation of Storage for testing.
type MockStorage struct {
	data         map[string][]byte
	saveErr      error
	getErr       error
	closeErr     error
	saveCalled   bool
	getCalled    bool
	closeCalled  bool
	saveReaderFn func(ctx context.Context, key CoverageKey, reader io.Reader, size int64) error
}

// NewMockStorage creates a new mock storage instance.
func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string][]byte),
	}
}

// SaveCoverage implements Storage.SaveCoverage.
func (m *MockStorage) SaveCoverage(ctx context.Context, key CoverageKey, data []byte) error {
	m.saveCalled = true
	if m.saveErr != nil {
		return m.saveErr
	}
	m.data[FormatObjectPath(key)] = data
	return nil
}

// GetCoverage implements Storage.GetCoverage.
func (m *MockStorage) GetCoverage(ctx context.Context, key CoverageKey) ([]byte, error) {
	m.getCalled = true
	if m.getErr != nil {
		return nil, m.getErr
	}
	data, exists := m.data[FormatObjectPath(key)]
	if !exists {
		return nil, nil
	}
	return data, nil
}

// SaveCoverageReader implements Storage.SaveCoverageReader.
func (m *MockStorage) SaveCoverageReader(ctx context.Context, key CoverageKey, reader io.Reader, size int64) error {
	if m.saveReaderFn != nil {
		return m.saveReaderFn(ctx, key, reader, size)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return m.SaveCoverage(ctx, key, data)
}

// Close implements Storage.Close.
func (m *MockStorage) Close() error {
	m.closeCalled = true
	return m.closeErr
}

// SetSaveError configures the mock to return an error on save.
func (m *MockStorage) SetSaveError(err error) {
	m.saveErr = err
}

// SetGetError configures the mock to return an error on get.
func (m *MockStorage) SetGetError(err error) {
	m.getErr = err
}

// SetCloseError configures the mock to return an error on close.
func (m *MockStorage) SetCloseError(err error) {
	m.closeErr = err
}

// TestFormatObjectPath tests the FormatObjectPath helper function.
func TestFormatObjectPath(t *testing.T) {
	tests := []struct {
		name     string
		key      CoverageKey
		expected string
	}{
		{
			name: "basic key",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "mimir",
				Branch: "main",
			},
			expected: "grafana/mimir/main/coverage.out",
		},
		{
			name: "feature branch",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "loki",
				Branch: "feature-branch",
			},
			expected: "grafana/loki/feature-branch/coverage.out",
		},
		{
			name: "branch with slashes",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "tempo",
				Branch: "release/v1.0.0",
			},
			expected: "grafana/tempo/release/v1.0.0/coverage.out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := FormatObjectPath(tt.key)
			assert.Equal(t, tt.expected, path)
		})
	}
}

// TestValidateCoverageKey tests the ValidateCoverageKey helper function.
func TestValidateCoverageKey(t *testing.T) {
	tests := []struct {
		name    string
		key     CoverageKey
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid key",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "mimir",
				Branch: "main",
			},
			wantErr: false,
		},
		{
			name: "empty org",
			key: CoverageKey{
				Org:    "",
				Repo:   "mimir",
				Branch: "main",
			},
			wantErr: true,
			errMsg:  "org is required",
		},
		{
			name: "empty repo",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "",
				Branch: "main",
			},
			wantErr: true,
			errMsg:  "repo is required",
		},
		{
			name: "empty branch",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "mimir",
				Branch: "",
			},
			wantErr: true,
			errMsg:  "branch is required",
		},
		{
			name:    "all empty",
			key:     CoverageKey{},
			wantErr: true,
			errMsg:  "org is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCoverageKey(tt.key)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMockStorage_SaveAndGet tests the mock storage implementation.
func TestMockStorage_SaveAndGet(t *testing.T) {
	ctx := context.Background()
	storage := NewMockStorage()

	key := CoverageKey{
		Org:    "grafana",
		Repo:   "mimir",
		Branch: "main",
	}
	data := []byte("mode: set\ngrafana.com/project/file.go:10.2,12.3 1 1")

	// Test SaveCoverage
	err := storage.SaveCoverage(ctx, key, data)
	require.NoError(t, err)
	assert.True(t, storage.saveCalled)

	// Test GetCoverage
	retrieved, err := storage.GetCoverage(ctx, key)
	require.NoError(t, err)
	assert.True(t, storage.getCalled)
	assert.Equal(t, data, retrieved)
}

// TestMockStorage_SaveCoverageReader tests saving coverage from a reader.
func TestMockStorage_SaveCoverageReader(t *testing.T) {
	ctx := context.Background()
	storage := NewMockStorage()

	key := CoverageKey{
		Org:    "grafana",
		Repo:   "mimir",
		Branch: "main",
	}
	data := "mode: set\ngrafana.com/project/file.go:10.2,12.3 1 1"

	reader := strings.NewReader(data)
	err := storage.SaveCoverageReader(ctx, key, reader, int64(len(data)))
	require.NoError(t, err)

	// Verify data was saved correctly
	retrieved, err := storage.GetCoverage(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, []byte(data), retrieved)
}
