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

	type fileStat struct {
		name    string
		added   int
		removed int
	}
	var stats []fileStat
	var current *fileStat

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			if current != nil {
				stats = append(stats, *current)
			}
			// Extract filename from "diff --git a/foo b/foo"
			parts := strings.SplitN(line, " b/", 2)
			name := line
			if len(parts) == 2 {
				name = parts[1]
			}
			current = &fileStat{name: name}
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			current.added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			current.removed++
		}
	}
	if current != nil {
		stats = append(stats, *current)
	}

	if len(stats) == 0 {
		return filterGitStat(trimmed)
	}

	var out strings.Builder
	totalAdded := 0
	totalRemoved := 0
	for _, s := range stats {
		fmt.Fprintf(&out, "%s: +%d -%d\n", s.name, s.added, s.removed)
		totalAdded += s.added
		totalRemoved += s.removed
	}
	fmt.Fprintf(&out, "%d files changed, +%d -%d", len(stats), totalAdded, totalRemoved)

	result := out.String()
	return outputSanityCheck(raw, result), nil
}
