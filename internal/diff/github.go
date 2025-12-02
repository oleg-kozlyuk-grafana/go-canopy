package diff

import (
	"context"
	"fmt"
)

// GitHubAPIDiffSource implements DiffSource by fetching diff data from the GitHub API.
// This is used in the worker service to get PR diffs without needing a local checkout.
type GitHubAPIDiffSource struct {
	// Client is the GitHub API client.
	// For now this is a placeholder - the actual type will be determined
	// when implementing the worker service.
	Client interface{}

	// Owner is the repository owner (org or user).
	Owner string
	// Repo is the repository name.
	Repo string
	// PRNumber is the pull request number (0 for non-PR diffs).
	PRNumber int
	// BaseSHA is the base commit SHA for comparison.
	BaseSHA string
	// HeadSHA is the head commit SHA for comparison.
	HeadSHA string
}

// NewGitHubAPIDiffSource creates a new GitHubAPIDiffSource.
// This is a placeholder implementation for the worker service.
func NewGitHubAPIDiffSource(owner, repo string, prNumber int, baseSHA, headSHA string) *GitHubAPIDiffSource {
	return &GitHubAPIDiffSource{
		Owner:    owner,
		Repo:     repo,
		PRNumber: prNumber,
		BaseSHA:  baseSHA,
		HeadSHA:  headSHA,
	}
}

// GetDiff fetches the diff from the GitHub API.
// This is a stub implementation - the actual implementation will use
// the go-github client to fetch PR diffs or compare commits.
func (s *GitHubAPIDiffSource) GetDiff(ctx context.Context) ([]byte, error) {
	// TODO: Implement when building the worker service
	// Options for implementation:
	// 1. For PRs: client.PullRequests.GetRaw(ctx, owner, repo, prNumber, opts)
	//    with MediaType set to "diff" to get unified diff format
	// 2. For commits: client.Repositories.CompareCommits(ctx, owner, repo, base, head, opts)
	//    and extract the diff data
	return nil, fmt.Errorf("GitHubAPIDiffSource not yet implemented")
}
