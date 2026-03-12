package filters

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/AgusRdz/chop/config"
)

// warnf writes a warning to stderr. Used for non-fatal config issues.
var warnf = func(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "chop: warning: "+format+"\n", args...)
}

// BuildUserFilter creates a FilterFunc from a user-defined CustomFilter.
// Returns nil if the filter definition is empty/invalid.
func BuildUserFilter(cf *config.CustomFilter) FilterFunc {
	if cf == nil {
		return nil
	}

	// Exec-based filter takes priority - it's a full pipeline replacement
	if cf.Exec != "" {
		return buildExecFilter(cf.Exec)
	}

	// Declarative rules: keep/drop regex + head/tail truncation
	if len(cf.Keep) == 0 && len(cf.Drop) == 0 && cf.Head == 0 && cf.Tail == 0 {
		return nil
	}

	return buildRuleFilter(cf.Keep, cf.Drop, cf.Head, cf.Tail)
}

// buildRuleFilter creates a FilterFunc from declarative keep/drop/head/tail rules.
func buildRuleFilter(keep, drop []string, head, tail int) FilterFunc {
	// Pre-compile regexes
	keepRe := compilePatterns(keep)
	dropRe := compilePatterns(drop)

	return func(raw string) (string, error) {
		if raw == "" {
			return "", nil
		}

		lines := strings.Split(raw, "\n")

		// Phase 1: Drop matching lines
		if len(dropRe) > 0 {
			var filtered []string
			for _, line := range lines {
				if matchesAny(line, dropRe) {
					continue
				}
				filtered = append(filtered, line)
			}
			lines = filtered
		}

		// Phase 2: Keep only matching lines
		if len(keepRe) > 0 {
			var filtered []string
			for _, line := range lines {
				// Skip empty lines - they add noise when filtering by pattern
				if strings.TrimSpace(line) == "" {
					continue
				}
				if matchesAny(line, keepRe) {
					filtered = append(filtered, line)
				}
			}
			lines = filtered
		}

		total := len(lines)

		// Phase 3: Head/tail truncation
		if head > 0 && tail > 0 && head+tail < total {
			headLines := lines[:head]
			tailLines := lines[total-tail:]
			hidden := total - head - tail
			result := strings.Join(headLines, "\n") +
				fmt.Sprintf("\n... (%d lines hidden)\n", hidden) +
				strings.Join(tailLines, "\n")
			return result, nil
		}

		if head > 0 && head < total {
			result := strings.Join(lines[:head], "\n")
			remaining := total - head
			result += fmt.Sprintf("\n... (%d more lines)", remaining)
			return result, nil
		}

		if tail > 0 && tail < total {
			skipped := total - tail
			result := fmt.Sprintf("... (%d lines skipped)\n", skipped)
			result += strings.Join(lines[total-tail:], "\n")
			return result, nil
		}

		return strings.Join(lines, "\n"), nil
	}
}

// buildExecFilter creates a FilterFunc that pipes output through an external command.
// The execCmd is passed to "sh -c", so it supports system commands (jq, python3, etc.),
// scripts with arguments, and shell expressions.
func buildExecFilter(execCmd string) FilterFunc {
	return func(raw string) (string, error) {
		// Expand ~ to home dir
		expanded := expandHome(execCmd)

		cmd := exec.Command("sh", "-c", expanded)
		cmd.Stdin = strings.NewReader(raw)

		out, err := cmd.Output()
		if err != nil {
			// On script failure, return raw output rather than losing data
			return raw, fmt.Errorf("exec filter failed (%s): %w", expanded, err)
		}

		return string(out), nil
	}
}

// compilePatterns compiles a list of regex pattern strings.
// Invalid patterns are skipped with a warning to stderr.
func compilePatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			warnf("invalid regex pattern %q: %v", p, err)
			continue
		}
		compiled = append(compiled, re)
	}
	return compiled
}

// matchesAny returns true if the line matches any of the compiled patterns.
func matchesAny(line string, patterns []*regexp.Regexp) bool {
	for _, re := range patterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return strings.Replace(path, "~", home, 1)
}
