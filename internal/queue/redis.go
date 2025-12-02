package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisQueue implements MessageQueue using Redis Streams.
// It uses consumer groups for reliable message processing with acknowledgment.
type RedisQueue struct {
	client        *redis.Client
	streamKey     string
	consumerGroup string
	consumerName  string
}

// RedisConfig holds configuration for creating a RedisQueue.
type RedisConfig struct {
	// Address is the Redis server address (host:port)
	Address string

	// Password is the Redis password (optional)
	Password string

	// DB is the Redis database number (default: 0)
	DB int

	// StreamKey is the Redis stream name
	StreamKey string

	// ConsumerGroup is the consumer group name
	ConsumerGroup string

	// ConsumerName is the consumer name within the group
	ConsumerName string

	// CreateIfNotExists creates the stream and consumer group if they don't exist
	CreateIfNotExists bool
}

// NewRedisQueue creates a new RedisQueue instance.
// The caller is responsible for calling Close() when done.
func NewRedisQueue(ctx context.Context, cfg RedisConfig) (*RedisQueue, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("redis address is required")
	}
	if cfg.StreamKey == "" {
		return nil, fmt.Errorf("stream key is required")
	}
	if cfg.ConsumerGroup == "" {
		return nil, fmt.Errorf("consumer group is required")
	}
	if cfg.ConsumerName == "" {
		return nil, fmt.Errorf("consumer name is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	q := &RedisQueue{
		client:        client,
		streamKey:     cfg.StreamKey,
		consumerGroup: cfg.ConsumerGroup,
		consumerName:  cfg.ConsumerName,
	}

	// Optionally create consumer group if it doesn't exist
	if cfg.CreateIfNotExists {
		// Create consumer group. MKSTREAM option creates the stream if it doesn't exist.
		// The "$" argument means the group will only see new messages (not historical ones).
		err := client.XGroupCreateMkStream(ctx, cfg.StreamKey, cfg.ConsumerGroup, "$").Err()
		if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return nil, fmt.Errorf("failed to create consumer group: %w", err)
		}
	}

	return q, nil
}

// Publish sends a WorkRequest message to the Redis stream.
func (q *RedisQueue) Publish(ctx context.Context, req *WorkRequest) error {
	if req == nil {
		return fmt.Errorf("work request cannot be nil")
	}

	// Serialize the work request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal work request: %w", err)
	}

	// Add message to the stream
	// The "*" argument tells Redis to auto-generate a message ID
	args := &redis.XAddArgs{
		Stream: q.streamKey,
		Values: map[string]interface{}{
			"data": string(data),
			"org":  req.Org,
			"repo": req.Repo,
		},
	}

	_, err = q.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to publish message to redis stream: %w", err)
	}

	return nil
}

// Subscribe starts consuming messages from the Redis stream using a consumer group.
// It calls the handler function for each received message.
// This method blocks until the context is cancelled or an error occurs.
func (q *RedisQueue) Subscribe(ctx context.Context, handler func(context.Context, *WorkRequest) error) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	// Process messages in a loop until context is cancelled
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		// Read messages from the consumer group
		// ">" means to receive only new messages that were never delivered to any other consumer
		streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    q.consumerGroup,
			Consumer: q.consumerName,
			Streams:  []string{q.streamKey, ">"},
			Count:    10, // Process up to 10 messages at a time
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				// No messages available, continue polling
				continue
			}
			if err == context.Canceled || err == context.DeadlineExceeded {
				return err
			}
			// For other errors, log and continue
			// In production, you'd want to log this error
			continue
		}

		// Process each message
		for _, stream := range streams {
			for _, message := range stream.Messages {
				if err := q.processMessage(ctx, message, handler); err != nil {
					// Error processing message, but continue with others
					// In production, you'd want to log this error
					continue
				}
			}
		}
	}
}

// processMessage handles a single message from the stream.
func (q *RedisQueue) processMessage(ctx context.Context, msg redis.XMessage, handler func(context.Context, *WorkRequest) error) error {
	// Extract the message data
	dataStr, ok := msg.Values["data"].(string)
	if !ok {
		// Invalid message format - acknowledge it to remove from pending
		_ = q.client.XAck(ctx, q.streamKey, q.consumerGroup, msg.ID)
		return fmt.Errorf("message data field is not a string")
	}

	// Parse the work request
	var req WorkRequest
	if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
		// Invalid JSON - acknowledge it to remove from pending
		_ = q.client.XAck(ctx, q.streamKey, q.consumerGroup, msg.ID)
		return fmt.Errorf("failed to unmarshal work request: %w", err)
	}

	// Call the handler to process the message
	if err := handler(ctx, &req); err != nil {
		// Processing failed - don't acknowledge, message will be retried
		return fmt.Errorf("handler failed to process message: %w", err)
	}

	// Processing succeeded - acknowledge the message
	if err := q.client.XAck(ctx, q.streamKey, q.consumerGroup, msg.ID).Err(); err != nil {
		return fmt.Errorf("failed to acknowledge message: %w", err)
	}

	return nil
}

// Close releases resources held by the RedisQueue.
func (q *RedisQueue) Close() error {
	return q.client.Close()
}
