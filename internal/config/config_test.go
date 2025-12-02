package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to set up environment variables for tests
func setupEnv(t *testing.T, envVars map[string]string) func() {
	// Store original values
	originalEnv := make(map[string]string)
	for key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Set test values
	for key, value := range envVars {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}

	// Return cleanup function
	return func() {
		for key, originalValue := range originalEnv {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
		// Also clean up any keys that weren't in originalEnv
		for key := range envVars {
			if _, exists := originalEnv[key]; !exists {
				os.Unsetenv(key)
			}
		}
	}
}

func TestLoad_InvalidMode(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{})
	defer cleanup()

	cfg, err := Load("invalid-mode")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid mode")
}

func TestLoad_InvalidPort(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		"CANOPY_PORT": "not-a-number",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid CANOPY_PORT")
}

func TestLoad_AllInOneMode_Success(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_PORT":                    "8080",
		"CANOPY_DISABLE_HMAC":            "false",
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_MINIO_BUCKET":            "canopy-coverage",
		"CANOPY_MINIO_USE_SSL":           "false",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org,another-org",
		"CANOPY_ALLOWED_WORKFLOWS":       "CI,Build",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	
	assert.Equal(t, 8080, cfg.Port)
	assert.False(t, cfg.DisableHMAC)
	assert.Equal(t, QueueTypeInMemory, cfg.Queue.Type)
	assert.Equal(t, StorageTypeMinio, cfg.Storage.Type)
	assert.Equal(t, "localhost:9000", cfg.Storage.MinIOEndpoint)
	assert.Equal(t, "minioadmin", cfg.Storage.MinIOAccessKey)
	assert.Equal(t, int64(123456), cfg.GitHub.AppID)
	assert.Equal(t, int64(789012), cfg.GitHub.InstallationID)
	assert.Equal(t, "my-secret", cfg.Initiator.WebhookSecret)
	assert.Equal(t, []string{"my-org", "another-org"}, cfg.Initiator.AllowedOrgs)
	assert.Equal(t, []string{"CI", "Build"}, cfg.Initiator.AllowedWorkflows)
}

func TestLoad_AllInOneMode_WithRedis(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "redis",
		"CANOPY_REDIS_ADDR":              "localhost:6379",
		"CANOPY_REDIS_PASSWORD":          "password",
		"CANOPY_REDIS_DB":                "1",
		"CANOPY_REDIS_STREAM":            "my-stream",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, QueueTypeRedis, cfg.Queue.Type)
	assert.Equal(t, "localhost:6379", cfg.Queue.RedisAddr)
	assert.Equal(t, "password", cfg.Queue.RedisPassword)
	assert.Equal(t, 1, cfg.Queue.RedisDB)
	assert.Equal(t, "my-stream", cfg.Queue.RedisStream)
}

func TestLoad_AllInOneMode_WithPubSub(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "pubsub",
		"CANOPY_PUBSUB_PROJECT_ID":       "my-project",
		"CANOPY_PUBSUB_TOPIC_ID":         "my-topic",
		"CANOPY_PUBSUB_SUBSCRIPTION":     "my-subscription",
		"CANOPY_STORAGE_TYPE":            "gcs",
		"CANOPY_GCS_BUCKET":              "my-bucket",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, QueueTypePubSub, cfg.Queue.Type)
	assert.Equal(t, "my-project", cfg.Queue.PubSubProjectID)
	assert.Equal(t, "my-topic", cfg.Queue.PubSubTopicID)
	assert.Equal(t, "my-subscription", cfg.Queue.PubSubSubscription)
	assert.Equal(t, StorageTypeGCS, cfg.Storage.Type)
	assert.Equal(t, "my-bucket", cfg.Storage.GCSBucket)
}

func TestLoad_InitiatorMode_Success(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_PORT":                "8080",
		"CANOPY_QUEUE_TYPE":          "pubsub",
		"CANOPY_PUBSUB_PROJECT_ID":   "my-project",
		"CANOPY_PUBSUB_TOPIC_ID":     "my-topic",
		"CANOPY_WEBHOOK_SECRET":      "my-secret",
		"CANOPY_ALLOWED_ORGS":        "my-org",
		"CANOPY_ALLOWED_WORKFLOWS":   "CI",
	})
	defer cleanup()

	cfg, err := Load(ModeInitiator)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	
	assert.Equal(t, QueueTypePubSub, cfg.Queue.Type)
	assert.Equal(t, "my-org", cfg.Initiator.AllowedOrgs[0])
}

func TestLoad_InitiatorMode_MissingQueueType(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_WEBHOOK_SECRET":   "my-secret",
		"CANOPY_ALLOWED_ORGS":     "my-org",
	})
	defer cleanup()

	cfg, err := Load(ModeInitiator)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "CANOPY_QUEUE_TYPE is required")
}

func TestLoad_InitiatorMode_InMemoryQueueNotAllowed(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":     "inmemory",
		"CANOPY_WEBHOOK_SECRET": "my-secret",
		"CANOPY_ALLOWED_ORGS":   "my-org",
	})
	defer cleanup()

	cfg, err := Load(ModeInitiator)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid queue type for initiator")
}

func TestLoad_InitiatorMode_MissingWebhookSecret(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":        "redis",
		"CANOPY_REDIS_ADDR":        "localhost:6379",
		"CANOPY_ALLOWED_ORGS":      "my-org",
		"CANOPY_DISABLE_HMAC":      "false",
	})
	defer cleanup()

	cfg, err := Load(ModeInitiator)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "CANOPY_WEBHOOK_SECRET is required")
}

func TestLoad_InitiatorMode_DisableHMAC(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":        "redis",
		"CANOPY_REDIS_ADDR":        "localhost:6379",
		"CANOPY_ALLOWED_ORGS":      "my-org",
		"CANOPY_DISABLE_HMAC":      "true",
	})
	defer cleanup()

	cfg, err := Load(ModeInitiator)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.DisableHMAC)
	assert.Equal(t, "", cfg.Initiator.WebhookSecret)
}

func TestLoad_WorkerMode_Success(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "redis",
		"CANOPY_REDIS_ADDR":              "localhost:6379",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
	})
	defer cleanup()

	cfg, err := Load(ModeWorker)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	
	assert.Equal(t, QueueTypeRedis, cfg.Queue.Type)
	assert.Equal(t, StorageTypeMinio, cfg.Storage.Type)
	assert.Equal(t, int64(123456), cfg.GitHub.AppID)
}

func TestLoad_WorkerMode_MissingStorage(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "redis",
		"CANOPY_REDIS_ADDR":              "localhost:6379",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
	})
	defer cleanup()

	cfg, err := Load(ModeWorker)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "CANOPY_STORAGE_TYPE is required")
}

func TestLoad_WorkerMode_MissingGitHub(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":        "redis",
		"CANOPY_REDIS_ADDR":        "localhost:6379",
		"CANOPY_STORAGE_TYPE":      "minio",
		"CANOPY_MINIO_ENDPOINT":    "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":  "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":  "minioadmin",
	})
	defer cleanup()

	cfg, err := Load(ModeWorker)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "CANOPY_GITHUB_APP_ID is required")
}

func TestLoad_WorkerMode_InMemoryQueueNotAllowed(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
	})
	defer cleanup()

	cfg, err := Load(ModeWorker)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid queue type for worker")
}

func TestLoad_MissingAllowedOrgs(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "CANOPY_ALLOWED_ORGS is required")
}

func TestLoad_DefaultPort(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
}

func TestLoad_CustomPort(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_PORT":                    "9090",
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.Port)
}

func TestValidate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port: tt.port,
			}
			err := cfg.Validate(ModeAllInOne)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid port")
		})
	}
}

func TestLoad_AllowedWorkflowsOptional(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	assert.Empty(t, cfg.Initiator.AllowedWorkflows)
}

func TestLoad_AllowedOrgsWithSpaces(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            " org1 , org2 , org3 ",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	assert.Equal(t, []string{"org1", "org2", "org3"}, cfg.Initiator.AllowedOrgs)
}

func TestLoad_InvalidGitHubAppID(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "redis",
		"CANOPY_REDIS_ADDR":              "localhost:6379",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "not-a-number",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid CANOPY_GITHUB_APP_ID")
}

func TestLoad_InvalidRedisDB(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "redis",
		"CANOPY_REDIS_ADDR":              "localhost:6379",
		"CANOPY_REDIS_DB":                "not-a-number",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid CANOPY_REDIS_DB")
}

func TestLoad_PubSubMissingSubscriptionForWorker(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "pubsub",
		"CANOPY_PUBSUB_PROJECT_ID":       "my-project",
		"CANOPY_PUBSUB_TOPIC_ID":         "my-topic",
		"CANOPY_STORAGE_TYPE":            "gcs",
		"CANOPY_GCS_BUCKET":              "my-bucket",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "CANOPY_PUBSUB_SUBSCRIPTION is required")
}

func TestLoad_MinIODefaultBucket(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "redis",
		"CANOPY_REDIS_ADDR":              "localhost:6379",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
	})
	defer cleanup()

	cfg, err := Load(ModeWorker)
	require.NoError(t, err)
	assert.Equal(t, "canopy-coverage", cfg.Storage.MinIOBucket)
}

func TestLoad_MinIOUseSSL(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"default", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := map[string]string{
				
				"CANOPY_QUEUE_TYPE":              "redis",
				"CANOPY_REDIS_ADDR":              "localhost:6379",
				"CANOPY_STORAGE_TYPE":            "minio",
				"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
				"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
				"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
				"CANOPY_GITHUB_APP_ID":           "123456",
				"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
				"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
			}
			if tt.value != "" {
				envVars["CANOPY_MINIO_USE_SSL"] = tt.value
			}

			cleanup := setupEnv(t, envVars)
			defer cleanup()

			cfg, err := Load(ModeWorker)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Storage.MinIOUseSSL)
		})
	}
}

func TestLoad_RedisDefaults(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "redis",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
	})
	defer cleanup()

	cfg, err := Load(ModeWorker)
	require.NoError(t, err)
	assert.Equal(t, "localhost:6379", cfg.Queue.RedisAddr)
	assert.Equal(t, "", cfg.Queue.RedisPassword)
	assert.Equal(t, 0, cfg.Queue.RedisDB)
	assert.Equal(t, "canopy-coverage-requests", cfg.Queue.RedisStream)
}

func TestLoad_GCSMissingBucket(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "redis",
		"CANOPY_REDIS_ADDR":              "localhost:6379",
		"CANOPY_STORAGE_TYPE":            "gcs",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "CANOPY_GCS_BUCKET is required")
}

func TestLoad_InvalidStorageType(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "redis",
		"CANOPY_REDIS_ADDR":              "localhost:6379",
		"CANOPY_STORAGE_TYPE":            "invalid-storage",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid storage type")
}

func TestLoad_InvalidQueueType(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_QUEUE_TYPE":              "invalid-queue",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	cfg, err := Load(ModeAllInOne)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid queue type")
}

// Flag precedence tests
// These tests verify that CLI flags properly override environment variables
// as implemented in cmd/canopy/main.go where flags set environment variables
// before config.Load() is called

func TestFlagPrecedence_Port(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_PORT":                    "8080", // Original env var value
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	// Simulate flag override (as done in main.go:67-69)
	os.Setenv("CANOPY_PORT", "9090")

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.Port, "flag should override env var")
}

func TestFlagPrecedence_DisableHMAC(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_DISABLE_HMAC":            "false", // Original env var value
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	// Simulate flag override (as done in main.go:70-72)
	os.Setenv("CANOPY_DISABLE_HMAC", "true")

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	assert.True(t, cfg.DisableHMAC, "flag should override env var")
}

func TestFlagPrecedence_MultipleFlags(t *testing.T) {
	cleanup := setupEnv(t, map[string]string{
		
		"CANOPY_PORT":                    "8080",
		"CANOPY_DISABLE_HMAC":            "false",
		"CANOPY_QUEUE_TYPE":              "inmemory",
		"CANOPY_STORAGE_TYPE":            "minio",
		"CANOPY_MINIO_ENDPOINT":          "localhost:9000",
		"CANOPY_MINIO_ACCESS_KEY":        "minioadmin",
		"CANOPY_MINIO_SECRET_KEY":        "minioadmin",
		"CANOPY_GITHUB_APP_ID":           "123456",
		"CANOPY_GITHUB_INSTALLATION_ID":  "789012",
		"CANOPY_GITHUB_PRIVATE_KEY":      "test-key",
		"CANOPY_WEBHOOK_SECRET":          "my-secret",
		"CANOPY_ALLOWED_ORGS":            "my-org",
	})
	defer cleanup()

	// Simulate multiple flag overrides
	os.Setenv("CANOPY_PORT", "3000")
	os.Setenv("CANOPY_DISABLE_HMAC", "true")

	cfg, err := Load(ModeAllInOne)
	require.NoError(t, err)
	
	assert.Equal(t, 3000, cfg.Port, "port flag should override env var")
	assert.True(t, cfg.DisableHMAC, "disable-hmac flag should override env var")
}
