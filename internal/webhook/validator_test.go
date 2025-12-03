package webhook

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEvent(t *testing.T) {
	tests := []struct {
		name        string
		event       *WorkflowRunEvent
		wantErr     error
		errContains string
	}{
		{
			name: "valid event - grafana org, ci.yml workflow",
			event: &WorkflowRunEvent{
				Action: "completed",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "ci.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "grafana/myrepo",
				},
				Organization: Organization{
					Login: "grafana",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid event - grafana org, build.yml workflow",
			event: &WorkflowRunEvent{
				Action: "completed",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "build.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "grafana/myrepo",
				},
				Organization: Organization{
					Login: "grafana",
				},
			},
			wantErr: nil,
		},
		{
			name: "invalid action - requested",
			event: &WorkflowRunEvent{
				Action: "requested",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "ci.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "grafana/myrepo",
				},
				Organization: Organization{
					Login: "grafana",
				},
			},
			wantErr:     ErrInvalidAction,
			errContains: "requested",
		},
		{
			name: "invalid action - in_progress",
			event: &WorkflowRunEvent{
				Action: "in_progress",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "ci.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "grafana/myrepo",
				},
				Organization: Organization{
					Login: "grafana",
				},
			},
			wantErr:     ErrInvalidAction,
			errContains: "in_progress",
		},
		{
			name: "invalid action - empty string",
			event: &WorkflowRunEvent{
				Action: "",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "ci.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "grafana/myrepo",
				},
				Organization: Organization{
					Login: "grafana",
				},
			},
			wantErr:     ErrInvalidAction,
			errContains: `got ""`,
		},
		{
			name: "disallowed org - different org",
			event: &WorkflowRunEvent{
				Action: "completed",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "ci.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "otherorg/myrepo",
				},
				Organization: Organization{
					Login: "otherorg",
				},
			},
			wantErr:     ErrDisallowedOrg,
			errContains: "otherorg",
		},
		{
			name: "disallowed org - empty string",
			event: &WorkflowRunEvent{
				Action: "completed",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "ci.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "myrepo",
				},
				Organization: Organization{
					Login: "",
				},
			},
			wantErr:     ErrDisallowedOrg,
			errContains: `""`,
		},
		{
			name: "disallowed workflow - test.yml",
			event: &WorkflowRunEvent{
				Action: "completed",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "test.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "grafana/myrepo",
				},
				Organization: Organization{
					Login: "grafana",
				},
			},
			wantErr:     ErrDisallowedWorkflow,
			errContains: "test.yml",
		},
		{
			name: "disallowed workflow - empty string",
			event: &WorkflowRunEvent{
				Action: "completed",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "grafana/myrepo",
				},
				Organization: Organization{
					Login: "grafana",
				},
			},
			wantErr:     ErrDisallowedWorkflow,
			errContains: `""`,
		},
		{
			name: "disallowed workflow - similar but wrong name",
			event: &WorkflowRunEvent{
				Action: "completed",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "ci-test.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "grafana/myrepo",
				},
				Organization: Organization{
					Login: "grafana",
				},
			},
			wantErr:     ErrDisallowedWorkflow,
			errContains: "ci-test.yml",
		},
		{
			name: "disallowed workflow - case sensitive",
			event: &WorkflowRunEvent{
				Action: "completed",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "CI.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "grafana/myrepo",
				},
				Organization: Organization{
					Login: "grafana",
				},
			},
			wantErr:     ErrDisallowedWorkflow,
			errContains: "CI.yml",
		},
		{
			name: "multiple validation failures - wrong action takes precedence",
			event: &WorkflowRunEvent{
				Action: "requested",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "test.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "otherorg/myrepo",
				},
				Organization: Organization{
					Login: "otherorg",
				},
			},
			wantErr:     ErrInvalidAction,
			errContains: "requested",
		},
		{
			name: "multiple validation failures - wrong org when action valid",
			event: &WorkflowRunEvent{
				Action: "completed",
				WorkflowRun: WorkflowRun{
					ID:   12345,
					Name: "test.yml",
				},
				Repository: Repository{
					Name:     "myrepo",
					FullName: "otherorg/myrepo",
				},
				Organization: Organization{
					Login: "otherorg",
				},
			},
			wantErr:     ErrDisallowedOrg,
			errContains: "otherorg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEvent(tt.event)

			if tt.wantErr == nil {
				assert.NoError(t, err, "expected no error but got: %v", err)
			} else {
				require.Error(t, err, "expected error but got none")
				assert.True(t, errors.Is(err, tt.wantErr),
					"expected error to be %v, got %v", tt.wantErr, err)

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains,
						"expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		str   string
		want  bool
	}{
		{
			name:  "empty slice",
			slice: []string{},
			str:   "test",
			want:  false,
		},
		{
			name:  "string present",
			slice: []string{"a", "b", "c"},
			str:   "b",
			want:  true,
		},
		{
			name:  "string not present",
			slice: []string{"a", "b", "c"},
			str:   "d",
			want:  false,
		},
		{
			name:  "empty string in slice",
			slice: []string{"a", "", "c"},
			str:   "",
			want:  true,
		},
		{
			name:  "case sensitive",
			slice: []string{"Test", "test"},
			str:   "TEST",
			want:  false,
		},
		{
			name:  "single element slice - match",
			slice: []string{"only"},
			str:   "only",
			want:  true,
		},
		{
			name:  "single element slice - no match",
			slice: []string{"only"},
			str:   "other",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.str)
			assert.Equal(t, tt.want, got,
				"contains(%v, %q) = %v, want %v", tt.slice, tt.str, got, tt.want)
		})
	}
}

func TestWorkflowRunEvent_Structure(t *testing.T) {
	// Test that the struct properly holds all required fields
	event := WorkflowRunEvent{
		Action: "completed",
		WorkflowRun: WorkflowRun{
			ID:   999,
			Name: "test-workflow.yml",
		},
		Repository: Repository{
			Name:     "test-repo",
			FullName: "test-org/test-repo",
		},
		Organization: Organization{
			Login: "test-org",
		},
	}

	assert.Equal(t, "completed", event.Action)
	assert.Equal(t, int64(999), event.WorkflowRun.ID)
	assert.Equal(t, "test-workflow.yml", event.WorkflowRun.Name)
	assert.Equal(t, "test-repo", event.Repository.Name)
	assert.Equal(t, "test-org/test-repo", event.Repository.FullName)
	assert.Equal(t, "test-org", event.Organization.Login)
}
