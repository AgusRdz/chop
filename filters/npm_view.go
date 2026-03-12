package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reNpmViewHeader = regexp.MustCompile(`^(\S+@\S+)\s*\|`)
	reNpmViewDep    = regexp.MustCompile(`^\s{2}(\S+):\s+\S+`)
)

func filterNpmView(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, nil
	}
	if !looksLikeNpmViewOutput(trimmed) {
		return raw, nil
	}

	lines := strings.Split(trimmed, "\n")

	var header, description string
	var deps []string
	var distTags []string
	inDeps := false
	inDistTags := false

	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			inDeps = false
			inDistTags = false
			continue
		}

		// First line: "name@version | LICENSE | deps: N | versions: N"
		if header == "" && reNpmViewHeader.MatchString(t) {
			header = t
			continue
		}

		// Description: second non-empty line after header, before any key:
		if header != "" && description == "" && !strings.Contains(t, ":") && !strings.HasPrefix(t, ".") {
			description = t
			continue
		}

		// Section headers
		if t == "dependencies:" {
			inDeps = true
			inDistTags = false
			continue
		}
		if t == "dist-tags:" {
			inDistTags = true
			inDeps = false
			continue
		}

		// Collect dependencies
		if inDeps && reNpmViewDep.MatchString(line) {
			if m := reNpmViewDep.FindStringSubmatch(line); m != nil {
				deps = append(deps, t)
			}
			continue
		}

		// Collect dist-tags (latest is most useful)
		if inDistTags && strings.Contains(t, ":") {
			distTags = append(distTags, t)
			continue
		}

		// Stop collecting on next section
		if strings.HasSuffix(t, ":") {
			inDeps = false
			inDistTags = false
		}
	}

	if header == "" {
		return raw, nil
	}

	var out strings.Builder
	out.WriteString(header + "\n")
	if description != "" {
		out.WriteString(description + "\n")
	}
	if len(deps) > 0 {
		fmt.Fprintf(&out, "\ndependencies: %s\n", strings.Join(deps, ", "))
	}
	if len(distTags) > 0 {
		out.WriteString("\ndist-tags:\n")
		for _, tag := range distTags {
			fmt.Fprintf(&out, "  %s\n", tag)
		}
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return raw, nil
	}
	return outputSanityCheck(raw, result), nil
}

func looksLikeNpmViewOutput(s string) bool {
	return reNpmViewHeader.MatchString(s) ||
		(strings.Contains(s, "dist-tags:") && strings.Contains(s, "latest:"))
}
