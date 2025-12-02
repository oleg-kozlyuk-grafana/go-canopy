package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPubSubConfig_Validation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  PubSubConfig
		wantErr string
	}{
		{
			name: "missing project ID",
			config: PubSubConfig{
				TopicName:        "test-topic",
				SubscriptionName: "test-sub",
			},
			wantErr: "project ID is required",
		},
		{
			name: "missing topic name",
			config: PubSubConfig{
				ProjectID:        "test-project",
				SubscriptionName: "test-sub",
			},
			wantErr: "topic name is required",
		},
		{
			name: "missing subscription name",
			config: PubSubConfig{
				ProjectID: "test-project",
				TopicName: "test-topic",
			},
			wantErr: "subscription name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPubSubQueue(ctx, tt.config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestPubSubQueue_PublishValidation(t *testing.T) {
	// Create a mock queue (without actual GCP connection)
	// This test focuses on validation logic
	q := &PubSubQueue{}

	ctx := context.Background()

	t.Run("nil work request", func(t *testing.T) {
		err := q.Publish(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work request cannot be nil")
	})
}

func TestPubSubQueue_SubscribeValidation(t *testing.T) {
	// Create a mock queue (without actual GCP connection)
	q := &PubSubQueue{}

	ctx := context.Background()

	t.Run("nil handler", func(t *testing.T) {
		err := q.Subscribe(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "handler cannot be nil")
	})
}

func TestWorkRequest_Serialization(t *testing.T) {
	req := &WorkRequest{
		Org:           "test-org",
		Repo:          "test-repo",
		WorkflowRunID: 12345,
	}

	// Test JSON marshaling (using encoding/json directly)
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
}

// Integration test (requires actual GCP Pub/Sub or emulator)
// This test is skipped by default but can be enabled with -integration flag
func TestPubSubQueue_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This test would require either:
	// 1. GCP Pub/Sub emulator running locally
	// 2. Actual GCP credentials and test project
	// For now, it's a placeholder

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := PubSubConfig{
		ProjectID:         "test-project",
		TopicName:         "test-topic",
		SubscriptionName:  "test-sub",
		CreateIfNotExists: true,
	}

	t.Run("publish and subscribe", func(t *testing.T) {
		t.Skip("requires GCP Pub/Sub emulator or credentials")

		queue, err := NewPubSubQueue(ctx, cfg)
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
			assert.NoError(t, err)
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
}
