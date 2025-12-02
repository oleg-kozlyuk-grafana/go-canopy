package storage

import (
	"context"
	"io"
)

// CoverageKey uniquely identifies a coverage file in storage.
// Storage path format: {org}/{repo}/{branch}/coverage.out
type CoverageKey struct {
	Org    string
	Repo   string
	Branch string
}

// Storage defines the interface for coverage data persistence.
// Implementations include GCS for production and MinIO for local development.
type Storage interface {
	// SaveCoverage stores coverage data for the given key.
	// The data parameter contains the raw coverage profile content.
	// Returns an error if the save operation fails.
	SaveCoverage(ctx context.Context, key CoverageKey, data []byte) error

	// GetCoverage retrieves coverage data for the given key.
	// Returns nil if the coverage file does not exist.
	// Returns an error if the retrieval operation fails (excluding not-found).
	GetCoverage(ctx context.Context, key CoverageKey) ([]byte, error)

	// SaveCoverageReader stores coverage data from a reader.
	// This is useful for streaming large coverage files without loading them into memory.
	// Returns an error if the save operation fails.
	SaveCoverageReader(ctx context.Context, key CoverageKey, reader io.Reader, size int64) error

	// Close releases any resources held by the storage client.
	// Should be called when the storage client is no longer needed.
	Close() error
}
