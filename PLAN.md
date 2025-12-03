# Canopy Implementation Plan

## Overview

Canopy is a Go service that provides code coverage annotations on GitHub pull requests. It receives GitHub workflow_run webhooks, processes Go coverage files, and posts check runs with annotations highlighting uncovered lines.

## Architecture

### Deployment Modes

1. **All-in-one mode**: Single process for local development (webhook + worker)
2. **Split mode**: Separate webhook and worker processes for production

### Production Flow

```
GitHub Webhook → Webhook Handler (Cloud Run) → Pub/Sub → Worker (Cloud Run) → GitHub API
                                                          ↓
                                                     GCS Bucket
```

### Security Model

- Webhook Handler: No GitHub credentials, validates HMAC, publishes to queue
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
│   ├── webhook/
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
│   ├── cloud_run.tf                   # Webhook & Worker services
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
  - `--mode` (all-in-one|webhook|worker)
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

- [x] **3.2** Implement GCS adapter (`internal/storage/gcs.go`)
  - Client initialization with credentials
  - Object path format: `{org}/{repo}/{branch}/coverage.out`
  - Upload with retry logic
  - Download with not-found handling
  - Tests for validation and error handling

- [x] **3.3** Implement MinIO adapter (`internal/storage/minio.go`)
  - Client initialization (endpoint, credentials, SSL)
  - S3-compatible API usage
  - Bucket existence check
  - Error handling

- [x] **3.4** Write storage tests
  - Test save/get cycle
  - Test not-found scenarios
  - Test error handling
  - Integration tests with testcontainers

### Phase 4: Coverage Processing

- [x] **4.1** Implement coverage parser (`internal/coverage/parser.go`)
  - Parse standard Go coverage format using `golang.org/x/tools/cover`
  - Handle coverage files from zip archives (workflow artifacts)
  - Validate coverage profile format
  - Error handling for malformed files
  - **Tests**:
    - Test parsing valid coverage files
    - Test handling malformed coverage data
    - Test empty files and edge cases

- [x] **4.2** Implement coverage merger (`internal/coverage/merger.go`)
  - Merge multiple coverage profiles (gocovmerge algorithm)
  - Handle overlapping coverage blocks
  - Serialize merged profiles back to standard format
  - Optimize for performance
  - **Tests**:
    - Test merging multiple coverage files with sample data in `testdata/coverage/`
    - Test overlapping blocks are merged correctly
    - Test single file merge (no-op)
    - Test empty coverage handling

- [x] **4.3** Implement diff parser (`internal/coverage/diff.go`)
  - Parse unified diff format
  - Extract added lines per file
  - Map GitHub CommitFile objects to line numbers
  - Handle binary files and renames
  - **Tests**:
    - Test parsing unified diffs with sample data in `testdata/diffs/`
    - Test extraction of added lines
    - Test binary file handling
    - Test file renames

- [x] **4.4** Implement coverage analysis (`internal/coverage/analysis.go`)
  - Calculate overall coverage percentage
  - Calculate per-file coverage
  - Cross-reference coverage with diff (find uncovered added lines)
  - Generate GitHub Check Run annotations
  - Compare base vs head coverage
  - **Tests**:
    - Test coverage percentage calculations
    - Test finding uncovered lines in diff
    - Test annotation generation
    - Test coverage comparison (increase/decrease)
    - Integration test for full analysis pipeline

### Phase 5: Webhook Handler Service

- [x] **5.1** Implement HMAC validation (`internal/webhook/hmac.go`)
  - Parse `X-Hub-Signature-256` header
  - Compute HMAC-SHA256 of payload
  - Constant-time comparison
  - Return specific errors for debugging
  - **Tests**:
    - Test valid HMAC signature
    - Test invalid HMAC signature → returns error
    - Test missing signature header → returns error
    - Test malformed header → returns error
    - Test constant-time comparison (timing attack resistance)

- [x] **5.2** Implement event validator (`internal/initiator/validator.go`)
  - Check event action == "completed"
  - Check org matches allowed list
  - Check workflow name in allowed list
  - Return specific validation errors
  - **Tests**:
    - Test valid event passes validation
    - Test wrong org → returns error
    - Test disallowed workflow → returns error
    - Test non-completed action → returns error

- [ ] **5.3** Implement webhook handler (`internal/webhook/handler.go`)
  - Parse webhook payload
  - Validate HMAC signature (unless disabled)
  - Validate event criteria
  - Build WorkRequest message
  - Publish to queue
  - Return appropriate HTTP status codes
  - **Tests**:
    - Test valid webhook end-to-end processing
    - Test HMAC disabled (--disable-hmac flag) bypasses validation
    - Test malformed JSON payload → 400
    - Test validation failures → appropriate status codes
    - Mock queue to verify message publishing

- [ ] **5.4** Wire up webhook handler in main.go
  - Initialize queue client
  - Create handler with dependencies
  - Register route: POST /webhook
  - Start HTTP server
  - **Tests**:
    - Integration test with test HTTP server
    - Test graceful shutdown

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
  - **Tests**:
    - Test GitHub App authentication flow (JWT generation)
    - Test each API wrapper method with httptest mock server
    - Test error handling (rate limiting, 404 not found, 403 forbidden, network errors)
    - Test token refresh logic

- [ ] **6.2** Implement mock GitHub client (`internal/github/mock.go`)
  - Mock implementation of Client interface
  - Configurable responses for testing
  - Error injection capabilities
  - **Tests**:
    - Test mock returns configured responses
    - Test error injection works as expected

### Phase 7: Worker Service

- [ ] **7.1** Implement artifact fetcher (`internal/worker/artifacts.go`)
  - List artifacts for workflow run
  - Filter by name pattern (e.g., "coverage*.out", "coverage*.txt")
  - Download artifact zip files
  - Extract coverage files from zip
  - Return raw coverage data
  - **Tests**:
    - Test listing and filtering artifacts by pattern
    - Test downloading and extracting zip files
    - Test no matching artifacts found
    - Test malformed zip files

- [ ] **7.2** Implement check run manager (`internal/worker/checkrun.go`)
  - Create initial check run with "in_progress" status
  - Update check run with annotations (batch in groups of 50)
  - Set final status (success/failure based on coverage change)
  - Format check run summary: "Project coverage X%, change Y%"
  - Handle GitHub API errors
  - **Tests**:
    - Test creating check run
    - Test updating check run with annotations (verify batching at 50)
    - Test setting final status (success/failure)
    - Test summary formatting
    - Test GitHub API error handling

- [ ] **7.3** Implement PR comment manager (`internal/worker/comment.go`)
  - Generate markdown table with coverage comparison
  - Search for existing bot comment
  - Create new comment or update existing
  - Format: main coverage, PR coverage, change delta
  - **Tests**:
    - Test markdown table generation
    - Test finding existing comment
    - Test creating new comment
    - Test updating existing comment

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
  - **Tests**:
    - Test default branch flow (save coverage to storage)
    - Test PR flow end-to-end (check run, annotations, comment)
    - Test no coverage artifacts found (log and exit gracefully)
    - Test no base coverage available (first PR on branch)
    - Test GitHub API errors (retry and error handling)
    - Test storage errors
    - Mock all external dependencies (GitHub, storage, coverage)

- [ ] **7.5** Wire up worker in main.go
  - Initialize GitHub client with App credentials
  - Initialize storage client
  - Initialize queue subscriber
  - Create worker with dependencies
  - Subscribe to queue with worker.ProcessWorkRequest handler
  - Handle graceful shutdown
  - **Tests**:
    - Integration test with mocked queue and dependencies
    - Test graceful shutdown on signal

### Phase 8: All-in-One Mode

- [ ] **8.1** Implement combined mode in main.go
  - Use in-memory queue
  - Start worker goroutine
  - Start webhook HTTP server
  - Handle graceful shutdown of both
  - **Tests**:
    - End-to-end integration test: webhook → coverage processing flow
    - Use real Redis and MinIO via testcontainers
    - Mock GitHub API
    - Test graceful shutdown of both components

### Phase 9: Testing & Quality

**Note:** Most tests are integrated into implementation phases (4-8). This phase covers cross-cutting concerns.

- [ ] **9.1** Create comprehensive test fixtures (`testdata/`)
  - Valid webhook payloads in `testdata/webhooks/`
    - workflow_run completed event (default branch)
    - workflow_run completed event (PR)
    - Various org/workflow combinations
  - Sample coverage files in `testdata/coverage/`
    - Single file coverage
    - Multi-file coverage for merging
    - Edge cases (empty, malformed)
  - Sample PR diffs in `testdata/diffs/`
    - Simple additions
    - File renames
    - Binary files
    - Mixed changes

- [ ] **9.2** Verify security test coverage (required by SPEC.md)
  - Ensure HMAC validation tests exist in Phase 5:
    - Missing HMAC signature → 401
    - Invalid HMAC signature → 401
    - Malformed signature header → 401
    - Wrong organization → 403
    - Disallowed workflow → 403
    - --disable-hmac flag bypasses validation
  - Review and run security tests

- [ ] **9.3** Enforce 80% code coverage threshold
  - Run coverage reports: `go test -cover ./...`
  - Identify untested code paths
  - Add missing unit tests where needed
  - Update Makefile coverage target to fail below 80%
  - Document coverage requirements in README

### Phase 10: Continuous Integration

- [ ] **10.1** Create GitHub Actions workflow (`.github/workflows/ci.yml`)
  - **Trigger**: Pull requests and pushes to main branch
  - **Jobs**:
    - **Lint**: Run golangci-lint
      - Use `golangci/golangci-lint-action@v4`
      - Configure timeout (5 minutes)
    - **Test**: Run tests with coverage
      - Go version matrix (1.22, 1.23)
      - Run `go test -race -coverprofile=coverage.out -covermode=atomic ./...`
      - Fail if coverage < 80% threshold
    - **Build**: Build binary and Docker image
      - Build for linux/amd64
      - Build Docker image
      - (Optional) Push to GitHub Container Registry on main branch
  - **Integration Tests** (optional separate job):
    - Use testcontainers for Redis and MinIO
    - Run integration tests with `go test -tags=integration`
  - **Tests**:
    - Verify workflow runs on sample PRs
    - Test that coverage threshold enforcement works
    - Test that build failures fail the workflow

- [ ] **10.2** Configure golangci-lint (`.golangci.yml`)
  - Enable linters: gofmt, govet, staticcheck, unused, gosimple, ineffassign
  - Configure rules for project style
  - Set timeout and concurrency
  - Define skip patterns (e.g., generated code)

- [ ] **10.3** Add status badges to README.md
  - CI status badge
  - Go Report Card badge (optional)

### Phase 11: Docker & Local Development

- [ ] **11.1** Create Dockerfile (`deployments/Dockerfile`)
  - Multi-stage build (builder + runtime)
  - Use golang:1.22-alpine for building
  - Use alpine:3.19 for runtime
  - Create non-root user
  - Copy binary only
  - Set entrypoint

- [ ] **11.2** Create docker-compose.yml (`deployments/docker-compose.yml`)
  - Redis service (message queue)
  - MinIO service (S3-compatible storage)
  - MinIO initialization (create bucket)
  - Canopy service in all-in-one mode
  - Health checks for all services
  - Volume persistence
  - Port mappings

- [ ] **11.3** Create .env.example (`deployments/.env.example`)
  - Document all required environment variables
  - Provide example values for local dev

- [ ] **11.4** Test local development setup
  - `docker-compose up` should start all services
  - Send test webhook to localhost:8080
  - Verify coverage processing works end-to-end
  - Check MinIO console for stored coverage
  - Verify Redis queue messages

### Phase 12: Terraform Infrastructure

- [ ] **12.1** Create Terraform providers config (`terraform/providers.tf`)
  - Google provider configuration
  - Required provider versions
  - Backend configuration (optional)

- [ ] **12.2** Create variables (`terraform/variables.tf`)
  - project_id, region
  - github_webhook_secret, github_app_id, github_installation_id, github_private_key
  - allowed_org, allowed_workflows
  - All other configurable values

- [ ] **12.3** Implement Pub/Sub resources (`terraform/pubsub.tf`)
  - Topic: canopy-coverage-requests
  - Subscription: canopy-worker-subscription
  - Dead letter topic and subscription
  - Retry and ack deadline configuration

- [ ] **12.4** Implement storage resources (`terraform/storage.tf`)
  - GCS bucket for coverage data
  - Versioning enabled
  - Lifecycle rules (90-day retention)
  - Uniform bucket-level access

- [ ] **12.5** Implement IAM resources (`terraform/iam.tf`)
  - Service account for webhook
  - Service account for worker
  - IAM bindings:
    - Webhook can publish to Pub/Sub
    - Worker can subscribe to Pub/Sub
    - Worker can read/write GCS
    - Worker can access secrets

- [ ] **12.6** Implement Secret Manager resources (`terraform/secrets.tf`)
  - Secret: webhook secret (for webhook handler)
  - Secret: GitHub App private key (for worker)
  - Secret versions
  - IAM bindings for secret access

- [ ] **12.7** Implement Cloud Run services (`terraform/cloud_run.tf`)
  - Artifact Registry repository
  - Webhook Cloud Run service:
    - Mode: webhook
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

- [ ] **12.8** Create outputs (`terraform/outputs.tf`)
  - Webhook URL (webhook endpoint)
  - Worker URL
  - Coverage bucket name

- [ ] **12.9** Create Terraform documentation
  - Required GCP APIs to enable
  - Terraform initialization and apply steps
  - Variable configuration
  - GitHub App setup instructions

- [ ] **12.10** Test Terraform deployment
  - terraform init
  - terraform plan
  - terraform apply
  - Verify all resources created
  - Test webhook delivery to Cloud Run
  - Verify worker processes messages

### Phase 13: Documentation & Polish

- [ ] **13.1** Create comprehensive README.md
  - Project overview
  - Architecture diagram
  - Prerequisites
  - Local development setup
  - Production deployment guide
  - Configuration reference
  - Troubleshooting guide

- [ ] **13.2** Create Makefile
  - `make build` - Build binary
  - `make test` - Run tests
  - `make test-coverage` - Run tests with coverage report
  - `make docker-build` - Build Docker image
  - `make local-up` - Start docker-compose
  - `make local-down` - Stop docker-compose
  - `make lint` - Run linter

- [ ] **13.3** Create GitHub App setup guide
  - App permissions required
  - Webhook event subscriptions
  - Installation instructions
  - Testing instructions

- [ ] **13.4** Add logging throughout
  - Structured logging (JSON format)
  - Log levels: debug, info, warn, error
  - Request ID tracking
  - Avoid logging secrets

- [ ] **13.5** Add metrics/observability (optional)
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
- Webhook handler passes minimal context (org, repo, run ID)
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
- 2 Cloud Run services (webhook, worker)
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

## Next Steps

1. Review this plan with team
2. Set up GitHub App for testing
3. Begin Phase 1 implementation
4. Set up CI/CD pipeline
5. Iterate based on testing feedback
