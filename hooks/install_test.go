package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writePluginHooksJSON creates a plugin hooks.json file at the given path with a
// Bash PreToolUse entry using the supplied command string.
func writePluginHooksJSON(t *testing.T, path string, bashCmd string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("failed to create plugin hooks dir: %v", err)
	}
	content := `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"` + bashCmd + `"}]}]}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write plugin hooks.json: %v", err)
	}
}

const testHookCmd = `"/usr/local/bin/chop" hook`

func tempSettingsPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, ".claude", "settings.json")
}

func readJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	return m
}

func writeJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
}

func TestInstallCreatesCorrectStructure(t *testing.T) {
	path := tempSettingsPath(t)

	if err := installWithCommand(path, testHookCmd); err != nil {
		t.Fatalf("installWithCommand failed: %v", err)
	}

	settings := readJSON(t, path)

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks not found or not an object")
	}

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		t.Fatal("PreToolUse not found or not an array")
	}

	if len(preToolUse) != 1 {
		t.Fatalf("expected 1 PreToolUse entry, got %d", len(preToolUse))
	}

	matcher := preToolUse[0].(map[string]interface{})
	if matcher["matcher"] != "Bash" {
		t.Errorf("expected matcher 'Bash', got %v", matcher["matcher"])
	}

	hooksArr := matcher["hooks"].([]interface{})
	if len(hooksArr) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooksArr))
	}

	hookEntry := hooksArr[0].(map[string]interface{})
	if hookEntry["type"] != "command" {
		t.Errorf("expected type 'command', got %v", hookEntry["type"])
	}

	cmd, _ := hookEntry["command"].(string)
	if cmd != testHookCmd {
		t.Errorf("expected command %q, got %q", testHookCmd, cmd)
	}
}

func TestInstallPreservesExistingSettings(t *testing.T) {
	path := tempSettingsPath(t)

	existing := map[string]interface{}{
		"apiKey":      "sk-test-123",
		"permissions": []interface{}{"allow_all"},
		"model":       "claude-opus-4-20250514",
	}
	writeJSON(t, path, existing)

	if err := installWithCommand(path, testHookCmd); err != nil {
		t.Fatalf("installWithCommand failed: %v", err)
	}

	settings := readJSON(t, path)

	if settings["apiKey"] != "sk-test-123" {
		t.Errorf("apiKey was lost, got %v", settings["apiKey"])
	}
	if settings["model"] != "claude-opus-4-20250514" {
		t.Errorf("model was lost, got %v", settings["model"])
	}
	perms, ok := settings["permissions"].([]interface{})
	if !ok || len(perms) != 1 || perms[0] != "allow_all" {
		t.Errorf("permissions were lost, got %v", settings["permissions"])
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks not added")
	}
	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok || len(preToolUse) == 0 {
		t.Fatal("PreToolUse not added")
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	path := tempSettingsPath(t)

	if err := installWithCommand(path, testHookCmd); err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	if err := installWithCommand(path, testHookCmd); err != nil {
		t.Fatalf("second install failed: %v", err)
	}

	settings := readJSON(t, path)
	hooks := settings["hooks"].(map[string]interface{})
	preToolUse := hooks["PreToolUse"].([]interface{})

	if len(preToolUse) != 1 {
		t.Fatalf("expected 1 PreToolUse entry after double install, got %d", len(preToolUse))
	}

	bashMatcher := preToolUse[0].(map[string]interface{})
	hooksArr := bashMatcher["hooks"].([]interface{})
	if len(hooksArr) != 1 {
		t.Fatalf("expected 1 hook after double install, got %d", len(hooksArr))
	}
}

func TestInstallPreservesOtherMatchers(t *testing.T) {
	path := tempSettingsPath(t)

	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Read",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "some-other-tool",
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, existing)

	if err := installWithCommand(path, testHookCmd); err != nil {
		t.Fatalf("installWithCommand failed: %v", err)
	}

	settings := readJSON(t, path)
	hooks := settings["hooks"].(map[string]interface{})
	preToolUse := hooks["PreToolUse"].([]interface{})

	if len(preToolUse) != 2 {
		t.Fatalf("expected 2 PreToolUse entries (Read + Bash), got %d", len(preToolUse))
	}

	found := false
	for _, entry := range preToolUse {
		m := entry.(map[string]interface{})
		if m["matcher"] == "Read" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Read matcher was lost")
	}
}

func TestInstallAddsToBashMatcherWithOtherHooks(t *testing.T) {
	path := tempSettingsPath(t)

	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "some-linter hook",
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, existing)

	if err := installWithCommand(path, testHookCmd); err != nil {
		t.Fatalf("installWithCommand failed: %v", err)
	}

	settings := readJSON(t, path)
	hooks := settings["hooks"].(map[string]interface{})
	preToolUse := hooks["PreToolUse"].([]interface{})
	bashMatcher := preToolUse[0].(map[string]interface{})
	hooksArr := bashMatcher["hooks"].([]interface{})

	if len(hooksArr) != 2 {
		t.Fatalf("expected 2 hooks in Bash matcher, got %d", len(hooksArr))
	}
}

func TestUninstallRemovesChopHook(t *testing.T) {
	path := tempSettingsPath(t)

	if err := installWithCommand(path, testHookCmd); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if err := uninstallFrom(path); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	settings := readJSON(t, path)

	if _, ok := settings["hooks"]; ok {
		t.Error("hooks key should have been removed when empty")
	}
}

func TestUninstallPreservesOtherHooks(t *testing.T) {
	path := tempSettingsPath(t)

	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "some-linter hook",
						},
						map[string]interface{}{
							"type":    "command",
							"command": `"/usr/local/bin/chop" hook`,
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, existing)

	if err := uninstallFrom(path); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	settings := readJSON(t, path)
	hooks := settings["hooks"].(map[string]interface{})
	preToolUse := hooks["PreToolUse"].([]interface{})

	if len(preToolUse) != 1 {
		t.Fatalf("expected 1 PreToolUse entry, got %d", len(preToolUse))
	}

	bashMatcher := preToolUse[0].(map[string]interface{})
	hooksArr := bashMatcher["hooks"].([]interface{})
	if len(hooksArr) != 1 {
		t.Fatalf("expected 1 hook after uninstall, got %d", len(hooksArr))
	}

	remaining := hooksArr[0].(map[string]interface{})
	if remaining["command"] != "some-linter hook" {
		t.Errorf("wrong hook preserved: %v", remaining["command"])
	}
}

func TestUninstallPreservesOtherMatchers(t *testing.T) {
	path := tempSettingsPath(t)

	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": `"/usr/local/bin/chop" hook`,
						},
					},
				},
				map[string]interface{}{
					"matcher": "Read",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "some-read-hook",
						},
					},
				},
			},
		},
	}
	writeJSON(t, path, existing)

	if err := uninstallFrom(path); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	settings := readJSON(t, path)
	hooks := settings["hooks"].(map[string]interface{})
	preToolUse := hooks["PreToolUse"].([]interface{})

	if len(preToolUse) != 1 {
		t.Fatalf("expected 1 PreToolUse entry (Read), got %d", len(preToolUse))
	}

	matcher := preToolUse[0].(map[string]interface{})
	if matcher["matcher"] != "Read" {
		t.Errorf("expected Read matcher to be preserved, got %v", matcher["matcher"])
	}
}

func TestUninstallNoSettingsFile(t *testing.T) {
	path := tempSettingsPath(t)

	if err := uninstallFrom(path); err != nil {
		t.Fatalf("uninstall on missing file failed: %v", err)
	}
}

func TestIsChopHook(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{`"/usr/local/bin/chop" hook`, true},
		{`"C:/Users/me/bin/chop" hook`, true},
		{`chop hook`, true},
		{`"/path/to/chop.exe" hook`, true},
		{"some-linter hook", false},
		{"other-command", false},
	}
	for _, tt := range tests {
		hookObj := map[string]interface{}{"command": tt.command}
		got := isChopHook(hookObj)
		if got != tt.want {
			t.Errorf("isChopHook(%q) = %v, want %v", tt.command, got, tt.want)
		}
	}
}

func TestBuildHookCommandFormat(t *testing.T) {
	cmd, err := buildHookCommand()
	if err != nil {
		t.Fatalf("buildHookCommand failed: %v", err)
	}
	if !strings.HasSuffix(cmd, " hook") {
		t.Errorf("hook command should end with ' hook', got %q", cmd)
	}
	if !strings.HasPrefix(cmd, `"`) {
		t.Errorf("hook command should start with quote, got %q", cmd)
	}
	if strings.Contains(cmd, `\`) {
		t.Errorf("hook command should not contain backslashes, got %q", cmd)
	}
}

func TestFindConflictingBashHooks_NoConflicts(t *testing.T) {
	home := t.TempDir()
	// Install only the chop hook — no conflicts expected.
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := installWithCommand(settingsPath, testHookCmd); err != nil {
		t.Fatalf("installWithCommand failed: %v", err)
	}

	conflicts, err := findConflictingBashHooksIn(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conflicts.HasConflict() {
		t.Errorf("expected no conflicts, got settings=%v plugins=%v",
			conflicts.SettingsConflicts, conflicts.PluginConflicts)
	}
}

func TestFindConflictingBashHooks_WrapperScriptNoConflict(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	// User-created chop wrapper (e.g. chop-verify.sh) — should not be reported as a conflict.
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "bash ~/.claude/hooks/chop-verify.sh"},
					},
				},
			},
		},
	}
	writeJSON(t, settingsPath, settings)

	conflicts, err := findConflictingBashHooksIn(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conflicts.HasConflict() {
		t.Errorf("expected no conflicts for chop wrapper script, got %+v", conflicts)
	}
}

func TestFindConflictingBashHooks_SettingsConflict(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	// Write settings.json with chop AND a second competing hook in the same Bash matcher.
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": testHookCmd},
						map[string]interface{}{"type": "command", "command": "bash ~/.claude/hooks/verify-branch.sh"},
					},
				},
			},
		},
	}
	writeJSON(t, settingsPath, settings)

	conflicts, err := findConflictingBashHooksIn(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts.SettingsConflicts) != 1 {
		t.Fatalf("expected 1 settings conflict, got %d: %v", len(conflicts.SettingsConflicts), conflicts.SettingsConflicts)
	}
	if conflicts.SettingsConflicts[0] != "bash ~/.claude/hooks/verify-branch.sh" {
		t.Errorf("unexpected conflict command: %q", conflicts.SettingsConflicts[0])
	}
	if len(conflicts.PluginConflicts) != 0 {
		t.Errorf("expected no plugin conflicts, got %v", conflicts.PluginConflicts)
	}
}

func TestFindConflictingBashHooks_PluginConflict(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := installWithCommand(settingsPath, testHookCmd); err != nil {
		t.Fatalf("installWithCommand failed: %v", err)
	}

	// Simulate a plugin with a Bash PreToolUse hook.
	pluginHooksPath := filepath.Join(home, ".claude", "plugins", "marketplaces",
		"some-marketplace", "plugins", "some-plugin", "hooks", "hooks.json")
	writePluginHooksJSON(t, pluginHooksPath, "/path/to/competing-hook.sh")

	conflicts, err := findConflictingBashHooksIn(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts.PluginConflicts) != 1 {
		t.Fatalf("expected 1 plugin conflict, got %d: %v", len(conflicts.PluginConflicts), conflicts.PluginConflicts)
	}
	if !strings.HasSuffix(conflicts.PluginConflicts[0], "hooks.json") {
		t.Errorf("expected plugin conflict to be a hooks.json path, got %q", conflicts.PluginConflicts[0])
	}
	if len(conflicts.SettingsConflicts) != 0 {
		t.Errorf("expected no settings conflicts, got %v", conflicts.SettingsConflicts)
	}
}

func TestFindConflictingBashHooks_EmptyPluginPreToolUse(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := installWithCommand(settingsPath, testHookCmd); err != nil {
		t.Fatalf("installWithCommand failed: %v", err)
	}

	// Plugin with PreToolUse:[] — the fixed state; should not be reported as a conflict.
	pluginHooksPath := filepath.Join(home, ".claude", "plugins", "cache",
		"some-marketplace", "some-plugin", "hash1234", "hooks", "hooks.json")
	if err := os.MkdirAll(filepath.Dir(pluginHooksPath), 0o700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(pluginHooksPath,
		[]byte(`{"hooks":{"PreToolUse":[]}}`), 0o600); err != nil {
		t.Fatalf("failed to write plugin hooks.json: %v", err)
	}

	conflicts, err := findConflictingBashHooksIn(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conflicts.HasConflict() {
		t.Errorf("expected no conflicts for empty PreToolUse, got %+v", conflicts)
	}
}
