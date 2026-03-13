package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func autoUpdateFlagPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auto-update"), nil
}

func updateAvailablePath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "update-available"), nil
}

// IsAutoUpdateEnabled reports whether automatic updates are turned on.
// Default is off — the flag file must be explicitly created.
func IsAutoUpdateEnabled() bool {
	p, err := autoUpdateFlagPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// SetAutoUpdate enables or disables automatic updates.
func SetAutoUpdate(enabled bool) error {
	p, err := autoUpdateFlagPath()
	if err != nil {
		return err
	}
	if enabled {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, nil, 0o644)
	}
	err = os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// NotifyIfUpdateAvailable prints a hint to stderr if a newer version is known.
// Called at startup when auto-update is off. Silent on all errors.
func NotifyIfUpdateAvailable(currentVersion string) {
	if IsDev(currentVersion) || IsAutoUpdateEnabled() {
		return
	}
	p, err := updateAvailablePath()
	if err != nil {
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return
	}
	latest := strings.TrimSpace(string(data))
	if latest == "" || latest == currentVersion {
		return
	}
	fmt.Fprintf(os.Stderr, "chop: update available %s -> %s (run 'chop update')\n", currentVersion, latest)
}

// clearUpdateAvailable removes the hint file (called after a successful manual update
// or when auto-update is enabled and handles the update itself).
func clearUpdateAvailable() {
	p, err := updateAvailablePath()
	if err != nil {
		return
	}
	os.Remove(p)
}

// recordUpdateAvailable writes the latest version to the hint file.
func recordUpdateAvailable(version string) {
	p, err := updateAvailablePath()
	if err != nil {
		return
	}
	os.WriteFile(p, []byte(version), 0o644)
}
