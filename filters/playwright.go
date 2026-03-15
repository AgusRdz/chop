package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	rePlaywrightPassLine    = regexp.MustCompile(`^\s+[✓✔]\s+\d+\s`)
	rePlaywrightFailLine    = regexp.MustCompile(`^\s+[✗✘]\s+\d+\s`)
	rePlaywrightBlockStart  = regexp.MustCompile(`^\s+\d+\)\s+\S`)
	rePlaywrightSummary     = regexp.MustCompile(`(\d+) tests? \((\d+) failed\)|(\d+) passed \(|(\d+) tests? passed`)
	rePlaywrightTestsFailed = regexp.MustCompile(`(\d+) tests? \((\d+) failed\)`)
	rePlaywrightPassed      = regexp.MustCompile(`(\d+) passed`)
)

func filterPlaywright(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, nil
	}
	if !looksLikePlaywrightOutput(trimmed) {
		return raw, nil
	}

	lines := strings.Split(trimmed, "\n")

	var failures []string
	inFailure := false
	passed, failed := 0, 0

	for _, line := range lines {
		// Pass: ✓ or ✔
		if rePlaywrightPassLine.MatchString(line) {
			passed++
			inFailure = false
			continue
		}

		// Fail: ✗ or ✘ (× is a retry attempt - skip, don't count)
		if rePlaywrightFailLine.MatchString(line) {
			failed++
			inFailure = false
			continue
		}

		// Numbered failure detail block: "  1) [browser] › ..."
		if rePlaywrightBlockStart.MatchString(line) {
			inFailure = true
			failures = append(failures, line)
			continue
		}

		if inFailure {
			if rePlaywrightSummary.MatchString(strings.TrimSpace(line)) {
				inFailure = false
				continue
			}
			failures = append(failures, line)
			continue
		}
	}

	total := passed + failed

	// Fallback: parse summary line for dot/line reporters
	if total == 0 {
		for _, line := range lines {
			t := strings.TrimSpace(line)
			if m := rePlaywrightTestsFailed.FindStringSubmatch(t); m != nil {
				fmt.Sscanf(m[1], "%d", &total)
				fmt.Sscanf(m[2], "%d", &failed)
				passed = total - failed
				break
			}
			if m := rePlaywrightPassed.FindStringSubmatch(t); m != nil {
				fmt.Sscanf(m[1], "%d", &passed)
				total = passed
			}
		}
	}

	if total == 0 {
		return raw, nil
	}

	if failed == 0 {
		return fmt.Sprintf("all %d tests passed", total), nil
	}

	var out strings.Builder
	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintln(&out, f)
		}
		fmt.Fprintln(&out)
	}
	fmt.Fprintf(&out, "%d passed, %d failed", passed, failed)

	result := strings.TrimSpace(out.String())
	if result == "" {
		return raw, nil
	}
	return outputSanityCheck(raw, result), nil
}

var rePlaywrightTimeSummary = regexp.MustCompile(`\d+ tests? \(\d+ failed\) - \d`)

func looksLikePlaywrightOutput(s string) bool {
	return (strings.Contains(s, "Running") && strings.Contains(s, "workers")) ||
		strings.Contains(s, "✓") ||
		strings.Contains(s, "✗") ||
		(strings.Contains(s, "passed (") && strings.Contains(s, "s)")) ||
		rePlaywrightTimeSummary.MatchString(s)
}
