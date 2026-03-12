package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reAnsi       = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	rePrListLine = regexp.MustCompile(`^#?(\d+)\s+(.+?)\s+(OPEN|CLOSED|MERGED|DRAFT)\s+(.+)$`)
)

func stripAnsi(s string) string {
	return reAnsi.ReplaceAllString(s, "")
}

func getGhPrFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "list":
		return filterGhPrList
	case "view":
		return filterGhPrView
	case "checks":
		return filterGhPrChecks
	default:
		return nil
	}
}

func filterGhPrList(raw string) (string, error) {
	raw = stripAnsi(strings.TrimSpace(raw))
	if raw == "" {
		return "no pull requests", nil
	}
	if !looksLikeGhPrOutput(raw) {
		return raw, nil
	}

	lines := strings.Split(raw, "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// gh pr list outputs tab-separated: NUMBER\tTITLE\tBRANCH\tDATE or similar
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			// Tab-separated format from gh CLI
			num := strings.TrimSpace(parts[0])
			title := strings.TrimSpace(parts[1])
			branch := strings.TrimSpace(parts[2])
			out = append(out, fmt.Sprintf("#%s %s (%s)", num, title, branch))
		} else {
			// Fallback: try regex for table format
			m := rePrListLine.FindStringSubmatch(line)
			if m != nil {
				out = append(out, fmt.Sprintf("#%s %s (%s) %s", m[1], m[2], m[3], m[4]))
			} else {
				out = append(out, line)
			}
		}
	}

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}

func filterGhPrView(raw string) (string, error) {
	raw = stripAnsi(strings.TrimSpace(raw))
	if raw == "" {
		return "no PR data", nil
	}
	if !looksLikeGhPrOutput(raw) {
		return raw, nil
	}

	lines := strings.Split(raw, "\n")
	var (
		title, state, author, branch string
		labels                       []string
		additions, deletions         string
		changedFiles                 string
		reviewStatus                 string
	)

	inBody := false
	bodyFirstLine := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		if strings.HasPrefix(lower, "title:") {
			title = strings.TrimSpace(trimmed[6:])
			inBody = false
		} else if strings.HasPrefix(lower, "state:") {
			state = strings.TrimSpace(trimmed[6:])
			inBody = false
		} else if strings.HasPrefix(lower, "author:") {
			author = strings.TrimSpace(trimmed[7:])
			inBody = false
		} else if strings.HasPrefix(lower, "base:") || strings.HasPrefix(lower, "head:") {
			if strings.HasPrefix(lower, "head:") {
				branch = strings.TrimSpace(trimmed[5:])
			}
			inBody = false
		} else if strings.HasPrefix(lower, "labels:") {
			labelStr := strings.TrimSpace(trimmed[7:])
			if labelStr != "" {
				for _, l := range strings.Split(labelStr, ",") {
					l = strings.TrimSpace(l)
					if l != "" {
						labels = append(labels, l)
					}
				}
			}
			inBody = false
		} else if strings.HasPrefix(lower, "additions:") {
			additions = strings.TrimSpace(trimmed[10:])
			inBody = false
		} else if strings.HasPrefix(lower, "deletions:") {
			deletions = strings.TrimSpace(trimmed[10:])
			inBody = false
		} else if strings.HasPrefix(lower, "changed files:") {
			changedFiles = strings.TrimSpace(trimmed[14:])
			inBody = false
		} else if strings.HasPrefix(lower, "review status:") || strings.HasPrefix(lower, "review decision:") {
			idx := strings.Index(trimmed, ":")
			reviewStatus = strings.TrimSpace(trimmed[idx+1:])
			inBody = false
		} else if strings.HasPrefix(lower, "--") && strings.Contains(lower, "body") {
			inBody = true
		} else if inBody && bodyFirstLine == "" && trimmed != "" {
			bodyFirstLine = trimmed
			if len(bodyFirstLine) > 120 {
				bodyFirstLine = bodyFirstLine[:120] + "..."
			}
			inBody = false
		}
	}

	// Also handle the common gh pr view format where title is the first line
	if title == "" && len(lines) > 0 {
		first := strings.TrimSpace(lines[0])
		if first != "" && !strings.Contains(first, ":") {
			title = first
		}
	}

	var out []string
	if title != "" {
		line := title
		if state != "" {
			line += " [" + state + "]"
		}
		out = append(out, line)
	}
	if author != "" {
		out = append(out, "author: "+author)
	}
	if branch != "" {
		out = append(out, "branch: "+branch)
	}
	if len(labels) > 0 {
		out = append(out, "labels: "+strings.Join(labels, ", "))
	}
	if changedFiles != "" {
		fileLine := changedFiles + " files"
		if additions != "" || deletions != "" {
			fileLine += " (+" + additions + "/-" + deletions + ")"
		}
		out = append(out, fileLine)
	}
	if reviewStatus != "" {
		out = append(out, "review: "+reviewStatus)
	}
	if bodyFirstLine != "" {
		out = append(out, bodyFirstLine)
	}

	if len(out) == 0 {
		return raw, nil
	}
	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}

func filterGhPrChecks(raw string) (string, error) {
	raw = stripAnsi(strings.TrimSpace(raw))
	if raw == "" {
		return "no checks", nil
	}

	lines := strings.Split(raw, "\n")
	var failures []string
	passed, failed, pending := 0, 0, 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)

		if strings.Contains(lower, "pass") || strings.Contains(lower, "success") || strings.Contains(lower, "neutral") || strings.Contains(lower, "skipping") {
			passed++
		} else if strings.Contains(lower, "fail") || strings.Contains(lower, "error") || strings.Contains(lower, "cancelled") || strings.Contains(lower, "timed out") || strings.Contains(lower, "action required") {
			failed++
			failures = append(failures, trimmed)
		} else if strings.Contains(lower, "pending") || strings.Contains(lower, "queued") || strings.Contains(lower, "in_progress") || strings.Contains(lower, "waiting") {
			pending++
		} else {
			// Unknown line - count as passed (often header lines)
			// But check for tab-delimited gh output format
			parts := strings.Split(trimmed, "\t")
			if len(parts) >= 2 {
				status := strings.ToLower(strings.TrimSpace(parts[1]))
				if strings.Contains(status, "pass") || strings.Contains(status, "success") {
					passed++
				} else if strings.Contains(status, "fail") {
					failed++
					failures = append(failures, trimmed)
				} else if strings.Contains(status, "pending") {
					pending++
				} else {
					passed++
				}
			}
		}
	}

	var out []string
	if len(failures) > 0 {
		out = append(out, "FAILING:")
		for _, f := range failures {
			out = append(out, "  "+f)
		}
	}
	summary := fmt.Sprintf("%d passed, %d failed, %d pending", passed, failed, pending)
	out = append(out, summary)

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}
