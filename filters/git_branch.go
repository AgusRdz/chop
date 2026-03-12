package filters

import (
	"fmt"
	"strings"
)

func filterGitBranch(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeGitBranchOutput(trimmed) {
		return raw, nil
	}

	lines := strings.Split(trimmed, "\n")

	var current string
	var local []string
	var remote []string

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "* ") {
			current = strings.TrimPrefix(name, "* ")
		} else if strings.HasPrefix(name, "remotes/") {
			remote = append(remote, name)
		} else {
			local = append(local, name)
		}
	}

	// Short output - show current first + local branches
	totalBranches := len(local) + len(remote)
	if current != "" {
		totalBranches++
	}
	if totalBranches <= 10 {
		var out strings.Builder
		if current != "" {
			out.WriteString("* ")
			out.WriteString(current)
			out.WriteString("\n")
		}
		for _, b := range local {
			out.WriteString("  ")
			out.WriteString(b)
			out.WriteString("\n")
		}
		for _, b := range remote {
			out.WriteString("  ")
			out.WriteString(b)
			out.WriteString("\n")
		}
		// Reordering (current first), not compressing - skip sanity check
		return strings.TrimSpace(out.String()), nil
	}

	// Summarize
	var out strings.Builder
	if current != "" {
		out.WriteString("* ")
		out.WriteString(current)
		out.WriteString("\n")
	}

	// Show local branches (usually fewer)
	for _, b := range local {
		out.WriteString("  ")
		out.WriteString(b)
		out.WriteString("\n")
	}

	// Summarize remote branches
	if len(remote) > 0 {
		out.WriteString(fmt.Sprintf("\n%d remote branches", len(remote)))
		// Group by prefix (e.g. feat/, fix/, release/)
		prefixCounts := make(map[string]int)
		for _, r := range remote {
			// Strip "remotes/origin/"
			name := r
			if idx := strings.Index(name, "/"); idx >= 0 {
				rest := name[idx+1:]
				if idx2 := strings.Index(rest, "/"); idx2 >= 0 {
					rest = rest[idx2+1:]
				}
				if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
					prefixCounts[rest[:slashIdx+1]]++
				} else {
					prefixCounts["other"]++
				}
			}
		}
		if len(prefixCounts) > 0 {
			out.WriteString(": ")
			var parts []string
			for prefix, count := range prefixCounts {
				parts = append(parts, fmt.Sprintf("%s(%d)", prefix, count))
			}
			out.WriteString(strings.Join(parts, ", "))
		}
	}

	result := strings.TrimSpace(out.String())
	return outputSanityCheck(trimmed, result), nil
}
