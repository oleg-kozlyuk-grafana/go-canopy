package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	storagepkg "github.com/oleg-kozlyuk-grafana/go-canopy/internal/storage"
)

// GCSStorage implements the Storage interface using Google Cloud Storage.
type GCSStorage struct {
	client *storage.Client
	bucket string
}

// NewGCSStorage creates a new GCS storage client.
// The bucket parameter specifies the GCS bucket name.
// It uses Application Default Credentials (ADC) for authentication.
func NewGCSStorage(ctx context.Context, bucket string) (*GCSStorage, error) {
	if bucket == "" {
		return nil, errors.New("bucket name is required")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCSStorage{
		client: client,
		bucket: bucket,
	}, nil
}

// SaveCoverage stores coverage data for the given key.
// Path format: {org}/{repo}/{branch}/coverage.out
func (g *GCSStorage) SaveCoverage(ctx context.Context, key storagepkg.CoverageKey, data []byte) error {
	if err := storagepkg.ValidateCoverageKey(key); err != nil {
		return err
	}

	objectPath := storagepkg.FormatObjectPath(key)
	obj := g.client.Bucket(g.bucket).Object(objectPath)

	w := obj.NewWriter(ctx)
	if _, err := w.Write(data); err != nil {
		w.Close()
		return fmt.Errorf("failed to write to GCS object %s: %w", objectPath, err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close GCS writer for %s: %w", objectPath, err)
	}

	return nil
}

// GetCoverage retrieves coverage data for the given key.
// Returns nil if the coverage file does not exist.
// Returns an error if the retrieval operation fails (excluding not-found).
func (g *GCSStorage) GetCoverage(ctx context.Context, key storagepkg.CoverageKey) ([]byte, error) {
	if err := storagepkg.ValidateCoverageKey(key); err != nil {
		return nil, err
	}

	objectPath := storagepkg.FormatObjectPath(key)
	obj := g.client.Bucket(g.bucket).Object(objectPath)

	r, err := obj.NewReader(ctx)
	if err != nil {
		// Return nil if object doesn't exist (not an error according to interface)
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open GCS object %s: %w", objectPath, err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read GCS object %s: %w", objectPath, err)
	}

	return data, nil
}

// SaveCoverageReader stores coverage data from a reader.
// This is useful for streaming large coverage files without loading them into memory.
// The size parameter helps GCS optimize the upload.
func (g *GCSStorage) SaveCoverageReader(ctx context.Context, key storagepkg.CoverageKey, reader io.Reader, size int64) error {
	if err := storagepkg.ValidateCoverageKey(key); err != nil {
		return err
	}

	if reader == nil {
		return errors.New("reader is nil")
	}

	objectPath := storagepkg.FormatObjectPath(key)
	obj := g.client.Bucket(g.bucket).Object(objectPath)

	w := obj.NewWriter(ctx)
	if size > 0 {
		// Set the size hint for better performance
		w.Size = size
	}

	if _, err := io.Copy(w, reader); err != nil {
		w.Close()
		return fmt.Errorf("failed to copy data to GCS object %s: %w", objectPath, err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close GCS writer for %s: %w", objectPath, err)
	}

	return nil
}

// Close releases resources held by the storage client.
func (g *GCSStorage) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}

// ListCoverageFiles lists all coverage files in the bucket.
// This is primarily useful for debugging and testing.
// Returns a slice of object paths.
func (g *GCSStorage) ListCoverageFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}
		files = append(files, attrs.Name)
	}

	return files, nil
}
