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
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			return err
		}
		return os.WriteFile(p, nil, 0o600)
	}
	err = os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// AvailableUpdate returns the newer version recorded by the last check, if any.
func AvailableUpdate(currentVersion string) (string, bool) {
	if IsDev(currentVersion) {
		return "", false
	}
	p, err := updateAvailablePath()
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	latest := strings.TrimSpace(string(data))
	if latest == "" || latest == currentVersion || !isNewer(latest, currentVersion) {
		return "", false
	}
	return latest, true
}

// NotifyIfUpdateAvailable prints a hint to stderr if a newer version is known.
// Called from gain when auto-update is off. Silent on all errors.
func NotifyIfUpdateAvailable(currentVersion string) {
	if IsAutoUpdateEnabled() {
		return
	}
	if latest, ok := AvailableUpdate(currentVersion); ok {
		fmt.Fprintf(os.Stderr, "chop: update available %s -> %s (run 'chop update')\n", currentVersion, latest)
	}
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
	os.WriteFile(p, []byte(version), 0o600)
}
