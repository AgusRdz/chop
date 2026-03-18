package hooks

import (
	"encoding/json"
	"fmt"
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func isChopHook(hookObj map[string]interface{}) bool {
	cmd, ok := hookObj["command"].(string)
	if !ok {
		return false
	}
	// Match commands that reference "chop" and end with " hook"
	return strings.Contains(cmd, chopBinaryName) && strings.HasSuffix(cmd, " hook")
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
