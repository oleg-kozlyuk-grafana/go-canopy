package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Mode represents the deployment mode of the service
type Mode string

const (
	ModeAllInOne  Mode = "all-in-one"
	ModeInitiator Mode = "initiator"
	ModeWorker    Mode = "worker"
	ModeLocal     Mode = "local"
)

// QueueType represents the type of message queue to use
type QueueType string

const (
	QueueTypeInMemory QueueType = "inmemory"
	QueueTypeRedis    QueueType = "redis"
	QueueTypePubSub   QueueType = "pubsub"
)

// StorageType represents the type of storage backend to use
type StorageType string

const (
	StorageTypeGCS   StorageType = "gcs"
	StorageTypeMinio StorageType = "minio"
)

// Config holds all configuration for the Canopy service
type Config struct {
	// Port for the HTTP server
	Port int

	// DisableHMAC disables webhook signature validation (dev only)
	DisableHMAC bool

	// Queue configuration
	Queue QueueConfig

	// Storage configuration
	Storage StorageConfig

	// GitHub configuration
	GitHub GitHubConfig

	// Initiator configuration
	Initiator InitiatorConfig
}

// QueueConfig holds message queue configuration
type QueueConfig struct {
	Type QueueType

	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisStream   string

	// Pub/Sub configuration
	PubSubProjectID    string
	PubSubTopicID      string
	PubSubSubscription string
}

// StorageConfig holds storage backend configuration
type StorageConfig struct {
	Type StorageType

	// GCS configuration
	GCSBucket string

	// MinIO configuration
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
}

// GitHubConfig holds GitHub API configuration
type GitHubConfig struct {
	// GitHub App credentials
	AppID          int64
	InstallationID int64
	PrivateKey     string // PEM-encoded private key
}

// InitiatorConfig holds initiator-specific configuration
type InitiatorConfig struct {
	// Webhook validation
	WebhookSecret string

	// Filtering
	AllowedOrgs      []string
	AllowedWorkflows []string
}

// Load loads configuration from environment variables for the specified mode
func Load(mode Mode) (*Config, error) {
	// Validate mode
	if err := validateMode(mode); err != nil {
		return nil, err
	}

	cfg := &Config{}

	// Port (optional, default 8080)
	port, err := strconv.Atoi(getEnv("CANOPY_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid CANOPY_PORT: %w", err)
	}
	cfg.Port = port

	// DisableHMAC (optional, default false)
	cfg.DisableHMAC = getEnv("CANOPY_DISABLE_HMAC", "false") == "true"

	// Load mode-specific config
	switch mode {
	case ModeAllInOne:
		if err := cfg.loadAllInOneConfig(mode); err != nil {
			return nil, err
		}
	case ModeInitiator:
		if err := cfg.loadInitiatorConfig(mode); err != nil {
			return nil, err
		}
	case ModeWorker:
		if err := cfg.loadWorkerConfig(mode); err != nil {
			return nil, err
		}
	case ModeLocal:
		// Local mode doesn't need any additional configuration
		// It just uses git and local coverage files
	}

	// Validate the complete configuration
	if err := cfg.Validate(mode); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadAllInOneConfig loads config for all-in-one mode
func (c *Config) loadAllInOneConfig(mode Mode) error {
	// Queue: in-memory by default, but can use Redis/Pub/Sub
	queueType := getEnv("CANOPY_QUEUE_TYPE", string(QueueTypeInMemory))
	c.Queue.Type = QueueType(queueType)

	switch c.Queue.Type {
	case QueueTypeInMemory:
		// No additional config needed
	case QueueTypeRedis:
		if err := c.loadRedisConfig(); err != nil {
			return err
		}
	case QueueTypePubSub:
		if err := c.loadPubSubConfig(mode); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid queue type: %s", queueType)
	}

	// Storage
	if err := c.loadStorageConfig(); err != nil {
		return err
	}

	// GitHub
	if err := c.loadGitHubConfig(); err != nil {
		return err
	}

	// Initiator settings
	if err := c.loadInitiatorSettings(); err != nil {
		return err
	}

	return nil
}

// loadInitiatorConfig loads config for initiator mode
func (c *Config) loadInitiatorConfig(mode Mode) error {
	// Queue (required)
	queueType := getEnv("CANOPY_QUEUE_TYPE", "")
	if queueType == "" {
		return fmt.Errorf("CANOPY_QUEUE_TYPE is required in initiator mode")
	}
	c.Queue.Type = QueueType(queueType)

	switch c.Queue.Type {
	case QueueTypeRedis:
		if err := c.loadRedisConfig(); err != nil {
			return err
		}
	case QueueTypePubSub:
		if err := c.loadPubSubConfig(mode); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid queue type for initiator: %s (must be redis or pubsub)", queueType)
	}

	// Initiator settings
	if err := c.loadInitiatorSettings(); err != nil {
		return err
	}

	return nil
}

// loadWorkerConfig loads config for worker mode
func (c *Config) loadWorkerConfig(mode Mode) error {
	// Queue (required)
	queueType := getEnv("CANOPY_QUEUE_TYPE", "")
	if queueType == "" {
		return fmt.Errorf("CANOPY_QUEUE_TYPE is required in worker mode")
	}
	c.Queue.Type = QueueType(queueType)

	switch c.Queue.Type {
	case QueueTypeRedis:
		if err := c.loadRedisConfig(); err != nil {
			return err
		}
	case QueueTypePubSub:
		if err := c.loadPubSubConfig(mode); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid queue type for worker: %s (must be redis or pubsub)", queueType)
	}

	// Storage (required)
	if err := c.loadStorageConfig(); err != nil {
		return err
	}

	// GitHub (required)
	if err := c.loadGitHubConfig(); err != nil {
		return err
	}

	return nil
}

// loadRedisConfig loads Redis queue configuration
func (c *Config) loadRedisConfig() error {
	c.Queue.RedisAddr = getEnv("CANOPY_REDIS_ADDR", "localhost:6379")
	c.Queue.RedisPassword = getEnv("CANOPY_REDIS_PASSWORD", "")

	redisDB, err := strconv.Atoi(getEnv("CANOPY_REDIS_DB", "0"))
	if err != nil {
		return fmt.Errorf("invalid CANOPY_REDIS_DB: %w", err)
	}
	c.Queue.RedisDB = redisDB
	c.Queue.RedisStream = getEnv("CANOPY_REDIS_STREAM", "canopy-coverage-requests")

	return nil
}

// loadPubSubConfig loads Pub/Sub queue configuration
func (c *Config) loadPubSubConfig(mode Mode) error {
	c.Queue.PubSubProjectID = getEnv("CANOPY_PUBSUB_PROJECT_ID", "")
	if c.Queue.PubSubProjectID == "" {
		return fmt.Errorf("CANOPY_PUBSUB_PROJECT_ID is required for pubsub queue")
	}

	c.Queue.PubSubTopicID = getEnv("CANOPY_PUBSUB_TOPIC_ID", "canopy-coverage-requests")

	// Subscription is only needed for worker/all-in-one
	if mode == ModeWorker || mode == ModeAllInOne {
		c.Queue.PubSubSubscription = getEnv("CANOPY_PUBSUB_SUBSCRIPTION", "")
		if c.Queue.PubSubSubscription == "" {
			return fmt.Errorf("CANOPY_PUBSUB_SUBSCRIPTION is required for worker/all-in-one mode")
		}
	}

	return nil
}

// loadStorageConfig loads storage backend configuration
func (c *Config) loadStorageConfig() error {
	storageType := getEnv("CANOPY_STORAGE_TYPE", "")
	if storageType == "" {
		return fmt.Errorf("CANOPY_STORAGE_TYPE is required")
	}
	c.Storage.Type = StorageType(storageType)

	switch c.Storage.Type {
	case StorageTypeGCS:
		c.Storage.GCSBucket = getEnv("CANOPY_GCS_BUCKET", "")
		if c.Storage.GCSBucket == "" {
			return fmt.Errorf("CANOPY_GCS_BUCKET is required for gcs storage")
		}
	case StorageTypeMinio:
		c.Storage.MinIOEndpoint = getEnv("CANOPY_MINIO_ENDPOINT", "")
		if c.Storage.MinIOEndpoint == "" {
			return fmt.Errorf("CANOPY_MINIO_ENDPOINT is required for minio storage")
		}
		c.Storage.MinIOAccessKey = getEnv("CANOPY_MINIO_ACCESS_KEY", "")
		if c.Storage.MinIOAccessKey == "" {
			return fmt.Errorf("CANOPY_MINIO_ACCESS_KEY is required for minio storage")
		}
		c.Storage.MinIOSecretKey = getEnv("CANOPY_MINIO_SECRET_KEY", "")
		if c.Storage.MinIOSecretKey == "" {
			return fmt.Errorf("CANOPY_MINIO_SECRET_KEY is required for minio storage")
		}
		c.Storage.MinIOBucket = getEnv("CANOPY_MINIO_BUCKET", "canopy-coverage")
		c.Storage.MinIOUseSSL = getEnv("CANOPY_MINIO_USE_SSL", "false") == "true"
	default:
		return fmt.Errorf("invalid storage type: %s", storageType)
	}

	return nil
}

// loadGitHubConfig loads GitHub API configuration
func (c *Config) loadGitHubConfig() error {
	appIDStr := getEnv("CANOPY_GITHUB_APP_ID", "")
	if appIDStr == "" {
		return fmt.Errorf("CANOPY_GITHUB_APP_ID is required")
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid CANOPY_GITHUB_APP_ID: %w", err)
	}
	c.GitHub.AppID = appID

	installIDStr := getEnv("CANOPY_GITHUB_INSTALLATION_ID", "")
	if installIDStr == "" {
		return fmt.Errorf("CANOPY_GITHUB_INSTALLATION_ID is required")
	}
	installID, err := strconv.ParseInt(installIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid CANOPY_GITHUB_INSTALLATION_ID: %w", err)
	}
	c.GitHub.InstallationID = installID

	c.GitHub.PrivateKey = getEnv("CANOPY_GITHUB_PRIVATE_KEY", "")
	if c.GitHub.PrivateKey == "" {
		return fmt.Errorf("CANOPY_GITHUB_PRIVATE_KEY is required")
	}

	return nil
}

// loadInitiatorSettings loads initiator-specific settings
func (c *Config) loadInitiatorSettings() error {
	// Webhook secret (optional if HMAC is disabled)
	c.Initiator.WebhookSecret = getEnv("CANOPY_WEBHOOK_SECRET", "")
	if !c.DisableHMAC && c.Initiator.WebhookSecret == "" {
		return fmt.Errorf("CANOPY_WEBHOOK_SECRET is required when HMAC validation is enabled")
	}

	// Allowed orgs (required)
	allowedOrgsStr := getEnv("CANOPY_ALLOWED_ORGS", "")
	if allowedOrgsStr == "" {
		return fmt.Errorf("CANOPY_ALLOWED_ORGS is required")
	}
	c.Initiator.AllowedOrgs = strings.Split(allowedOrgsStr, ",")
	for i := range c.Initiator.AllowedOrgs {
		c.Initiator.AllowedOrgs[i] = strings.TrimSpace(c.Initiator.AllowedOrgs[i])
	}

	// Allowed workflows (optional, empty means all workflows allowed)
	allowedWorkflowsStr := getEnv("CANOPY_ALLOWED_WORKFLOWS", "")
	if allowedWorkflowsStr != "" {
		c.Initiator.AllowedWorkflows = strings.Split(allowedWorkflowsStr, ",")
		for i := range c.Initiator.AllowedWorkflows {
			c.Initiator.AllowedWorkflows[i] = strings.TrimSpace(c.Initiator.AllowedWorkflows[i])
		}
	}

	return nil
}

// validateMode validates that the mode is valid
func validateMode(mode Mode) error {
	switch mode {
	case ModeAllInOne, ModeInitiator, ModeWorker, ModeLocal:
		return nil
	default:
		return fmt.Errorf("invalid mode: %s (must be all-in-one, initiator, worker, or local)", mode)
	}
}

// Validate validates the complete configuration for the specified mode
func (c *Config) Validate(mode Mode) error {
	// Port validation
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", c.Port)
	}

	// Mode-specific validation
	switch mode {
	case ModeAllInOne:
		// All-in-one needs everything
		if c.Queue.Type == "" {
			return fmt.Errorf("queue type is required")
		}
		if c.Storage.Type == "" {
			return fmt.Errorf("storage type is required")
		}
		if c.GitHub.AppID == 0 {
			return fmt.Errorf("GitHub App ID is required")
		}
		if len(c.Initiator.AllowedOrgs) == 0 {
			return fmt.Errorf("at least one allowed org is required")
		}

	case ModeInitiator:
		// Initiator needs queue and filtering config
		if c.Queue.Type == "" {
			return fmt.Errorf("queue type is required")
		}
		if c.Queue.Type == QueueTypeInMemory {
			return fmt.Errorf("in-memory queue cannot be used in initiator mode")
		}
		if len(c.Initiator.AllowedOrgs) == 0 {
			return fmt.Errorf("at least one allowed org is required")
		}

	case ModeWorker:
		// Worker needs queue, storage, and GitHub
		if c.Queue.Type == "" {
			return fmt.Errorf("queue type is required")
		}
		if c.Queue.Type == QueueTypeInMemory {
			return fmt.Errorf("in-memory queue cannot be used in worker mode")
		}
		if c.Storage.Type == "" {
			return fmt.Errorf("storage type is required")
		}
		if c.GitHub.AppID == 0 {
			return fmt.Errorf("GitHub App ID is required")
		}

	case ModeLocal:
		// Local mode doesn't need any cloud configuration
		// No validation required beyond port validation (done above)
	}

	return nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
