package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- indexOutsideQuotes ----

func TestIndexOutsideQuotes(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		needle string
		want   int
	}{
		{"found outside quotes", "git add . && git commit", " && ", 9},
		{"found at start", " && rest", " && ", 0},
		{"not found", "git status", " && ", -1},
		{"inside double quotes", `grep "foo && bar" file`, " && ", -1},
		{"inside single quotes", `grep 'foo && bar' file`, " && ", -1},
		{"after closing double quote", `grep "foo" && bar`, " && ", 10},
		{"after closing single quote", `grep 'foo' && bar`, " && ", 10},
		{"escaped double quote inside string", `grep "say \" && end" file`, " && ", -1},
		{"real op after escaped quote string", `grep "say \" end" file && echo ok`, " && ", 22},
		{"empty string", "", " && ", -1},
		{"needle longer than string", "ab", "abcd", -1},
		{"pipe outside quotes", "git log | head", " | ", 7},
		{"pipe inside double quotes", `echo "a | b"`, " | ", -1},
		{"semicolon inside single quotes", `echo 'a ; b'`, " ; ", -1},
		{"semicolon outside quotes", "cd /tmp ; ls", " ; ", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indexOutsideQuotes(tt.s, tt.needle)
			if got != tt.want {
				t.Errorf("indexOutsideQuotes(%q, %q) = %d, want %d", tt.s, tt.needle, got, tt.want)
			}
		})
	}
}

// ---- containsOutsideQuotes ----

func TestContainsOutsideQuotes(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		needle string
		want   bool
	}{
		{"found outside", "git add . && git commit", " && ", true},
		{"inside double quotes", `grep "foo && bar" file`, " && ", false},
		{"inside single quotes", `grep 'foo && bar' file`, " && ", false},
		{"not present", "git status", " && ", false},
		{"pipe inside quotes", `echo "a | b"`, " | ", false},
		{"pipe outside quotes", "git log | head", " | ", true},
		{"redirect inside single quote", `grep '$1 > 0' data`, " > ", false},
		{"redirect outside quotes", "git diff > out.txt", " > ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsOutsideQuotes(tt.s, tt.needle)
			if got != tt.want {
				t.Errorf("containsOutsideQuotes(%q, %q) = %v, want %v", tt.s, tt.needle, got, tt.want)
			}
		})
	}
}

// ---- buildOutput ----

func TestBuildOutput(t *testing.T) {
	original := "git status"
	wrapped := "chop git status"

	data, modified, orig := buildOutput(original, wrapped)

	if !modified {
		t.Fatal("buildOutput should return modified=true")
	}
	if orig != original {
		t.Errorf("buildOutput original = %q, want %q", orig, original)
	}

	var out hookOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("buildOutput produced invalid JSON: %v", err)
	}

	if out.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want PreToolUse", out.HookSpecificOutput.HookEventName)
	}
	if out.HookSpecificOutput.PermissionDecision != "" {
		t.Errorf("PermissionDecision = %q, want empty (no override)", out.HookSpecificOutput.PermissionDecision)
	}
	if out.HookSpecificOutput.UpdatedInput.Command != wrapped {
		t.Errorf("UpdatedInput.Command = %q, want %q", out.HookSpecificOutput.UpdatedInput.Command, wrapped)
	}
}

// ---- processHookInput ----

func TestProcessHookInputInvalidToolInput(t *testing.T) {
	// tool_input is not a valid toolInput JSON
	input := map[string]interface{}{
		"session_id":      "s",
		"cwd":             "/tmp",
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
		"tool_input":      "not-an-object",
	}
	data, _ := json.Marshal(input)
	_, shouldModify, _ := processHookInput(data)
	if shouldModify {
		t.Error("should not modify when tool_input is invalid")
	}
}

func TestProcessHookInputNonBashTool(t *testing.T) {
	input := map[string]interface{}{
		"session_id":      "s",
		"cwd":             "/tmp",
		"hook_event_name": "PreToolUse",
		"tool_name":       "Read",
		"tool_input":      map[string]string{"file_path": "/some/file"},
	}
	data, _ := json.Marshal(input)
	_, shouldModify, _ := processHookInput(data)
	if shouldModify {
		t.Error("should not modify non-Bash tool")
	}
}

func TestProcessHookInputWhenDisabled(t *testing.T) {
	// Redirect HOME to a temp dir so Disable/Enable don't touch real files
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := Disable(); err != nil {
		t.Fatalf("Disable() failed: %v", err)
	}
	defer Enable() //nolint

	_, shouldModify, _ := processHookInput(makeInput("git status"))
	if shouldModify {
		t.Error("should not modify when globally disabled")
	}
}

// ---- IsDisabledGlobally / Disable / Enable ----

func TestDisableEnableCycle(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Initially not disabled
	if IsDisabledGlobally() {
		t.Fatal("should not be disabled initially")
	}

	// Disable
	if err := Disable(); err != nil {
		t.Fatalf("Disable() error: %v", err)
	}
	if !IsDisabledGlobally() {
		t.Fatal("should be disabled after Disable()")
	}

	// Verify flag file exists
	flagPath := filepath.Join(tmp, ".local", "share", "chop", "disabled")
	if _, err := os.Stat(flagPath); err != nil {
		t.Fatalf("flag file not created: %v", err)
	}

	// Enable
	if err := Enable(); err != nil {
		t.Fatalf("Enable() error: %v", err)
	}
	if IsDisabledGlobally() {
		t.Fatal("should not be disabled after Enable()")
	}
}

func TestDisableIsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := Disable(); err != nil {
		t.Fatalf("first Disable() error: %v", err)
	}
	if err := Disable(); err != nil {
		t.Fatalf("second Disable() error: %v", err)
	}
	if !IsDisabledGlobally() {
		t.Fatal("should still be disabled")
	}
}

func TestEnableWhenNotDisabled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Enable without prior Disable — should be a no-op
	if err := Enable(); err != nil {
		t.Fatalf("Enable() on non-existent flag returned error: %v", err)
	}
}

// ---- readSettings ----

func TestReadSettingsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	settings, err := readSettings(path)
	if err != nil {
		t.Fatalf("readSettings on missing file should return empty map, got error: %v", err)
	}
	if settings == nil {
		t.Fatal("expected empty map, got nil")
	}
	if len(settings) != 0 {
		t.Errorf("expected empty map, got %v", settings)
	}
}

func TestReadSettingsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("   "), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := readSettings(path)
	if err != nil {
		t.Fatalf("readSettings on empty file error: %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("expected empty map, got %v", settings)
	}
}

func TestReadSettingsValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	content := `{"apiKey": "sk-123", "model": "claude-opus"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := readSettings(path)
	if err != nil {
		t.Fatalf("readSettings error: %v", err)
	}
	if settings["apiKey"] != "sk-123" {
		t.Errorf("apiKey = %v, want sk-123", settings["apiKey"])
	}
	if settings["model"] != "claude-opus" {
		t.Errorf("model = %v, want claude-opus", settings["model"])
	}
}

func TestReadSettingsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := readSettings(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---- writeSettings ----

func TestWriteSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "settings.json")

	settings := map[string]interface{}{
		"apiKey": "test-key",
		"count":  float64(42),
	}
	if err := writeSettings(path, settings); err != nil {
		t.Fatalf("writeSettings error: %v", err)
	}

	// File should exist with correct content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if !strings.Contains(string(data), "test-key") {
		t.Errorf("written file doesn't contain expected key: %s", data)
	}

	// Should end with newline
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("written file should end with newline")
	}

	// Round-trip: read back what we wrote
	read, err := readSettings(path)
	if err != nil {
		t.Fatalf("readSettings after write error: %v", err)
	}
	if read["apiKey"] != "test-key" {
		t.Errorf("round-trip apiKey = %v, want test-key", read["apiKey"])
	}
}

func TestWriteSettingsCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Nested directory that doesn't exist yet
	path := filepath.Join(dir, "a", "b", "c", "settings.json")
	if err := writeSettings(path, map[string]interface{}{"x": 1}); err != nil {
		t.Fatalf("writeSettings should create parent dirs: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

// ---- isChopHook ----

func TestIsChopHookAdditionalCases(t *testing.T) {
	tests := []struct {
		name    string
		hookObj map[string]interface{}
		want    bool
	}{
		{
			name:    "no command key",
			hookObj: map[string]interface{}{"type": "command"},
			want:    false,
		},
		{
			name:    "command is not a string",
			hookObj: map[string]interface{}{"command": 42},
			want:    false,
		},
		{
			name:    "contains chop but does not end with hook",
			hookObj: map[string]interface{}{"command": "/path/to/chop run"},
			want:    false,
		},
		{
			name:    "ends with hook but no chop",
			hookObj: map[string]interface{}{"command": "some-tool hook"},
			want:    false,
		},
		{
			name:    "valid chop hook with full path",
			hookObj: map[string]interface{}{"command": `"/home/user/bin/chop" hook`},
			want:    true,
		},
		{
			name:    "bare chop hook",
			hookObj: map[string]interface{}{"command": "chop hook"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isChopHook(tt.hookObj)
			if got != tt.want {
				t.Errorf("isChopHook(%v) = %v, want %v", tt.hookObj, got, tt.want)
			}
		})
	}
}

// ---- GetHookCommand ----

func TestGetHookCommandWhenInstalled(t *testing.T) {
	path := tempSettingsPath(t)
	if err := installWithCommand(path, testHookCmd); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// GetHookCommand reads from the real home dir — we redirect via HOME
	// so it won't find anything, but we can test the underlying data structure
	// by calling readSettings directly and walking the structure.
	settings, err := readSettings(path)
	if err != nil {
		t.Fatalf("readSettings failed: %v", err)
	}

	hooksMap := settings["hooks"].(map[string]interface{})
	preToolUse := hooksMap["PreToolUse"].([]interface{})
	bashMatcher := preToolUse[0].(map[string]interface{})
	hooksArr := bashMatcher["hooks"].([]interface{})
	hookEntry := hooksArr[0].(map[string]interface{})

	cmd, _ := hookEntry["command"].(string)
	if cmd != testHookCmd {
		t.Errorf("stored hook command = %q, want %q", cmd, testHookCmd)
	}
}

// ---- shouldWrap edge cases ----

func TestShouldWrapEdgeCases(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"chop git status", false},      // already chopped
		{". ~/.bashrc", false},           // source shorthand
		{"cd /tmp", false},               // shell builtin
		{"export FOO=bar", false},        // shell builtin
		{"git status", true},             // supported
		{"npm install", true},            // supported
		{"vim file.txt", false},          // unsupported
		{"ls -la", false},                // unsupported
		{`npm.exe install`, true},        // .exe suffix stripped
		{"/usr/bin/git status", true},    // absolute path
		{`"git" status`, true},           // quoted command name
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := shouldWrap(tt.cmd)
			if got != tt.want {
				t.Errorf("shouldWrap(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

// ---- rewriteCommand edge cases ----

func TestRewriteCommandEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		wantModified bool
		wantWrapped  string
	}{
		{"empty", "", false, ""},
		{"already chopped", "chop git status", false, ""},
		{"source shorthand", ". ~/.bashrc", false, ""},
		{"pipe passthrough", "git log | head -5", false, ""},
		{"redirect passthrough", "git diff > out.txt", false, ""},
		{"supported single", "git status", true, "chop git status"},
		{"unsupported single", "vim file.txt", false, ""},
		{"compound with supported", "git add . && git commit", true, "chop git add . && chop git commit"},
		{"compound all unsupported", "cd /tmp && ls", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped, modified, _ := rewriteCommand(tt.cmd)
			if modified != tt.wantModified {
				t.Errorf("rewriteCommand(%q) modified=%v, want %v", tt.cmd, modified, tt.wantModified)
			}
			if tt.wantModified && wrapped != tt.wantWrapped {
				t.Errorf("rewriteCommand(%q) wrapped=%q, want %q", tt.cmd, wrapped, tt.wantWrapped)
			}
		})
	}
}
