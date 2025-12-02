package coverage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeCoverage(t *testing.T) {
	tests := []struct {
		name                   string
		profiles               []*Profile
		addedLinesByFile       map[string][]int
		expectedUncoveredByFile map[string][]int
		expectedAdded          int
		expectedTotalUncovered int
	}{
		{
			name: "all lines covered",
			profiles: []*Profile{
				{
					FileName: "github.com/org/repo/main.go",
					Mode:     "set",
					Blocks: []ProfileBlock{
						{StartLine: 1, EndLine: 5, Count: 1},
						{StartLine: 6, EndLine: 10, Count: 1},
					},
				},
			},
			addedLinesByFile: map[string][]int{
				"main.go": {3, 7},
			},
			expectedUncoveredByFile: map[string][]int{},
			expectedAdded:      2,
			expectedTotalUncovered:  0,
		},
		{
			name: "some lines uncovered",
			profiles: []*Profile{
				{
					FileName: "github.com/org/repo/main.go",
					Mode:     "set",
					Blocks: []ProfileBlock{
						{StartLine: 1, EndLine: 5, Count: 1},  // covered
						{StartLine: 6, EndLine: 10, Count: 0}, // not covered
					},
				},
			},
			addedLinesByFile: map[string][]int{
				"main.go": {3, 7, 9},
			},
			expectedUncoveredByFile: map[string][]int{
				"main.go": {7, 9},
			},
			expectedAdded:     3,
			expectedTotalUncovered: 2,
		},
		{
			name: "no coverage for file",
			profiles: []*Profile{
				{
					FileName: "github.com/org/repo/other.go",
					Mode:     "set",
					Blocks: []ProfileBlock{
						{StartLine: 1, EndLine: 10, Count: 1},
					},
				},
			},
			addedLinesByFile: map[string][]int{
				"main.go": {1, 2, 3},
			},
			expectedUncoveredByFile: map[string][]int{},
			expectedAdded:           0,
			expectedTotalUncovered:  0,
		},
		{
			name: "multiple files mixed coverage",
			profiles: []*Profile{
				{
					FileName: "github.com/org/repo/server.go",
					Mode:     "set",
					Blocks: []ProfileBlock{
						{StartLine: 10, EndLine: 20, Count: 1},
						{StartLine: 30, EndLine: 40, Count: 0},
					},
				},
				{
					FileName: "github.com/org/repo/handler.go",
					Mode:     "set",
					Blocks: []ProfileBlock{
						{StartLine: 5, EndLine: 15, Count: 1},
					},
				},
			},
			addedLinesByFile: map[string][]int{
				"server.go":  {15, 35},
				"handler.go": {10, 20}, // line 20 not in any block, will be ignored
			},
			expectedUncoveredByFile: map[string][]int{
				"server.go": {35}, // only server.go line 35 is uncovered
			},
			expectedAdded:          4,
			expectedTotalUncovered: 1, // only 1 uncovered (line 20 is ignored as non-instrumented)
		},
		{
			name:             "empty diff",
			profiles:         []*Profile{},
			addedLinesByFile: map[string][]int{},
			expectedUncoveredByFile: map[string][]int{},
			expectedAdded:0,
			expectedTotalUncovered:0,
		},
		{
			name: "file with coverage not in diff",
			profiles: []*Profile{
				{
					FileName: "github.com/org/repo/other.go",
					Mode:     "set",
					Blocks: []ProfileBlock{
						{StartLine: 1, EndLine: 5, Count: 0},
						{StartLine: 6, EndLine: 10, Count: 1},
					},
				},
			},
			addedLinesByFile: map[string][]int{
				"main.go": {1, 2, 3},
			},
			expectedUncoveredByFile: map[string][]int{},
			expectedAdded:           0,
			expectedTotalUncovered:  0,
		},
		{
			name:     "no coverage data",
			profiles: []*Profile{},
			addedLinesByFile: map[string][]int{
				"main.go": {1, 2, 3},
			},
			expectedUncoveredByFile: map[string][]int{},
			expectedAdded:           0,
			expectedTotalUncovered:  0,
		},
		{
			name: "nested file paths",
			profiles: []*Profile{
				{
					FileName: "github.com/org/repo/internal/server/handler.go",
					Mode:     "set",
					Blocks: []ProfileBlock{
						{StartLine: 1, EndLine: 10, Count: 1},
						{StartLine: 11, EndLine: 20, Count: 0},
					},
				},
			},
			addedLinesByFile: map[string][]int{
				"internal/server/handler.go": {5, 15},
			},
			expectedUncoveredByFile: map[string][]int{
				"internal/server/handler.go": {15},
			},
			expectedAdded:2,
			expectedTotalUncovered:1,
		},
		{
			name: "non-instrumented lines are ignored",
			profiles: []*Profile{
				{
					FileName: "github.com/org/repo/main.go",
					Mode:     "set",
					Blocks: []ProfileBlock{
						{StartLine: 5, EndLine: 7, Count: 1},   // covered: lines 5-7
						{StartLine: 10, EndLine: 12, Count: 0}, // uncovered: lines 10-12
					},
				},
			},
			addedLinesByFile: map[string][]int{
				"main.go": {1, 3, 6, 8, 11, 15}, // lines 1,3,8,15 are not in any block
			},
			expectedUncoveredByFile: map[string][]int{
				"main.go": {11}, // only line 11 is uncovered (in block with Count=0)
			},
			expectedAdded:          6,
			expectedTotalUncovered: 1, // only 1 uncovered line (11), others are ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnalyzeCoverage(tt.profiles, tt.addedLinesByFile)

			assert.Equal(t, tt.expectedUncoveredByFile, result.UncoveredByFile)
			assert.Equal(t, tt.expectedAdded, result.TotalAdded)
			assert.Equal(t, tt.expectedTotalUncovered, result.TotalUncovered)
		})
	}
}

func TestFindMatchingProfile(t *testing.T) {
	profiles := []*Profile{
		{FileName: "github.com/org/repo/main.go"},
		{FileName: "github.com/org/repo/internal/server/handler.go"},
		{FileName: "github.com/org/repo/internal/coverage/parser.go"},
		{FileName: "simple.go"}, // relative path
	}

	profilesByFile := make(map[string]*Profile)
	for _, p := range profiles {
		profilesByFile[p.FileName] = p
	}

	tests := []struct {
		name          string
		diffFile      string
		expectedFound bool
		expectedFile  string
	}{
		{
			name:          "exact match",
			diffFile:      "simple.go",
			expectedFound: true,
			expectedFile:  "simple.go",
		},
		{
			name:          "suffix match - simple",
			diffFile:      "main.go",
			expectedFound: true,
			expectedFile:  "github.com/org/repo/main.go",
		},
		{
			name:          "suffix match - nested path",
			diffFile:      "internal/server/handler.go",
			expectedFound: true,
			expectedFile:  "github.com/org/repo/internal/server/handler.go",
		},
		{
			name:          "suffix match - deeper nested",
			diffFile:      "coverage/parser.go",
			expectedFound: true,
			expectedFile:  "github.com/org/repo/internal/coverage/parser.go",
		},
		{
			name:          "no match",
			diffFile:      "notfound.go",
			expectedFound: false,
		},
		{
			name:          "partial filename no match",
			diffFile:      "handler.go",
			expectedFound: true,
			expectedFile:  "github.com/org/repo/internal/server/handler.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := findMatchingProfile(profilesByFile, tt.diffFile)

			if tt.expectedFound {
				require.NotNil(t, profile, "expected to find a matching profile")
				assert.Equal(t, tt.expectedFile, profile.FileName)
			} else {
				assert.Nil(t, profile, "expected no matching profile")
			}
		})
	}
}

func TestIsLineInstrumented(t *testing.T) {
	profile := &Profile{
		FileName: "test.go",
		Mode:     "set",
		Blocks: []ProfileBlock{
			{StartLine: 5, EndLine: 10, Count: 1},  // covered
			{StartLine: 15, EndLine: 20, Count: 0}, // not covered
			{StartLine: 25, EndLine: 30, Count: 5}, // covered (count mode)
		},
	}

	tests := []struct {
		name     string
		profile  *Profile
		line     int
		expected bool
	}{
		{
			name:     "line in covered block",
			profile:  profile,
			line:     7,
			expected: true,
		},
		{
			name:     "line in uncovered block",
			profile:  profile,
			line:     17,
			expected: true,
		},
		{
			name:     "line not in any block",
			profile:  profile,
			line:     12,
			expected: false,
		},
		{
			name:     "line before all blocks",
			profile:  profile,
			line:     1,
			expected: false,
		},
		{
			name:     "line after all blocks",
			profile:  profile,
			line:     50,
			expected: false,
		},
		{
			name:     "line at block start",
			profile:  profile,
			line:     5,
			expected: true,
		},
		{
			name:     "line at block end",
			profile:  profile,
			line:     10,
			expected: true,
		},
		{
			name:     "nil profile",
			profile:  nil,
			line:     5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLineInstrumented(tt.profile, tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsLineCovered(t *testing.T) {
	profile := &Profile{
		FileName: "test.go",
		Mode:     "set",
		Blocks: []ProfileBlock{
			{StartLine: 5, EndLine: 10, Count: 1},  // covered
			{StartLine: 15, EndLine: 20, Count: 0}, // not covered
			{StartLine: 25, EndLine: 30, Count: 5}, // covered (count mode)
		},
	}

	tests := []struct {
		name     string
		profile  *Profile
		line     int
		expected bool
	}{
		{
			name:     "line in covered block - start",
			profile:  profile,
			line:     5,
			expected: true,
		},
		{
			name:     "line in covered block - middle",
			profile:  profile,
			line:     7,
			expected: true,
		},
		{
			name:     "line in covered block - end",
			profile:  profile,
			line:     10,
			expected: true,
		},
		{
			name:     "line in uncovered block",
			profile:  profile,
			line:     17,
			expected: false,
		},
		{
			name:     "line not in any block",
			profile:  profile,
			line:     12,
			expected: false,
		},
		{
			name:     "line before all blocks",
			profile:  profile,
			line:     1,
			expected: false,
		},
		{
			name:     "line after all blocks",
			profile:  profile,
			line:     50,
			expected: false,
		},
		{
			name:     "line in covered block with high count",
			profile:  profile,
			line:     27,
			expected: true,
		},
		{
			name:     "nil profile",
			profile:  nil,
			line:     5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLineCovered(tt.profile, tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalysisResult_HasUncoveredLines(t *testing.T) {
	tests := []struct {
		name     string
		result   *AnalysisResult
		expected bool
	}{
		{
			name: "has uncovered lines",
			result: &AnalysisResult{
				TotalUncovered: 5,
			},
			expected: true,
		},
		{
			name: "no uncovered lines",
			result: &AnalysisResult{
				TotalUncovered: 0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.HasUncoveredLines())
		})
	}
}

func TestAnalysisResult_GetSortedFiles(t *testing.T) {
	result := &AnalysisResult{
		UncoveredByFile: map[string][]int{
			"zebra.go":   {1, 2},
			"alpha.go":   {3, 4},
			"charlie.go": {5, 6},
		},
	}

	files := result.GetSortedFiles()
	expected := []string{"alpha.go", "charlie.go", "zebra.go"}

	assert.Equal(t, expected, files)
}

func TestAnalyzeCoverage_EdgeCases(t *testing.T) {
	t.Run("line at block boundary", func(t *testing.T) {
		profiles := []*Profile{
			{
				FileName: "test.go",
				Blocks: []ProfileBlock{
					{StartLine: 1, EndLine: 5, Count: 1},
					{StartLine: 6, EndLine: 10, Count: 0},
				},
			},
		}

		addedLines := map[string][]int{
			"test.go": {5, 6}, // boundary lines
		}

		result := AnalyzeCoverage(profiles, addedLines)

		// Line 5 should be covered (end of first block)
		// Line 6 should be uncovered (start of second block with Count=0)
		assert.Equal(t, map[string][]int{"test.go": {6}}, result.UncoveredByFile)
		assert.Equal(t, 2, result.TotalAdded)
		assert.Equal(t, 1, result.TotalUncovered)
	})

	t.Run("single line block", func(t *testing.T) {
		profiles := []*Profile{
			{
				FileName: "test.go",
				Blocks: []ProfileBlock{
					{StartLine: 5, EndLine: 5, Count: 1},
				},
			},
		}

		addedLines := map[string][]int{
			"test.go": {5},
		}

		result := AnalyzeCoverage(profiles, addedLines)

		assert.Equal(t, map[string][]int{}, result.UncoveredByFile)
		assert.Equal(t, 1, result.TotalAdded)
		assert.Equal(t, 0, result.TotalUncovered)
	})
}
