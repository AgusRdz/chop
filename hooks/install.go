package hooks

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/AgusRdz/chop/config"
)

// chopHookIdentifier is used to detect chop hook entries in settings.json.
// We look for commands containing "chop" that end with " hook".
const chopBinaryName = "chop"

// Install registers the chop hook in ~/.claude/settings.json.
func Install(version string) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to get home directory: %v\n", err)
		os.Exit(1)
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := installTo(settingsPath); err != nil {
		fmt.Fprintf(os.Stderr, "chop: %v\n", err)
		os.Exit(1)
	}
	_ = config.WriteDiscoveryInfo(version)
	fmt.Printf("chop hook installed in %s\n", settingsPath)

	// Warn if competing Bash PreToolUse hooks exist — they will silently disable compression.
	if conflicts, err := FindConflictingBashHooks(); err == nil && conflicts.HasConflict() {
		fmt.Println()
		fmt.Println("WARNING: competing Bash PreToolUse hooks detected.")
		fmt.Println("Claude Code will silently drop chop's output due to a known bug:")
		fmt.Println("  https://github.com/anthropics/claude-code/issues/15897")
		if len(conflicts.SettingsConflicts) > 0 {
			fmt.Println("Conflicts in ~/.claude/settings.json:")
			for _, cmd := range conflicts.SettingsConflicts {
				fmt.Printf("  - %s\n", cmd)
			}
		}
		if len(conflicts.PluginConflicts) > 0 {
			fmt.Println("Conflicts from plugins:")
			for _, path := range conflicts.PluginConflicts {
				fmt.Printf("  - %s\n", path)
			}
		}
		fmt.Println("Run `chop fix-hooks` to generate a combined wrapper script automatically.")
	}

	binPath, _ := chopBinaryPath()
	fmt.Printf("\nInstallation complete! Please tell your Claude Code: 'Remember that chop is installed at %s and use it for CLI compression.' This will prevent the agent from searching for it in the future.\n", binPath)
}

// Uninstall removes the chop hook from ~/.claude/settings.json.
func Uninstall() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to get home directory: %v\n", err)
		os.Exit(1)
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := uninstallFrom(settingsPath); err != nil {
		fmt.Fprintf(os.Stderr, "chop: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("chop hook removed from %s\n", settingsPath)
}

// IsInstalled checks whether the chop hook is registered in ~/.claude/settings.json.
func IsInstalled() (bool, string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, ""
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settings, err := readSettings(settingsPath)
	if err != nil {
		return false, settingsPath
	}

	hooksRaw, ok := settings["hooks"]
	if !ok {
		return false, settingsPath
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return false, settingsPath
	}

	preToolUseRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		return false, settingsPath
	}
	preToolUse, ok := preToolUseRaw.([]interface{})
	if !ok {
		return false, settingsPath
	}

	for _, entry := range preToolUse {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if matcher, _ := m["matcher"].(string); matcher != "Bash" {
			continue
		}
		hooksArrayRaw, ok := m["hooks"]
		if !ok {
			continue
		}
		hooksArray, ok := hooksArrayRaw.([]interface{})
		if !ok {
			continue
		}
		for _, h := range hooksArray {
			hMap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if isChopHook(hMap) {
				return true, settingsPath
			}
		}
	}

	return false, settingsPath
}

// GetHookCommand returns the command string currently registered in the hook, or "" if not found.
func GetHookCommand() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settings, err := readSettings(settingsPath)
	if err != nil {
		return ""
	}

	hooksRaw, ok := settings["hooks"]
	if !ok {
		return ""
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return ""
	}
	preToolUseRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		return ""
	}
	preToolUse, ok := preToolUseRaw.([]interface{})
	if !ok {
		return ""
	}

	for _, entry := range preToolUse {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if matcher, _ := m["matcher"].(string); matcher != "Bash" {
			continue
		}
		hooksArrayRaw, ok := m["hooks"]
		if !ok {
			continue
		}
		hooksArray, ok := hooksArrayRaw.([]interface{})
		if !ok {
			continue
		}
		for _, h := range hooksArray {
			hMap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if isChopHook(hMap) {
				cmd, _ := hMap["command"].(string)
				return cmd
			}
		}
	}
	return ""
}

func chopBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}
	// Convert backslashes to forward slashes for Claude Code compatibility
	return strings.ReplaceAll(exe, "\\", "/"), nil
}

func buildHookCommand() (string, error) {
	binPath, err := chopBinaryPath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%s" hook`, binPath), nil
}

func readSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return make(map[string]interface{}), nil
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

// isChopHook returns true for direct chop binary invocations: `"<path>/chop" hook`.
// Used by IsInstalled, GetHookCommand, install, and uninstall — must stay strict.
func isChopHook(hookObj map[string]interface{}) bool {
	cmd, ok := hookObj["command"].(string)
	if !ok {
		return false
	}
	return strings.Contains(cmd, chopBinaryName) && strings.HasSuffix(cmd, " hook")
}

// isChopAwareHook returns true if the hook is either a direct chop invocation or a
// wrapper script whose filename contains "chop" (e.g. chop-verify.sh).
// Used only by conflict detection to avoid false positives on user-created wrappers.
func isChopAwareHook(hookObj map[string]interface{}) bool {
	if isChopHook(hookObj) {
		return true
	}
	cmd, ok := hookObj["command"].(string)
	if !ok {
		return false
	}
	fields := strings.Fields(cmd)
	if len(fields) >= 1 {
		script := filepath.Base(fields[len(fields)-1])
		return strings.Contains(strings.ToLower(script), chopBinaryName)
	}
	return false
}

// ConflictingBashHooks describes hooks that compete with chop's updatedInput output.
// When multiple Bash PreToolUse hooks are active, Claude Code silently drops updatedInput
// from all of them (https://github.com/anthropics/claude-code/issues/15897).
type ConflictingBashHooks struct {
	// SettingsConflicts are non-chop hook commands found in the Bash PreToolUse
	// section of ~/.claude/settings.json.
	SettingsConflicts []string
	// PluginConflicts are plugin hooks.json file paths that declare a Bash
	// PreToolUse hook, adding a competing hook matcher at the Claude Code level.
	PluginConflicts []string
}

// HasConflict returns true if any conflicting hooks were found.
func (c ConflictingBashHooks) HasConflict() bool {
	return len(c.SettingsConflicts) > 0 || len(c.PluginConflicts) > 0
}

// HasChopAwareHook returns true if a Bash PreToolUse hook that is either a direct
// chop invocation or a chop wrapper script is registered in settings.json.
// Unlike IsInstalled, this does not require the strict `"<binary>" hook` form.
func HasChopAwareHook() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settings, err := readSettings(settingsPath)
	if err != nil {
		return false
	}
	hooksRaw, ok := settings["hooks"]
	if !ok {
		return false
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return false
	}
	ptuRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		return false
	}
	ptu, ok := ptuRaw.([]interface{})
	if !ok {
		return false
	}
	for _, entry := range ptu {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if matcher, _ := m["matcher"].(string); matcher != "Bash" {
			continue
		}
		hooksArrayRaw, ok := m["hooks"]
		if !ok {
			continue
		}
		hooksArray, ok := hooksArrayRaw.([]interface{})
		if !ok {
			continue
		}
		for _, h := range hooksArray {
			hMap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if isChopAwareHook(hMap) {
				return true
			}
		}
	}
	return false
}

// FindConflictingBashHooks scans settings.json and plugin hooks.json files for
// Bash PreToolUse hooks that would compete with chop's updatedInput output.
func FindConflictingBashHooks() (ConflictingBashHooks, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return ConflictingBashHooks{}, fmt.Errorf("failed to get home directory: %w", err)
	}
	return findConflictingBashHooksIn(home)
}

// findConflictingBashHooksIn is the testable core of FindConflictingBashHooks.
func findConflictingBashHooksIn(home string) (ConflictingBashHooks, error) {
	result := ConflictingBashHooks{}

	// --- scan settings.json for non-chop Bash PreToolUse hooks ---
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settings, err := readSettings(settingsPath)
	if err == nil {
		if hooksRaw, ok := settings["hooks"]; ok {
			if hooksMap, ok := hooksRaw.(map[string]interface{}); ok {
				if ptuRaw, ok := hooksMap["PreToolUse"]; ok {
					if ptu, ok := ptuRaw.([]interface{}); ok {
						for _, entry := range ptu {
							m, ok := entry.(map[string]interface{})
							if !ok {
								continue
							}
							if matcher, _ := m["matcher"].(string); matcher != "Bash" {
								continue
							}
							hooksArrayRaw, ok := m["hooks"]
							if !ok {
								continue
							}
							hooksArray, ok := hooksArrayRaw.([]interface{})
							if !ok {
								continue
							}
							for _, h := range hooksArray {
								hMap, ok := h.(map[string]interface{})
								if !ok {
									continue
								}
								if isChopAwareHook(hMap) {
									continue // chop or a chop wrapper — not a conflict
								}
								if cmd, ok := hMap["command"].(string); ok && cmd != "" {
									result.SettingsConflicts = append(result.SettingsConflicts, cmd)
								}
							}
						}
					}
				}
			}
		}
	}

	// --- scan plugin hooks.json files for Bash PreToolUse entries ---
	pluginsDir := filepath.Join(home, ".claude", "plugins")
	if _, err := os.Stat(pluginsDir); err == nil {
		_ = filepath.WalkDir(pluginsDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			// Only care about hooks/hooks.json files inside the plugins tree
			if d.Name() != "hooks.json" {
				return nil
			}
			if filepath.Base(filepath.Dir(path)) != "hooks" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var pluginHooks struct {
				Hooks struct {
					PreToolUse []struct {
						Matcher string `json:"matcher"`
						Hooks   []struct {
							Command string `json:"command"`
						} `json:"hooks"`
					} `json:"PreToolUse"`
				} `json:"hooks"`
			}
			if json.Unmarshal(data, &pluginHooks) != nil {
				return nil
			}
			for _, entry := range pluginHooks.Hooks.PreToolUse {
				if entry.Matcher != "Bash" {
					continue
				}
				for _, h := range entry.Hooks {
					if h.Command != "" {
						result.PluginConflicts = append(result.PluginConflicts, path)
						return nil // one report per file is enough
					}
				}
			}
			return nil
		})
	}

	return result, nil
}

// WrapperScriptPath returns the canonical path for the chop-generated wrapper script.
func WrapperScriptPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "hooks", "chop-wrapper.sh"), nil
}

// GenerateConflictFixScript writes a combined Bash PreToolUse wrapper script to
// ~/.claude/hooks/chop-wrapper.sh. The script runs each competing hook first (forwarding
// any denial), then invokes chop for command rewriting. Only settings.json conflicts can
// be auto-fixed; plugin conflicts require manual intervention.
func GenerateConflictFixScript(conflicts ConflictingBashHooks, chopBinPath string) (string, error) {
	scriptPath, err := WrapperScriptPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o700); err != nil {
		return "", fmt.Errorf("failed to create hooks directory: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("# Combined Bash PreToolUse hook — generated by chop fix-hooks\n")
	sb.WriteString("# Workaround for: https://github.com/anthropics/claude-code/issues/15897\n")
	sb.WriteString("INPUT=$(cat)\n\n")
	sb.WriteString("_run_hook() {\n")
	sb.WriteString("  local out\n")
	sb.WriteString("  out=$(printf '%s' \"$INPUT\" | eval \"$1\" 2>/dev/null) || return 0\n")
	sb.WriteString("  if printf '%s' \"$out\" | grep -q '\"permissionDecision\":\"deny\"'; then\n")
	sb.WriteString("    printf '%s\\n' \"$out\"\n")
	sb.WriteString("    exit 0\n")
	sb.WriteString("  fi\n")
	sb.WriteString("}\n\n")

	if len(conflicts.SettingsConflicts) > 0 {
		sb.WriteString("# Competing hooks (run before chop)\n")
		for _, cmd := range conflicts.SettingsConflicts {
			sb.WriteString(fmt.Sprintf("_run_hook %s\n", shellQuote(cmd)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("# chop: rewrite supported commands for compression\n")
	sb.WriteString(fmt.Sprintf("printf '%%s' \"$INPUT\" | %s hook\n", shellQuote(chopBinPath)))

	if err := os.WriteFile(scriptPath, []byte(sb.String()), 0o755); err != nil {
		return "", fmt.Errorf("failed to write wrapper script: %w", err)
	}
	return scriptPath, nil
}

// ApplyConflictFix replaces all Bash PreToolUse hooks in settings.json with a single
// wrapper hook pointing to wrapperPath.
func ApplyConflictFix(wrapperPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	hooksRaw, ok := settings["hooks"]
	if !ok {
		hooksRaw = make(map[string]interface{})
		settings["hooks"] = hooksRaw
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("hooks field is not an object in %s", settingsPath)
	}

	ptuRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		ptuRaw = []interface{}{}
	}
	ptu, ok := ptuRaw.([]interface{})
	if !ok {
		return fmt.Errorf("hooks.PreToolUse is not an array in %s", settingsPath)
	}

	// Convert path to forward slashes for shell compatibility
	wrapperCmd := "bash " + strings.ReplaceAll(wrapperPath, "\\", "/")

	// Replace all Bash matcher entries with a single wrapper entry
	newPTU := make([]interface{}, 0, len(ptu))
	bashReplaced := false
	for _, entry := range ptu {
		m, ok := entry.(map[string]interface{})
		if !ok {
			newPTU = append(newPTU, entry)
			continue
		}
		if matcher, _ := m["matcher"].(string); matcher != "Bash" {
			newPTU = append(newPTU, entry)
			continue
		}
		if !bashReplaced {
			newPTU = append(newPTU, map[string]interface{}{
				"matcher": "Bash",
				"hooks": []interface{}{
					map[string]interface{}{"type": "command", "command": wrapperCmd},
				},
			})
			bashReplaced = true
		}
		// Additional Bash matcher entries are dropped (merged into wrapper)
	}
	if !bashReplaced {
		newPTU = append(newPTU, map[string]interface{}{
			"matcher": "Bash",
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "command": wrapperCmd},
			},
		})
	}

	hooksMap["PreToolUse"] = newPTU
	settings["hooks"] = hooksMap
	return writeSettings(settingsPath, settings)
}

// shellQuote wraps a string in single quotes, escaping any single quotes within.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// ExtractWrappedHooks parses a chop-generated wrapper script and returns the
// competing hook commands that were embedded in it via _run_hook lines.
// Returns nil if the file does not exist or contains no _run_hook lines.
func ExtractWrappedHooks(scriptPath string) ([]string, error) {
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cmds []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "_run_hook ") {
			continue
		}
		arg := strings.TrimPrefix(line, "_run_hook ")
		// Unquote single-quoted argument (reverse of shellQuote)
		if len(arg) >= 2 && arg[0] == '\'' && arg[len(arg)-1] == '\'' {
			arg = arg[1 : len(arg)-1]
			arg = strings.ReplaceAll(arg, "'\\''", "'")
		}
		cmds = append(cmds, arg)
	}
	return cmds, nil
}

// UnwrapHooks reverses fix-hooks: reinstalls the direct chop hook in settings.json
// and removes chop-wrapper.sh. Returns the list of commands that were in the wrapper
// so the caller can inform the user what to re-enable elsewhere.
func UnwrapHooks(version string) ([]string, error) {
	wrapperPath, err := WrapperScriptPath()
	if err != nil {
		return nil, err
	}

	wrapped, err := ExtractWrappedHooks(wrapperPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wrapper script: %w", err)
	}

	// Remove all chop-aware hooks from settings.json before reinstalling the direct hook.
	// installWithCommand only recognises strict chop hooks, so wrapper entries linger
	// alongside the new entry unless we remove them first.
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := removeChopAwareHooks(settingsPath); err != nil {
		return nil, fmt.Errorf("failed to remove wrapper hook: %w", err)
	}

	// Reinstall direct chop hook
	Install(version)

	// Remove the wrapper script if it exists
	if _, err := os.Stat(wrapperPath); err == nil {
		_ = os.Remove(wrapperPath)
	}

	return wrapped, nil
}

// removeChopAwareHooks strips all chop-aware entries (direct or wrapper) from the
// Bash PreToolUse matcher in settings.json. Called before reinstalling the direct hook.
func removeChopAwareHooks(settingsPath string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}
	hooksRaw, ok := settings["hooks"]
	if !ok {
		return nil
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return nil
	}
	ptuRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		return nil
	}
	ptu, ok := ptuRaw.([]interface{})
	if !ok {
		return nil
	}

	newPTU := make([]interface{}, 0, len(ptu))
	for _, entry := range ptu {
		m, ok := entry.(map[string]interface{})
		if !ok {
			newPTU = append(newPTU, entry)
			continue
		}
		if matcher, _ := m["matcher"].(string); matcher != "Bash" {
			newPTU = append(newPTU, entry)
			continue
		}
		hooksArrayRaw, _ := m["hooks"].([]interface{})
		newHooks := make([]interface{}, 0, len(hooksArrayRaw))
		for _, h := range hooksArrayRaw {
			hMap, ok := h.(map[string]interface{})
			if !ok || isChopAwareHook(hMap) {
				continue
			}
			newHooks = append(newHooks, h)
		}
		if len(newHooks) > 0 {
			m["hooks"] = newHooks
			newPTU = append(newPTU, m)
		}
	}

	hooksMap["PreToolUse"] = newPTU
	settings["hooks"] = hooksMap
	return writeSettings(settingsPath, settings)
}

func installTo(settingsPath string) error {
	hookCmd, err := buildHookCommand()
	if err != nil {
		return err
	}
	return installWithCommand(settingsPath, hookCmd)
}

func installWithCommand(settingsPath string, hookCmd string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	// Ensure hooks map exists
	hooksRaw, ok := settings["hooks"]
	if !ok {
		hooksRaw = make(map[string]interface{})
		settings["hooks"] = hooksRaw
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("hooks field is not an object in %s", settingsPath)
	}

	// Ensure PreToolUse array exists
	preToolUseRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		preToolUseRaw = []interface{}{}
	}
	preToolUse, ok := preToolUseRaw.([]interface{})
	if !ok {
		return fmt.Errorf("hooks.PreToolUse is not an array in %s", settingsPath)
	}

	chopHookEntry := map[string]interface{}{
		"type":    "command",
		"command": hookCmd,
	}

	// Find existing Bash matcher
	bashIdx := -1
	for i, entry := range preToolUse {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if matcher, _ := m["matcher"].(string); matcher == "Bash" {
			bashIdx = i
			break
		}
	}

	if bashIdx >= 0 {
		bashMatcher := preToolUse[bashIdx].(map[string]interface{})
		hooksArrayRaw, ok := bashMatcher["hooks"]
		if !ok {
			hooksArrayRaw = []interface{}{}
		}
		hooksArray, ok := hooksArrayRaw.([]interface{})
		if !ok {
			return fmt.Errorf("Bash matcher hooks is not an array")
		}

		// Check if chop hook already exists - update it
		chopIdx := -1
		for i, h := range hooksArray {
			hMap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if isChopHook(hMap) {
				chopIdx = i
				break
			}
		}

		if chopIdx >= 0 {
			hooksArray[chopIdx] = chopHookEntry
		} else {
			hooksArray = append(hooksArray, chopHookEntry)
		}
		bashMatcher["hooks"] = hooksArray
	} else {
		// Create new Bash matcher
		bashMatcher := map[string]interface{}{
			"matcher": "Bash",
			"hooks": []interface{}{
				chopHookEntry,
			},
		}
		preToolUse = append(preToolUse, bashMatcher)
	}

	hooksMap["PreToolUse"] = preToolUse
	settings["hooks"] = hooksMap

	return writeSettings(settingsPath, settings)
}

func uninstallFrom(settingsPath string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	hooksRaw, ok := settings["hooks"]
	if !ok {
		return nil // nothing to remove
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	preToolUseRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		return nil
	}
	preToolUse, ok := preToolUseRaw.([]interface{})
	if !ok {
		return nil
	}

	// Find Bash matcher and remove chop hook
	newPreToolUse := make([]interface{}, 0, len(preToolUse))
	for _, entry := range preToolUse {
		m, ok := entry.(map[string]interface{})
		if !ok {
			newPreToolUse = append(newPreToolUse, entry)
			continue
		}
		matcher, _ := m["matcher"].(string)
		if matcher != "Bash" {
			newPreToolUse = append(newPreToolUse, entry)
			continue
		}

		// Filter out chop hooks from this Bash matcher
		hooksArrayRaw, ok := m["hooks"]
		if !ok {
			newPreToolUse = append(newPreToolUse, entry)
			continue
		}
		hooksArray, ok := hooksArrayRaw.([]interface{})
		if !ok {
			newPreToolUse = append(newPreToolUse, entry)
			continue
		}

		newHooks := make([]interface{}, 0, len(hooksArray))
		for _, h := range hooksArray {
			hMap, ok := h.(map[string]interface{})
			if !ok {
				newHooks = append(newHooks, h)
				continue
			}
			if !isChopHook(hMap) {
				newHooks = append(newHooks, h)
			}
		}

		if len(newHooks) > 0 {
			m["hooks"] = newHooks
			newPreToolUse = append(newPreToolUse, m)
		}
		// If no hooks left, drop the entire Bash matcher
	}

	if len(newPreToolUse) > 0 {
		hooksMap["PreToolUse"] = newPreToolUse
	} else {
		delete(hooksMap, "PreToolUse")
	}

	if len(hooksMap) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooksMap
	}

	return writeSettings(settingsPath, settings)
}
