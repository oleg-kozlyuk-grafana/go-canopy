package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage implements the Storage interface using MinIO (S3-compatible storage).
type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// MinIOConfig holds the configuration for MinIO client initialization.
type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	Bucket          string
}

// NewMinIOStorage creates a new MinIO storage client.
// The config parameter specifies the MinIO connection details.
func NewMinIOStorage(ctx context.Context, config MinIOConfig) (*MinIOStorage, error) {
	if config.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	if config.AccessKeyID == "" {
		return nil, errors.New("access key ID is required")
	}
	if config.SecretAccessKey == "" {
		return nil, errors.New("secret access key is required")
	}
	if config.Bucket == "" {
		return nil, errors.New("bucket name is required")
	}

	// Initialize MinIO client
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Check if bucket exists, create if it doesn't
	exists, err := client.BucketExists(ctx, config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, config.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %w", config.Bucket, err)
		}
	}

	return &MinIOStorage{
		client: client,
		bucket: config.Bucket,
	}, nil
}

// SaveCoverage stores coverage data for the given key.
// Path format: {org}/{repo}/{branch}/coverage.out
func (m *MinIOStorage) SaveCoverage(ctx context.Context, key CoverageKey, data []byte) error {
	if err := validateCoverageKey(key); err != nil {
		return err
	}

	objectPath := formatObjectPath(key)
	reader := bytes.NewReader(data)

	_, err := m.client.PutObject(ctx, m.bucket, objectPath, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO object %s: %w", objectPath, err)
	}

	return nil
}

// GetCoverage retrieves coverage data for the given key.
// Returns nil if the coverage file does not exist.
// Returns an error if the retrieval operation fails (excluding not-found).
func (m *MinIOStorage) GetCoverage(ctx context.Context, key CoverageKey) ([]byte, error) {
	if err := validateCoverageKey(key); err != nil {
		return nil, err
	}

	objectPath := formatObjectPath(key)

	obj, err := m.client.GetObject(ctx, m.bucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get MinIO object %s: %w", objectPath, err)
	}
	defer obj.Close()

	// Try to read the object to check if it exists
	data, err := io.ReadAll(obj)
	if err != nil {
		// Check if the error is because the object doesn't exist
		var minioErr minio.ErrorResponse
		if errors.As(err, &minioErr) && minioErr.Code == "NoSuchKey" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read MinIO object %s: %w", objectPath, err)
	}

	return data, nil
}

// SaveCoverageReader stores coverage data from a reader.
// This is useful for streaming large coverage files without loading them into memory.
// The size parameter helps MinIO optimize the upload.
func (m *MinIOStorage) SaveCoverageReader(ctx context.Context, key CoverageKey, reader io.Reader, size int64) error {
	if err := validateCoverageKey(key); err != nil {
		return err
	}

	if reader == nil {
		return errors.New("reader is nil")
	}

	objectPath := formatObjectPath(key)

	// If size is not provided, MinIO will use chunked upload
	uploadSize := size
	if uploadSize < 0 {
		uploadSize = -1 // MinIO uses -1 to indicate unknown size
	}

	_, err := m.client.PutObject(ctx, m.bucket, objectPath, reader, uploadSize, minio.PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		return fmt.Errorf("failed to upload stream to MinIO object %s: %w", objectPath, err)
	}

	return nil
}

// Close releases resources held by the storage client.
// MinIO client doesn't require explicit cleanup, but we implement this for interface compliance.
func (m *MinIOStorage) Close() error {
	// MinIO client doesn't require explicit cleanup
	return nil
}

// ListCoverageFiles lists all coverage files in the bucket.
// This is primarily useful for debugging and testing.
// Returns a slice of object paths.
func (m *MinIOStorage) ListCoverageFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	objectCh := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		files = append(files, object.Key)
	}

	return files, nil
}
