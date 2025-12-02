# Canopy Implementation Plan

## Overview

Canopy is a Go service that provides code coverage annotations on GitHub pull requests. It receives GitHub workflow_run webhooks, processes Go coverage files, and posts check runs with annotations highlighting uncovered lines.

## Architecture

### Deployment Modes

1. **All-in-one mode**: Single process for local development (initiator + worker)
2. **Split mode**: Separate initiator and worker processes for production

### Production Flow

```
GitHub Webhook → Initiator (Cloud Run) → Pub/Sub → Worker (Cloud Run) → GitHub API
                                                          ↓
                                                     GCS Bucket
```

### Security Model

- Initiator: No GitHub credentials, validates HMAC, publishes to queue
- Worker: Has GitHub credentials, processes coverage, updates PRs

## Technology Stack

- **Language**: Go 1.22+
- **GitHub API**: google/go-github/v57
- **Coverage**: golang.org/x/tools/cover
- **Message Queue**: GCP Pub/Sub (prod), Redis (local)
- **Storage**: GCS (prod), MinIO (local)
- **Infrastructure**: Terraform, GCP Cloud Run
- **Local Dev**: Docker Compose

## Project Structure

```
canopy/
├── cmd/
│   └── canopy/
│       └── main.go                    # Entry point, CLI
├── internal/
│   ├── config/
│   │   ├── config.go                  # Configuration management
│   │   └── config_test.go
│   ├── initiator/
│   │   ├── handler.go                 # Webhook HTTP handler
│   │   ├── hmac.go                    # HMAC signature validation
│   │   └── validator.go               # Event validation (org, workflow)
│   ├── worker/
│   │   ├── worker.go                  # Main orchestration logic
│   │   ├── artifacts.go               # Artifact fetching
│   │   ├── checkrun.go                # Check run creation/updates
│   │   └── comment.go                 # PR comment management
│   ├── coverage/
│   │   ├── parser.go                  # Coverage file parsing
│   │   ├── merger.go                  # gocovmerge-style merging
│   │   ├── diff.go                    # PR diff parsing
│   │   └── analysis.go                # Coverage calculations
│   ├── queue/
│   │   ├── interface.go               # MessageQueue interface
│   │   ├── pubsub.go                  # GCP Pub/Sub implementation
│   │   ├── redis.go                   # Redis implementation
│   │   └── inmemory.go                # In-memory (all-in-one mode)
│   ├── storage/
│   │   ├── interface.go               # Storage interface
│   │   ├── gcs.go                     # GCS implementation
│   │   └── minio.go                   # MinIO implementation
│   ├── github/
│   │   ├── client.go                  # GitHub API wrapper
│   │   └── mock.go                    # Mock for testing
│   └── server/
│       ├── server.go                  # HTTP server
│       └── middleware.go              # Logging, recovery
├── terraform/
│   ├── main.tf
│   ├── variables.tf
│   ├── outputs.tf
│   ├── providers.tf
│   ├── cloud_run.tf                   # Initiator & Worker services
│   ├── pubsub.tf                      # Topic, subscription, DLQ
│   ├── storage.tf                     # GCS bucket
│   ├── iam.tf                         # Service accounts, IAM bindings
│   └── secrets.tf                     # Secret Manager
├── deployments/
│   ├── docker-compose.yml             # Local dev setup
│   ├── Dockerfile                     # Multi-stage build
│   └── .env.example
├── testdata/
│   ├── coverage/                      # Sample coverage files
│   ├── webhooks/                      # Sample webhook payloads
│   └── diffs/                         # Sample PR diffs
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Implementation Task List

### Phase 1: Foundation & Configuration

- [x] **1.1** Initialize Go module with dependencies
  - `github.com/google/go-github/v57`
  - `golang.org/x/tools/cover`
  - `cloud.google.com/go/pubsub`
  - `cloud.google.com/go/storage`
  - `github.com/redis/go-redis/v9`
  - `github.com/minio/minio-go/v7`
  - `github.com/spf13/cobra` (CLI)
  - `github.com/stretchr/testify` (testing)

- [x] **1.2** Create configuration management (`internal/config/`)
  - Config struct with all settings (mode, queue, storage, GitHub)
  - Environment variable loading
  - Config file support (optional)
  - Validation logic

- [x] **1.3** Implement CLI with flags (`cmd/canopy/main.go`)
  - `--mode` (all-in-one|initiator|worker)
  - `--port` (default: 8080)
  - `--disable-hmac` for local dev
  - Environment variable support
  - Version and help commands

- [x] **1.4** Create HTTP server framework (`internal/server/`)
  - Basic server setup with graceful shutdown
  - Middleware: logging, recovery, request ID
  - Health check endpoint
  - Metrics endpoint (optional)

- [x] **1.5** Write configuration tests
  - Test env var loading
  - Test flag precedence
  - Test validation errors

- [x] **1.6** Create Makefile
  - `make build` - Build binary
  - `make test` - Run tests
  - `make test-coverage` - Run tests with coverage report
  - `make docker-build` - Build Docker image
  - `make local-up` - Start docker-compose
  - `make local-down` - Stop docker-compose
  - `make lint` - Run linter

### Phase 2: Message Queue Abstraction

- [x] **2.1** Define MessageQueue interface (`internal/queue/interface.go`)
  - WorkRequest message struct (org, repo, workflow_run_id)
  - Publish() method
  - Subscribe() method with handler function
  - Close() method

- [x] **2.2** Implement GCP Pub/Sub adapter (`internal/queue/pubsub.go`)
  - Client initialization
  - Publish with context and error handling
  - Subscribe with message acknowledgment
  - Handle graceful shutdown

- [x] **2.3** Implement Redis adapter (`internal/queue/redis.go`)
  - Use Redis Streams for reliability
  - Consumer group support
  - Message acknowledgment
  - Handle connection errors

- [x] **2.4** Implement in-memory queue (`internal/queue/inmemory.go`)
  - Simple channel-based queue for all-in-one mode
  - Synchronous processing

- [x] **2.5** Write queue tests
  - Test publish/subscribe cycle
  - Test error handling
  - Test message serialization
  - Mock tests for each implementation

### Phase 3: Storage Abstraction

- [x] **3.1** Define Storage interface (`internal/storage/interface.go`)
  - CoverageKey struct (org, repo, branch)
  - SaveCoverage() method
  - GetCoverage() method (returns nil if not found)
  - Close() method

- [ ] **3.2** Implement GCS adapter (`internal/storage/gcs.go`)
  - Client initialization with credentials
  - Object path format: `{org}/{repo}/{branch}/coverage.out`
  - Upload with retry logic
  - Download with not-found handling

- [ ] **3.3** Implement MinIO adapter (`internal/storage/minio.go`)
  - Client initialization (endpoint, credentials, SSL)
  - S3-compatible API usage
  - Bucket existence check
  - Error handling

- [ ] **3.4** Write storage tests
  - Test save/get cycle
  - Test not-found scenarios
  - Test error handling
  - Integration tests with testcontainers

### Phase 4: Coverage Processing

- [ ] **4.1** Implement coverage parser (`internal/coverage/parser.go`)
  - Parse standard Go coverage format using `golang.org/x/tools/cover`
  - Handle coverage files from zip archives (workflow artifacts)
  - Validate coverage profile format
  - Error handling for malformed files

- [ ] **4.2** Implement coverage merger (`internal/coverage/merger.go`)
  - Merge multiple coverage profiles (gocovmerge algorithm)
  - Handle overlapping coverage blocks
  - Serialize merged profiles back to standard format
  - Optimize for performance

- [ ] **4.3** Implement diff parser (`internal/coverage/diff.go`)
  - Parse unified diff format
  - Extract added lines per file
  - Map GitHub CommitFile objects to line numbers
  - Handle binary files and renames

- [ ] **4.4** Implement coverage analysis (`internal/coverage/analysis.go`)
  - Calculate overall coverage percentage
  - Calculate per-file coverage
  - Cross-reference coverage with diff (find uncovered added lines)
  - Generate GitHub Check Run annotations
  - Compare base vs head coverage

- [ ] **4.5** Create test fixtures and tests
  - Sample coverage files in `testdata/coverage/`
  - Sample diffs in `testdata/diffs/`
  - Unit tests for each function
  - Integration tests for full pipeline

### Phase 5: Initiator Service

- [ ] **5.1** Implement HMAC validation (`internal/initiator/hmac.go`)
  - Parse `X-Hub-Signature-256` header
  - Compute HMAC-SHA256 of payload
  - Constant-time comparison
  - Return specific errors for debugging

- [ ] **5.2** Implement event validator (`internal/initiator/validator.go`)
  - Check event action == "completed"
  - Check org matches allowed list
  - Check workflow name in allowed list
  - Return specific validation errors

- [ ] **5.3** Implement webhook handler (`internal/initiator/handler.go`)
  - Parse webhook payload
  - Validate HMAC signature (unless disabled)
  - Validate event criteria
  - Build WorkRequest message
  - Publish to queue
  - Return appropriate HTTP status codes

- [ ] **5.4** Wire up initiator in main.go
  - Initialize queue client
  - Create handler with dependencies
  - Register route: POST /webhook
  - Start HTTP server

- [ ] **5.5** Write initiator tests
  - Test valid webhook processing
  - Test HMAC validation (valid, invalid, missing)
  - Test org/workflow filtering
  - Test non-completed events
  - Test malformed payloads
  - Mock queue for testing

### Phase 6: GitHub Client Wrapper

- [ ] **6.1** Implement GitHub client wrapper (`internal/github/client.go`)
  - Initialize with GitHub App credentials (App ID, Installation ID, private key)
  - JWT token generation for GitHub App authentication
  - Wrapper methods for all needed API calls:
    - GetWorkflowRun()
    - ListArtifacts() and DownloadArtifact()
    - GetPullRequest() and ListPullRequestFiles()
    - GetDefaultBranch()
    - CreateCheckRun() and UpdateCheckRun()
    - CreateIssueComment(), ListIssueComments(), UpdateIssueComment()

- [ ] **6.2** Implement mock GitHub client (`internal/github/mock.go`)
  - Mock implementation of Client interface
  - Configurable responses for testing
  - Error injection capabilities

- [ ] **6.3** Write GitHub client tests
  - Test authentication flow
  - Test each API method with mock responses
  - Test error handling (rate limiting, not found, etc.)

### Phase 7: Worker Service

- [ ] **7.1** Implement artifact fetcher (`internal/worker/artifacts.go`)
  - List artifacts for workflow run
  - Filter by name pattern (e.g., "coverage*.out", "coverage*.txt")
  - Download artifact zip files
  - Extract coverage files from zip
  - Return raw coverage data

- [ ] **7.2** Implement check run manager (`internal/worker/checkrun.go`)
  - Create initial check run with "in_progress" status
  - Update check run with annotations (batch in groups of 50)
  - Set final status (success/failure based on coverage change)
  - Format check run summary: "Project coverage X%, change Y%"
  - Handle GitHub API errors

- [ ] **7.3** Implement PR comment manager (`internal/worker/comment.go`)
  - Generate markdown table with coverage comparison
  - Search for existing bot comment
  - Create new comment or update existing
  - Format: main coverage, PR coverage, change delta

- [ ] **7.4** Implement main worker orchestration (`internal/worker/worker.go`)
  - Fetch workflow run details
  - Download and merge coverage artifacts
  - Detect if run is on default branch or PR
  - **Default branch flow**:
    - Save merged coverage to storage
    - Exit
  - **PR flow**:
    - Get PR number from workflow run
    - Get PR diff (files changed)
    - Get base branch coverage from storage
    - Create check run
    - Analyze coverage, find uncovered added lines
    - Create annotations for uncovered lines
    - Update check run with annotations and summary
    - Post/update PR comment with coverage table
    - Set check run status (fail if coverage decreased)

- [ ] **7.5** Wire up worker in main.go
  - Initialize GitHub client with App credentials
  - Initialize storage client
  - Initialize queue subscriber
  - Create worker with dependencies
  - Subscribe to queue with worker.ProcessWorkRequest handler
  - Handle graceful shutdown

- [ ] **7.6** Write worker tests
  - Test default branch flow (save coverage)
  - Test PR flow (create check run, annotations, comment)
  - Test no coverage artifacts found
  - Test no base coverage available (first PR)
  - Test GitHub API errors
  - Test storage errors
  - Mock all external dependencies

### Phase 8: All-in-One Mode

- [ ] **8.1** Implement combined mode in main.go
  - Use in-memory queue
  - Start worker goroutine
  - Start initiator HTTP server
  - Handle graceful shutdown of both

- [ ] **8.2** Create integration tests for all-in-one mode
  - End-to-end webhook → coverage processing flow
  - Use real Redis and MinIO via testcontainers
  - Mock GitHub API

### Phase 9: Testing & Quality

- [ ] **9.1** Achieve 80% code coverage
  - Run coverage reports: `go test -cover ./...`
  - Identify untested code paths
  - Add missing tests
  - Add coverage check to CI

- [ ] **9.2** Bad auth scenario tests (required by spec)
  - Missing HMAC signature → 401
  - Invalid HMAC signature → 401
  - Malformed signature header → 401
  - Wrong organization → 403
  - Disallowed workflow → 403
  - Ensure --disable-hmac flag bypasses checks

- [ ] **9.3** Edge case tests
  - Empty coverage files
  - No coverage artifacts in workflow run
  - PR with no Go files changed
  - Storage unavailable
  - GitHub API rate limiting
  - Malformed webhook payloads

- [ ] **9.4** Create comprehensive test fixtures
  - Valid webhook payloads in `testdata/webhooks/`
  - Sample coverage files with various scenarios
  - Sample PR diffs

### Phase 10: Docker & Local Development

- [ ] **10.1** Create Dockerfile (`deployments/Dockerfile`)
  - Multi-stage build (builder + runtime)
  - Use golang:1.22-alpine for building
  - Use alpine:3.19 for runtime
  - Create non-root user
  - Copy binary only
  - Set entrypoint

- [ ] **10.2** Create docker-compose.yml (`deployments/docker-compose.yml`)
  - Redis service (message queue)
  - MinIO service (S3-compatible storage)
  - MinIO initialization (create bucket)
  - Canopy service in all-in-one mode
  - Health checks for all services
  - Volume persistence
  - Port mappings

- [ ] **10.3** Create .env.example (`deployments/.env.example`)
  - Document all required environment variables
  - Provide example values for local dev

- [ ] **10.4** Test local development setup
  - `docker-compose up` should start all services
  - Send test webhook to localhost:8080
  - Verify coverage processing works end-to-end
  - Check MinIO console for stored coverage
  - Verify Redis queue messages

### Phase 11: Terraform Infrastructure

- [ ] **11.1** Create Terraform providers config (`terraform/providers.tf`)
  - Google provider configuration
  - Required provider versions
  - Backend configuration (optional)

- [ ] **11.2** Create variables (`terraform/variables.tf`)
  - project_id, region
  - github_webhook_secret, github_app_id, github_installation_id, github_private_key
  - allowed_org, allowed_workflows
  - All other configurable values

- [ ] **11.3** Implement Pub/Sub resources (`terraform/pubsub.tf`)
  - Topic: canopy-coverage-requests
  - Subscription: canopy-worker-subscription
  - Dead letter topic and subscription
  - Retry and ack deadline configuration

- [ ] **11.4** Implement storage resources (`terraform/storage.tf`)
  - GCS bucket for coverage data
  - Versioning enabled
  - Lifecycle rules (90-day retention)
  - Uniform bucket-level access

- [ ] **11.5** Implement IAM resources (`terraform/iam.tf`)
  - Service account for initiator
  - Service account for worker
  - IAM bindings:
    - Initiator can publish to Pub/Sub
    - Worker can subscribe to Pub/Sub
    - Worker can read/write GCS
    - Worker can access secrets

- [ ] **11.6** Implement Secret Manager resources (`terraform/secrets.tf`)
  - Secret: webhook secret (for initiator)
  - Secret: GitHub App private key (for worker)
  - Secret versions
  - IAM bindings for secret access

- [ ] **11.7** Implement Cloud Run services (`terraform/cloud_run.tf`)
  - Artifact Registry repository
  - Initiator Cloud Run service:
    - Mode: initiator
    - Env vars: allowed_org, allowed_workflows, pubsub config
    - Service account
    - Public access (for GitHub webhooks)
    - Scaling: min 0, max 10
  - Worker Cloud Run service:
    - Mode: worker
    - Env vars: pubsub subscription, GCS bucket, GitHub credentials
    - Service account
    - Private (no public access)
    - Scaling: min 0, max 5
    - Longer timeout (10 minutes)

- [ ] **11.8** Create outputs (`terraform/outputs.tf`)
  - Initiator URL (webhook endpoint)
  - Worker URL
  - Coverage bucket name

- [ ] **11.9** Create Terraform documentation
  - Required GCP APIs to enable
  - Terraform initialization and apply steps
  - Variable configuration
  - GitHub App setup instructions

- [ ] **11.10** Test Terraform deployment
  - terraform init
  - terraform plan
  - terraform apply
  - Verify all resources created
  - Test webhook delivery to Cloud Run
  - Verify worker processes messages

### Phase 12: Documentation & Polish

- [ ] **12.1** Create comprehensive README.md
  - Project overview
  - Architecture diagram
  - Prerequisites
  - Local development setup
  - Production deployment guide
  - Configuration reference
  - Troubleshooting guide

- [ ] **12.2** Create Makefile
  - `make build` - Build binary
  - `make test` - Run tests
  - `make test-coverage` - Run tests with coverage report
  - `make docker-build` - Build Docker image
  - `make local-up` - Start docker-compose
  - `make local-down` - Stop docker-compose
  - `make lint` - Run linter

- [ ] **12.3** Create GitHub App setup guide
  - App permissions required
  - Webhook event subscriptions
  - Installation instructions
  - Testing instructions

- [ ] **12.4** Add logging throughout
  - Structured logging (JSON format)
  - Log levels: debug, info, warn, error
  - Request ID tracking
  - Avoid logging secrets

- [ ] **12.5** Add metrics/observability (optional)
  - Prometheus metrics endpoint
  - Key metrics: webhook requests, processing time, errors
  - Cloud Monitoring integration

## Key Technical Decisions

### Message Queue Interface
- Abstraction allows swapping Pub/Sub ↔ Redis without code changes
- WorkRequest is simple JSON-serializable struct
- Handler function pattern for processing

### Storage Interface
- Simple key-value semantics (org/repo/branch → coverage data)
- Abstraction allows GCS ↔ MinIO without code changes
- Coverage stored as serialized Go coverage profile format

### Coverage Merging
- Use algorithm from gocovmerge (proven, well-tested)
- Leverage `golang.org/x/tools/cover` for parsing/serialization
- Merge at block level for accuracy

### GitHub API Client
- Wrap google/go-github to simplify mocking
- GitHub App authentication (JWT + installation token)
- Interface-based design for testability

### Check Run Annotations
- GitHub limits 50 annotations per API call → batch updates
- Annotation level: "notice" for uncovered lines
- Include line ranges for multi-line blocks

### Security
- HMAC validation prevents unauthorized webhooks
- Only worker has GitHub credentials (principle of least privilege)
- Initiator passes minimal context (org, repo, run ID)
- Secrets stored in Secret Manager, not env vars

## Testing Strategy

### Unit Tests
- Every package has `_test.go` files
- Table-driven tests for edge cases
- Mock external dependencies (GitHub, storage, queue)
- Target: 80%+ code coverage

### Integration Tests
- Use testcontainers for Redis and MinIO
- Mock GitHub API with httptest
- Test full request/response cycles

### Required Bad Auth Tests
- Missing HMAC signature → 401
- Invalid HMAC signature → 401
- Wrong org → 403
- Disallowed workflow → 403
- Test --disable-hmac flag bypasses validation

### Edge Cases
- Empty coverage files
- No coverage artifacts
- PR with no Go files
- Storage/GitHub API failures

### Coverage Reporting
```makefile
test-coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
    @go tool cover -func=coverage.out | grep total | \
        awk '{print $$3}' | awk -F% '{if ($$1 < 80) exit 1}'
```

## Infrastructure Requirements

### GCP Resources
- 2 Cloud Run services (initiator, worker)
- 1 Pub/Sub topic + subscription + DLQ
- 1 GCS bucket
- 2 service accounts
- 2 Secret Manager secrets
- 1 Artifact Registry repository

### Local Development
- Redis (Docker)
- MinIO (Docker)
- GitHub App with test repository

### GitHub App Permissions
- **Repository permissions**:
  - Checks: Read & Write (create check runs)
  - Contents: Read (download artifacts, get files)
  - Pull requests: Read & Write (comment on PRs)
  - Metadata: Read (required)
  - Actions: Read (get workflow run info)

- **Webhook events**:
  - Workflow run (completed)

## Success Criteria

- [ ] All phases completed
- [ ] 80%+ test coverage achieved
- [ ] All bad auth scenarios tested
- [ ] Local development setup works (docker-compose up)
- [ ] Terraform deploys successfully to GCP
- [ ] End-to-end flow works: webhook → annotations on PR
- [ ] Documentation complete and accurate

## Timeline Estimate

- Phase 1-3 (Foundation): ~2 days
- Phase 4 (Coverage): ~2 days
- Phase 5-7 (Services): ~3 days
- Phase 8-9 (Testing): ~2 days
- Phase 10-11 (Infra): ~2 days
- Phase 12 (Docs): ~1 day

**Total: ~12 days** (for single developer, working full-time)

## Next Steps

1. Review this plan with team
2. Set up GitHub App for testing
3. Begin Phase 1 implementation
4. Set up CI/CD pipeline
5. Iterate based on testing feedback
