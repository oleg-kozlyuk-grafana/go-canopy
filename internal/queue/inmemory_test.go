package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryQueue(t *testing.T) {
	t.Run("default buffer size", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{})
		require.NotNil(t, q)
		assert.NotNil(t, q.ch)
		assert.False(t, q.closed)
		assert.Equal(t, 100, cap(q.ch)) // Default buffer size
	})

	t.Run("custom buffer size", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{BufferSize: 50})
		require.NotNil(t, q)
		assert.Equal(t, 50, cap(q.ch))
	})

	t.Run("zero buffer size uses default", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{BufferSize: 0})
		require.NotNil(t, q)
		assert.Equal(t, 100, cap(q.ch))
	})

	t.Run("negative buffer size uses default", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{BufferSize: -10})
		require.NotNil(t, q)
		assert.Equal(t, 100, cap(q.ch))
	})
}

func TestInMemoryQueue_PublishValidation(t *testing.T) {
	q := NewInMemoryQueue(InMemoryConfig{})
	ctx := context.Background()

	t.Run("nil work request", func(t *testing.T) {
		err := q.Publish(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work request cannot be nil")
	})

	t.Run("publish to closed queue", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{})
		err := q.Close()
		require.NoError(t, err)

		req := &WorkRequest{
			Org:           "test-org",
			Repo:          "test-repo",
			WorkflowRunID: 12345,
		}

		err = q.Publish(ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "queue is closed")
	})

	t.Run("publish with cancelled context", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{BufferSize: 1})

		// Fill the buffer
		req1 := &WorkRequest{Org: "org1", Repo: "repo1", WorkflowRunID: 1}
		err := q.Publish(context.Background(), req1)
		require.NoError(t, err)

		// Try to publish with cancelled context (buffer is full)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req2 := &WorkRequest{Org: "org2", Repo: "repo2", WorkflowRunID: 2}
		err = q.Publish(ctx, req2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "publish cancelled")
	})
}

func TestInMemoryQueue_SubscribeValidation(t *testing.T) {
	q := NewInMemoryQueue(InMemoryConfig{})
	ctx := context.Background()

	t.Run("nil handler", func(t *testing.T) {
		err := q.Subscribe(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "handler cannot be nil")
	})
}

func TestInMemoryQueue_PublishAndSubscribe(t *testing.T) {
	t.Run("single message", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{})
		ctx := context.Background()

		// Publish a message
		req := &WorkRequest{
			Org:           "test-org",
			Repo:          "test-repo",
			WorkflowRunID: 12345,
		}

		err := q.Publish(ctx, req)
		require.NoError(t, err)

		// Subscribe and receive the message
		received := make(chan *WorkRequest, 1)
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		go func() {
			err := q.Subscribe(subCtx, func(ctx context.Context, r *WorkRequest) error {
				received <- r
				subCancel() // Stop after receiving one message
				return nil
			})
			assert.ErrorIs(t, err, context.Canceled)
		}()

		// Wait for the message
		select {
		case r := <-received:
			assert.Equal(t, req.Org, r.Org)
			assert.Equal(t, req.Repo, r.Repo)
			assert.Equal(t, req.WorkflowRunID, r.WorkflowRunID)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("multiple messages", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{})
		ctx := context.Background()

		// Publish multiple messages
		messages := []*WorkRequest{
			{Org: "org1", Repo: "repo1", WorkflowRunID: 1},
			{Org: "org2", Repo: "repo2", WorkflowRunID: 2},
			{Org: "org3", Repo: "repo3", WorkflowRunID: 3},
		}

		for _, msg := range messages {
			err := q.Publish(ctx, msg)
			require.NoError(t, err)
		}

		// Subscribe and receive all messages
		received := make([]*WorkRequest, 0, len(messages))
		var mu sync.Mutex
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		go func() {
			err := q.Subscribe(subCtx, func(ctx context.Context, r *WorkRequest) error {
				mu.Lock()
				received = append(received, r)
				if len(received) == len(messages) {
					subCancel() // Stop after receiving all messages
				}
				mu.Unlock()
				return nil
			})
			assert.ErrorIs(t, err, context.Canceled)
		}()

		// Wait for all messages
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(received) == len(messages)
		}, 1*time.Second, 10*time.Millisecond)

		// Verify all messages were received in order
		for i, msg := range messages {
			assert.Equal(t, msg.Org, received[i].Org)
			assert.Equal(t, msg.Repo, received[i].Repo)
			assert.Equal(t, msg.WorkflowRunID, received[i].WorkflowRunID)
		}
	})

	t.Run("handler error doesn't stop subscription", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{})
		ctx := context.Background()

		// Publish multiple messages
		messages := []*WorkRequest{
			{Org: "org1", Repo: "repo1", WorkflowRunID: 1},
			{Org: "org2", Repo: "repo2", WorkflowRunID: 2},
			{Org: "org3", Repo: "repo3", WorkflowRunID: 3},
		}

		for _, msg := range messages {
			err := q.Publish(ctx, msg)
			require.NoError(t, err)
		}

		// Subscribe with handler that fails on second message
		received := make([]*WorkRequest, 0)
		var mu sync.Mutex
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		go func() {
			err := q.Subscribe(subCtx, func(ctx context.Context, r *WorkRequest) error {
				mu.Lock()
				received = append(received, r)
				count := len(received)
				mu.Unlock()

				if count == len(messages) {
					subCancel()
				}

				// Return error for second message
				if count == 2 {
					return assert.AnError
				}
				return nil
			})
			assert.ErrorIs(t, err, context.Canceled)
		}()

		// Wait for all messages (including the one with error)
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(received) == len(messages)
		}, 1*time.Second, 10*time.Millisecond)

		// Verify all messages were processed despite the error
		assert.Len(t, received, len(messages))
	})

	t.Run("context cancellation stops subscription", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{})

		subCtx, subCancel := context.WithCancel(context.Background())
		subCancel() // Cancel immediately

		err := q.Subscribe(subCtx, func(ctx context.Context, r *WorkRequest) error {
			return nil
		})

		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("closing queue stops subscription", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{})
		ctx := context.Background()

		done := make(chan struct{})
		go func() {
			err := q.Subscribe(ctx, func(ctx context.Context, r *WorkRequest) error {
				return nil
			})
			assert.NoError(t, err) // Should return nil when channel is closed
			close(done)
		}()

		// Give the subscriber time to start
		time.Sleep(10 * time.Millisecond)

		// Close the queue
		err := q.Close()
		require.NoError(t, err)

		// Wait for subscription to stop
		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("subscription didn't stop after queue close")
		}
	})
}

func TestInMemoryQueue_Close(t *testing.T) {
	t.Run("close once", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{})
		err := q.Close()
		require.NoError(t, err)
		assert.True(t, q.closed)
	})

	t.Run("close twice is safe", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{})
		err := q.Close()
		require.NoError(t, err)

		// Second close should not panic or error
		err = q.Close()
		require.NoError(t, err)
	})
}

func TestInMemoryQueue_Concurrency(t *testing.T) {
	t.Run("concurrent publishers", func(t *testing.T) {
		q := NewInMemoryQueue(InMemoryConfig{BufferSize: 1000})
		ctx := context.Background()

		numPublishers := 10
		messagesPerPublisher := 100
		totalMessages := numPublishers * messagesPerPublisher

		// Start multiple publishers
		var wg sync.WaitGroup
		for i := 0; i < numPublishers; i++ {
			wg.Add(1)
			go func(publisherID int) {
				defer wg.Done()
				for j := 0; j < messagesPerPublisher; j++ {
					req := &WorkRequest{
						Org:           "org",
						Repo:          "repo",
						WorkflowRunID: int64(publisherID*messagesPerPublisher + j),
					}
					err := q.Publish(ctx, req)
					assert.NoError(t, err)
				}
			}(i)
		}

		// Start subscriber
		received := make([]*WorkRequest, 0, totalMessages)
		var mu sync.Mutex
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		go func() {
			err := q.Subscribe(subCtx, func(ctx context.Context, r *WorkRequest) error {
				mu.Lock()
				received = append(received, r)
				if len(received) == totalMessages {
					subCancel()
				}
				mu.Unlock()
				return nil
			})
			assert.ErrorIs(t, err, context.Canceled)
		}()

		// Wait for all publishers to finish
		wg.Wait()

		// Wait for all messages to be received
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(received) == totalMessages
		}, 5*time.Second, 10*time.Millisecond)

		// Verify count
		mu.Lock()
		defer mu.Unlock()
		assert.Len(t, received, totalMessages)
	})
}
