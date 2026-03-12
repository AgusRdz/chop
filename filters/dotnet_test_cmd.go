package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// "Passed!" or "Failed!" result line with summary: "Passed!  - Failed:     0, Passed:    28, ..."
	reDotnetTestResultLine = regexp.MustCompile(`(?i)^\s*(Passed!|Failed!)\s*-\s*Failed:\s*(\d+),\s*Passed:\s*(\d+),\s*Skipped:\s*(\d+),\s*Total:\s*(\d+)`)
	// Standalone result line without counts
	reDotnetTestResultOnly = regexp.MustCompile(`(?i)^\s*(Passed!|Failed!)\s*$`)
	// Separate summary lines: "Total tests: 42"
	reDotnetTestTotal = regexp.MustCompile(`(?i)Total tests:\s*(\d+)`)
	// Separate count lines
	reDotnetTestPassedCount  = regexp.MustCompile(`(?i)^\s*Passed:\s*(\d+)`)
	reDotnetTestFailedCount  = regexp.MustCompile(`(?i)^\s*Failed:\s*(\d+)`)
	reDotnetTestSkippedCount = regexp.MustCompile(`(?i)^\s*Skipped:\s*(\d+)`)
	// Failed test name: "  Failed MethodName [time]"
	reDotnetFailedTest = regexp.MustCompile(`^\s*Failed\s+(\S.+?)(?:\s*\[\d+\s*ms\])?\s*$`)
	// Passing test line: "Passed  TestName [time]" - to skip
	reDotnetPassedTest = regexp.MustCompile(`(?i)^\s*Passed\s+\S`)
	// Noise lines to skip
	reDotnetTestNoise = regexp.MustCompile(`(?i)(^Microsoft \(R\)|^Copyright \(C\)|^\s*Starting test execution|^\s*Restore complete|^\s*Determining projects|^\s*Build started|^\s*NuGet |^\s*A total of|^Test run for|^\s*Results File|^\s*Attachments?:|^  Duration:)`)
)

func filterDotnetTestCmd(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeDotnetTestOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	var (
		failures    []string
		inFailure   bool
		passed      int
		failed      int
		skipped     int
		total       int
		foundCounts bool
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Blank lines: end failure block context
		if trimmed == "" {
			if inFailure {
				failures = append(failures, "")
			}
			continue
		}

		// Result line with inline counts: "Passed!  - Failed: 0, Passed: 28, Skipped: 0, Total: 28, ..."
		if m := reDotnetTestResultLine.FindStringSubmatch(trimmed); m != nil {
			failed = atoi(m[2])
			passed = atoi(m[3])
			skipped = atoi(m[4])
			total = atoi(m[5])
			foundCounts = true
			continue
		}

		// Standalone "Passed!" or "Failed!" without counts
		if reDotnetTestResultOnly.MatchString(trimmed) {
			continue
		}

		// Skip noise
		if reDotnetTestNoise.MatchString(trimmed) {
			continue
		}

		// Skip passing test lines
		if reDotnetPassedTest.MatchString(trimmed) {
			if inFailure {
				inFailure = false
			}
			continue
		}

		// Separate summary count lines
		if m := reDotnetTestTotal.FindStringSubmatch(trimmed); m != nil {
			total = atoi(m[1])
			foundCounts = true
			continue
		}
		if m := reDotnetTestFailedCount.FindStringSubmatch(trimmed); m != nil {
			failed = atoi(m[1])
			foundCounts = true
			continue
		}
		if m := reDotnetTestPassedCount.FindStringSubmatch(trimmed); m != nil {
			passed = atoi(m[1])
			foundCounts = true
			continue
		}
		if m := reDotnetTestSkippedCount.FindStringSubmatch(trimmed); m != nil {
			skipped = atoi(m[1])
			foundCounts = true
			continue
		}

		// Failed test marker: "  Failed TestName [42 ms]"
		if reDotnetFailedTest.MatchString(trimmed) {
			inFailure = true
			failures = append(failures, trimmed)
			continue
		}

		// Collect failure context (stack traces, error messages)
		if inFailure {
			failures = append(failures, line)
			continue
		}
	}

	// If we found no counts at all, return raw
	if !foundCounts {
		return raw, nil
	}

	if total == 0 {
		total = passed + failed + skipped
	}

	// All passed - ultra compact
	if failed == 0 && total > 0 {
		return fmt.Sprintf("all %d tests passed", total), nil
	}

	// Build output with failures + summary
	var out strings.Builder

	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintln(&out, f)
		}
		out.WriteString("\n")
	}

	var parts []string
	if passed > 0 {
		parts = append(parts, fmt.Sprintf("%d passed", passed))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if len(parts) > 0 {
		fmt.Fprintf(&out, "%s", strings.Join(parts, ", "))
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return raw, nil
	}
	return outputSanityCheck(raw, result), nil
}
