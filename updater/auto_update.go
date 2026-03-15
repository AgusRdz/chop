package updater

import (
	"fmt"
	"os"
	"os/exec"
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
// If found, replaces the current binary. The update takes effect on the next invocation.
// Only applies when auto-update is enabled. Cleans up stale pending files otherwise.
// Silent on all errors - never disrupts the current command.
func ApplyPendingUpdate(currentVersion string) {
	if IsDev(currentVersion) {
		return
	}

	pending, err := pendingUpdatePath()
	if err != nil {
		return
	}

	data, err := os.ReadFile(pending)
	if err != nil {
		return
	}

	// If auto-update is off, clean up any leftover pending files
	if !IsAutoUpdateEnabled() {
		parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
		os.Remove(pending)
		if len(parts) == 2 {
			os.Remove(parts[1]) // remove downloaded binary
		}
		return
	}

	// Format: "version\ntmpBinaryPath"
	parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	if len(parts) != 2 {
		os.Remove(pending)
		return
	}

	newVersion := parts[0]
	tmpBinary := parts[1]

	// Verify the temp binary still exists and is valid
	info, err := os.Stat(tmpBinary)
	if err != nil || info.Size() < 1024 {
		os.Remove(pending)
		os.Remove(tmpBinary)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		os.Remove(pending)
		return
	}

	// Replace the current binary
	if err := replaceBinary(exe, tmpBinary); err != nil {
		os.Remove(pending)
		os.Remove(tmpBinary)
		return
	}

	os.Remove(pending)
	fmt.Fprintf(os.Stderr, "chop: auto-updated %s -> %s\n", currentVersion, newVersion)
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

// BackgroundCheck spawns a detached subprocess to check for updates.
// When auto-update is on, the subprocess downloads the new binary.
// When auto-update is off, it only records the available version for a hint message.
// Silent on all errors - never disrupts command output.
func BackgroundCheck(currentVersion string) {
	if IsDev(currentVersion) {
		return
	}
	if !shouldCheck() {
		return
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}

	// Spawn detached subprocess — parent exits immediately, child runs independently.
	cmd := exec.Command(exe, "--_bg-update", currentVersion)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if cmd.Start() == nil {
		// Mark check initiated so we don't spawn again within the interval.
		touchLastCheck()
	}
}

// RunBackgroundUpdate performs the version check and optionally downloads.
// When auto-update is on: checks version + downloads binary for next-run apply.
// When auto-update is off: checks version + records it so a hint is shown.
// Called by the subprocess spawned from BackgroundCheck — runs after parent exits.
func RunBackgroundUpdate(currentVersion string) {
	latest, err := latestVersion()
	if err != nil || latest == currentVersion {
		clearUpdateAvailable()
		return
	}

	// Always record that an update is available (for the hint message)
	recordUpdateAvailable(latest)

	// Only download when auto-update is on
	if !IsAutoUpdateEnabled() {
		return
	}

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

	// Verify checksum before staging the pending update
	if err := verifyChecksum(tmpPath, latest, binaryName); err != nil {
		os.Remove(tmpPath)
		return
	}

	pending, err := pendingUpdatePath()
	if err != nil {
		os.Remove(tmpPath)
		return
	}

	content := fmt.Sprintf("%s\n%s", latest, tmpPath)
	os.WriteFile(pending, []byte(content), 0o644)
}
