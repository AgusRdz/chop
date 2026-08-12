package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// "> Task :compileJava" or "> Task :app:compileJava"
	reGradleTask = regexp.MustCompile(`^>\s*Task\s+:\S+`)
	// "> Task :compileJava FAILED"
	reGradleTaskFailed = regexp.MustCompile(`^>\s*Task\s+(:\S+)\s+FAILED`)
	// "BUILD SUCCESSFUL in 5s"
	reGradleBuildSuccess = regexp.MustCompile(`(?i)BUILD SUCCESSFUL\s+in\s+(.+)`)
	// "BUILD FAILED"
	reGradleBuildFailed = regexp.MustCompile(`(?i)BUILD FAILED`)
	// "N actionable tasks: M executed, K up-to-date"
	reGradleActionable = regexp.MustCompile(`(\d+)\s+actionable\s+task`)
	// Compilation error: "file.java:12: error: ..."
	reGradleCompileError = regexp.MustCompile(`^(.+\.(?:java|kt|groovy)):(\d+):\s*(error|warning):\s*(.+)`)
	// FAILURE: Build failed with an exception
	reGradleFailureHeader = regexp.MustCompile(`(?i)^FAILURE:\s*`)
	// "* What went wrong:" section
	reGradleWhatWentWrong = regexp.MustCompile(`^\*\s*What went wrong:`)
	// "* Try:" section - skip
	reGradleTrySection = regexp.MustCompile(`^\*\s*Try:`)
	// "* Get more help" - skip
	reGradleGetHelp = regexp.MustCompile(`^\*\s*Get more help`)
	// Configuration cache / daemon info noise
	reGradleNoise = regexp.MustCompile(`(?i)(^Downloading |^Download |configuration cache|daemon|^\s*$|^Starting a Gradle|^To honour the JVM)`)
	// Deprecated Gradle features warning
	reGradleDeprecated = regexp.MustCompile(`(?i)^Deprecated Gradle features`)

	// --- gradle test specific ---
	// Test result line: "X tests completed, Y failed" or "X tests completed, Y failed, Z skipped"
	reGradleTestResult  = regexp.MustCompile(`(\d+)\s+tests?\s+completed`)
	reGradleTestFailed  = regexp.MustCompile(`(\d+)\s+failed`)
	reGradleTestSkipped = regexp.MustCompile(`(\d+)\s+skipped`)
	// Individual test failure: "ClassName > testMethod FAILED"
	reGradleTestFailLine = regexp.MustCompile(`^\s*(\S+)\s*>\s*(\S+.*?)\s+FAILED\s*$`)
	// HTML report line: "HTML test report ..."
	reGradleHTMLReport = regexp.MustCompile(`(?i)HTML\s+test\s+report`)

	// --- gradle dependencies specific ---
	// Direct dep: "+--- group:artifact:version" or "\--- group:artifact:version"
	reGradleDirectDep = regexp.MustCompile(`^[+\\]---\s+(\S+:\S+:\S+)`)
	// Transitive dep (indented with |)
	reGradleTransDep = regexp.MustCompile(`^\|?\s+[+\\]---\s+(\S+:\S+:\S+)`)
	// Config header like "implementation - ..." or "compileClasspath - ..."
	reGradleDepConfig = regexp.MustCompile(`^\S+.*\s+-\s+`)
)

func getGradleFilter(args []string) FilterFunc {
	selected := FilterFunc(filterGradleBuild)
	for _, task := range gradleTasks(args) {
		switch gradleTaskName(task) {
		case "test":
			return filterGradleTest
		case "dependencies":
			selected = filterGradleDeps
		}
	}
	return selected
}

// gradleTasks returns task selectors while excluding CLI options and their
// separate values. Task selectors may be qualified (for example :app:test)
// and may appear after global options or other tasks.
func gradleTasks(args []string) []string {
	var tasks []string
	skipNext := false

	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(arg, "-") {
			skipNext = gradleOptionTakesValue(arg)
			continue
		}
		tasks = append(tasks, arg)
	}

	return tasks
}

func gradleTaskName(selector string) string {
	if i := strings.LastIndex(selector, ":"); i >= 0 {
		return selector[i+1:]
	}
	return selector
}

func gradleOptionTakesValue(arg string) bool {
	if strings.Contains(arg, "=") {
		return false
	}

	switch arg {
	case "-D", "--system-prop",
		"-P", "--project-prop",
		"-g", "--gradle-user-home",
		"-p", "--project-dir",
		"--project-cache-dir",
		"-I", "--init-script",
		"-x", "--exclude-task",
		"--include-build",
		"-M", "--write-verification-metadata",
		"-F", "--dependency-verification",
		"--update-locks",
		"--console", "--warning-mode", "--priority",
		"--max-workers",
		"--configuration-cache-problems",
		"--configuration-cache-entries-per-key",
		"-b", "--build-file",
		"-c", "--settings-file",
		"--configuration", "--tests", "--args":
		return true
	default:
		return false
	}
}

func filterGradleBuild(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeGradleOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	var (
		errors        []string
		warnings      []string
		failedTasks   []string
		errorDetails  []string
		result        string
		elapsed       string
		taskCount     string
		inWhatWrong   bool
		inSkipSection bool
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// BUILD SUCCESSFUL
		if m := reGradleBuildSuccess.FindStringSubmatch(trimmed); m != nil {
			result = "SUCCESS"
			elapsed = strings.TrimSpace(m[1])
			continue
		}

		// BUILD FAILED
		if reGradleBuildFailed.MatchString(trimmed) {
			result = "FAILED"
			continue
		}

		// Actionable tasks count
		if m := reGradleActionable.FindStringSubmatch(trimmed); m != nil {
			taskCount = m[1]
			continue
		}

		// Task FAILED
		if m := reGradleTaskFailed.FindStringSubmatch(trimmed); m != nil {
			failedTasks = append(failedTasks, m[1])
			continue
		}

		// Skip regular task lines
		if reGradleTask.MatchString(trimmed) {
			continue
		}

		// Skip noise
		if reGradleNoise.MatchString(trimmed) || reGradleDeprecated.MatchString(trimmed) {
			continue
		}

		// FAILURE: header
		if reGradleFailureHeader.MatchString(trimmed) {
			continue
		}

		// "* What went wrong:" section
		if reGradleWhatWentWrong.MatchString(trimmed) {
			inWhatWrong = true
			inSkipSection = false
			continue
		}

		// "* Try:" or "* Get more help" - skip sections
		if reGradleTrySection.MatchString(trimmed) || reGradleGetHelp.MatchString(trimmed) {
			inWhatWrong = false
			inSkipSection = true
			continue
		}

		if inSkipSection {
			continue
		}

		// Collect "What went wrong" details
		if inWhatWrong {
			errorDetails = append(errorDetails, trimmed)
			continue
		}

		// Compilation errors
		if m := reGradleCompileError.FindStringSubmatch(trimmed); m != nil {
			loc := fmt.Sprintf("%s:%s", m[1], m[2])
			severity := m[3]
			msg := m[4]
			entry := fmt.Sprintf("%s: %s -> %s", severity, msg, loc)
			if severity == "error" {
				errors = append(errors, entry)
			} else {
				warnings = append(warnings, entry)
			}
			continue
		}
	}

	if result == "" {
		if len(errors) > 0 || len(failedTasks) > 0 {
			result = "FAILED"
		} else {
			result = "SUCCESS"
		}
	}

	var out []string

	if result == "FAILED" {
		out = append(out, "BUILD FAILED")
		if len(errors) > 0 {
			for _, e := range errors {
				out = append(out, "  "+e)
			}
		}
		if len(errorDetails) > 0 {
			for _, d := range errorDetails {
				out = append(out, "  "+d)
			}
		}
		if len(failedTasks) > 0 {
			out = append(out, "")
			out = append(out, fmt.Sprintf("failed tasks: %s", strings.Join(failedTasks, ", ")))
		}
	} else {
		header := "BUILD SUCCESSFUL"
		if elapsed != "" {
			header += fmt.Sprintf(" (%s)", elapsed)
		}
		if taskCount != "" {
			header += fmt.Sprintf(", %s tasks", taskCount)
		}
		out = append(out, header)
		if len(warnings) > 0 {
			for _, w := range warnings {
				out = append(out, "  "+w)
			}
		}
	}

	output := strings.Join(out, "\n")
	return outputSanityCheck(raw, output), nil
}

func filterGradleTest(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeGradleOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	var (
		failures     []string
		failMsgs     []string
		totalTests   int
		failedCount  int
		skippedCount int
		inFailure    bool
		foundResult  bool
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Test result summary
		if m := reGradleTestResult.FindStringSubmatch(trimmed); m != nil {
			totalTests += atoi(m[1])
			foundResult = true
			if m2 := reGradleTestFailed.FindStringSubmatch(trimmed); m2 != nil {
				failedCount += atoi(m2[1])
			}
			if m2 := reGradleTestSkipped.FindStringSubmatch(trimmed); m2 != nil {
				skippedCount += atoi(m2[1])
			}
			inFailure = false
			continue
		}

		// Individual test failure
		if m := reGradleTestFailLine.FindStringSubmatch(trimmed); m != nil {
			inFailure = true
			failures = append(failures, fmt.Sprintf("%s > %s", m[1], m[2]))
			continue
		}

		// Skip task lines, noise, HTML report
		if reGradleTask.MatchString(trimmed) ||
			reGradleNoise.MatchString(trimmed) ||
			reGradleHTMLReport.MatchString(trimmed) ||
			reGradleBuildSuccess.MatchString(trimmed) ||
			reGradleBuildFailed.MatchString(trimmed) ||
			reGradleActionable.MatchString(trimmed) ||
			reGradleDeprecated.MatchString(trimmed) ||
			reGradleFailureHeader.MatchString(trimmed) ||
			reGradleTrySection.MatchString(trimmed) ||
			reGradleGetHelp.MatchString(trimmed) ||
			reGradleWhatWentWrong.MatchString(trimmed) {
			inFailure = false
			continue
		}

		// Collect failure context
		if inFailure && trimmed != "" {
			failMsgs = append(failMsgs, "  "+trimmed)
		}
	}

	if !foundResult {
		return raw, nil
	}

	// All passed
	if failedCount == 0 && totalTests > 0 {
		return fmt.Sprintf("all %d tests passed", totalTests), nil
	}

	// Has failures
	var out []string

	if len(failures) > 0 {
		out = append(out, "Failures:")
		for i, f := range failures {
			out = append(out, "  "+f)
			// Add any collected messages for this failure
			if i < len(failMsgs) {
				// failMsgs are interleaved, just append all at the end
			}
		}
		if len(failMsgs) > 0 {
			for _, m := range failMsgs {
				out = append(out, m)
			}
		}
		out = append(out, "")
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%d tests", totalTests))
	if failedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failedCount))
	}
	if skippedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skippedCount))
	}
	out = append(out, strings.Join(parts, ", "))

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}

func filterGradleDeps(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !looksLikeGradleOutput(trimmed) {
		return raw, nil
	}

	raw = trimmed
	lines := strings.Split(raw, "\n")

	var (
		directDeps      []string
		transitiveCount int
		seen            = make(map[string]bool)
	)

	for _, line := range lines {
		// Direct dep (first level)
		if m := reGradleDirectDep.FindStringSubmatch(line); m != nil {
			dep := m[1]
			// Strip version constraints like " -> 2.0" or " (*)"
			dep = cleanGradleDep(dep)
			if !seen[dep] {
				directDeps = append(directDeps, dep)
				seen[dep] = true
			}
			continue
		}

		// Transitive dep (deeper)
		if m := reGradleTransDep.FindStringSubmatch(line); m != nil {
			transitiveCount++
			continue
		}
	}

	if len(directDeps) == 0 {
		return raw, nil
	}

	var out []string
	for _, dep := range directDeps {
		out = append(out, dep)
	}

	out = append(out, "")
	out = append(out, fmt.Sprintf("%d direct, %d transitive dependencies", len(directDeps), transitiveCount))

	result := strings.Join(out, "\n")
	return outputSanityCheck(raw, result), nil
}

func cleanGradleDep(dep string) string {
	// Remove trailing " -> version" or " (*)" or " (c)" markers
	if idx := strings.Index(dep, " ->"); idx != -1 {
		dep = dep[:idx]
	}
	dep = strings.TrimSuffix(dep, " (*)")
	dep = strings.TrimSuffix(dep, " (c)")
	return strings.TrimSpace(dep)
}
