package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the platform-specific directory for configuration files.
// On Unix: $XDG_CONFIG_HOME/chop or ~/.config/chop
// On Windows: %AppData%\chop
func ConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "chop")
		}
		return filepath.Join(home, ".config", "chop")
	}
	return filepath.Join(dir, "chop")
}

// DataDir returns the platform-specific directory for state, logs, and database.
// On Unix: $XDG_DATA_HOME/chop or ~/.local/share/chop
// On Windows: %LocalAppData%\chop
func DataDir() string {
	if runtime.GOOS == "windows" {
		dir, err := os.UserCacheDir()
		if err == nil {
			return filepath.Join(dir, "chop")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Local", "chop")
	}

	// Unix-like
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "chop")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "chop")
}
