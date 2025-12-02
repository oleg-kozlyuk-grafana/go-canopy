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

// findMatchingDiffFile finds the diff file that matches a coverage profile.
// Returns the diff filename, the added lines, and true if found; empty values and false otherwise.
func findMatchingDiffFile(profile *Profile, addedLinesByFile map[string][]int) (string, []int, bool) {
	profileFile := profile.FileName

	// First try exact match
	if addedLines, ok := addedLinesByFile[profileFile]; ok {
		return profileFile, addedLines, true
	}

	// Try to find a diff file whose path is a suffix of the profile filename
	// Coverage uses full module paths, diff uses relative paths
	for diffFile, addedLines := range addedLinesByFile {
		if strings.HasSuffix(profileFile, diffFile) {
			return diffFile, addedLines, true
		}
		if strings.HasSuffix(profileFile, "/"+diffFile) {
			return diffFile, addedLines, true
		}
	}

	return "", nil, false
}

// AnalyzeCoverage cross-references coverage profiles with diff to find uncovered added lines.
// Coverage profiles are the primary source - we extract uncovered lines from them,
// then filter by the diff to only report lines that were added.
// It takes coverage profiles and a map of added lines by file (from GetAddedLinesByFile).
// Returns an AnalysisResult with uncovered lines grouped by file.
func AnalyzeCoverage(profiles []*Profile, addedLinesByFile map[string][]int) *AnalysisResult {
	result := &AnalysisResult{
		UncoveredByFile: make(map[string][]int),
		TotalAdded:      0,
		TotalUncovered:  0,
	}

	// Track which diff files have been processed via coverage
	processedDiffFiles := make(map[string]bool)

	// Step 1: Process files that have coverage data
	for _, profile := range profiles {
		// Find the matching diff file
		diffFile, addedLines, found := findMatchingDiffFile(profile, addedLinesByFile)
		if !found {
			// File has coverage but is not in the diff - skip it
			continue
		}

		// Mark this diff file as processed
		processedDiffFiles[diffFile] = true

		// Count total added lines for this file
		result.TotalAdded += len(addedLines)

		// Check each added line to see if it's covered
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

	// Step 2: Handle diff files with NO coverage data (flag all added lines as uncovered)
	for diffFile, addedLines := range addedLinesByFile {
		if !processedDiffFiles[diffFile] {
			// This file is in the diff but has no coverage data
			result.TotalAdded += len(addedLines)
			result.TotalUncovered += len(addedLines)
			result.UncoveredByFile[diffFile] = addedLines
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
