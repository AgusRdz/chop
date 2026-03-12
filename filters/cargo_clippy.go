package filters

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	// Clippy warning with lint name: "warning: unused variable `x`"
	reClippyWarning = regexp.MustCompile(`^warning:\s*(.+)`)
	// Clippy error
	reClippyError = regexp.MustCompile(`^error(\[E\d+\])?:\s*(.+)`)
	// Location: "  --> src/main.rs:12:5"
	reClippyLocation = regexp.MustCompile(`^\s*-->\s*(.+?):(\d+):\d+`)
	// Lint name in note: "  = note: `#[warn(clippy::some_lint)]` on by default"
	// or: "  = note: `#[warn(unused_variables)]` on by default"
	reClippyLintName = regexp.MustCompile(`#\[(?:warn|deny|allow)\(([^)]+)\)\]`)
	// Help/note lines
	reClippyHelpNote = regexp.MustCompile(`^\s*=\s*(help|note):\s*`)
	// "for further information visit ..."
	reClippyFurtherInfo = regexp.MustCompile(`(?i)for further information`)
	// Warning summary: "warning: `crate` (lib) generated N warnings"
	reClippyWarnSummary = regexp.MustCompile(`^warning:.*generated \d+ warnings?`)
	// "N warnings emitted"
	reClippyEmitted = regexp.MustCompile(`^warning: \d+ warnings? emitted`)
)

type clippyDiag struct {
	severity string // "error" or "warning"
	message  string
	location string // "file.rs:12"
	lint     string // e.g. "clippy::some_lint" or "unused_variables"
}

func filterCargoClippy(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeCargoClippyOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	var (
		diags   []clippyDiag
		current *clippyDiag
	)

	flushCurrent := func() {
		if current != nil {
			diags = append(diags, *current)
			current = nil
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Skip build noise
		if reCargoBuildNoise.MatchString(trimmed) {
			continue
		}

		// Skip warning summaries
		if reClippyWarnSummary.MatchString(trimmed) || reClippyEmitted.MatchString(trimmed) {
			continue
		}

		// Skip "for further information..."
		if reClippyFurtherInfo.MatchString(trimmed) {
			continue
		}

		// Skip "could not compile" and "aborting"
		if reCargoCouldNotCompile.MatchString(trimmed) {
			continue
		}
		if reCargoAborting.MatchString(trimmed) {
			continue
		}

		// Skip "For more information..."
		if reCargoMoreInfo.MatchString(trimmed) {
			continue
		}

		// Error line
		if m := reClippyError.FindStringSubmatch(trimmed); m != nil {
			flushCurrent()
			code := ""
			if m[1] != "" {
				code = strings.Trim(m[1], "[]")
			}
			msg := m[2]
			if code != "" {
				msg = fmt.Sprintf("%s: %s", code, msg)
			}
			current = &clippyDiag{
				severity: "error",
				message:  msg,
			}
			continue
		}

		// Warning line
		if m := reClippyWarning.FindStringSubmatch(trimmed); m != nil {
			flushCurrent()
			current = &clippyDiag{
				severity: "warning",
				message:  m[1],
			}
			continue
		}

		// Location line
		if m := reClippyLocation.FindStringSubmatch(trimmed); m != nil {
			if current != nil {
				current.location = fmt.Sprintf("%s:%s", m[1], m[2])
			}
			continue
		}

		// Help/note lines - extract lint name if present
		if reClippyHelpNote.MatchString(trimmed) {
			if current != nil {
				if m := reClippyLintName.FindStringSubmatch(trimmed); m != nil {
					current.lint = m[1]
				}
			}
			continue
		}
	}

	flushCurrent()

	if len(diags) == 0 {
		// Check if it was all noise
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if reCargoBuildNoise.MatchString(trimmed) {
				continue
			}
			return raw, nil
		}
		return "no warnings", nil
	}

	// Separate errors and warnings
	var errors []clippyDiag
	// Group warnings by lint rule
	ruleMap := make(map[string][]string) // rule -> []location
	ruleMsg := make(map[string]string)   // rule -> message
	ungrouped := 0

	for _, d := range diags {
		if d.severity == "error" {
			errors = append(errors, d)
			continue
		}
		rule := d.lint
		if rule == "" {
			rule = d.message
		}
		loc := d.location
		if loc == "" {
			loc = "(unknown)"
		}
		ruleMap[rule] = append(ruleMap[rule], loc)
		if _, ok := ruleMsg[rule]; !ok {
			ruleMsg[rule] = d.message
		}
		ungrouped++
	}

	var out []string

	// Show errors in full
	if len(errors) > 0 {
		out = append(out, "Errors:")
		for _, e := range errors {
			line := "  error: " + e.message
			if e.location != "" {
				line += " -> " + e.location
			}
			out = append(out, line)
		}
		out = append(out, "")
	}

	// Show warnings grouped by rule
	if len(ruleMap) > 0 {
		out = append(out, "Warnings:")

		// Sort rules for deterministic output
		rules := make([]string, 0, len(ruleMap))
		for r := range ruleMap {
			rules = append(rules, r)
		}
		sort.Strings(rules)

		for _, rule := range rules {
			locs := ruleMap[rule]
			out = append(out, fmt.Sprintf("  %s (%d): %s", rule, len(locs), strings.Join(locs, ", ")))
		}
		out = append(out, "")
	}

	// Summary
	out = append(out, fmt.Sprintf("%d warning(s), %d error(s)", ungrouped, len(errors)))

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}
