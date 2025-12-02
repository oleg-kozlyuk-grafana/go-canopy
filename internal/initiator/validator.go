package initiator

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidAction is returned when the workflow run action is not "completed"
	ErrInvalidAction = errors.New("workflow run action must be 'completed'")

	// ErrDisallowedOrg is returned when the organization is not in the allowed list
	ErrDisallowedOrg = errors.New("organization not allowed")

	// ErrDisallowedWorkflow is returned when the workflow name is not in the allowed list
	ErrDisallowedWorkflow = errors.New("workflow not allowed")
)

// Hardcoded allowed values per SPEC.md
var (
	allowedOrgs      = []string{"grafana"}
	allowedWorkflows = []string{"ci.yml", "build.yml"}
)

// WorkflowRunEvent represents the minimal structure of a GitHub workflow_run webhook event
// needed for validation. This matches the structure from GitHub's webhook payloads.
type WorkflowRunEvent struct {
	Action      string       `json:"action"`
	WorkflowRun WorkflowRun  `json:"workflow_run"`
	Repository  Repository   `json:"repository"`
	Organization Organization `json:"organization"`
}

// WorkflowRun contains workflow run details
type WorkflowRun struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Repository contains repository information
type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// Organization contains organization information
type Organization struct {
	Login string `json:"login"`
}

// ValidateEvent validates a GitHub workflow_run webhook event against the configured criteria.
// It checks:
// 1. Action is "completed"
// 2. Organization is in the allowed list
// 3. Workflow name is in the allowed list
//
// Returns nil if the event is valid, or a specific error otherwise.
func ValidateEvent(event *WorkflowRunEvent) error {
	// Check action is "completed"
	if event.Action != "completed" {
		return fmt.Errorf("%w: got %q", ErrInvalidAction, event.Action)
	}

	// Check organization is allowed
	org := event.Organization.Login
	if !contains(allowedOrgs, org) {
		return fmt.Errorf("%w: %q", ErrDisallowedOrg, org)
	}

	// Check workflow name is allowed
	// The workflow name from the event is the workflow file path (e.g., ".github/workflows/ci.yml")
	// We need to extract just the filename
	workflowName := event.WorkflowRun.Name
	if !contains(allowedWorkflows, workflowName) {
		return fmt.Errorf("%w: %q", ErrDisallowedWorkflow, workflowName)
	}

	return nil
}

// contains checks if a string slice contains a given string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
