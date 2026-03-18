package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AgusRdz/chop/config"
)

// GeminiInstall registers the chop hook in .gemini/settings.json.
func GeminiInstall(version string) {
	settingsPath := geminiSettingsPath()
	if err := geminiInstallTo(settingsPath); err != nil {
		fmt.Fprintf(os.Stderr, "chop: %v\n", err)
		os.Exit(1)
	}
	_ = config.WriteDiscoveryInfo(version)
	fmt.Printf("chop hook installed in %s\n", settingsPath)

	binPath, _ := chopBinaryPath()
	fmt.Printf("\nInstallation complete! Please tell your Gemini CLI: 'Remember that chop is installed at %s and use it for CLI compression.' This will prevent the agent from searching for it in the future.\n", binPath)
}

// GeminiUninstall removes the chop hook from .gemini/settings.json.
func GeminiUninstall() {
	settingsPath := geminiSettingsPath()
	if err := geminiUninstallFrom(settingsPath); err != nil {
		fmt.Fprintf(os.Stderr, "chop: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("chop hook removed from %s\n", settingsPath)
}

// GeminiIsInstalled checks whether the chop hook is registered in .gemini/settings.json.
func GeminiIsInstalled() (bool, string) {
	settingsPath := geminiSettingsPath()
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

	beforeToolRaw, ok := hooksMap["BeforeTool"]
	if !ok {
		return false, settingsPath
	}
	beforeTool, ok := beforeToolRaw.([]interface{})
	if !ok {
		return false, settingsPath
	}

	for _, entry := range beforeTool {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if matcher, _ := m["matcher"].(string); matcher != "run_shell_command" {
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
			if isChopGeminiHook(hMap) {
				return true, settingsPath
			}
		}
	}

	return false, settingsPath
}

func geminiSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gemini/settings.json"
	}
	return filepath.Join(home, ".gemini", "settings.json")
}

func buildGeminiHookCommand() (string, error) {
	binPath, err := chopBinaryPath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%s" hook --gemini`, binPath), nil
}

func isChopGeminiHook(hookObj map[string]interface{}) bool {
	cmd, ok := hookObj["command"].(string)
	if !ok {
		return false
	}
	return strings.Contains(cmd, chopBinaryName) && strings.Contains(cmd, "hook")
}

func geminiInstallTo(settingsPath string) error {
	hookCmd, err := buildGeminiHookCommand()
	if err != nil {
		return err
	}
	return geminiInstallWithCommand(settingsPath, hookCmd)
}

func geminiInstallWithCommand(settingsPath string, hookCmd string) error {
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

	// Ensure BeforeTool array exists
	beforeToolRaw, ok := hooksMap["BeforeTool"]
	if !ok {
		beforeToolRaw = []interface{}{}
	}
	beforeTool, ok := beforeToolRaw.([]interface{})
	if !ok {
		return fmt.Errorf("hooks.BeforeTool is not an array in %s", settingsPath)
	}

	chopHookEntry := map[string]interface{}{
		"type":    "command",
		"command": hookCmd,
	}

	// Find existing run_shell_command matcher
	matcherIdx := -1
	for i, entry := range beforeTool {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if matcher, _ := m["matcher"].(string); matcher == "run_shell_command" {
			matcherIdx = i
			break
		}
	}

	if matcherIdx >= 0 {
		matcher := beforeTool[matcherIdx].(map[string]interface{})
		hooksArrayRaw, ok := matcher["hooks"]
		if !ok {
			hooksArrayRaw = []interface{}{}
		}
		hooksArray, ok := hooksArrayRaw.([]interface{})
		if !ok {
			return fmt.Errorf("run_shell_command matcher hooks is not an array")
		}

		// Check if chop hook already exists — update it
		chopIdx := -1
		for i, h := range hooksArray {
			hMap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if isChopGeminiHook(hMap) {
				chopIdx = i
				break
			}
		}

		if chopIdx >= 0 {
			hooksArray[chopIdx] = chopHookEntry
		} else {
			hooksArray = append(hooksArray, chopHookEntry)
		}
		matcher["hooks"] = hooksArray
	} else {
		// Create new run_shell_command matcher
		newMatcher := map[string]interface{}{
			"matcher": "run_shell_command",
			"hooks": []interface{}{
				chopHookEntry,
			},
		}
		beforeTool = append(beforeTool, newMatcher)
	}

	hooksMap["BeforeTool"] = beforeTool
	settings["hooks"] = hooksMap

	return writeSettings(settingsPath, settings)
}

func geminiUninstallFrom(settingsPath string) error {
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

	beforeToolRaw, ok := hooksMap["BeforeTool"]
	if !ok {
		return nil
	}
	beforeTool, ok := beforeToolRaw.([]interface{})
	if !ok {
		return nil
	}

	newBeforeTool := make([]interface{}, 0, len(beforeTool))
	for _, entry := range beforeTool {
		m, ok := entry.(map[string]interface{})
		if !ok {
			newBeforeTool = append(newBeforeTool, entry)
			continue
		}
		matcher, _ := m["matcher"].(string)
		if matcher != "run_shell_command" {
			newBeforeTool = append(newBeforeTool, entry)
			continue
		}

		hooksArrayRaw, ok := m["hooks"]
		if !ok {
			newBeforeTool = append(newBeforeTool, entry)
			continue
		}
		hooksArray, ok := hooksArrayRaw.([]interface{})
		if !ok {
			newBeforeTool = append(newBeforeTool, entry)
			continue
		}

		newHooks := make([]interface{}, 0, len(hooksArray))
		for _, h := range hooksArray {
			hMap, ok := h.(map[string]interface{})
			if !ok {
				newHooks = append(newHooks, h)
				continue
			}
			if !isChopGeminiHook(hMap) {
				newHooks = append(newHooks, h)
			}
		}

		if len(newHooks) > 0 {
			m["hooks"] = newHooks
			newBeforeTool = append(newBeforeTool, m)
		}
	}

	if len(newBeforeTool) > 0 {
		hooksMap["BeforeTool"] = newBeforeTool
	} else {
		delete(hooksMap, "BeforeTool")
	}

	if len(hooksMap) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooksMap
	}

	return writeSettings(settingsPath, settings)
}
