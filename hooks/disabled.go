package hooks

import (
	"os"
	"path/filepath"
)

func disabledPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "chop", "disabled")
}

// IsDisabledGlobally reports whether chop hook wrapping is disabled.
func IsDisabledGlobally() bool {
	p := disabledPath()
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

// Disable creates the flag file to stop hook wrapping.
func Disable() error {
	p := disabledPath()
	if p == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, nil, 0o644)
}

// Enable removes the flag file to resume hook wrapping.
func Enable() error {
	p := disabledPath()
	if p == "" {
		return nil
	}
	err := os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
