# Canopy

This folder contains scaffold for project Canopy, a small service that helps improve Go code coverage.

## Architecture

This project consists of separate executables for different deployment scenarios:

* `canopy` - Local coverage analysis tool (analyzes local coverage against git diff)
* `canopy-initiator` - GitHub webhook handler for production
* `canopy-worker` - Coverage processor for production
* `canopy-all-in-one` - Combined initiator + worker for local development/testing

### Production Architecture

In PROD, the architecture is:

```
GitHub --webhook-> canopy-initiator --message queue-> canopy-worker
                        |                                    |
                        |                                    |
                    (No GitHub creds)              (GitHub creds + Storage)
```

Deployment is planned in GCP using Terraform, Cloud Run and Pub/Sub as message queue

### Executables

**canopy** - Local development tool
- Analyzes local coverage files against current git diff
- No server, no external dependencies
- Usage: `canopy --coverage .coverage`

**canopy-initiator** - Production webhook handler
- Receives GitHub workflow_run webhooks
- Validates HMAC signatures
- Publishes work requests to message queue
- Does NOT have GitHub credentials (least privilege)
- Flags: `--port`, `--disable-hmac`

**canopy-worker** - Production coverage processor
- Subscribes to message queue
- Downloads artifacts, processes coverage
- Creates GitHub check runs and comments
- Has GitHub credentials and storage access
- No CLI flags (configured via environment variables)

**canopy-all-in-one** - Local development mode
- Runs both initiator and worker in single process
- Typically uses in-memory queue and MinIO storage
- Flags: `--port`, `--disable-hmac`

## Security

Only worker has GitHub credentials, initiator builds and sends a message with only org/repo name and workflow ID, providing isolated context. App should support authentication both as an app and via PAT.

## Workflow

### Initiator

1.Initiator receives a workflow_run webhook from GitHub
1.HMAC signature is validated and rejected if needed
1.Event type is matched == completed
1.Org is matched == grafana
1.Workflow name is validated against hardcoded list (ci.yml, build.yml)

If all conditions are satisfied, it sends a work request to worker via message queue - org/repo, workflow run ID

### Worker

1.Upon receiving the message, worker fetches run information and coverage artifacts - any artifacts named coverage*
1.If there is no coverage data, log info, quit
1.Coverage files are merged similar to gocovmerge (check source code)
1.If run is on default branch, merged coverage is saved to bucket & exit
1.If run is on PR
  1.get the PR diff via API
  1.get saved default branch coverage
  1.create check run on commit
  1.cross-reference PR diff with merged coverage, add "notice" annotations on uncovered lines
  1.calculate coverage rate of default coverage and PR
  1.set check text as "Project coverage {}%, change {}%" and set status as failed if change is negative
  1.leave a comment on PR with table that summarizes: `main` coverage, `pr` coverage, change %

## Tests

* Enable code coverage on the project and ensure 80% code coverage
* Ensure that tests cover the scenarios:
  * wrong repository
  * payload tampering

## Developer modes

### Local Coverage Analysis
```bash
# Analyze local coverage against git diff
canopy --coverage .coverage
```

### All-in-One Development Mode
```bash
# Run both initiator and worker together
canopy-all-in-one --port 8080 --disable-hmac
```

- Uses in-memory queue or Redis (configurable via env vars)
- Uses MinIO for storage (via docker-compose)
- `--disable-hmac` flag disables HMAC validation for local webhooks

### Docker Compose
```bash
# Start Redis and MinIO for local testing
make local-up

# Then run all-in-one mode
make run-all-in-one
```

### Building
```bash
# Build all executables
make build-all

# Or build individually
make build-canopy
make build-initiator
make build-worker
make build-all-in-one
```
