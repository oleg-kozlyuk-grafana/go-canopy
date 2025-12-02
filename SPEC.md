# Canopy

This folder contains scaffold for project Canopy, a small service that helps improve Go code coverage.

## Architecture

This project should be a monolithic executable, deployed in 2 modes:

* all-in-one mode for local deployments and testing
* split mode with "initiator" and "worker"

In PROD, the architecture is:

GitHub --webhook-> Initiator --message queue->worker

Deployment is planned in GCP using Terraform, Cloud Run and Pub/Sub as message queue

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

- all-in-one mode
- -disable-hmac to disable HMAC validation for local development
- docker compose with Redis as message queue and minio as s3
