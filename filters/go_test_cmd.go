package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// "=== RUN   TestName"
	reGoTestRun = regexp.MustCompile(`^=== RUN\s+(\S+)`)
	// "--- PASS: TestName (0.00s)"
	reGoTestPass = regexp.MustCompile(`^--- PASS:\s+(\S+)\s+\(([^)]+)\)`)
	// "--- FAIL: TestName (0.00s)"
	reGoTestFail = regexp.MustCompile(`^--- FAIL:\s+(\S+)\s+\(([^)]+)\)`)
	// "--- SKIP: TestName (0.00s)"
	reGoTestSkip = regexp.MustCompile(`^--- SKIP:\s+(\S+)`)
	// "ok  	package	0.123s"
	reGoTestOkPkg = regexp.MustCompile(`^ok\s+\S+\s+[\d.]+s`)
	// "FAIL	package	0.123s"
	reGoTestFailPkg = regexp.MustCompile(`^FAIL\s+\S+\s+[\d.]+s`)
	// "PASS"
	reGoTestPASS = regexp.MustCompile(`^PASS$`)
	// "FAIL"
	reGoTestFAIL = regexp.MustCompile(`^FAIL$`)
	// "=== PAUSE" / "=== CONT"
	reGoTestPauseCont = regexp.MustCompile(`^=== (PAUSE|CONT)\s+`)
	// "?   	package	[no test files]"
	reGoTestNoFiles = regexp.MustCompile(`^\?\s+\S+\s+\[no test files\]`)
)

func filterGoTestCmd(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeGoTestOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	var (
		passed    int
		failed    int
		skipped   int
		totalTime string
		// Track current test output: between "=== RUN" / "--- FAIL" and the next marker.
		currentTestName   string
		currentTestOutput []string
		// When true, currentTestOutput holds post-FAIL detail lines (non-verbose mode).
		collectingFailDetails bool
		// Collected failure blocks
		failures []string
	)

	flushTest := func(isFail bool) {
		if (isFail || collectingFailDetails) && len(currentTestOutput) > 0 {
			for _, line := range currentTestOutput {
				failures = append(failures, "  "+line)
			}
		}
		currentTestName = ""
		currentTestOutput = nil
		collectingFailDetails = false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// "=== PAUSE" / "=== CONT" - skip
		if reGoTestPauseCont.MatchString(trimmed) {
			continue
		}

		// "=== RUN TestName" - start tracking a new test
		if m := reGoTestRun.FindStringSubmatch(trimmed); m != nil {
			flushTest(false) // previous test wasn't FAIL if we hit a new RUN
			currentTestName = m[1]
			currentTestOutput = nil
			continue
		}

		// "--- PASS" - count and discard output (also flushes any pending fail details)
		if reGoTestPass.MatchString(trimmed) {
			passed++
			flushTest(collectingFailDetails)
			continue
		}

		// "--- SKIP"
		if reGoTestSkip.MatchString(trimmed) {
			skipped++
			flushTest(collectingFailDetails)
			continue
		}

		// "--- FAIL" - flush verbose pre-marker output, then capture post-marker details
		if m := reGoTestFail.FindStringSubmatch(trimmed); m != nil {
			failed++
			flushTest(true) // flush any pre-marker lines (verbose mode)
			failures = append(failures, fmt.Sprintf("FAIL: %s (%s)", m[1], m[2]))
			// Enter detail-capture mode for non-verbose output (details follow the marker)
			currentTestName = m[1]
			collectingFailDetails = true
			continue
		}

		// "ok  package  0.123s"
		if reGoTestOkPkg.MatchString(trimmed) {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				totalTime = parts[len(parts)-1]
			}
			flushTest(true) // flush any pending non-verbose fail details
			continue
		}

		// "FAIL  package  0.123s"
		if reGoTestFailPkg.MatchString(trimmed) {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				totalTime = parts[len(parts)-1]
			}
			flushTest(true) // flush any pending non-verbose fail details
			continue
		}

		// Bare "PASS" / "FAIL"
		if reGoTestPASS.MatchString(trimmed) || reGoTestFAIL.MatchString(trimmed) {
			flushTest(collectingFailDetails)
			continue
		}

		// "? package [no test files]"
		if reGoTestNoFiles.MatchString(trimmed) {
			continue
		}

		// Accumulate test output lines
		if currentTestName != "" {
			currentTestOutput = append(currentTestOutput, trimmed)
			continue
		}
	}
	// Flush any trailing non-verbose fail details not terminated by a package line.
	flushTest(collectingFailDetails)

	total := passed + failed + skipped

	// Fallback if we couldn't parse anything
	if total == 0 {
		return raw, nil
	}

	// All passed
	if failed == 0 {
		summary := fmt.Sprintf("all %d tests passed", total)
		if totalTime != "" {
			summary += fmt.Sprintf(" (%s)", totalTime)
		}
		return summary, nil
	}

	// Has failures
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
	fmt.Fprint(&out, strings.Join(parts, ", "))

	result := strings.TrimSpace(out.String())
	return outputSanityCheck(raw, result), nil
}
