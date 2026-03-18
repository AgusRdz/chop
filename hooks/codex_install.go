package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AgusRdz/chop/config"
)

// CodexInstall registers the chop hook in .codex/settings.json.
func CodexInstall(version string) {
	settingsPath := codexSettingsPath()
	if err := codexInstallTo(settingsPath); err != nil {
		fmt.Fprintf(os.Stderr, "chop: %v\n", err)
		os.Exit(1)
	}
	_ = config.WriteDiscoveryInfo(version)
	fmt.Printf("chop hook installed in %s\n", settingsPath)

	binPath, _ := chopBinaryPath()
	fmt.Printf("\nInstallation complete! Please tell your Codex CLI: 'Remember that chop is installed at %s and use it for CLI compression.' This will prevent the agent from searching for it in the future.\n", binPath)
}

// CodexUninstall removes the chop hook from .codex/settings.json.
func CodexUninstall() {
	settingsPath := codexSettingsPath()
	if err := codexUninstallFrom(settingsPath); err != nil {
		fmt.Fprintf(os.Stderr, "chop: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("chop hook removed from %s\n", settingsPath)
}

// CodexIsInstalled checks whether the chop hook is registered in .codex/settings.json.
func CodexIsInstalled() (bool, string) {
	settingsPath := codexSettingsPath()
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
		matcher, _ := m["matcher"].(string)
		if matcher != "bash" && matcher != "Bash" {
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
			if isChopCodexHook(hMap) {
				return true, settingsPath
			}
		}
	}

	return false, settingsPath
}

func codexSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".codex/settings.json"
	}
	return filepath.Join(home, ".codex", "settings.json")
}

func buildCodexHookCommand() (string, error) {
	binPath, err := chopBinaryPath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%s" hook --codex`, binPath), nil
}

func isChopCodexHook(hookObj map[string]interface{}) bool {
	cmd, ok := hookObj["command"].(string)
	if !ok {
		return false
	}
	return strings.Contains(cmd, chopBinaryName) && strings.Contains(cmd, "hook") && strings.Contains(cmd, "--codex")
}

func codexInstallTo(settingsPath string) error {
	hookCmd, err := buildCodexHookCommand()
	if err != nil {
		return err
	}
	return codexInstallWithCommand(settingsPath, hookCmd)
}

func codexInstallWithCommand(settingsPath string, hookCmd string) error {
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

	// Find existing bash matcher
	bashIdx := -1
	for i, entry := range preToolUse {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		matcher, _ := m["matcher"].(string)
		if matcher == "bash" || matcher == "Bash" {
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
			return fmt.Errorf("bash matcher hooks is not an array")
		}

		// Check if chop hook already exists — update it
		chopIdx := -1
		for i, h := range hooksArray {
			hMap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if isChopCodexHook(hMap) {
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
		// Create new bash matcher
		newMatcher := map[string]interface{}{
			"matcher": "bash",
			"hooks": []interface{}{
				chopHookEntry,
			},
		}
		preToolUse = append(preToolUse, newMatcher)
	}

	hooksMap["PreToolUse"] = preToolUse
	settings["hooks"] = hooksMap

	return writeSettings(settingsPath, settings)
}

func codexUninstallFrom(settingsPath string) error {
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

	preToolUseRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		return nil
	}
	preToolUse, ok := preToolUseRaw.([]interface{})
	if !ok {
		return nil
	}

	newPreToolUse := make([]interface{}, 0, len(preToolUse))
	for _, entry := range preToolUse {
		m, ok := entry.(map[string]interface{})
		if !ok {
			newPreToolUse = append(newPreToolUse, entry)
			continue
		}
		matcher, _ := m["matcher"].(string)
		if matcher != "bash" && matcher != "Bash" {
			newPreToolUse = append(newPreToolUse, entry)
			continue
		}

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
			if !isChopCodexHook(hMap) {
				newHooks = append(newHooks, h)
			}
		}

		if len(newHooks) > 0 {
			m["hooks"] = newHooks
			newPreToolUse = append(newPreToolUse, m)
		}
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
