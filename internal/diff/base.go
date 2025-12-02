package diff

import (
	"context"
	"fmt"
	"os/exec"
)

// GitBaseDiffSource implements DiffSource by running git diff to compare
// against a base reference (branch, tag, or commit SHA).
type GitBaseDiffSource struct {
	// BaseRef is the base reference to compare against (e.g., SHA, branch name, tag).
	BaseRef string
	// WorkDir is the directory to run git commands in.
	// If empty, uses the current working directory.
	WorkDir string
}

// NewGitBaseDiffSource creates a new GitBaseDiffSource for comparing against the specified base.
func NewGitBaseDiffSource(baseRef string, workDir string) *GitBaseDiffSource {
	return &GitBaseDiffSource{
		BaseRef: baseRef,
		WorkDir: workDir,
	}
}

// GetDiff executes `git diff <base>...HEAD` and returns the unified diff output.
// The triple-dot syntax finds the merge-base between base and HEAD, then diffs from there.
// This is ideal for PR contexts where you want all changes from the PR branch.
func (s *GitBaseDiffSource) GetDiff(ctx context.Context) ([]byte, error) {
	if s.BaseRef == "" {
		return nil, fmt.Errorf("base ref is required")
	}

	// git diff <base>...HEAD shows all changes from the merge-base to HEAD
	// This captures all commits in a PR branch relative to the base branch
	cmd := exec.CommandContext(ctx, "git", "diff", s.BaseRef+"...HEAD")
	if s.WorkDir != "" {
		cmd.Dir = s.WorkDir
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git diff failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	return output, nil
}
