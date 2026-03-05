package filters

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// [INFO] --- maven-compiler-plugin:3.11.0:compile (default-compile) @ myapp ---
	reMvnSeparator = regexp.MustCompile(`^\[INFO\]\s*---.*---\s*$`)
	// [INFO] Downloading/Downloaded artifact lines
	reMvnDownload = regexp.MustCompile(`(?i)^\[INFO\]\s*(Downloading|Downloaded)\s+`)
	// [INFO] Scanning for projects...
	reMvnScanning = regexp.MustCompile(`(?i)^\[INFO\]\s*Scanning for projects`)
	// [INFO] BUILD SUCCESS / BUILD FAILURE
	reMvnBuildResult = regexp.MustCompile(`^\[INFO\]\s*BUILD\s+(SUCCESS|FAILURE)`)
	// [INFO] Total time: 5.123 s
	reMvnTotalTime = regexp.MustCompile(`(?i)^\[INFO\]\s*Total time:\s*(.+)`)
	// [INFO] ========= line separators
	reMvnBannerSeparator = regexp.MustCompile(`^\[INFO\]\s*[-=]{5,}\s*$`)
	// [WARNING] ... lines
	reMvnWarning = regexp.MustCompile(`^\[WARNING\]\s*(.+)`)
	// [ERROR] ... lines
	reMvnError = regexp.MustCompile(`^\[ERROR\]\s*(.+)`)
	// Reactor summary noise for single-module: "[INFO] Reactor Summary:"
	reMvnReactorSummary = regexp.MustCompile(`(?i)^\[INFO\]\s*Reactor Summary`)
	// Reactor module line: "[INFO] myapp 1.0 ........... SUCCESS [5s]"
	reMvnReactorModule = regexp.MustCompile(`^\[INFO\]\s*\S+.*\.\.\.*\s*(SUCCESS|FAILURE)`)
	// [INFO] blank or empty info lines
	reMvnInfoEmpty = regexp.MustCompile(`^\[INFO\]\s*$`)
	// Generic [INFO] prefix
	reMvnInfoLine = regexp.MustCompile(`^\[INFO\]\s*(.*)`)

	// --- mvn test specific ---
	// "Tests run: 42, Failures: 0, Errors: 0, Skipped: 2" (summary line, final one without " -- in ClassName")
	reMvnTestSummary = regexp.MustCompile(`Tests run:\s*(\d+),\s*Failures:\s*(\d+),\s*Errors:\s*(\d+),\s*Skipped:\s*(\d+)`)
	// Per-class summary has " -- in com.example.AppTest" at the end
	reMvnTestPerClass = regexp.MustCompile(`--\s+in\s+\S+`)
	// Failed test name from surefire: "  testMethodName(com.example.MyTest)  Time elapsed: 0.5 s  <<< FAILURE!"
	// Also matches [ERROR] prefixed lines
	reMvnFailedTest = regexp.MustCompile(`^(?:\[ERROR\]\s*)?(\S+.*?)\s+Time elapsed:.*<<<\s*(FAILURE|ERROR)`)
	// T E S T S header
	reMvnTestsHeader = regexp.MustCompile(`(?i)^\s*-*\s*T\s*E\s*S\s*T\s*S\s*-*\s*$`)
	// "Running com.example.MyTest"
	reMvnRunningTest = regexp.MustCompile(`(?i)^\s*Running\s+`)
	// Surefire report noise / results section markers
	reMvnSurefireNoise = regexp.MustCompile(`(?i)(Results\s*:|^\s*Tests run:.*Time elapsed)`)
	// "[ERROR] Failures:" or "[INFO] Results:" section headers
	reMvnResultsSection = regexp.MustCompile(`^(?:\[ERROR\]|\[INFO\])\s*(Failures|Errors|Results)\s*:\s*$`)

	// --- mvn dependency:tree specific ---
	// Direct dependency line (first-level, prefixed with exactly "+- " or "\- " right after [INFO] )
	reMvnDirectDep = regexp.MustCompile(`^\[INFO\] [+\\]\-\s+(\S+)`)
	// Transitive dep (any line with dependency artifact deeper than first level)
	reMvnTransitiveDep = regexp.MustCompile(`^\[INFO\]\s+[\s|]+[+\\]\-\s+\S+:\S+`)
	// Project root line in dep tree: "[INFO] com.example:myapp:jar:1.0-SNAPSHOT"
	reMvnDepRoot = regexp.MustCompile(`^\[INFO\]\s*\S+:\S+:\S+:\S+`)
)

func getMavenFilter(args []string) FilterFunc {
	if len(args) == 0 {
		return filterMavenBuild
	}
	switch args[0] {
	case "test":
		return filterMavenTest
	case "dependency:tree":
		return filterMavenDepTree
	case "compile", "package", "install", "clean", "verify":
		return filterMavenBuild
	default:
		return filterMavenBuild
	}
}

func filterMavenBuild(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	lines := strings.Split(raw, "\n")

	var (
		warnings []string
		errors   []string
		result   string
		elapsed  string
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Build result
		if m := reMvnBuildResult.FindStringSubmatch(trimmed); m != nil {
			result = m[1]
			continue
		}

		// Total time
		if m := reMvnTotalTime.FindStringSubmatch(trimmed); m != nil {
			elapsed = strings.TrimSpace(m[1])
			continue
		}

		// Skip noise
		if reMvnSeparator.MatchString(trimmed) ||
			reMvnDownload.MatchString(trimmed) ||
			reMvnScanning.MatchString(trimmed) ||
			reMvnBannerSeparator.MatchString(trimmed) ||
			reMvnInfoEmpty.MatchString(trimmed) ||
			reMvnReactorSummary.MatchString(trimmed) ||
			reMvnReactorModule.MatchString(trimmed) {
			continue
		}

		// Warnings
		if m := reMvnWarning.FindStringSubmatch(trimmed); m != nil {
			warnings = append(warnings, m[1])
			continue
		}

		// Errors
		if m := reMvnError.FindStringSubmatch(trimmed); m != nil {
			errors = append(errors, m[1])
			continue
		}
	}

	// Determine result
	if result == "" {
		if len(errors) > 0 {
			result = "FAILURE"
		} else {
			result = "SUCCESS"
		}
	}

	var out []string

	if result == "FAILURE" {
		out = append(out, "BUILD FAILURE")
		if len(errors) > 0 {
			for _, e := range errors {
				out = append(out, "  "+e)
			}
		}
	} else {
		header := "BUILD SUCCESS"
		if elapsed != "" {
			header += fmt.Sprintf(" (%s)", elapsed)
		}
		if len(warnings) > 0 {
			header += fmt.Sprintf(", %d warning(s)", len(warnings))
		}
		out = append(out, header)
	}

	if len(warnings) > 0 && result != "FAILURE" {
		for _, w := range warnings {
			out = append(out, "  warning: "+w)
		}
	}

	return strings.Join(out, "\n"), nil
}

func filterMavenTest(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	lines := strings.Split(raw, "\n")

	var (
		failures    []string
		inFailure   bool
		totalRun    int
		totalFail   int
		totalErr    int
		totalSkip   int
		foundSummary bool
		elapsed     string
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Total time
		if m := reMvnTotalTime.FindStringSubmatch(trimmed); m != nil {
			elapsed = strings.TrimSpace(m[1])
			continue
		}

		// Test summary line — skip per-class summaries (contain " -- in ClassName")
		if m := reMvnTestSummary.FindStringSubmatch(trimmed); m != nil {
			if !reMvnTestPerClass.MatchString(trimmed) {
				totalRun = atoi(m[1])
				totalFail = atoi(m[2])
				totalErr = atoi(m[3])
				totalSkip = atoi(m[4])
				foundSummary = true
			}
			inFailure = false
			continue
		}

		// Failed test marker
		if m := reMvnFailedTest.FindStringSubmatch(trimmed); m != nil {
			inFailure = true
			failures = append(failures, m[1])
			continue
		}

		// Skip noise
		if reMvnDownload.MatchString(trimmed) ||
			reMvnSeparator.MatchString(trimmed) ||
			reMvnBannerSeparator.MatchString(trimmed) ||
			reMvnScanning.MatchString(trimmed) ||
			reMvnInfoEmpty.MatchString(trimmed) ||
			reMvnTestsHeader.MatchString(trimmed) ||
			reMvnRunningTest.MatchString(trimmed) ||
			reMvnReactorSummary.MatchString(trimmed) ||
			reMvnReactorModule.MatchString(trimmed) ||
			reMvnBuildResult.MatchString(trimmed) {
			inFailure = false
			continue
		}

		// Results section headers end failure context
		if reMvnResultsSection.MatchString(trimmed) {
			inFailure = false
			continue
		}

		// Collect failure context
		if inFailure && trimmed != "" {
			// Stop collecting on [INFO] lines that aren't error details
			if reMvnInfoLine.MatchString(trimmed) {
				inFailure = false
				continue
			}
			failures = append(failures, "  "+trimmed)
		}
	}

	if !foundSummary {
		return raw, nil
	}

	totalFailures := totalFail + totalErr

	// All passed
	if totalFailures == 0 && totalRun > 0 {
		result := fmt.Sprintf("all %d tests passed", totalRun)
		if elapsed != "" {
			result += fmt.Sprintf(" (%s)", elapsed)
		}
		return result, nil
	}

	// Has failures
	var out []string

	if len(failures) > 0 {
		out = append(out, "Failures:")
		for _, f := range failures {
			out = append(out, f)
		}
		out = append(out, "")
	}

	out = append(out, fmt.Sprintf("Tests run: %d, Failures: %d, Errors: %d, Skipped: %d",
		totalRun, totalFail, totalErr, totalSkip))

	return strings.Join(out, "\n"), nil
}

func filterMavenDepTree(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	lines := strings.Split(raw, "\n")

	var (
		directDeps    []string
		transitiveCount int
	)

	for _, line := range lines {
		// Direct dependency (first level: "[INFO] +- " or "[INFO] \- ")
		if m := reMvnDirectDep.FindStringSubmatch(line); m != nil {
			directDeps = append(directDeps, m[1])
			continue
		}

		// Transitive dependency (deeper nesting with | or spaces before +- or \-)
		if reMvnTransitiveDep.MatchString(line) {
			transitiveCount++
			continue
		}
	}

	if len(directDeps) == 0 {
		return raw, nil
	}

	var out []string
	for _, dep := range directDeps {
		if transitiveCount > 0 {
			out = append(out, dep)
		} else {
			out = append(out, dep)
		}
	}

	out = append(out, "")
	out = append(out, fmt.Sprintf("%d direct, %d transitive dependencies", len(directDeps), transitiveCount))

	return strings.Join(out, "\n"), nil
}
