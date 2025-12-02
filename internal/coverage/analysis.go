package coverage

import (
	"sort"
	"strings"
)

// AnalysisResult contains the results of coverage analysis.
type AnalysisResult struct {
	// UncoveredByFile maps filenames to their uncovered line numbers
	UncoveredByFile map[string][]int
	// TotalAdded is the total number of lines added in the diff
	TotalAdded int
	// TotalUncovered is the total number of uncovered added lines
	TotalUncovered int
}

// AnalyzeCoverage cross-references coverage profiles with diff to find uncovered added lines.
// It takes coverage profiles and a map of added lines by file (from GetAddedLinesByFile).
// Returns an AnalysisResult with uncovered lines grouped by file.
func AnalyzeCoverage(profiles []*Profile, addedLinesByFile map[string][]int) *AnalysisResult {
	result := &AnalysisResult{
		UncoveredByFile: make(map[string][]int),
		TotalAdded:      0,
		TotalUncovered:  0,
	}

	// Build a map of profiles by filename for efficient lookup
	profilesByFile := make(map[string]*Profile)
	for _, profile := range profiles {
		profilesByFile[profile.FileName] = profile
	}

	// For each file in the diff, check coverage for added lines
	for diffFile, addedLines := range addedLinesByFile {
		result.TotalAdded += len(addedLines)

		// Find the matching coverage profile
		profile := findMatchingProfile(profilesByFile, diffFile)

		var uncoveredLines []int
		for _, line := range addedLines {
			if !isLineCovered(profile, line) {
				uncoveredLines = append(uncoveredLines, line)
				result.TotalUncovered++
			}
		}

		// Only add to result if there are uncovered lines
		if len(uncoveredLines) > 0 {
			result.UncoveredByFile[diffFile] = uncoveredLines
		}
	}

	return result
}

// findMatchingProfile finds a coverage profile that matches the given diff filename.
// Coverage files use full module paths (e.g., "github.com/org/repo/internal/server/handler.go")
// while diff files use relative paths (e.g., "internal/server/handler.go").
// Returns the matching profile or nil if not found.
func findMatchingProfile(profilesByFile map[string]*Profile, diffFile string) *Profile {
	// First try exact match (in case coverage uses relative paths too)
	if profile, ok := profilesByFile[diffFile]; ok {
		return profile
	}

	// Try suffix match: find a profile whose filename ends with the diff filename
	for coverageFile, profile := range profilesByFile {
		if strings.HasSuffix(coverageFile, diffFile) {
			return profile
		}
		// Also try with "/" prefix to avoid partial matches
		// e.g., "handler.go" shouldn't match "myhandler.go"
		if strings.HasSuffix(coverageFile, "/"+diffFile) {
			return profile
		}
	}

	return nil
}

// isLineCovered checks if a specific line is covered by the profile.
// A line is covered if it falls within a coverage block with Count > 0.
// Returns false if profile is nil or line is not in any covered block.
func isLineCovered(profile *Profile, line int) bool {
	if profile == nil {
		return false
	}

	// Check each block to see if the line falls within a covered block
	for _, block := range profile.Blocks {
		// A line is covered if:
		// 1. It falls within the block's line range (StartLine to EndLine)
		// 2. The block was executed (Count > 0)
		if line >= block.StartLine && line <= block.EndLine && block.Count > 0 {
			return true
		}
	}

	return false
}

// HasUncoveredLines returns true if there are any uncovered lines in the result.
func (r *AnalysisResult) HasUncoveredLines() bool {
	return r.TotalUncovered > 0
}

// GetSortedFiles returns a sorted list of files with uncovered lines.
// Useful for consistent output ordering.
func (r *AnalysisResult) GetSortedFiles() []string {
	files := make([]string, 0, len(r.UncoveredByFile))
	for file := range r.UncoveredByFile {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}
