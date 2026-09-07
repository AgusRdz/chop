package filters

import (
	"fmt"
	"strings"
)

// filterGitShow preserves the commit header (hash, author, date, subject) from
// "git show <sha>" and then delegates the diff body to filterGitDiff.
func filterGitShow(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	diffIdx := strings.Index(trimmed, "\ndiff --git")
	if diffIdx < 0 {
		// No diff section — pass through filterGitDiff which handles stat-only output.
		return filterGitDiff(raw)
	}

	header := trimmed[:diffIdx+1] // commit/author/date/message block
	diffPart := trimmed[diffIdx+1:]

	diffSummary, err := filterGitDiff(diffPart)
	if err != nil {
		return outputSanityCheck(raw, raw), nil
	}

	result := header + "\n" + diffSummary
	return outputSanityCheck(raw, result), nil
}

// contextWindow is how many unchanged lines to keep immediately around a
// run of changed lines within a hunk before eliding the rest.
const gitDiffContextWindow = 3

// hugeHunkLineThreshold is the unchanged-line-run length (within a single
// hunk) above which lines get elided. Below this, hunks pass through
// untouched — most real diffs never hit this since `git diff` already
// limits context to a few lines by default.
const gitDiffHugeHunkRunThreshold = 12

func filterGitDiff(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeGitDiffOutput(trimmed) {
		return filterGitStat(trimmed)
	}

	// No "diff --git" headers — likely --stat or --numstat output
	if !strings.Contains(trimmed, "diff --git") {
		return filterGitStat(trimmed)
	}

	lines := strings.Split(trimmed, "\n")

	// Short diff: pass through as-is
	if len(lines) < 10 {
		return trimmed, nil
	}

	result := elideLongContextRuns(lines)
	return outputSanityCheck(raw, result), nil
}

// elideLongContextRuns keeps every diff header, hunk header, and +/- line
// verbatim, but collapses long runs of unchanged context lines (beyond
// gitDiffContextWindow on each side of a change) into a single elision
// marker so the actual changes stay reviewable while huge hunks shrink.
func elideLongContextRuns(lines []string) string {
	isHeader := func(line string) bool {
		return strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") ||
			strings.HasPrefix(line, "@@")
	}
	isContext := func(line string) bool {
		if isHeader(line) {
			return false
		}
		return line == "" || (line[0] != '+' && line[0] != '-')
	}

	var out []string
	i := 0
	for i < len(lines) {
		line := lines[i]
		if !isContext(line) {
			out = append(out, line)
			i++
			continue
		}

		// Gather the run of consecutive unchanged context lines.
		start := i
		for i < len(lines) && isContext(lines[i]) {
			i++
		}
		run := lines[start:i]

		if len(run) <= gitDiffHugeHunkRunThreshold {
			out = append(out, run...)
			continue
		}

		out = append(out, run[:gitDiffContextWindow]...)
		elided := len(run) - 2*gitDiffContextWindow
		out = append(out, fmt.Sprintf("@@ ... %d unchanged lines elided ... @@", elided))
		out = append(out, run[len(run)-gitDiffContextWindow:]...)
	}

	return strings.Join(out, "\n")
}
