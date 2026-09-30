package hooks

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

// conflictFixedIn is the first Claude Code release where multiple Bash
// PreToolUse hooks no longer drop updatedInput (anthropics/claude-code#15897).
var conflictFixedIn = [3]int{2, 1, 168}

var claudeVersionRe = regexp.MustCompile(`^\s*(\d+)\.(\d+)\.(\d+)`)

// claudeCodeVersion returns the installed Claude Code version ("2.1.286").
// It is a variable so tests can stub it without spawning a process.
var claudeCodeVersion = func() (string, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return "", err
	}
	return parseClaudeVersion(string(out))
}

// parseClaudeVersion extracts the leading a.b.c from `claude --version` output
// such as "2.1.286 (Claude Code)".
func parseClaudeVersion(out string) (string, error) {
	m := claudeVersionRe.FindStringSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("unrecognized claude version output: %q", out)
	}
	return m[1] + "." + m[2] + "." + m[3], nil
}

// versionAtLeast reports whether version ("a.b.c") is >= min, component-wise.
func versionAtLeast(version string, min [3]int) bool {
	m := claudeVersionRe.FindStringSubmatch(version)
	if m == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return false
		}
		if n != min[i] {
			return n > min[i]
		}
	}
	return true
}

// ConflictBugFixed reports the installed Claude Code version and whether it
// includes the fix for the competing-hook bug. Any failure to determine the
// version returns fixed=false so callers keep showing the warning.
func ConflictBugFixed() (version string, fixed bool) {
	version, err := claudeCodeVersion()
	if err != nil {
		return "", false
	}
	return version, versionAtLeast(version, conflictFixedIn)
}
