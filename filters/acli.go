package filters

import (
	"fmt"
	"strings"
)

func getAcliFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "jira":
		return getAcliJiraFilter(args[1:])
	default:
		return nil
	}
}

func getAcliJiraFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "workitem":
		return getAcliJiraWorkitemFilter(args[1:])
	default:
		return filterAutoDetect
	}
}

func getAcliJiraWorkitemFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "view":
		return filterAcliJiraWorkitemView
	case "search":
		return filterAcliJiraWorkitemSearch
	default:
		return filterAutoDetect
	}
}

// filterAcliJiraWorkitemView compresses "acli jira workitem view" output.
// Input format: "Key: Value" pairs, with a long multi-paragraph Description.
func filterAcliJiraWorkitemView(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, nil
	}

	lines := strings.Split(trimmed, "\n")
	fields := map[string]string{}
	var currentKey string
	var descLines []string

	for _, line := range lines {
		if idx := strings.Index(line, ": "); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+2:])
			if key == "Description" {
				currentKey = "Description"
				descLines = append(descLines, val)
			} else {
				currentKey = key
				fields[key] = val
			}
		} else if currentKey == "Description" && strings.TrimSpace(line) != "" {
			descLines = append(descLines, strings.TrimSpace(line))
		}
	}

	key := fields["Key"]
	typ := fields["Type"]
	status := fields["Status"]
	summary := fields["Summary"]
	assignee := fields["Assignee"]

	if key == "" && summary == "" {
		return raw, nil
	}

	var out strings.Builder
	fmt.Fprintf(&out, "%s [%s] %s\n", key, typ, status)
	fmt.Fprintf(&out, "Summary: %s\n", summary)
	if assignee != "" {
		fmt.Fprintf(&out, "Assignee: %s\n", assignee)
	}
	if len(descLines) > 0 {
		desc := strings.Join(descLines, " ")
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		fmt.Fprintf(&out, "Description: %s\n", desc)
	}

	result := strings.TrimSpace(out.String())
	return outputSanityCheck(raw, result), nil
}

// filterAcliJiraWorkitemSearch compresses "acli jira workitem search" output.
// Input format: Unicode box-drawing table with wrapped cell content.
func filterAcliJiraWorkitemSearch(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, nil
	}

	lines := strings.Split(trimmed, "\n")

	// Collect lines that are table rows (start with │)
	var tableLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "│") {
			tableLines = append(tableLines, line)
		}
	}

	if len(tableLines) < 2 {
		return raw, nil
	}

	parseCells := func(line string) []string {
		parts := strings.Split(line, "│")
		var cells []string
		// parts[0] is empty (before first │), parts[len-1] is empty (after last │)
		for _, p := range parts[1 : len(parts)-1] {
			cells = append(cells, strings.TrimSpace(p))
		}
		return cells
	}

	headers := parseCells(tableLines[0])
	if len(headers) == 0 {
		return raw, nil
	}

	// Find column indices
	idxOf := func(name string) int {
		for i, h := range headers {
			if strings.EqualFold(h, name) {
				return i
			}
		}
		return -1
	}

	keyIdx := idxOf("Key")
	typeIdx := idxOf("Type")
	statusIdx := idxOf("Status")
	summaryIdx := idxOf("Summary")

	if keyIdx < 0 || summaryIdx < 0 {
		return raw, nil
	}

	// Build logical rows by merging wrapped continuation lines.
	// A continuation row has empty Key column.
	type row []string
	var rows []row

	for _, line := range tableLines[1:] {
		cells := parseCells(line)
		if len(cells) != len(headers) {
			continue
		}
		if len(rows) > 0 && cells[keyIdx] == "" {
			// Continuation: concatenate non-empty cells
			last := rows[len(rows)-1]
			for i, cell := range cells {
				if cell != "" {
					last[i] += cell
				}
			}
			rows[len(rows)-1] = last
		} else {
			r := make(row, len(cells))
			copy(r, cells)
			rows = append(rows, r)
		}
	}

	if len(rows) == 0 {
		return raw, nil
	}

	var out strings.Builder
	for _, r := range rows {
		k := safeIdx(r, keyIdx)
		typ := safeIdx(r, typeIdx)
		status := safeIdx(r, statusIdx)
		summary := safeIdx(r, summaryIdx)
		if k == "" {
			continue
		}
		fmt.Fprintf(&out, "%-10s %-10s %-7s %s\n", k, status, typ, summary)
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return raw, nil
	}
	return outputSanityCheck(raw, result), nil
}

func safeIdx(s []string, i int) string {
	if i < 0 || i >= len(s) {
		return ""
	}
	return s[i]
}
