package queue

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisConfig_Validation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  RedisConfig
		wantErr string
	}{
		{
			name: "missing address",
			config: RedisConfig{
				StreamKey:     "test-stream",
				ConsumerGroup: "test-group",
				ConsumerName:  "test-consumer",
			},
			wantErr: "redis address is required",
		},
		{
			name: "missing stream key",
			config: RedisConfig{
				Address:       "localhost:6379",
				ConsumerGroup: "test-group",
				ConsumerName:  "test-consumer",
			},
			wantErr: "stream key is required",
		},
		{
			name: "missing consumer group",
			config: RedisConfig{
				Address:      "localhost:6379",
				StreamKey:    "test-stream",
				ConsumerName: "test-consumer",
			},
			wantErr: "consumer group is required",
		},
		{
			name: "missing consumer name",
			config: RedisConfig{
				Address:       "localhost:6379",
				StreamKey:     "test-stream",
				ConsumerGroup: "test-group",
			},
			wantErr: "consumer name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRedisQueue(ctx, tt.config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestRedisQueue_PublishValidation(t *testing.T) {
	// Create a mock queue (without actual Redis connection)
	// This test focuses on validation logic
	q := &RedisQueue{}

	ctx := context.Background()

	t.Run("nil work request", func(t *testing.T) {
		err := q.Publish(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work request cannot be nil")
	})
}

func TestRedisQueue_SubscribeValidation(t *testing.T) {
	// Create a mock queue (without actual Redis connection)
	q := &RedisQueue{}

	ctx := context.Background()

	t.Run("nil handler", func(t *testing.T) {
		err := q.Subscribe(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "handler cannot be nil")
	})
}

func TestRedisQueue_MessageSerialization(t *testing.T) {
	t.Run("marshal and unmarshal work request", func(t *testing.T) {
		req := &WorkRequest{
			Org:           "test-org",
			Repo:          "test-repo",
			WorkflowRunID: 12345,
		}

		// Test JSON marshaling
		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Test JSON unmarshaling
		var decoded WorkRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, req.Org, decoded.Org)
		assert.Equal(t, req.Repo, decoded.Repo)
		assert.Equal(t, req.WorkflowRunID, decoded.WorkflowRunID)
	})

	t.Run("unmarshal invalid JSON", func(t *testing.T) {
		invalidJSON := []byte(`{"org": "test", invalid}`)

		var decoded WorkRequest
		err := json.Unmarshal(invalidJSON, &decoded)
		assert.Error(t, err)
	})

	t.Run("unmarshal empty JSON", func(t *testing.T) {
		emptyJSON := []byte(`{}`)

		var decoded WorkRequest
		err := json.Unmarshal(emptyJSON, &decoded)
		require.NoError(t, err)
		// Fields should be zero values
		assert.Empty(t, decoded.Org)
		assert.Empty(t, decoded.Repo)
		assert.Equal(t, int64(0), decoded.WorkflowRunID)
	})
}

func TestRedisQueue_ConnectionError(t *testing.T) {
	ctx := context.Background()

	t.Run("connection to invalid address fails", func(t *testing.T) {
		cfg := RedisConfig{
			Address:       "localhost:9999", // Invalid port
			StreamKey:     "test-stream",
			ConsumerGroup: "test-group",
			ConsumerName:  "test-consumer",
		}

		// Set a short timeout to fail fast
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		_, err := NewRedisQueue(ctx, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to redis")
	})
}

// Integration test (requires actual Redis or Redis container)
// This test is skipped by default but can be enabled with integration testing
func TestRedisQueue_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This test would require either:
	// 1. Redis server running locally on localhost:6379
	// 2. Redis container started via testcontainers
	// For now, it's a placeholder that can be enabled for local testing

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	streamKey := "test-stream-" + time.Now().Format("20060102150405")
	cfg := RedisConfig{
		Address:           "localhost:6379",
		StreamKey:         streamKey,
		ConsumerGroup:     "test-group",
		ConsumerName:      "test-consumer",
		CreateIfNotExists: true,
	}

	t.Run("publish and subscribe", func(t *testing.T) {
		t.Skip("requires Redis server or container")

		queue, err := NewRedisQueue(ctx, cfg)
		require.NoError(t, err)
		defer queue.Close()

		// Publish a message
		req := &WorkRequest{
			Org:           "test-org",
			Repo:          "test-repo",
			WorkflowRunID: 12345,
		}

		err = queue.Publish(ctx, req)
		require.NoError(t, err)

		// Subscribe and receive the message
		received := make(chan *WorkRequest, 1)
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		go func() {
			err := queue.Subscribe(subCtx, func(ctx context.Context, r *WorkRequest) error {
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
		case <-time.After(10 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("multiple messages in order", func(t *testing.T) {
		t.Skip("requires Redis server or container")

		queue, err := NewRedisQueue(ctx, cfg)
		require.NoError(t, err)
		defer queue.Close()

		// Publish multiple messages
		messages := []*WorkRequest{
			{Org: "org1", Repo: "repo1", WorkflowRunID: 1},
			{Org: "org2", Repo: "repo2", WorkflowRunID: 2},
			{Org: "org3", Repo: "repo3", WorkflowRunID: 3},
		}

		for _, msg := range messages {
			err := queue.Publish(ctx, msg)
			require.NoError(t, err)
		}

		// Subscribe and collect messages
		received := make([]*WorkRequest, 0, len(messages))
		var mu sync.Mutex
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		go func() {
			err := queue.Subscribe(subCtx, func(ctx context.Context, r *WorkRequest) error {
				mu.Lock()
				received = append(received, r)
				if len(received) == len(messages) {
					subCancel()
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
		}, 10*time.Second, 100*time.Millisecond)

		// Verify messages
		mu.Lock()
		defer mu.Unlock()
		require.Len(t, received, len(messages))
		for i, msg := range messages {
			assert.Equal(t, msg.Org, received[i].Org)
			assert.Equal(t, msg.Repo, received[i].Repo)
			assert.Equal(t, msg.WorkflowRunID, received[i].WorkflowRunID)
		}
	})

	t.Run("handler error triggers retry", func(t *testing.T) {
		t.Skip("requires Redis server or container")

		queue, err := NewRedisQueue(ctx, cfg)
		require.NoError(t, err)
		defer queue.Close()

		// Publish a message
		req := &WorkRequest{
			Org:           "test-org",
			Repo:          "test-repo",
			WorkflowRunID: 12345,
		}

		err = queue.Publish(ctx, req)
		require.NoError(t, err)

		// Subscribe with handler that fails first time
		attempts := 0
		var mu sync.Mutex
		received := make(chan *WorkRequest, 1)
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		go func() {
			err := queue.Subscribe(subCtx, func(ctx context.Context, r *WorkRequest) error {
				mu.Lock()
				attempts++
				currentAttempt := attempts
				mu.Unlock()

				// Fail first attempt, succeed on second
				if currentAttempt == 1 {
					return assert.AnError
				}

				received <- r
				subCancel()
				return nil
			})
			assert.ErrorIs(t, err, context.Canceled)
		}()

		// Wait for message to be processed successfully
		select {
		case r := <-received:
			assert.Equal(t, req.Org, r.Org)
			mu.Lock()
			assert.GreaterOrEqual(t, attempts, 2, "message should be retried after failure")
			mu.Unlock()
		case <-time.After(10 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("concurrent publishers", func(t *testing.T) {
		t.Skip("requires Redis server or container")

		queue, err := NewRedisQueue(ctx, cfg)
		require.NoError(t, err)
		defer queue.Close()

		numPublishers := 5
		messagesPerPublisher := 10
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
					err := queue.Publish(ctx, req)
					assert.NoError(t, err)
				}
			}(i)
		}

		// Wait for all publishers
		wg.Wait()

		// Subscribe and collect all messages
		received := make([]*WorkRequest, 0, totalMessages)
		var mu sync.Mutex
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		go func() {
			err := queue.Subscribe(subCtx, func(ctx context.Context, r *WorkRequest) error {
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

		// Wait for all messages
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(received) == totalMessages
		}, 15*time.Second, 100*time.Millisecond)

		// Verify count
		mu.Lock()
		defer mu.Unlock()
		assert.Len(t, received, totalMessages)
	})
}
