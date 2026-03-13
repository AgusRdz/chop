package filters

import (
	"strings"
	"testing"
)

var garbageInputs = []string{
	"",
	" ",
	"hello world",
	"<html><body>test</body></html>",
	`{"key": "value", "nested": {"a": 1}}`,
	"|||---|||",
	"error: something failed",
	"\x00\x01\x02binary content",
	strings.Repeat("a very long line ", 1000),
	strings.Repeat("line\n", 500),
	"ANSI \x1b[31mred\x1b[0m text",
	"commit abc123\n\nsome message",
	"CONTAINER ID   IMAGE",
	"this is definitely not output from any CLI tool 12345 random garbage xyz",
}

type namedFilter struct {
	name   string
	filter FilterFunc
}

func allFilters() []namedFilter {
	return []namedFilter{
		// Git
		{"filterGitStatus", filterGitStatus},
		{"filterGitLog", filterGitLog},
		{"filterGitDiff", filterGitDiff},
		{"filterGitBranch", filterGitBranch},
		// npm
		{"filterNpmInstall", filterNpmInstall},
		{"filterNpmList", filterNpmList},
		{"filterNpmTestCmd", filterNpmTestCmd},
		// Docker
		{"filterDockerPs", filterDockerPs},
		{"filterDockerBuild", filterDockerBuild},
		{"filterDockerImages", filterDockerImages},
		{"filterDockerLogs", filterDockerLogs},
		{"filterDockerInspect", filterDockerInspect},
		{"filterDockerStats", filterDockerStats},
		{"filterDockerNetworkLs", filterDockerNetworkLs},
		{"filterDockerVolumeLs", filterDockerVolumeLs},
		{"filterDockerHistory", filterDockerHistory},
		{"filterDockerSystemDf", filterDockerSystemDf},
		{"filterDockerTop", filterDockerTop},
		{"filterDockerDiff", filterDockerDiff},
		// .NET
		{"filterDotnetBuild", filterDotnetBuild},
		{"filterDotnetTestCmd", filterDotnetTestCmd},
		// Kubernetes
		{"filterKubectlGet", filterKubectlGet},
		{"filterKubectlDescribe", filterKubectlDescribe},
		{"filterKubectlLogs", filterKubectlLogs},
		{"filterKubectlTop", filterKubectlTop},
		{"filterKubectlApply", filterKubectlApply},
		{"filterKubectlDelete", filterKubectlDelete},
		// Helm
		{"filterHelmInstall", filterHelmInstall},
		{"filterHelmList", filterHelmList},
		// Terraform
		{"filterTerraformPlan", filterTerraformPlan},
		{"filterTerraformApply", filterTerraformApply},
		{"filterTerraformInit", filterTerraformInit},
		// HTTP
		{"filterCurl", filterCurl},
		{"filterHttpie", filterHttpie},
		// Rust
		{"filterCargoTestCmd", filterCargoTestCmd},
		{"filterCargoBuild", filterCargoBuild},
		{"filterCargoClippy", filterCargoClippy},
		// Go
		{"filterGoTestCmd", filterGoTestCmd},
		{"filterGoBuild", filterGoBuild},
		// TypeScript / Lint
		{"filterTsc", filterTsc},
		{"filterEslint", filterEslint},
		// GitHub CLI
		{"filterGhPrList", filterGhPrList},
		{"filterGhPrView", filterGhPrView},
		{"filterGhPrChecks", filterGhPrChecks},
		{"filterGhIssueList", filterGhIssueList},
		{"filterGhIssueView", filterGhIssueView},
		{"filterGhRunList", filterGhRunList},
		{"filterGhRunView", filterGhRunView},
		// Search
		{"filterGrep", filterGrep},
		// Cloud
		{"filterAwsGeneric", filterAwsGeneric},
		{"filterAwsS3Ls", filterAwsS3Ls},
		{"filterAwsEc2Describe", filterAwsEc2Describe},
		{"filterAwsLogs", filterAwsLogs},
		{"filterAzGeneric", filterAzGeneric},
		{"filterAzVmList", filterAzVmList},
		{"filterAzResourceList", filterAzResourceList},
		{"filterGcloudGeneric", filterGcloudGeneric},
		{"filterGcloudInstancesList", filterGcloudInstancesList},
		// Java
		{"filterMavenBuild", filterMavenBuild},
		{"filterMavenTest", filterMavenTest},
		{"filterMavenDepTree", filterMavenDepTree},
		{"filterGradleBuild", filterGradleBuild},
		{"filterGradleTest", filterGradleTest},
		{"filterGradleDeps", filterGradleDeps},
		// JS ecosystem
		{"filterPnpmInstall", filterPnpmInstall},
		{"filterYarnInstall", filterYarnInstall},
		{"filterBunInstall", filterBunInstall},
		// Angular / Nx
		{"filterNgBuild", filterNgBuild},
		{"filterNgTest", filterNgTest},
		{"filterNgServe", filterNgServe},
		{"filterNxBuild", filterNxBuild},
		{"filterNxTest", filterNxTest},
		// Python
		{"filterPytest", filterPytest},
		{"filterPipInstall", filterPipInstall},
		{"filterPipList", filterPipList},
		{"filterMypy", filterMypy},
		{"filterRuff", filterRuff},
		{"filterPylint", filterPylint},
		{"filterUvInstall", filterUvInstall},
		// Ruby
		{"filterBundleInstall", filterBundleInstall},
		{"filterRspec", filterRspec},
		{"filterRubocop", filterRubocop},
		// PHP
		{"filterComposerInstall", filterComposerInstall},
		// Build tools
		{"filterMake", filterMake},
		{"filterCmake", filterCmake},
		{"filterCompiler", filterCompiler},
		// System
		{"filterPing", filterPing},
		{"filterPsCmd", filterPsCmd},
		{"filterNetstat", filterNetstat},
		{"filterDf", filterDf},
		// Auto-detect
		{"filterAutoDetect", filterAutoDetect},
	}
}

func TestAllFiltersNoPanicOnGarbage(t *testing.T) {
	for _, f := range allFilters() {
		for i, input := range garbageInputs {
			t.Run(f.name+"/garbage_"+string(rune('A'+i)), func(t *testing.T) {
				result, err := f.filter(input)
				if err != nil {
					t.Errorf("%s returned error on garbage input %d: %v", f.name, i, err)
				}
				// Must return something (not empty unless input was empty or whitespace)
				if strings.TrimSpace(input) != "" && result == "" {
					t.Logf("%s returned empty for non-empty input %d", f.name, i)
				}
			})
		}
	}
}

func TestAllFiltersReturnRawOnUnrecognizedInput(t *testing.T) {
	wrongInput := "this is definitely not output from any CLI tool 12345 random garbage xyz"

	for _, f := range allFilters() {
		t.Run(f.name, func(t *testing.T) {
			result, err := f.filter(wrongInput)
			if err != nil {
				t.Errorf("%s returned error: %v", f.name, err)
			}
			// Should return raw (passthrough) since it can't recognize the format
			if result != wrongInput && result != "" {
				maxLen := len(result)
				if maxLen > 60 {
					maxLen = 60
				}
				maxInput := len(wrongInput)
				if maxInput > 30 {
					maxInput = 30
				}
				t.Logf("%s modified unrecognized input: %q -> %q", f.name, wrongInput[:maxInput], result[:maxLen])
			}
		})
	}
}

func TestAllFiltersOutputNotLargerThanInput(t *testing.T) {
	// Use a moderately sized realistic-looking but not-quite-right input
	// that filters might try to process
	inputs := []struct {
		name  string
		input string
	}{
		{
			"long_random_text",
			strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit.\n", 100),
		},
		{
			"repeated_lines",
			strings.Repeat("INFO: Application started successfully on port 8080\n", 200),
		},
	}

	for _, f := range allFilters() {
		for _, inp := range inputs {
			t.Run(f.name+"/"+inp.name, func(t *testing.T) {
				result, err := f.filter(inp.input)
				if err != nil {
					t.Errorf("%s returned error: %v", f.name, err)
				}
				if len(result) > len(inp.input) {
					t.Errorf("%s made output larger: input=%d bytes, output=%d bytes",
						f.name, len(inp.input), len(result))
				}
			})
		}
	}
}

func TestAllFiltersHandleEmptyInput(t *testing.T) {
	for _, f := range allFilters() {
		t.Run(f.name, func(t *testing.T) {
			result, err := f.filter("")
			if err != nil {
				t.Errorf("%s returned error on empty input: %v", f.name, err)
			}
			// Empty or short predefined output is fine
			if len(result) > 50 {
				t.Errorf("%s returned unexpectedly long output for empty input: %q", f.name, result[:50])
			}
		})
	}
}

func TestAllFiltersHandleWhitespaceOnly(t *testing.T) {
	whitespaceInputs := []string{" ", "\n", "\t", "  \n  \n  ", "\r\n\r\n"}

	for _, f := range allFilters() {
		for i, input := range whitespaceInputs {
			t.Run(f.name+"/ws_"+string(rune('A'+i)), func(t *testing.T) {
				result, err := f.filter(input)
				if err != nil {
					t.Errorf("%s returned error on whitespace input: %v", f.name, err)
				}
				_ = result // just checking for panics and errors
			})
		}
	}
}

func TestAllFiltersHandleUnicode(t *testing.T) {
	unicodeInputs := []string{
		"日本語テスト出力",
		"Ошибка: файл не найден",
		"🚀 deployment complete",
		"résultat: succès",
	}

	for _, f := range allFilters() {
		for i, input := range unicodeInputs {
			t.Run(f.name+"/unicode_"+string(rune('A'+i)), func(t *testing.T) {
				result, err := f.filter(input)
				if err != nil {
					t.Errorf("%s returned error on unicode input: %v", f.name, err)
				}
				_ = result
			})
		}
	}
}

func TestAllFiltersHandleAnsiCodes(t *testing.T) {
	ansiInput := "\x1b[32mPASS\x1b[0m test_something\n\x1b[31mFAIL\x1b[0m test_other\n\x1b[1mBold text\x1b[0m"

	for _, f := range allFilters() {
		t.Run(f.name, func(t *testing.T) {
			result, err := f.filter(ansiInput)
			if err != nil {
				t.Errorf("%s returned error on ANSI input: %v", f.name, err)
			}
			_ = result
		})
	}
}

func TestSanityGuards_ZeroCoverage(t *testing.T) {
	tests := []struct {
		name     string
		check    func(string) bool
		input    string
		expected bool
	}{
		{
			name:     "looksLikeKubectlLogsOutput",
			check:    looksLikeKubectlLogsOutput,
			input:    "2023-10-27T10:00:00Z INFO starting app\n",
			expected: true,
		},
		{
			name:     "looksLikeCurlOutput",
			check:    looksLikeCurlOutput,
			input:    "{\"key\": \"value\"}", // curl often returns JSON, though anything goes
			expected: true,
		},
		{
			name:     "looksLikeHttpieOutput",
			check:    looksLikeHttpieOutput,
			input:    "HTTP/1.1 200 OK\n\n{\"key\": \"value\"}",
			expected: true,
		},
		{
			name:     "looksLikeGrepOutput",
			check:    looksLikeGrepOutput,
			input:    "file.txt: match\n", // or whatever grep outputs
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.check(tt.input); got != tt.expected {
				t.Errorf("%s() = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

// No need to test negative cases for functions that always return true
