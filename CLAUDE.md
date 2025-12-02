# Canopy Development Guide

This file provides context and guidelines for working on the Canopy project.

## Project Overview

Canopy is a Go service that provides code coverage annotations on GitHub pull requests. It receives GitHub workflow_run webhooks, processes Go coverage files, and posts check runs with annotations highlighting uncovered lines.

**Deployment Modes:**
- **All-in-one**: Single process for local development (initiator + worker)
- **Split mode**: Separate initiator and worker processes for production

**Production Flow:**
```
GitHub Webhook → Initiator → Message Queue → Worker → GitHub API + Storage
```

## Architecture Principles

### Security Model
- **Initiator**: No GitHub credentials, validates HMAC, publishes to queue
- **Worker**: Has GitHub credentials, processes coverage, updates PRs
- This separation follows principle of least privilege

### Interface-Based Design
All external dependencies use interfaces to enable testing and flexibility:
- `MessageQueue` interface: Pub/Sub (prod), Redis (local), in-memory (all-in-one)
- `Storage` interface: GCS (prod), MinIO (local)
- `GitHubClient` interface: Real client or mock for testing

## Code Organization

```
internal/
├── config/         # Configuration management (env vars, flags, validation)
├── initiator/      # Webhook handler, HMAC validation, event filtering
├── worker/         # Coverage processing orchestration
├── coverage/       # Coverage parsing, merging (gocovmerge algorithm), analysis
├── queue/          # Message queue abstraction and implementations
├── storage/        # Storage abstraction and implementations
├── github/         # GitHub API client wrapper
└── server/         # HTTP server with middleware
```

## Development Guidelines

### Testing Requirements
- **Coverage target**: 80% minimum
- **Required security tests** (per SPEC.md):
  - Invalid HMAC signatures
  - Payload tampering
  - Wrong repository/organization
  - --disable-hmac flag behavior
- **Test strategy**:
  - Unit tests for all packages with table-driven approach
  - Mock all external dependencies (GitHub, storage, queue)
  - Integration tests using testcontainers for Redis/MinIO
- **Plan tests alongside implementation** (per user instructions)

### Code Style
- Follow standard Go conventions
- Use structured logging (JSON format)
- Never log secrets
- Implement graceful shutdown for all services
- Use context for cancellation and timeouts

### Key Implementation Details

#### Coverage Merging
- Use algorithm from gocovmerge (proven, well-tested)
- Leverage `golang.org/x/tools/cover` for parsing
- Merge at block level for accuracy

#### GitHub Check Runs
- Batch annotations in groups of 50 (GitHub API limit)
- Use "notice" level for uncovered lines
- Summary format: "Project coverage X%, change Y%"
- Set status to failed if coverage decreases

#### HMAC Validation
- Parse `X-Hub-Signature-256` header
- Use constant-time comparison
- Support --disable-hmac flag for local dev

#### Initiator Validation Flow
1. Validate HMAC signature (unless disabled)
2. Check event action == "completed"
3. Check org in allowed list (hardcoded: "grafana")
4. Check workflow name in allowed list (hardcoded: "ci.yml", "build.yml")
5. Publish WorkRequest to queue

#### Worker Processing Flow
1. Fetch workflow run details
2. Download coverage artifacts (pattern: coverage*)
3. If no coverage found: log and exit
4. Merge coverage files
5. **If default branch**: Save to storage and exit
6. **If PR**:
   - Get PR diff
   - Get base coverage from storage
   - Create check run
   - Analyze coverage, find uncovered added lines
   - Add annotations for uncovered lines
   - Update check run with summary
   - Post/update PR comment with coverage table
   - Set check status (fail if coverage decreased)

### Storage Paths
Coverage stored at: `{org}/{repo}/{branch}/coverage.out`

### Common Commands
```bash
make build              # Build binary
make test               # Run tests
make test-coverage      # Run tests with coverage report
make docker-build       # Build Docker image
make local-up           # Start docker-compose (Redis + MinIO)
make local-down         # Stop docker-compose
```

## Current Progress

Refer to PLAN.md for detailed phase-by-phase task list. Completed phases:
- Phase 1: Foundation & Configuration ✓
- Phase 2: Message Queue Abstraction ✓

Next phases:
- Phase 3: Storage Abstraction
- Phase 4: Coverage Processing
- Phase 5-7: Service Implementation

## Important Constraints

1. **Hardcoded values** (per SPEC.md):
   - Allowed org: "grafana"
   - Allowed workflows: "ci.yml", "build.yml"

2. **Artifact naming**: Coverage artifacts must match pattern "coverage*"

3. **Message format**: WorkRequest contains only org, repo, workflow_run_id (minimal context)

4. **Configuration sources** (precedence order):
   - Command-line flags (highest)
   - Environment variables
   - Config file (optional, lowest)

5. **Error handling**:
   - Log errors with context
   - Return appropriate HTTP status codes
   - Handle GitHub API rate limiting
   - Implement retry logic for transient failures

## Dependencies

Key packages:
- `github.com/google/go-github/v57` - GitHub API
- `golang.org/x/tools/cover` - Coverage parsing
- `cloud.google.com/go/pubsub` - Pub/Sub
- `cloud.google.com/go/storage` - GCS
- `github.com/redis/go-redis/v9` - Redis
- `github.com/minio/minio-go/v7` - MinIO
- `github.com/spf13/cobra` - CLI framework

## Local Development

Use docker-compose for local testing:
```bash
docker-compose -f deployments/docker-compose.yml up
```

This starts:
- Redis (message queue)
- MinIO (S3-compatible storage)
- Canopy in all-in-one mode

Access MinIO console at http://localhost:9001 to inspect stored coverage files.

## When Working on This Project

1. **Before modifying code**: Read existing implementation to understand patterns
2. **When adding features**: Follow existing interface patterns
3. **When adding tests**: Aim for table-driven tests with clear test case names
4. **When implementing a phase**: Check PLAN.md task list and mark items complete
5. **When uncertain**: Refer to SPEC.md for requirements, PLAN.md for approach
6. **Always plan tests alongside implementation**

## Questions or Issues?

- Check SPEC.md for requirements clarification
- Check PLAN.md for implementation approach
- Refer to existing code for patterns
- Test edge cases thoroughly (empty files, missing data, API errors)
