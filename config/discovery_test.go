package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPreferredExecutablePathPreservesStableSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}

	dir := t.TempDir()
	versionedPath := filepath.Join(dir, "Cellar", "chop", "1.0.0", "bin", "chop")
	if err := os.MkdirAll(filepath.Dir(versionedPath), 0o755); err != nil {
		t.Fatalf("failed to create versioned directory: %v", err)
	}
	if err := os.WriteFile(versionedPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("failed to create versioned binary: %v", err)
	}

	stablePath := filepath.Join(dir, "bin", "chop")
	if err := os.MkdirAll(filepath.Dir(stablePath), 0o755); err != nil {
		t.Fatalf("failed to create stable directory: %v", err)
	}
	if err := os.Symlink(versionedPath, stablePath); err != nil {
		t.Fatalf("failed to create stable symlink: %v", err)
	}

	got, err := preferredExecutablePath(versionedPath, "chop", func(name string) (string, error) {
		return stablePath, nil
	})
	if err != nil {
		t.Fatalf("preferredExecutablePath failed: %v", err)
	}

	want := strings.ReplaceAll(stablePath, "\\", "/")
	if got != want {
		t.Fatalf("got %q, want stable invocation path %q", got, want)
	}
}

func TestPreferredExecutablePathRejectsDifferentExecutable(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "chop")
	otherExecutable := filepath.Join(dir, "other", "chop")
	if err := os.WriteFile(executable, []byte("current"), 0o755); err != nil {
		t.Fatalf("failed to create current executable: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(otherExecutable), 0o755); err != nil {
		t.Fatalf("failed to create other directory: %v", err)
	}
	if err := os.WriteFile(otherExecutable, []byte("other"), 0o755); err != nil {
		t.Fatalf("failed to create other executable: %v", err)
	}

	got, err := preferredExecutablePath(executable, "chop", func(name string) (string, error) {
		return otherExecutable, nil
	})
	if err != nil {
		t.Fatalf("preferredExecutablePath failed: %v", err)
	}

	want := strings.ReplaceAll(executable, "\\", "/")
	if got != want {
		t.Fatalf("got %q, want resolved current executable %q", got, want)
	}
}

func TestPreferredExecutablePathFallsBackWhenInvocationIsMissing(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "chop")
	if err := os.WriteFile(executable, []byte("current"), 0o755); err != nil {
		t.Fatalf("failed to create executable: %v", err)
	}

	got, err := preferredExecutablePath(executable, "chop", func(name string) (string, error) {
		return "", os.ErrNotExist
	})
	if err != nil {
		t.Fatalf("preferredExecutablePath failed: %v", err)
	}

	want := strings.ReplaceAll(executable, "\\", "/")
	if got != want {
		t.Fatalf("got %q, want resolved executable fallback %q", got, want)
	}
}

// --- DiscoveryPath ---

func TestDiscoveryPath_ReturnsPathJSONUnderDotChop(t *testing.T) {
	path, err := DiscoveryPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(path) != "path.json" {
		t.Errorf("expected file name 'path.json', got %q", filepath.Base(path))
	}
	if !strings.Contains(filepath.ToSlash(path), ".chop/") {
		t.Errorf("expected path to contain '.chop/', got %s", path)
	}
}

func TestDiscoveryPath_UnderHomeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	path, err := DiscoveryPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(path, home) {
		t.Errorf("expected path to start with home dir %s, got %s", home, path)
	}
}

// --- WriteDiscoveryInfo ---

func TestWriteDiscoveryInfo_CreatesFileWithCorrectContent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	const testVersion = "v1.2.3-test"

	err := WriteDiscoveryInfo(testVersion)
	if err != nil {
		t.Fatalf("WriteDiscoveryInfo returned error: %v", err)
	}

	// Locate the file using DiscoveryPath so the path logic is consistent.
	path, err := DiscoveryPath()
	if err != nil {
		t.Fatalf("DiscoveryPath error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read discovery file %s: %v", path, err)
	}

	var info DiscoveryInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("discovery file is not valid JSON: %v", err)
	}

	if info.Version != testVersion {
		t.Errorf("expected version %q, got %q", testVersion, info.Version)
	}
	if info.Path == "" {
		t.Error("expected non-empty Path in discovery info")
	}
	wantPath, err := ExecutablePath()
	if err != nil {
		t.Fatalf("ExecutablePath error: %v", err)
	}
	if info.Path != wantPath {
		t.Errorf("expected executable path %q, got %q", wantPath, info.Path)
	}
}

func TestWriteDiscoveryInfo_FileIsValidJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := WriteDiscoveryInfo("vtest")
	if err != nil {
		t.Fatalf("WriteDiscoveryInfo returned error: %v", err)
	}

	path, err := DiscoveryPath()
	if err != nil {
		t.Fatalf("DiscoveryPath error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Errorf("file is not valid JSON: %v\ncontent: %s", err, string(data))
	}

	if _, ok := raw["version"]; !ok {
		t.Error("expected 'version' key in JSON output")
	}
	if _, ok := raw["path"]; !ok {
		t.Error("expected 'path' key in JSON output")
	}
}

// --- ValidateFilters ---

func writeFilterTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "filters.yml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateFilters_MissingFile(t *testing.T) {
	errs := ValidateFilters("/nonexistent/path/filters.yml")
	if len(errs) == 0 {
		t.Error("expected error for missing file")
	}
}

func TestValidateFilters_InvalidYAML(t *testing.T) {
	path := writeFilterTemp(t, "{{invalid yaml")
	errs := ValidateFilters(path)
	if len(errs) == 0 {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidateFilters_ValidFilters(t *testing.T) {
	content := `
filters:
  "mycli build":
    keep: ["ERROR", "^BUILD"]
    drop: ["DEBUG"]
  terraform:
    head: 10
    tail: 5
`
	path := writeFilterTemp(t, content)
	errs := ValidateFilters(path)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid filters, got: %v", errs)
	}
}

func TestValidateFilters_InvalidKeepRegex(t *testing.T) {
	content := `
filters:
  "mycli build":
    keep: ["[invalid regex"]
`
	path := writeFilterTemp(t, content)
	errs := ValidateFilters(path)
	if len(errs) == 0 {
		t.Error("expected error for invalid keep regex")
	}
}

func TestValidateFilters_InvalidDropRegex(t *testing.T) {
	content := `
filters:
  "mycli build":
    drop: ["(unclosed"]
`
	path := writeFilterTemp(t, content)
	errs := ValidateFilters(path)
	if len(errs) == 0 {
		t.Error("expected error for invalid drop regex")
	}
}

func TestValidateFilters_MissingExecScript(t *testing.T) {
	content := `
filters:
  "mycli build":
    exec: "/nonexistent/path/to/script.sh"
`
	path := writeFilterTemp(t, content)
	errs := ValidateFilters(path)
	if len(errs) == 0 {
		t.Error("expected error for missing exec script")
	}
}

func TestValidateFilters_ValidExecScript(t *testing.T) {
	// Create an actual script file so validation passes.
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "filter.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\ncat\n"), 0700); err != nil {
		t.Fatal(err)
	}

	content := "filters:\n  \"mycli\":\n    exec: \"" + filepath.ToSlash(scriptPath) + "\"\n"
	filterPath := filepath.Join(dir, "filters.yml")
	if err := os.WriteFile(filterPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	errs := ValidateFilters(filterPath)
	if len(errs) != 0 {
		t.Errorf("expected no errors for existing exec script, got: %v", errs)
	}
}

func TestValidateFilters_EmptyFile(t *testing.T) {
	path := writeFilterTemp(t, "")
	errs := ValidateFilters(path)
	if len(errs) != 0 {
		t.Errorf("expected no errors for empty file, got: %v", errs)
	}
}

func TestValidateFilters_MultipleErrors(t *testing.T) {
	content := `
filters:
  "tool1":
    keep: ["[bad regex"]
  "tool2":
    drop: ["(unclosed"]
`
	path := writeFilterTemp(t, content)
	errs := ValidateFilters(path)
	if len(errs) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(errs), errs)
	}
}
