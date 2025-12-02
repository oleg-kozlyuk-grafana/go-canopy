package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
)

// PubSubQueue implements MessageQueue using Google Cloud Pub/Sub.
type PubSubQueue struct {
	client       *pubsub.Client
	topic        *pubsub.Topic
	subscription *pubsub.Subscription
	projectID    string
	topicName    string
	subName      string
}

// PubSubConfig holds configuration for creating a PubSubQueue.
type PubSubConfig struct {
	// ProjectID is the GCP project ID
	ProjectID string

	// TopicName is the Pub/Sub topic name
	TopicName string

	// SubscriptionName is the Pub/Sub subscription name
	SubscriptionName string

	// CreateIfNotExists creates the topic and subscription if they don't exist
	CreateIfNotExists bool
}

// NewPubSubQueue creates a new PubSubQueue instance.
// The caller is responsible for calling Close() when done.
func NewPubSubQueue(ctx context.Context, cfg PubSubConfig) (*PubSubQueue, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}
	if cfg.TopicName == "" {
		return nil, fmt.Errorf("topic name is required")
	}
	if cfg.SubscriptionName == "" {
		return nil, fmt.Errorf("subscription name is required")
	}

	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	topic := client.Topic(cfg.TopicName)
	sub := client.Subscription(cfg.SubscriptionName)

	// Optionally create topic and subscription if they don't exist
	if cfg.CreateIfNotExists {
		// Check if topic exists, create if not
		exists, err := topic.Exists(ctx)
		if err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to check topic existence: %w", err)
		}
		if !exists {
			topic, err = client.CreateTopic(ctx, cfg.TopicName)
			if err != nil {
				client.Close()
				return nil, fmt.Errorf("failed to create topic: %w", err)
			}
		}

		// Check if subscription exists, create if not
		exists, err = sub.Exists(ctx)
		if err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to check subscription existence: %w", err)
		}
		if !exists {
			sub, err = client.CreateSubscription(ctx, cfg.SubscriptionName, pubsub.SubscriptionConfig{
				Topic:       topic,
				AckDeadline: 60 * time.Second, // 60 seconds to process message
			})
			if err != nil {
				client.Close()
				return nil, fmt.Errorf("failed to create subscription: %w", err)
			}
		}
	}

	return &PubSubQueue{
		client:       client,
		topic:        topic,
		subscription: sub,
		projectID:    cfg.ProjectID,
		topicName:    cfg.TopicName,
		subName:      cfg.SubscriptionName,
	}, nil
}

// Publish sends a WorkRequest message to the Pub/Sub topic.
func (q *PubSubQueue) Publish(ctx context.Context, req *WorkRequest) error {
	if req == nil {
		return fmt.Errorf("work request cannot be nil")
	}

	// Serialize the work request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal work request: %w", err)
	}

	// Publish the message
	result := q.topic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"org":  req.Org,
			"repo": req.Repo,
		},
	})

	// Wait for the result
	_, err = result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Subscribe starts consuming messages from the Pub/Sub subscription.
// It calls the handler function for each received message.
// This method blocks until the context is cancelled or an error occurs.
func (q *PubSubQueue) Subscribe(ctx context.Context, handler func(context.Context, *WorkRequest) error) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	// Configure subscription receive settings
	q.subscription.ReceiveSettings.MaxOutstandingMessages = 10
	q.subscription.ReceiveSettings.NumGoroutines = 4

	// Start receiving messages
	err := q.subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// Parse the work request from message data
		var req WorkRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			// Invalid message format - nack and log error
			msg.Nack()
			// In production, you'd want to log this error
			return
		}

		// Call the handler to process the message
		if err := handler(ctx, &req); err != nil {
			// Processing failed - nack the message for retry
			msg.Nack()
			// In production, you'd want to log this error
			return
		}

		// Processing succeeded - acknowledge the message
		msg.Ack()
	})

	if err != nil {
		return fmt.Errorf("subscription receive error: %w", err)
	}

	return nil
}

// Close releases resources held by the PubSubQueue.
func (q *PubSubQueue) Close() error {
	// Stop the topic from accepting new messages
	q.topic.Stop()

	// Close the client
	return q.client.Close()
}
