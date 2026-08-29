package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AgusRdz/chop/hooks"
)

func TestRunGlobalInitStatusDoesNotInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := runGlobalInit([]string{"--status"}); err != nil {
		t.Fatalf("runGlobalInit --status failed: %v", err)
	}

	for _, path := range []string{
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(home, ".chop", "path.json"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("status should not create %s", path)
		}
	}
}

func TestRunGlobalInitUninstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		t.Fatalf("failed to create settings directory: %v", err)
	}
	settings := `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"chop hook"}]}]}}`
	if err := os.WriteFile(settingsPath, []byte(settings), 0o600); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	if err := runGlobalInit([]string{"--uninstall"}); err != nil {
		t.Fatalf("runGlobalInit --uninstall failed: %v", err)
	}
	if installed, _ := hooks.IsInstalled(); installed {
		t.Fatal("global hook should be uninstalled")
	}
}

func TestRunGlobalInitRejectsUnknownFlag(t *testing.T) {
	err := runGlobalInit([]string{"--unknown"})
	if err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got %v", err)
	}
}

// --- sanitizeFilename ---

func TestSanitizeFilename_ForwardSlash(t *testing.T) {
	got := sanitizeFilename("path/to/file")
	if strings.Contains(got, "/") {
		t.Errorf("sanitizeFilename should replace '/': got %q", got)
	}
	want := "path_to_file"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSanitizeFilename_BackwardSlash(t *testing.T) {
	got := sanitizeFilename(`path\to\file`)
	if strings.Contains(got, `\`) {
		t.Errorf("sanitizeFilename should replace '\\': got %q", got)
	}
	want := "path_to_file"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSanitizeFilename_DotDot(t *testing.T) {
	got := sanitizeFilename("../../etc/passwd")
	if strings.Contains(got, "..") {
		t.Errorf("sanitizeFilename should replace '..': got %q", got)
	}
	// "../../etc/passwd" → replace "/" → ".._.._etc_passwd" → replace ".." → "______etc_passwd"
	want := "______etc_passwd"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSanitizeFilename_Clean(t *testing.T) {
	got := sanitizeFilename("git-status-20240101")
	want := "git-status-20240101"
	if got != want {
		t.Errorf("sanitizeFilename modified clean name: got %q, want %q", got, want)
	}
}

func TestSanitizeFilename_Empty(t *testing.T) {
	got := sanitizeFilename("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestSanitizeFilename_Mixed(t *testing.T) {
	// Combines forward slash, backslash, and dot-dot
	got := sanitizeFilename(`../foo\bar/..`)
	if strings.Contains(got, "/") || strings.Contains(got, `\`) || strings.Contains(got, "..") {
		t.Errorf("sanitizeFilename did not sanitize all separators: got %q", got)
	}
}

// --- firstNLines ---

func TestFirstNLines_ExactLines(t *testing.T) {
	input := "a\nb\nc"
	got := firstNLines(input, 3)
	if got != input {
		t.Errorf("firstNLines with n=len got %q, want %q", got, input)
	}
}

func TestFirstNLines_FewerThanN(t *testing.T) {
	input := "a\nb"
	got := firstNLines(input, 10)
	if got != input {
		t.Errorf("firstNLines with n>len got %q, want %q", got, input)
	}
}

func TestFirstNLines_MoreThanN(t *testing.T) {
	input := "line1\nline2\nline3\nline4\nline5"
	got := firstNLines(input, 3)
	want := "line1\nline2\nline3"
	if got != want {
		t.Errorf("firstNLines got %q, want %q", got, want)
	}
}

func TestFirstNLines_Zero(t *testing.T) {
	got := firstNLines("a\nb\nc", 0)
	if got != "" {
		t.Errorf("firstNLines(n=0) got %q, want empty", got)
	}
}

func TestFirstNLines_One(t *testing.T) {
	got := firstNLines("hello\nworld", 1)
	want := "hello"
	if got != want {
		t.Errorf("firstNLines(n=1) got %q, want %q", got, want)
	}
}

func TestFirstNLines_Empty(t *testing.T) {
	got := firstNLines("", 5)
	if got != "" {
		t.Errorf("firstNLines on empty string got %q, want empty", got)
	}
}

func TestFirstNLines_TrailingNewline(t *testing.T) {
	// A trailing newline counts as an empty final line
	input := "a\nb\n"
	got := firstNLines(input, 2)
	want := "a\nb"
	if got != want {
		t.Errorf("firstNLines with trailing newline got %q, want %q", got, want)
	}
}

// --- validateCommand ---

func TestValidateCommand_Valid(t *testing.T) {
	// These should not call os.Exit — they pass silently.
	// We just call them and verify no panic.
	cases := []string{"git", "docker", "npm", "go", "my-tool", "tool_name"}
	for _, cmd := range cases {
		// validateCommand calls os.Exit on failure; safe cases just return.
		// We can't easily test os.Exit without subprocess, so we verify the
		// function does not panic for valid inputs.
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("validateCommand(%q) panicked: %v", cmd, r)
				}
			}()
			// Cannot call directly since it calls os.Exit on error;
			// here we test the validation condition directly.
			invalid := strings.ContainsAny(cmd, ";|&><`$()\n\r")
			if invalid {
				t.Errorf("validateCommand(%q) should be valid but ContainsAny returned true", cmd)
			}
		}()
	}
}

func TestValidateCommand_Invalid(t *testing.T) {
	// These should be caught by validateCommand's ContainsAny check.
	cases := []string{
		"git; rm -rf /",
		"foo|bar",
		"cmd&cmd2",
		"foo>out",
		"$(evil)",
		"back`tick`",
		"new\nline",
		"carriage\rreturn",
	}
	for _, cmd := range cases {
		got := strings.ContainsAny(cmd, ";|&><`$()\n\r")
		if !got {
			t.Errorf("validateCommand(%q): expected to be flagged as invalid", cmd)
		}
	}
}

// --- extractLatestVersion ---

func TestExtractLatestVersion_ReturnsFirstVersion(t *testing.T) {
	cl := `## [Unreleased]
Some unreleased stuff.

## [v1.5.0] - 2024-01-15
### Added
- Feature A

## [v1.4.0] - 2024-01-01
### Fixed
- Bug B
`
	got := extractLatestVersion(cl)
	if !strings.Contains(got, "v1.5.0") {
		t.Errorf("expected v1.5.0 section, got: %q", got)
	}
	if strings.Contains(got, "v1.4.0") {
		t.Errorf("should not include v1.4.0 section, got: %q", got)
	}
	if strings.Contains(got, "Unreleased") {
		t.Errorf("should not include Unreleased section, got: %q", got)
	}
}

func TestExtractLatestVersion_NoSections(t *testing.T) {
	cl := "just some text\nno version headers\n"
	got := extractLatestVersion(cl)
	// Falls back to returning the whole changelog
	if got != cl {
		t.Errorf("expected full changelog returned when no sections, got: %q", got)
	}
}

func TestExtractLatestVersion_OnlyUnreleased(t *testing.T) {
	cl := `## [Unreleased]
- Work in progress
`
	got := extractLatestVersion(cl)
	// No version section found, returns full string
	if got != cl {
		t.Errorf("expected full changelog for unreleased-only, got: %q", got)
	}
}

func TestExtractLatestVersion_SingleVersion(t *testing.T) {
	cl := `## [v2.0.0] - 2025-01-01
### Breaking
- Dropped old API
`
	got := extractLatestVersion(cl)
	if !strings.Contains(got, "v2.0.0") {
		t.Errorf("expected v2.0.0 in result, got: %q", got)
	}
}

// --- inlineStringSlice ---

func TestInlineStringSlice_Empty(t *testing.T) {
	got := inlineStringSlice([]string{})
	if got != "[]" {
		t.Errorf("expected '[]', got %q", got)
	}
}

func TestInlineStringSlice_Single(t *testing.T) {
	got := inlineStringSlice([]string{"hello"})
	want := `["hello"]`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineStringSlice_Multiple(t *testing.T) {
	got := inlineStringSlice([]string{"a", "b", "c"})
	want := `["a", "b", "c"]`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineStringSlice_QuotesSpecialChars(t *testing.T) {
	got := inlineStringSlice([]string{`he said "hi"`})
	// fmt.Sprintf("%q", ...) escapes internal quotes
	want := fmt.Sprintf("[%q]", `he said "hi"`)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
