package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Jest/Vitest patterns
	reSummaryLine  = regexp.MustCompile(`(?i)(Tests?|Test Suites?):\s+.*(\d+)\s+(passed|failed|skipped)`)
	reTestsFailed  = regexp.MustCompile(`(?i)(\d+) failed`)
	reTestsPassed  = regexp.MustCompile(`(?i)(\d+) passed`)
	reTestsSkipped = regexp.MustCompile(`(?i)(\d+) skipped`)
	reTestsTotal   = regexp.MustCompile(`(?i)(\d+) total`)

	// FAIL marker
	reFailBlock = regexp.MustCompile(`(?i)^\s*(FAIL|FAILED|x)\s+`)
	reFailSuite = regexp.MustCompile(`(?i)^FAIL\s+(.+)`)

	// Jest/Vitest summary sections
	reSummarySection = regexp.MustCompile(`(?i)^(Tests?|Test Suites?|Snapshots?|Time):`)
)

func filterNpmTestCmd(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, nil
	}
	if !looksLikeNpmTestOutput(trimmed) {
		return raw, nil
	}

	lines := strings.Split(raw, "\n")

	var failures []string
	var summaryLines []string
	inFailure := false
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	foundSummary := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Collect summary section lines
		if reSummarySection.MatchString(trimmed) {
			summaryLines = append(summaryLines, trimmed)
			foundSummary = true

			// Extract counts from summary
			if m := reTestsFailed.FindStringSubmatch(trimmed); m != nil {
				fmt.Sscanf(m[1], "%d", &totalFailed)
			}
			if m := reTestsPassed.FindStringSubmatch(trimmed); m != nil {
				fmt.Sscanf(m[1], "%d", &totalPassed)
			}
			if m := reTestsSkipped.FindStringSubmatch(trimmed); m != nil {
				fmt.Sscanf(m[1], "%d", &totalSkipped)
			}
			continue
		}

		// Detect start of failure block
		if reFailBlock.MatchString(trimmed) || reFailSuite.MatchString(trimmed) ||
			strings.Contains(trimmed, "FAIL") && (strings.Contains(trimmed, ".test.") || strings.Contains(trimmed, ".spec.")) {
			inFailure = true
			failures = append(failures, line)
			continue
		}

		// Detect error assertion lines (expect/received)
		if strings.Contains(trimmed, "Expected:") || strings.Contains(trimmed, "Received:") ||
			strings.Contains(trimmed, "expect(") || strings.Contains(trimmed, "AssertionError") ||
			strings.Contains(trimmed, "Error:") || strings.Contains(trimmed, "thrown:") {
			inFailure = true
			failures = append(failures, line)
			continue
		}

		// Continue collecting failure context
		if inFailure {
			// Stop on blank line or next test marker, but include a few context lines
			if trimmed == "" {
				failures = append(failures, "")
				inFailure = false
				continue
			}
			// Stop at next PASS marker
			if strings.HasPrefix(trimmed, "PASS") || strings.HasPrefix(trimmed, "Test Suites:") {
				inFailure = false
				// Process this line again
				if reSummarySection.MatchString(trimmed) {
					summaryLines = append(summaryLines, trimmed)
					foundSummary = true
				}
				continue
			}
			failures = append(failures, line)
			continue
		}

	}

	// If no summary found, try to count from raw output
	if !foundSummary {
		// Try simple pass/fail counting for mocha-style output
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "passing") {
				if m := regexp.MustCompile(`(\d+) passing`).FindStringSubmatch(trimmed); m != nil {
					fmt.Sscanf(m[1], "%d", &totalPassed)
				}
			}
			if strings.Contains(trimmed, "failing") {
				if m := regexp.MustCompile(`(\d+) failing`).FindStringSubmatch(trimmed); m != nil {
					fmt.Sscanf(m[1], "%d", &totalFailed)
				}
			}
		}
	}

	// All passed - ultra compact
	total := totalPassed + totalFailed + totalSkipped
	if totalFailed == 0 && total > 0 {
		return fmt.Sprintf("all %d tests passed", total), nil
	}

	// Build output with failures + summary
	var out strings.Builder

	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintln(&out, f)
		}
	}

	// Summary line
	var parts []string
	if totalPassed > 0 {
		parts = append(parts, fmt.Sprintf("%d passed", totalPassed))
	}
	if totalFailed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", totalFailed))
	}
	if totalSkipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", totalSkipped))
	}
	if len(parts) > 0 {
		fmt.Fprintf(&out, "\n%s", strings.Join(parts, ", "))
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return raw, nil
	}
	return outputSanityCheck(raw, result), nil
}
