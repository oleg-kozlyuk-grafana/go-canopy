package storage

import (
	"context"
	"errors"
	"fmt"
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

// FormatObjectPath creates the object path from a coverage key.
// Format: {org}/{repo}/{branch}/coverage.out
func FormatObjectPath(key CoverageKey) string {
	return fmt.Sprintf("%s/%s/%s/coverage.out", key.Org, key.Repo, key.Branch)
}

// ValidateCoverageKey validates that the coverage key fields are not empty.
func ValidateCoverageKey(key CoverageKey) error {
	if key.Org == "" {
		return errors.New("org is required")
	}
	if key.Repo == "" {
		return errors.New("repo is required")
	}
	if key.Branch == "" {
		return errors.New("branch is required")
	}
	return nil
}
