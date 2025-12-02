package format

import (
	"fmt"
	"io"
	"sort"

	"github.com/oleg-kozlyuk-grafana/go-canopy/internal/coverage"
)

// GitHubAnnotationsFormatter formats analysis results as GitHub Actions workflow commands.
// Outputs one ::notice annotation per block of consecutive uncovered lines.
type GitHubAnnotationsFormatter struct{}

// lineRange represents a range of consecutive line numbers.
type lineRange struct {
	start int
	end   int
}

// Format formats the analysis result as GitHub Actions annotations.
func (f *GitHubAnnotationsFormatter) Format(result *coverage.AnalysisResult, w io.Writer) error {
	if result == nil {
		return fmt.Errorf("result is nil")
	}

	// Handle case where all lines are covered
	if !result.HasUncoveredLines() {
		if result.TotalAdded == 0 {
			fmt.Fprintln(w, "No lines added in diff")
			return nil
		}
		fmt.Fprintln(w, "All added lines are covered!")
		return nil
	}

	// Print annotations for each file (sorted alphabetically)
	sortedFiles := result.GetSortedFiles()
	for _, file := range sortedFiles {
		lines := result.UncoveredByFile[file]

		// Ensure lines are sorted
		sortedLines := make([]int, len(lines))
		copy(sortedLines, lines)
		sort.Ints(sortedLines)

		// Group consecutive lines into ranges
		ranges := groupIntoRanges(sortedLines)

		// Create one annotation per range
		for _, r := range ranges {
			if r.start == r.end {
				// Single line
				fmt.Fprintf(w, "::notice file=%s,line=%d,title=Uncovered line::Line %d is not covered by tests\n",
					file, r.start, r.start)
			} else {
				// Range of lines
				fmt.Fprintf(w, "::notice file=%s,line=%d,endLine=%d,title=Uncovered lines::Lines %d-%d are not covered by tests\n",
					file, r.start, r.end, r.start, r.end)
			}
		}
	}

	return nil
}

// groupIntoRanges groups consecutive line numbers into ranges.
func groupIntoRanges(lines []int) []lineRange {
	if len(lines) == 0 {
		return nil
	}

	var ranges []lineRange
	rangeStart := lines[0]
	rangeEnd := lines[0]

	for i := 1; i < len(lines); i++ {
		if lines[i] == rangeEnd+1 {
			// Continue the range
			rangeEnd = lines[i]
		} else {
			// End the current range and start a new one
			ranges = append(ranges, lineRange{start: rangeStart, end: rangeEnd})
			rangeStart = lines[i]
			rangeEnd = lines[i]
		}
	}

	// Append the final range
	ranges = append(ranges, lineRange{start: rangeStart, end: rangeEnd})

	return ranges
}
