package diff

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitHubAPIDiffSource_GetDiff_NotImplemented(t *testing.T) {
	source := NewGitHubAPIDiffSource("owner", "repo", 123, "base", "head")
	_, err := source.GetDiff(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestNewGitHubAPIDiffSource(t *testing.T) {
	source := NewGitHubAPIDiffSource("owner", "repo", 456, "abc123", "def456")

	assert.NotNil(t, source)
	assert.Equal(t, "owner", source.Owner)
	assert.Equal(t, "repo", source.Repo)
	assert.Equal(t, 456, source.PRNumber)
	assert.Equal(t, "abc123", source.BaseSHA)
	assert.Equal(t, "def456", source.HeadSHA)
}
