package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFrom_MissingFile(t *testing.T) {
	cfg := LoadFrom("/nonexistent/path/config.yml")
	if len(cfg.Disabled) != 0 {
		t.Error("expected empty Disabled for missing file")
	}
}

func TestLoadFrom_EmptyFile(t *testing.T) {
	tmp := writeTemp(t, "")
	cfg := LoadFrom(tmp)
	if len(cfg.Disabled) != 0 {
		t.Error("expected empty Disabled for empty file")
	}
}

func TestLoadFrom_CommentsOnly(t *testing.T) {
	content := "# this is a comment\n# another comment\n"
	tmp := writeTemp(t, content)
	cfg := LoadFrom(tmp)
	if len(cfg.Disabled) != 0 {
		t.Error("expected empty Disabled for comments-only file")
	}
}

func TestLoadFrom_DisabledList(t *testing.T) {
	content := "disabled: [git, docker]\n"
	tmp := writeTemp(t, content)
	cfg := LoadFrom(tmp)
	if len(cfg.Disabled) != 2 {
		t.Fatalf("expected 2 disabled items, got %d", len(cfg.Disabled))
	}
	if cfg.Disabled[0] != "git" {
		t.Errorf("expected 'git', got %q", cfg.Disabled[0])
	}
	if cfg.Disabled[1] != "docker" {
		t.Errorf("expected 'docker', got %q", cfg.Disabled[1])
	}
}

func TestLoadFrom_DisabledEmpty(t *testing.T) {
	content := "disabled: []\n"
	tmp := writeTemp(t, content)
	cfg := LoadFrom(tmp)
	if len(cfg.Disabled) != 0 {
		t.Errorf("expected empty disabled list, got %d", len(cfg.Disabled))
	}
}

func TestLoadFrom_DisabledWithQuotes(t *testing.T) {
	content := `disabled: ["git", "docker"]` + "\n"
	tmp := writeTemp(t, content)
	cfg := LoadFrom(tmp)
	if len(cfg.Disabled) != 2 {
		t.Fatalf("expected 2 disabled items, got %d", len(cfg.Disabled))
	}
	if cfg.Disabled[0] != "git" {
		t.Errorf("expected 'git', got %q", cfg.Disabled[0])
	}
}

func TestLoadFrom_FullConfig(t *testing.T) {
	content := "# chop config\ndisabled: [git, docker, kubectl]\n"
	tmp := writeTemp(t, content)
	cfg := LoadFrom(tmp)
	if len(cfg.Disabled) != 3 {
		t.Fatalf("expected 3 disabled items, got %d", len(cfg.Disabled))
	}
}

func TestLoadFrom_InlineComments(t *testing.T) {
	content := "disabled: [git] # skip git\n"
	tmp := writeTemp(t, content)
	cfg := LoadFrom(tmp)
	if len(cfg.Disabled) != 1 || cfg.Disabled[0] != "git" {
		t.Errorf("expected [git], got %v", cfg.Disabled)
	}
}

func TestIsDisabled(t *testing.T) {
	cfg := Config{Disabled: []string{"git", "docker"}}

	if !cfg.IsDisabled("git") {
		t.Error("expected git to be disabled")
	}
	if !cfg.IsDisabled("Git") {
		t.Error("expected Git (case-insensitive) to be disabled")
	}
	if !cfg.IsDisabled("docker") {
		t.Error("expected docker to be disabled")
	}
	if cfg.IsDisabled("npm") {
		t.Error("expected npm to NOT be disabled")
	}
}

func TestIsDisabled_Subcommand(t *testing.T) {
	cfg := Config{Disabled: []string{"git diff", "docker ps"}}

	// Exact subcommand match
	if !cfg.IsDisabled("git", "diff") {
		t.Error("expected 'git diff' to be disabled")
	}
	if !cfg.IsDisabled("docker", "ps") {
		t.Error("expected 'docker ps' to be disabled")
	}

	// Different subcommand - not disabled
	if cfg.IsDisabled("git", "status") {
		t.Error("expected 'git status' to NOT be disabled")
	}

	// Base command alone - not disabled (only subcommand-level entries)
	if cfg.IsDisabled("git") {
		t.Error("expected bare 'git' to NOT be disabled when only 'git diff' is listed")
	}
}

func TestIsDisabled_BaseMatchesAllSubcommands(t *testing.T) {
	cfg := Config{Disabled: []string{"git"}}

	// Base "git" disables all git subcommands
	if !cfg.IsDisabled("git", "diff") {
		t.Error("expected 'git diff' disabled when 'git' is in disabled list")
	}
	if !cfg.IsDisabled("git", "status") {
		t.Error("expected 'git status' disabled when 'git' is in disabled list")
	}
	if !cfg.IsDisabled("git") {
		t.Error("expected bare 'git' disabled")
	}
}

func TestIsDisabled_Empty(t *testing.T) {
	cfg := Config{}
	if cfg.IsDisabled("git") {
		t.Error("expected nothing disabled on empty config")
	}
}

func TestLoadWithLocal_NoLocalFile(t *testing.T) {
	// Set up global config
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "config.yml")
	os.WriteFile(globalPath, []byte("disabled: [git, docker]\n"), 0o644)

	cfg := LoadFrom(globalPath)
	if len(cfg.Disabled) != 2 {
		t.Fatalf("expected 2 disabled, got %d", len(cfg.Disabled))
	}
}

func TestLoadWithLocal_OverridesGlobal(t *testing.T) {
	// Create a project dir with .chop.yml
	projectDir := t.TempDir()
	localPath := filepath.Join(projectDir, ".chop.yml")
	os.WriteFile(localPath, []byte(`disabled: ["git diff"]`+"\n"), 0o644)

	// LoadFrom for the local file
	local := LoadFrom(localPath)
	if len(local.Disabled) != 1 {
		t.Fatalf("expected 1 disabled, got %d", len(local.Disabled))
	}
	if local.Disabled[0] != "git diff" {
		t.Errorf("expected 'git diff', got %q", local.Disabled[0])
	}
}

func TestLoadFrom_SubcommandQuoted(t *testing.T) {
	content := `disabled: ["git diff", "git show"]` + "\n"
	tmp := writeTemp(t, content)
	cfg := LoadFrom(tmp)
	if len(cfg.Disabled) != 2 {
		t.Fatalf("expected 2, got %d", len(cfg.Disabled))
	}
	if cfg.Disabled[0] != "git diff" {
		t.Errorf("expected 'git diff', got %q", cfg.Disabled[0])
	}
	if cfg.Disabled[1] != "git show" {
		t.Errorf("expected 'git show', got %q", cfg.Disabled[1])
	}
}

func TestPath_Default(t *testing.T) {
	old := os.Getenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", old)

	p := Path()
	if filepath.Base(p) != "config.yml" {
		t.Errorf("expected config.yml, got %s", filepath.Base(p))
	}
}

func TestPath_XDG(t *testing.T) {
	old := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	defer os.Setenv("XDG_CONFIG_HOME", old)

	p := Path()
	expected := filepath.Join("/tmp/xdg", "chop", "config.yml")
	if p != expected {
		t.Errorf("expected %s, got %s", expected, p)
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
