package filters

import (
	"strings"
)

func filterGitPush(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	lines := strings.Split(trimmed, "\n")
	var out []string

	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}

		// Always keep
		if t == "Everything up-to-date" {
			out = append(out, t)
			continue
		}
		if strings.HasPrefix(t, "To ") {
			out = append(out, t)
			continue
		}
		// Ref update lines: "   abc..def  main -> main" or " * [new branch]" or " + [forced]"
		if strings.Contains(t, "->") || strings.HasPrefix(t, "* [new") || strings.HasPrefix(t, "+ [forced") || strings.HasPrefix(t, "- [deleted") {
			out = append(out, t)
			continue
		}
		// "Branch 'x' set up to track..."
		if strings.HasPrefix(t, "Branch '") {
			out = append(out, t)
			continue
		}
		// Error/hint lines
		if strings.HasPrefix(t, "error:") || strings.HasPrefix(t, "fatal:") || strings.HasPrefix(t, "hint:") {
			out = append(out, t)
			continue
		}
		// remote: lines — keep if they contain meaningful content (URL, PR, non-empty)
		if strings.HasPrefix(t, "remote:") {
			content := strings.TrimSpace(strings.TrimPrefix(t, "remote:"))
			if content == "" || strings.HasPrefix(content, "Resolving deltas") {
				continue
			}
			out = append(out, t)
			continue
		}
		// Skip progress noise: Enumerating, Counting, Compressing, Writing, Total, Delta compression
		// (anything else we haven't matched is likely progress — skip it)
	}

	if len(out) == 0 {
		return raw, nil
	}

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}
