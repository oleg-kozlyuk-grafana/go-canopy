package queue

import (
	"context"
)

// WorkRequest represents a message containing information about a workflow run
// that needs coverage processing.
type WorkRequest struct {
	// Organization name (e.g., "myorg")
	Org string `json:"org"`

	// Repository name (e.g., "myrepo")
	Repo string `json:"repo"`

	// GitHub workflow run ID
	WorkflowRunID int64 `json:"workflow_run_id"`
}

// MessageQueue defines the interface for queue operations.
// Implementations include GCP Pub/Sub, Redis, and in-memory queues.
type MessageQueue interface {
	// Publish sends a WorkRequest message to the queue.
	// Returns an error if the publish operation fails.
	Publish(ctx context.Context, req *WorkRequest) error

	// Subscribe starts consuming messages from the queue and calls the handler
	// function for each received message. The handler should process the message
	// and return an error if processing fails (which may trigger retries depending
	// on the queue implementation).
	//
	// This method blocks until the context is cancelled or an unrecoverable error occurs.
	Subscribe(ctx context.Context, handler func(context.Context, *WorkRequest) error) error

	// Close releases any resources held by the queue client.
	// After Close is called, the MessageQueue should not be used.
	Close() error
}
