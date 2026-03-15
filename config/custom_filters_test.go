package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCustomFilters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantKeys []string
	}{
		{
			name:    "empty",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "empty filters",
			input:   "filters: {}",
			wantLen: 0,
		},
		{
			name: "single filter with keep/drop",
			input: `
filters:
  "mycli build":
    keep: ["ERROR", "WARN"]
    drop: ["DEBUG"]
`,
			wantLen:  1,
			wantKeys: []string{"mycli build"},
		},
		{
			name: "filter with head/tail",
			input: `
filters:
  terraform:
    head: 10
    tail: 5
`,
			wantLen:  1,
			wantKeys: []string{"terraform"},
		},
		{
			name: "filter with exec",
			input: `
filters:
  "ansible-playbook":
    exec: "~/.config/chop/scripts/ansible.sh"
`,
			wantLen:  1,
			wantKeys: []string{"ansible-playbook"},
		},
		{
			name: "multiple filters",
			input: `
filters:
  "mycli build":
    keep: ["ERROR"]
  "mycli test":
    drop: ["PASS"]
    tail: 20
  custom-tool:
    exec: "/usr/local/bin/filter.sh"
`,
			wantLen:  3,
			wantKeys: []string{"mycli build", "mycli test", "custom-tool"},
		},
		{
			name:    "invalid yaml",
			input:   "{{invalid",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCustomFilters([]byte(tt.input))

			if tt.wantLen == 0 {
				if len(result) != 0 {
					t.Fatalf("expected nil/empty, got %d filters", len(result))
				}
				return
			}

			if len(result) != tt.wantLen {
				t.Fatalf("expected %d filters, got %d", tt.wantLen, len(result))
			}

			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("missing expected key %q", key)
				}
			}
		})
	}
}

func TestParseCustomFilterFields(t *testing.T) {
	input := `
filters:
  "mycli build":
    keep: ["ERROR", "^BUILD"]
    drop: ["DEBUG", "^\\s*$"]
    head: 10
    tail: 5
`
	result := ParseCustomFilters([]byte(input))
	cf, ok := result["mycli build"]
	if !ok {
		t.Fatal("missing 'mycli build' filter")
	}

	if len(cf.Keep) != 2 || cf.Keep[0] != "ERROR" || cf.Keep[1] != "^BUILD" {
		t.Errorf("keep: got %v", cf.Keep)
	}
	if len(cf.Drop) != 2 || cf.Drop[0] != "DEBUG" {
		t.Errorf("drop: got %v", cf.Drop)
	}
	if cf.Head != 10 {
		t.Errorf("head: got %d", cf.Head)
	}
	if cf.Tail != 5 {
		t.Errorf("tail: got %d", cf.Tail)
	}
}

func TestLookupCustomFilter(t *testing.T) {
	filters := map[string]CustomFilter{
		"mycli build": {Keep: []string{"ERROR"}},
		"mycli":       {Drop: []string{"DEBUG"}},
		"terraform":   {Head: 10},
	}

	tests := []struct {
		name    string
		command string
		args    []string
		wantNil bool
		wantKey string // identify which filter we expect by checking a field
	}{
		{
			name:    "exact subcmd match",
			command: "mycli",
			args:    []string{"build", "--verbose"},
			wantKey: "keep",
		},
		{
			name:    "base command fallback",
			command: "mycli",
			args:    []string{"deploy"},
			wantKey: "drop",
		},
		{
			name:    "base command no args",
			command: "terraform",
			args:    nil,
			wantKey: "head",
		},
		{
			name:    "no match",
			command: "unknown",
			args:    nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LookupCustomFilter(filters, tt.command, tt.args)
			if tt.wantNil {
				if result != nil {
					t.Fatal("expected nil, got filter")
				}
				return
			}
			if result == nil {
				t.Fatal("expected filter, got nil")
			}

			switch tt.wantKey {
			case "keep":
				if len(result.Keep) == 0 {
					t.Error("expected keep rules")
				}
			case "drop":
				if len(result.Drop) == 0 {
					t.Error("expected drop rules")
				}
			case "head":
				if result.Head == 0 {
					t.Error("expected head > 0")
				}
			}
		})
	}
}

func TestLookupCustomFilterNilMap(t *testing.T) {
	result := LookupCustomFilter(nil, "anything", nil)
	if result != nil {
		t.Fatal("expected nil for nil map")
	}
}

func TestLoadCustomFiltersFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "filters.yml")

	content := `
filters:
  "mycli build":
    keep: ["ERROR"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadCustomFiltersFrom(path)
	if len(result) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(result))
	}
	if _, ok := result["mycli build"]; !ok {
		t.Error("missing 'mycli build' filter")
	}
}

func TestLoadCustomFiltersFromMissing(t *testing.T) {
	result := LoadCustomFiltersFrom("/nonexistent/path/filters.yml")
	if result != nil {
		t.Fatal("expected nil for missing file")
	}
}

func TestLoadCustomFiltersWithLocal(t *testing.T) {
	// Create a temp dir with a local filters file
	dir := t.TempDir()
	localPath := filepath.Join(dir, ".chop-filters.yml")

	content := `
filters:
  "local-tool":
    drop: ["noise"]
`
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadCustomFiltersWithLocal(dir)
	if _, ok := result["local-tool"]; !ok {
		t.Error("missing local filter 'local-tool'")
	}
}

func TestFiltersConfigPath(t *testing.T) {
	// Test with XDG_CONFIG_HOME
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	expected := filepath.Join("/tmp/xdg", "chop", "filters.yml")
	if p := FiltersConfigPath(); p != expected {
		t.Errorf("expected %s, got %s", expected, p)
	}

	// Test without XDG_CONFIG_HOME
	os.Unsetenv("XDG_CONFIG_HOME")
	p := FiltersConfigPath()
	if filepath.Base(p) != "filters.yml" {
		t.Errorf("expected filters.yml, got %s", filepath.Base(p))
	}
}

func TestLoadCustomFilters(t *testing.T) {
	// Setup a temporary XDG_CONFIG_HOME
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	// Create the expected config file path
	configPath := filepath.Join(tmpDir, "chop", "filters.yml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}

	content := `
filters:
  "mycli build":
    keep: ["ERROR"]
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadCustomFilters()
	if len(result) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(result))
	}
	if _, ok := result["mycli build"]; !ok {
		t.Error("missing 'mycli build' filter")
	}
}
