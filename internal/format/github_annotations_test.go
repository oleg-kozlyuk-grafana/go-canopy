package format

import (
	"bytes"
	"testing"

	"github.com/oleg-kozlyuk-grafana/go-canopy/internal/coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubAnnotationsFormatter_Format(t *testing.T) {
	tests := []struct {
		name           string
		result         *coverage.AnalysisResult
		expectedOutput string
		expectError    bool
	}{
		{
			name: "single file with non-consecutive uncovered lines",
			result: &coverage.AnalysisResult{
				UncoveredByFile: map[string][]int{
					"main.go": {5, 10, 15},
				},
				TotalAdded:     20,
				TotalUncovered: 3,
			},
			expectedOutput: `::notice file=main.go,line=5,title=Uncovered line::Line 5 is not covered by tests
::notice file=main.go,line=10,title=Uncovered line::Line 10 is not covered by tests
::notice file=main.go,line=15,title=Uncovered line::Line 15 is not covered by tests
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
			expectedOutput: `::notice file=handler.go,line=10,title=Uncovered line::Line 10 is not covered by tests
::notice file=handler.go,line=15,title=Uncovered line::Line 15 is not covered by tests
::notice file=handler.go,line=20,title=Uncovered line::Line 20 is not covered by tests
::notice file=main.go,line=5,title=Uncovered line::Line 5 is not covered by tests
::notice file=main.go,line=7,title=Uncovered line::Line 7 is not covered by tests
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
			name: "consecutive lines grouped into ranges",
			result: &coverage.AnalysisResult{
				UncoveredByFile: map[string][]int{
					"server.go": {5, 6, 7, 10, 11, 15, 20, 21, 22, 23},
				},
				TotalAdded:     15,
				TotalUncovered: 10,
			},
			expectedOutput: `::notice file=server.go,line=5,endLine=7,title=Uncovered lines::Lines 5-7 are not covered by tests
::notice file=server.go,line=10,endLine=11,title=Uncovered lines::Lines 10-11 are not covered by tests
::notice file=server.go,line=15,title=Uncovered line::Line 15 is not covered by tests
::notice file=server.go,line=20,endLine=23,title=Uncovered lines::Lines 20-23 are not covered by tests
`,
		},
		{
			name: "single consecutive line range",
			result: &coverage.AnalysisResult{
				UncoveredByFile: map[string][]int{
					"test.go": {1, 2, 3, 4, 5},
				},
				TotalAdded:     10,
				TotalUncovered: 5,
			},
			expectedOutput: `::notice file=test.go,line=1,endLine=5,title=Uncovered lines::Lines 1-5 are not covered by tests
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
			formatter := &GitHubAnnotationsFormatter{}
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

func TestGroupIntoRanges(t *testing.T) {
	tests := []struct {
		name     string
		lines    []int
		expected []lineRange
	}{
		{
			name:     "empty list",
			lines:    []int{},
			expected: nil,
		},
		{
			name:  "single line",
			lines: []int{5},
			expected: []lineRange{
				{start: 5, end: 5},
			},
		},
		{
			name:  "two consecutive lines",
			lines: []int{5, 6},
			expected: []lineRange{
				{start: 5, end: 6},
			},
		},
		{
			name:  "three consecutive lines",
			lines: []int{5, 6, 7},
			expected: []lineRange{
				{start: 5, end: 7},
			},
		},
		{
			name:  "non-consecutive lines",
			lines: []int{5, 10, 15},
			expected: []lineRange{
				{start: 5, end: 5},
				{start: 10, end: 10},
				{start: 15, end: 15},
			},
		},
		{
			name:  "mixed consecutive and non-consecutive",
			lines: []int{5, 6, 7, 10, 15, 16},
			expected: []lineRange{
				{start: 5, end: 7},
				{start: 10, end: 10},
				{start: 15, end: 16},
			},
		},
		{
			name:  "all consecutive",
			lines: []int{1, 2, 3, 4, 5},
			expected: []lineRange{
				{start: 1, end: 5},
			},
		},
		{
			name:  "complex pattern",
			lines: []int{1, 3, 4, 5, 8, 9, 12},
			expected: []lineRange{
				{start: 1, end: 1},
				{start: 3, end: 5},
				{start: 8, end: 9},
				{start: 12, end: 12},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupIntoRanges(tt.lines)
			assert.Equal(t, tt.expected, result)
		})
	}
}
