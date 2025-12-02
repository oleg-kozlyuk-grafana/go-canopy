package queue

import (
	"context"
	"fmt"
	"sync"
)

// InMemoryQueue implements MessageQueue using an in-memory channel.
// This is designed for all-in-one mode where the initiator and worker
// run in the same process. It provides synchronous processing without
// external dependencies.
type InMemoryQueue struct {
	ch     chan *WorkRequest
	closed bool
	mu     sync.RWMutex
}

// InMemoryConfig holds configuration for creating an InMemoryQueue.
type InMemoryConfig struct {
	// BufferSize is the channel buffer size (default: 100)
	BufferSize int
}

// NewInMemoryQueue creates a new InMemoryQueue instance.
// The caller is responsible for calling Close() when done.
func NewInMemoryQueue(cfg InMemoryConfig) *InMemoryQueue {
	bufferSize := cfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 100 // Default buffer size
	}

	return &InMemoryQueue{
		ch:     make(chan *WorkRequest, bufferSize),
		closed: false,
	}
}

// Publish sends a WorkRequest message to the in-memory queue.
func (q *InMemoryQueue) Publish(ctx context.Context, req *WorkRequest) error {
	if req == nil {
		return fmt.Errorf("work request cannot be nil")
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	select {
	case q.ch <- req:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("publish cancelled: %w", ctx.Err())
	}
}

// Subscribe starts consuming messages from the in-memory queue.
// It calls the handler function for each received message.
// This method blocks until the context is cancelled, the queue is closed,
// or an error occurs.
func (q *InMemoryQueue) Subscribe(ctx context.Context, handler func(context.Context, *WorkRequest) error) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	for {
		select {
		case req, ok := <-q.ch:
			if !ok {
				// Channel closed, exit gracefully
				return nil
			}

			// Call the handler to process the message
			// Note: In-memory queue doesn't support retries like Pub/Sub or Redis
			// If the handler returns an error, we just continue to the next message
			if err := handler(ctx, req); err != nil {
				// In production, you'd want to log this error
				// For in-memory queue, we don't retry failed messages
				continue
			}

		case <-ctx.Done():
			// Context cancelled, exit
			return ctx.Err()
		}
	}
}

// Close releases resources held by the InMemoryQueue.
// It closes the channel and prevents further publishing.
func (q *InMemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.ch)
	return nil
}
