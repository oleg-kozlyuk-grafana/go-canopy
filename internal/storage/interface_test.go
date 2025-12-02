package storage

import (
	"context"
	"errors"
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

// keyToPath converts a CoverageKey to a storage path.
func keyToPath(key CoverageKey) string {
	return key.Org + "/" + key.Repo + "/" + key.Branch + "/coverage.out"
}

// SaveCoverage implements Storage.SaveCoverage.
func (m *MockStorage) SaveCoverage(ctx context.Context, key CoverageKey, data []byte) error {
	m.saveCalled = true
	if m.saveErr != nil {
		return m.saveErr
	}
	m.data[keyToPath(key)] = data
	return nil
}

// GetCoverage implements Storage.GetCoverage.
func (m *MockStorage) GetCoverage(ctx context.Context, key CoverageKey) ([]byte, error) {
	m.getCalled = true
	if m.getErr != nil {
		return nil, m.getErr
	}
	data, exists := m.data[keyToPath(key)]
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

// TestCoverageKey tests the CoverageKey struct.
func TestCoverageKey(t *testing.T) {
	tests := []struct {
		name     string
		key      CoverageKey
		expected string
	}{
		{
			name: "valid key",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "mimir",
				Branch: "main",
			},
			expected: "grafana/mimir/main/coverage.out",
		},
		{
			name: "different branch",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "loki",
				Branch: "feature-branch",
			},
			expected: "grafana/loki/feature-branch/coverage.out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := keyToPath(tt.key)
			assert.Equal(t, tt.expected, path)
		})
	}
}

// TestMockStorage_SaveAndGet tests save and get operations.
func TestMockStorage_SaveAndGet(t *testing.T) {
	tests := []struct {
		name     string
		key      CoverageKey
		data     []byte
		wantData []byte
		wantErr  bool
	}{
		{
			name: "save and retrieve coverage",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "mimir",
				Branch: "main",
			},
			data:     []byte("mode: set\ngrafana.com/project/file.go:10.2,12.3 1 1"),
			wantData: []byte("mode: set\ngrafana.com/project/file.go:10.2,12.3 1 1"),
			wantErr:  false,
		},
		{
			name: "save empty coverage",
			key: CoverageKey{
				Org:    "grafana",
				Repo:   "loki",
				Branch: "develop",
			},
			data:     []byte{},
			wantData: []byte{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := NewMockStorage()

			// Test SaveCoverage
			err := storage.SaveCoverage(ctx, tt.key, tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, storage.saveCalled)

			// Test GetCoverage
			retrieved, err := storage.GetCoverage(ctx, tt.key)
			require.NoError(t, err)
			assert.True(t, storage.getCalled)
			assert.Equal(t, tt.wantData, retrieved)
		})
	}
}

// TestMockStorage_GetNotFound tests getting non-existent coverage.
func TestMockStorage_GetNotFound(t *testing.T) {
	ctx := context.Background()
	storage := NewMockStorage()

	key := CoverageKey{
		Org:    "grafana",
		Repo:   "mimir",
		Branch: "main",
	}

	// Get coverage that doesn't exist
	data, err := storage.GetCoverage(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, data, "expected nil data for non-existent coverage")
	assert.True(t, storage.getCalled)
}

// TestMockStorage_SaveError tests save operation errors.
func TestMockStorage_SaveError(t *testing.T) {
	ctx := context.Background()
	storage := NewMockStorage()

	expectedErr := errors.New("storage unavailable")
	storage.SetSaveError(expectedErr)

	key := CoverageKey{
		Org:    "grafana",
		Repo:   "mimir",
		Branch: "main",
	}
	data := []byte("coverage data")

	err := storage.SaveCoverage(ctx, key, data)
	assert.ErrorIs(t, err, expectedErr)
	assert.True(t, storage.saveCalled)
}

// TestMockStorage_GetError tests get operation errors.
func TestMockStorage_GetError(t *testing.T) {
	ctx := context.Background()
	storage := NewMockStorage()

	expectedErr := errors.New("network error")
	storage.SetGetError(expectedErr)

	key := CoverageKey{
		Org:    "grafana",
		Repo:   "mimir",
		Branch: "main",
	}

	data, err := storage.GetCoverage(ctx, key)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, data)
	assert.True(t, storage.getCalled)
}

// TestMockStorage_Close tests the close operation.
func TestMockStorage_Close(t *testing.T) {
	tests := []struct {
		name     string
		closeErr error
		wantErr  bool
	}{
		{
			name:     "successful close",
			closeErr: nil,
			wantErr:  false,
		},
		{
			name:     "close error",
			closeErr: errors.New("close failed"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMockStorage()
			if tt.closeErr != nil {
				storage.SetCloseError(tt.closeErr)
			}

			err := storage.Close()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.True(t, storage.closeCalled)
		})
	}
}

// TestMockStorage_SaveCoverageReader tests saving coverage from a reader.
func TestMockStorage_SaveCoverageReader(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		size    int64
		wantErr bool
	}{
		{
			name:    "save from reader",
			data:    "mode: set\ngrafana.com/project/file.go:10.2,12.3 1 1",
			size:    55,
			wantErr: false,
		},
		{
			name:    "save empty from reader",
			data:    "",
			size:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := NewMockStorage()

			key := CoverageKey{
				Org:    "grafana",
				Repo:   "mimir",
				Branch: "main",
			}

			reader := strings.NewReader(tt.data)
			err := storage.SaveCoverageReader(ctx, key, reader, tt.size)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify data was saved correctly
			retrieved, err := storage.GetCoverage(ctx, key)
			require.NoError(t, err)
			assert.Equal(t, []byte(tt.data), retrieved)
		})
	}
}

// TestMockStorage_MultipleKeys tests storing coverage for multiple keys.
func TestMockStorage_MultipleKeys(t *testing.T) {
	ctx := context.Background()
	storage := NewMockStorage()

	keys := []CoverageKey{
		{Org: "grafana", Repo: "mimir", Branch: "main"},
		{Org: "grafana", Repo: "loki", Branch: "main"},
		{Org: "grafana", Repo: "mimir", Branch: "develop"},
	}

	// Save different data for each key
	for i, key := range keys {
		data := []byte("coverage data " + string(rune('A'+i)))
		err := storage.SaveCoverage(ctx, key, data)
		require.NoError(t, err)
	}

	// Verify each key returns its own data
	for i, key := range keys {
		expected := []byte("coverage data " + string(rune('A'+i)))
		retrieved, err := storage.GetCoverage(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, expected, retrieved)
	}
}

// TestMockStorage_OverwriteData tests overwriting existing coverage data.
func TestMockStorage_OverwriteData(t *testing.T) {
	ctx := context.Background()
	storage := NewMockStorage()

	key := CoverageKey{
		Org:    "grafana",
		Repo:   "mimir",
		Branch: "main",
	}

	// Save initial data
	initialData := []byte("initial coverage")
	err := storage.SaveCoverage(ctx, key, initialData)
	require.NoError(t, err)

	// Verify initial data
	retrieved, err := storage.GetCoverage(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, initialData, retrieved)

	// Overwrite with new data
	newData := []byte("updated coverage")
	err = storage.SaveCoverage(ctx, key, newData)
	require.NoError(t, err)

	// Verify new data
	retrieved, err = storage.GetCoverage(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, newData, retrieved)
}
