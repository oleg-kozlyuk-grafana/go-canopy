package format

import (
	"bytes"
	"testing"

	"github.com/oleg-kozlyuk/canopy/internal/coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownFormatter_Format(t *testing.T) {
	tests := []struct {
		name           string
		result         *coverage.AnalysisResult
		expectedOutput string
		expectError    bool
	}{
		{
			name: "single file with uncovered lines",
			result: &coverage.AnalysisResult{
				UncoveredByFile: map[string][]int{
					"main.go": {5, 10, 15},
				},
				TotalAdded:     20,
				TotalUncovered: 3,
			},
			expectedOutput: `## Uncovered Lines in Diff

| File | Lines |
|------|-------|
| main.go | 5, 10, 15 |

**Summary:** 3 uncovered lines out of 20 added (85.0% coverage)
`,
		},
		{
			name: "multiple files with uncovered lines",
			result: &coverage.AnalysisResult{
				UncoveredByFile: map[string][]int{
					"handler.go": {10, 15, 20},
					"main.go":    {5, 7},
				},
				TotalAdded:     25,
				TotalUncovered: 5,
			},
			expectedOutput: `## Uncovered Lines in Diff

| File | Lines |
|------|-------|
| handler.go | 10, 15, 20 |
| main.go | 5, 7 |

**Summary:** 5 uncovered lines out of 25 added (80.0% coverage)
`,
		},
		{
			name: "all lines covered",
			result: &coverage.AnalysisResult{
				UncoveredByFile: map[string][]int{},
				TotalAdded:      10,
				TotalUncovered:  0,
			},
			expectedOutput: "All added lines are covered!\n",
		},
		{
			name: "no lines added",
			result: &coverage.AnalysisResult{
				UncoveredByFile: map[string][]int{},
				TotalAdded:      0,
				TotalUncovered:  0,
			},
			expectedOutput: "No lines added in diff\n",
		},
		{
			name: "lines with ranges",
			result: &coverage.AnalysisResult{
				UncoveredByFile: map[string][]int{
					"server.go": {5, 6, 7, 10, 11, 15, 20, 21, 22, 23},
				},
				TotalAdded:     15,
				TotalUncovered: 10,
			},
			expectedOutput: `## Uncovered Lines in Diff

| File | Lines |
|------|-------|
| server.go | 5-7, 10-11, 15, 20-23 |

**Summary:** 10 uncovered lines out of 15 added (33.3% coverage)
`,
		},
		{
			name:        "nil result",
			result:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &MarkdownFormatter{}
			var buf bytes.Buffer
			err := formatter.Format(tt.result, &buf)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}
