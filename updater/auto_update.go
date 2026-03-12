package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const checkInterval = 24 * time.Hour

// dataDir returns ~/.local/share/chop, creating it if needed.
func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "chop")
	os.MkdirAll(dir, 0o755)
	return dir, nil
}

func lastCheckPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "last-update-check"), nil
}

func pendingUpdatePath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pending-update"), nil
}

// shouldCheck returns true if enough time has passed since the last update check.
func shouldCheck() bool {
	path, err := lastCheckPath()
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return true // never checked
	}
	return time.Since(info.ModTime()) > checkInterval
}

// touchLastCheck updates the timestamp of the last check file.
func touchLastCheck() {
	path, err := lastCheckPath()
	if err != nil {
		return
	}
	os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0o644)
}

// ApplyPendingUpdate checks for a pending update downloaded in a previous run.
// If found, replaces the current binary and re-execs with the same args.
// Returns true if an update was applied (caller should exit).
func ApplyPendingUpdate(currentVersion string) bool {
	if IsDev(currentVersion) {
		return false
	}

	pending, err := pendingUpdatePath()
	if err != nil {
		return false
	}

	data, err := os.ReadFile(pending)
	if err != nil {
		return false
	}

	// Format: "version\ntmpBinaryPath"
	parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	if len(parts) != 2 {
		os.Remove(pending)
		return false
	}

	newVersion := parts[0]
	tmpBinary := parts[1]

	// Verify the temp binary still exists and is valid
	info, err := os.Stat(tmpBinary)
	if err != nil || info.Size() < 1024 {
		os.Remove(pending)
		os.Remove(tmpBinary)
		return false
	}

	exe, err := os.Executable()
	if err != nil {
		os.Remove(pending)
		return false
	}

	// Replace the current binary
	if err := replaceBinary(exe, tmpBinary); err != nil {
		os.Remove(pending)
		os.Remove(tmpBinary)
		return false
	}

	os.Remove(pending)
	fmt.Fprintf(os.Stderr, "chop: auto-updated %s -> %s\n", currentVersion, newVersion)
	return true
}

// replaceBinary atomically replaces the binary at destPath with srcPath.
func replaceBinary(destPath, srcPath string) error {
	if runtime.GOOS == "windows" {
		// Windows can't replace a running binary - rename dance
		oldPath := destPath + ".old"
		os.Remove(oldPath)
		if err := os.Rename(destPath, oldPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(srcPath, destPath); err != nil {
			os.Rename(oldPath, destPath) // restore
			return err
		}
		os.Remove(oldPath)
		return nil
	}

	// Linux/macOS: rename works even on running binaries
	return os.Rename(srcPath, destPath)
}

// BackgroundCheck runs a non-blocking update check after the command has finished.
// If a new version is available, downloads the binary to a temp location
// and writes a pending-update marker for the next invocation to apply.
// Silent on all errors - never disrupts command output.
func BackgroundCheck(currentVersion string) {
	if IsDev(currentVersion) {
		return
	}

	if !shouldCheck() {
		return
	}

	touchLastCheck()

	latest, err := latestVersion()
	if err != nil || latest == currentVersion {
		return
	}

	// Download new binary to temp location next to current binary
	exe, err := os.Executable()
	if err != nil {
		return
	}

	tmpPath := exe + ".new"
	binaryName := buildBinaryName()
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, latest, binaryName)

	if err := download(url, tmpPath); err != nil {
		os.Remove(tmpPath)
		return
	}

	// Write pending update marker
	pending, err := pendingUpdatePath()
	if err != nil {
		os.Remove(tmpPath)
		return
	}

	content := fmt.Sprintf("%s\n%s", latest, tmpPath)
	os.WriteFile(pending, []byte(content), 0o644)
}
