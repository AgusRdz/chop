package filters

import "strings"

// Format recognition functions for graceful fallback.
// Each returns true if the input looks like it could be output from the expected command.
// Uses simple string checks (Contains, HasPrefix) - no regex for speed.

func looksLikeGitStatusOutput(s string) bool {
	return strings.Contains(s, "On branch") ||
		strings.Contains(s, "Changes") ||
		strings.Contains(s, "nothing to commit") ||
		strings.Contains(s, "Untracked") ||
		strings.Contains(s, "modified:") ||
		strings.Contains(s, "new file:") ||
		strings.Contains(s, "deleted:") ||
		strings.Contains(s, "renamed:") ||
		strings.Contains(s, "??") // short format untracked
}

func looksLikeGitLogOutput(s string) bool {
	return strings.Contains(s, "commit ") ||
		strings.Contains(s, "Author:") ||
		strings.Contains(s, "Date:") ||
		// Oneline format: short hash followed by space and message
		(len(s) > 8 && isHexPrefix(s))
}

func looksLikeGitDiffOutput(s string) bool {
	return strings.Contains(s, "diff --git") ||
		strings.Contains(s, "+++") ||
		strings.Contains(s, "---") ||
		strings.Contains(s, "@@") ||
		strings.HasPrefix(s, "index ")
}

func looksLikeGitBranchOutput(s string) bool {
	// Branch output has lines starting with "* " (current) or "  " (others)
	// or just branch names
	lines := strings.SplitN(s, "\n", 5)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "  ") || len(trimmed) > 0 {
			return true
		}
	}
	return false
}

func looksLikeNpmInstallOutput(s string) bool {
	return strings.Contains(s, "added") ||
		strings.Contains(s, "npm") ||
		strings.Contains(s, "packages") ||
		strings.Contains(s, "up to date") ||
		strings.Contains(s, "vulnerabilities") ||
		strings.Contains(s, "WARN") ||
		strings.Contains(s, "ERR!")
}

func looksLikeNpmListOutput(s string) bool {
	return strings.Contains(s, "@") ||
		strings.Contains(s, "+--") ||
		strings.Contains(s, "`--") ||
		strings.Contains(s, "├") ||
		strings.Contains(s, "└")
}

func looksLikeNpmTestOutput(s string) bool {
	return strings.Contains(s, "PASS") ||
		strings.Contains(s, "FAIL") ||
		strings.Contains(s, "Tests:") ||
		strings.Contains(s, "Test Suites:") ||
		strings.Contains(s, "passing") ||
		strings.Contains(s, "failing") ||
		strings.Contains(s, "test") ||
		strings.Contains(s, "expect")
}

func looksLikeDockerPsOutput(s string) bool {
	return strings.Contains(s, "CONTAINER") ||
		strings.Contains(s, "IMAGE") ||
		strings.Contains(s, "STATUS")
}

func looksLikeDockerBuildOutput(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "step") ||
		strings.Contains(s, "#") ||
		strings.Contains(lower, "successfully built") ||
		strings.Contains(lower, "successfully tagged") ||
		strings.Contains(lower, "writing image") ||
		strings.Contains(lower, "naming to") ||
		strings.Contains(lower, "building") ||
		strings.Contains(lower, "dockerfile")
}

func looksLikeDockerImagesOutput(s string) bool {
	return strings.Contains(s, "REPOSITORY") ||
		strings.Contains(s, "IMAGE ID") ||
		strings.Contains(s, "TAG") ||
		strings.Contains(s, "IMAGE") && strings.Contains(s, "DISK USAGE")
}

func looksLikeDockerStatsOutput(s string) bool {
	return strings.Contains(s, "CPU %") ||
		strings.Contains(s, "MEM USAGE") ||
		strings.Contains(s, "NET I/O")
}

func looksLikeDockerNetworkLsOutput(s string) bool {
	return strings.Contains(s, "NETWORK ID") ||
		(strings.Contains(s, "NAME") && strings.Contains(s, "DRIVER") && strings.Contains(s, "SCOPE"))
}

func looksLikeDockerVolumeLsOutput(s string) bool {
	return strings.Contains(s, "VOLUME NAME") ||
		(strings.Contains(s, "DRIVER") && strings.Contains(s, "VOLUME"))
}

func looksLikeDockerHistoryOutput(s string) bool {
	return strings.Contains(s, "CREATED BY") ||
		(strings.Contains(s, "IMAGE") && strings.Contains(s, "SIZE") && strings.Contains(s, "CREATED"))
}

func looksLikeDockerSystemDfOutput(s string) bool {
	return strings.Contains(s, "RECLAIMABLE") ||
		(strings.Contains(s, "TYPE") && strings.Contains(s, "TOTAL") && strings.Contains(s, "SIZE"))
}

func looksLikeDockerDiffOutput(s string) bool {
	lines := strings.SplitN(s, "\n", 5)
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if len(t) > 2 && (t[0] == 'A' || t[0] == 'C' || t[0] == 'D') && t[1] == ' ' {
			return true
		}
	}
	return false
}

func looksLikeDotnetBuildOutput(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "build") ||
		strings.Contains(s, "error") ||
		strings.Contains(s, "warning") ||
		strings.Contains(s, "Microsoft") ||
		strings.Contains(s, "Restore") ||
		strings.Contains(s, ".csproj") ||
		strings.Contains(s, ".sln")
}

func looksLikeDotnetTestOutput(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "passed") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "test") ||
		strings.Contains(lower, "total tests") ||
		strings.Contains(s, "Microsoft")
}

func looksLikeKubectlGetOutput(s string) bool {
	return strings.Contains(s, "NAME") ||
		strings.Contains(s, "STATUS") ||
		strings.Contains(s, "READY") ||
		strings.Contains(s, "AGE") ||
		strings.HasPrefix(s, "No resources found") ||
		strings.HasPrefix(s, "{") ||
		strings.HasPrefix(s, "[") ||
		strings.HasPrefix(s, "apiVersion:") ||
		strings.HasPrefix(s, "kind:")
}

func looksLikeKubectlDescribeOutput(s string) bool {
	return strings.Contains(s, "Name:") ||
		strings.Contains(s, "Namespace:") ||
		strings.Contains(s, "Labels:") ||
		strings.Contains(s, "Status:") ||
		strings.Contains(s, "Node:")
}

func looksLikeKubectlLogsOutput(_ string) bool {
	// Logs can be anything - always attempt to filter
	return true
}

func looksLikeKubectlTopOutput(s string) bool {
	return strings.Contains(s, "CPU(cores)") || strings.Contains(s, "MEMORY(bytes)")
}

func looksLikeKubectlApplyOutput(s string) bool {
	return strings.Contains(s, "created") ||
		strings.Contains(s, "configured") ||
		strings.Contains(s, "unchanged") ||
		strings.Contains(s, "replaced")
}

func looksLikeKubectlDeleteOutput(s string) bool {
	return strings.Contains(s, "deleted")
}

func looksLikeHelmInstallOutput(s string) bool {
	return strings.Contains(s, "NAME:") ||
		strings.Contains(s, "STATUS:") ||
		strings.Contains(s, "REVISION:") ||
		strings.Contains(s, "Happy Helming") ||
		strings.Contains(s, "NAMESPACE:")
}

func looksLikeHelmListOutput(s string) bool {
	return strings.Contains(s, "NAME") &&
		strings.Contains(s, "NAMESPACE") &&
		strings.Contains(s, "STATUS") &&
		strings.Contains(s, "CHART")
}

func looksLikeTerraformPlanOutput(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "plan") ||
		strings.Contains(s, "no changes") ||
		strings.Contains(s, "# ") ||
		strings.Contains(s, "will be") ||
		strings.Contains(s, "must be") ||
		strings.Contains(lower, "terraform") ||
		strings.Contains(lower, "error") ||
		strings.Contains(s, "to add") ||
		strings.Contains(s, "to change") ||
		strings.Contains(s, "to destroy")
}

func looksLikeTerraformApplyOutput(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "apply") ||
		strings.Contains(lower, "creating") ||
		strings.Contains(lower, "modifying") ||
		strings.Contains(lower, "destroying") ||
		strings.Contains(lower, "complete") ||
		strings.Contains(lower, "terraform") ||
		strings.Contains(lower, "error")
}

func looksLikeTerraformInitOutput(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "terraform") ||
		strings.Contains(lower, "initializ") ||
		strings.Contains(lower, "provider") ||
		strings.Contains(lower, "installing") ||
		strings.Contains(lower, "installed") ||
		strings.Contains(lower, "error")
}

func looksLikeCurlOutput(_ string) bool {
	// curl output can be anything (JSON, HTML, text, headers) - always attempt
	return true
}

func looksLikeHttpieOutput(_ string) bool {
	// httpie output can be anything - always attempt
	return true
}

func looksLikeCargoTestOutput(s string) bool {
	return strings.Contains(s, "test result:") ||
		strings.Contains(s, "test ") ||
		strings.Contains(s, "running") ||
		strings.Contains(s, "FAILED") ||
		strings.Contains(s, "Compiling") ||
		strings.Contains(s, "Doc-tests")
}

func looksLikeCargoBuildOutput(s string) bool {
	return strings.Contains(s, "Compiling") ||
		strings.Contains(s, "Checking") ||
		strings.Contains(s, "Finished") ||
		strings.Contains(s, "error") ||
		strings.Contains(s, "warning:") ||
		strings.Contains(s, "-->")
}

func looksLikeCargoClippyOutput(s string) bool {
	return strings.Contains(s, "Checking") ||
		strings.Contains(s, "Compiling") ||
		strings.Contains(s, "warning:") ||
		strings.Contains(s, "error") ||
		strings.Contains(s, "clippy") ||
		strings.Contains(s, "-->")
}

func looksLikeGoTestOutput(s string) bool {
	return strings.Contains(s, "=== RUN") ||
		strings.Contains(s, "--- PASS") ||
		strings.Contains(s, "--- FAIL") ||
		strings.Contains(s, "--- SKIP") ||
		strings.Contains(s, "PASS") ||
		strings.Contains(s, "FAIL") ||
		strings.Contains(s, "ok  \t") ||
		strings.Contains(s, "[no test files]")
}

func looksLikeGoBuildOutput(s string) bool {
	return strings.Contains(s, ".go:") ||
		strings.Contains(s, "# ") ||
		strings.Contains(s, "build") ||
		strings.Contains(s, "vet")
}

func looksLikeTscOutput(s string) bool {
	return strings.Contains(s, "error TS") ||
		strings.Contains(s, "Found") ||
		strings.Contains(s, ".ts") ||
		strings.Contains(s, ".tsx")
}

func looksLikeEslintOutput(s string) bool {
	return strings.Contains(s, "error") ||
		strings.Contains(s, "warning") ||
		strings.Contains(s, "problems") ||
		strings.Contains(s, "✖") ||
		strings.Contains(s, "✓") ||
		strings.Contains(s, "/") // file paths
}

func looksLikeGhPrOutput(s string) bool {
	return strings.Contains(s, "title") ||
		strings.Contains(s, "Title") ||
		strings.Contains(s, "#") ||
		strings.Contains(s, "OPEN") ||
		strings.Contains(s, "CLOSED") ||
		strings.Contains(s, "MERGED") ||
		strings.Contains(s, "DRAFT") ||
		strings.Contains(s, "\t") || // tab-delimited gh output
		strings.Contains(s, "state") ||
		strings.Contains(s, "author")
}

func looksLikeGhIssueOutput(s string) bool {
	return strings.Contains(s, "title") ||
		strings.Contains(s, "Title") ||
		strings.Contains(s, "#") ||
		strings.Contains(s, "\t") ||
		strings.Contains(s, "state") ||
		strings.Contains(s, "author") ||
		strings.Contains(s, "OPEN") ||
		strings.Contains(s, "CLOSED")
}

func looksLikeGhRunOutput(s string) bool {
	return strings.Contains(s, "status") ||
		strings.Contains(s, "Status") ||
		strings.Contains(s, "workflow") ||
		strings.Contains(s, "Workflow") ||
		strings.Contains(s, "\t") ||
		strings.Contains(s, "completed") ||
		strings.Contains(s, "in_progress") ||
		strings.Contains(s, "success") ||
		strings.Contains(s, "failure")
}

func looksLikeGrepOutput(s string) bool {
	// grep output is either file:line:content or plain text - always process
	return true
}

func looksLikeMavenOutput(s string) bool {
	return strings.Contains(s, "[INFO]") ||
		strings.Contains(s, "[WARNING]") ||
		strings.Contains(s, "[ERROR]") ||
		strings.Contains(s, "BUILD") ||
		strings.Contains(s, "Maven")
}

func looksLikeGradleOutput(s string) bool {
	return strings.Contains(s, "> Task") ||
		strings.Contains(s, "BUILD") ||
		strings.Contains(s, "Gradle") ||
		strings.Contains(s, "actionable") ||
		strings.Contains(s, "+---") ||
		strings.Contains(s, "\\---")
}

// --- New sanity guards ---

func looksLikeNgBuildOutput(s string) bool {
	return strings.Contains(s, "Initial Chunk Files") ||
		strings.Contains(s, "Initial Total") ||
		strings.Contains(s, "bundle generation") ||
		strings.Contains(s, "Build at:")
}

func looksLikeNgTestOutput(s string) bool {
	return strings.Contains(s, "TOTAL:") ||
		strings.Contains(s, "Executed") ||
		strings.Contains(s, "Karma") ||
		strings.Contains(s, "Chrome")
}

func looksLikeNgServeOutput(s string) bool {
	return strings.Contains(s, "listening on") ||
		strings.Contains(s, "Live Development Server") ||
		strings.Contains(s, "Angular")
}

func looksLikeNxBuildOutput(s string) bool {
	return strings.Contains(s, "NX") ||
		strings.Contains(s, "nx run") ||
		strings.Contains(s, "Successfully ran target")
}

func looksLikeNxTestOutput(s string) bool {
	return (strings.Contains(s, "NX") || strings.Contains(s, "nx run")) &&
		(strings.Contains(s, "test") || strings.Contains(s, "Test"))
}

func looksLikePytestOutput(s string) bool {
	return strings.Contains(s, "test session starts") ||
		strings.Contains(s, "passed") ||
		strings.Contains(s, "failed") ||
		strings.Contains(s, "collected") ||
		strings.Contains(s, "pytest")
}

func looksLikePipInstallOutput(s string) bool {
	return strings.Contains(s, "Collecting") ||
		strings.Contains(s, "Successfully installed") ||
		strings.Contains(s, "already satisfied") ||
		strings.Contains(s, "Downloading") ||
		strings.Contains(s, "Installing collected")
}

func looksLikePipListOutput(s string) bool {
	return strings.Contains(s, "Package") && strings.Contains(s, "Version")
}

func looksLikeMypyOutput(s string) bool {
	return strings.Contains(s, ": error:") ||
		strings.Contains(s, ": warning:") ||
		strings.Contains(s, ": note:") ||
		strings.Contains(s, "Found") ||
		strings.Contains(s, "mypy") ||
		strings.Contains(s, "Success:")
}

func looksLikeRuffOutput(s string) bool {
	return strings.Contains(s, ": F") ||
		strings.Contains(s, ": E") ||
		strings.Contains(s, ": W") ||
		strings.Contains(s, "Found") ||
		strings.Contains(s, "fixable")
}

func looksLikePylintOutput(s string) bool {
	return strings.Contains(s, "************") ||
		strings.Contains(s, "(missing-") ||
		strings.Contains(s, "C0") ||
		strings.Contains(s, "W0") ||
		strings.Contains(s, "E0") ||
		strings.Contains(s, "R0")
}

func looksLikeUvInstallOutput(s string) bool {
	return strings.Contains(s, "Resolved") ||
		strings.Contains(s, "Installed") ||
		strings.Contains(s, "Prepared")
}

func looksLikePnpmInstallOutput(s string) bool {
	return strings.Contains(s, "Packages:") ||
		strings.Contains(s, "pnpm") ||
		strings.Contains(s, "Progress:") ||
		strings.Contains(s, "devDependencies")
}

func looksLikeYarnInstallOutput(s string) bool {
	return strings.Contains(s, "yarn") ||
		strings.Contains(s, "Done in") ||
		strings.Contains(s, "Resolving") ||
		strings.Contains(s, "Fetching") ||
		strings.Contains(s, "YN0000")
}

func looksLikeBunInstallOutput(s string) bool {
	return strings.Contains(s, "bun") ||
		strings.Contains(s, "Installed") ||
		strings.Contains(s, "Resolved")
}

func looksLikeBundleInstallOutput(s string) bool {
	return strings.Contains(s, "Fetching gem metadata") ||
		strings.Contains(s, "Bundle complete") ||
		strings.Contains(s, "Installing") ||
		strings.Contains(s, "bundler") ||
		strings.Contains(s, "Gemfile")
}

func looksLikeRspecOutput(s string) bool {
	return strings.Contains(s, "examples") ||
		strings.Contains(s, "Finished in") ||
		strings.Contains(s, "rspec") ||
		strings.Contains(s, "Failures:") ||
		strings.Contains(s, "Randomized with seed")
}

func looksLikeRubocopOutput(s string) bool {
	return strings.Contains(s, "Inspecting") ||
		strings.Contains(s, "offenses") ||
		strings.Contains(s, "Offenses:") ||
		strings.Contains(s, "Style/") ||
		strings.Contains(s, "Lint/") ||
		strings.Contains(s, "Layout/")
}

func looksLikeComposerOutput(s string) bool {
	return strings.Contains(s, "Installing dependencies") ||
		strings.Contains(s, "Package operations") ||
		strings.Contains(s, "composer") ||
		strings.Contains(s, "autoload") ||
		strings.Contains(s, "Verifying lock file")
}

func looksLikeMakeOutput(s string) bool {
	return strings.Contains(s, "make[") ||
		strings.Contains(s, "Entering directory") ||
		strings.Contains(s, "Leaving directory") ||
		strings.Contains(s, "make:")
}

func looksLikeCmakeOutput(s string) bool {
	return strings.Contains(s, "-- ") ||
		strings.Contains(s, "Building CXX") ||
		strings.Contains(s, "Building C ") ||
		strings.Contains(s, "Built target") ||
		strings.Contains(s, "Configuring done")
}

func looksLikeCompilerOutput(s string) bool {
	return strings.Contains(s, ": error:") ||
		strings.Contains(s, ": warning:") ||
		strings.Contains(s, ": note:") ||
		strings.Contains(s, "In function") ||
		strings.Contains(s, "In file included from")
}

func looksLikePingOutput(s string) bool {
	return strings.Contains(s, "PING") ||
		strings.Contains(s, "Pinging") ||
		strings.Contains(s, "icmp_seq") ||
		strings.Contains(s, "ping statistics") ||
		strings.Contains(s, "Reply from") ||
		strings.Contains(s, "packets transmitted")
}

func looksLikePsCmdOutput(s string) bool {
	return (strings.Contains(s, "PID") &&
		(strings.Contains(s, "%CPU") || strings.Contains(s, "%MEM") || strings.Contains(s, "COMMAND") || strings.Contains(s, "CMD")))
}

func looksLikeNetstatOutput(s string) bool {
	return strings.Contains(s, "LISTEN") ||
		strings.Contains(s, "ESTAB") ||
		strings.Contains(s, "ESTABLISHED") ||
		(strings.Contains(s, "Proto") && strings.Contains(s, "Local Address"))
}

func looksLikeDfOutput(s string) bool {
	return strings.Contains(s, "Filesystem") ||
		strings.Contains(s, "Mounted on") ||
		strings.Contains(s, "Use%")
}

// isHexPrefix checks if string starts with what looks like a hex hash (git oneline format)
func isHexPrefix(s string) bool {
	if len(s) < 7 {
		return false
	}
	for i := 0; i < 7; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// outputSanityCheck returns raw if the filtered result is strictly longer than the raw input.
// Equal length is allowed since some filters reformat without compressing (e.g., git branch reorder).
func outputSanityCheck(raw, result string) string {
	if len(result) > len(raw) {
		return raw
	}
	return result
}
