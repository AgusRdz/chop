package updater

import (
	"fmt"
	"os"
	"runtime"
	"path/filepath"
	"strings"
	"time"

	"github.com/AgusRdz/chop/config"
)

const checkInterval = 24 * time.Hour

// dataDir returns the platform-appropriate chop data directory, creating it if needed.
func dataDir() (string, error) {
	dir := config.DataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
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
	os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0o600)
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

	// Format: "version\ntmpBinaryPath\nsha256hash"
	parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 3)
	if len(parts) != 3 {
		os.Remove(pending)
		return
	}

	newVersion := parts[0]
	tmpBinary := parts[1]
	expectedHash := parts[2]

	// Validate the binary path is inside our data directory to prevent path traversal.
	safeDir, err := dataDir()
	if err != nil {
		os.Remove(pending)
		return
	}
	cleanBinary := filepath.Clean(tmpBinary)
	if !strings.HasPrefix(cleanBinary, safeDir+string(filepath.Separator)) {
		os.Remove(pending)
		return
	}
	tmpBinary = cleanBinary

	// Verify the temp binary still exists and matches the stored hash.
	// Re-hashing at apply time closes the TOCTOU window between download and install.
	info, err := os.Stat(tmpBinary)
	if err != nil || info.Size() < 1024 {
		os.Remove(pending)
		os.Remove(tmpBinary)
		return
	}
	actualHash, err := hashFile(tmpBinary)
	if err != nil || actualHash != expectedHash {
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

// CleanupOldBinary removes the ".old" copy left behind by a previous Windows update.
// The rename-aside in replaceBinary can't delete it while the old process is running,
// so the next doctor or gain run does it. Silent on all errors.
func CleanupOldBinary() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	removeOldBinary(exe)
}

func removeOldBinary(exe string) {
	os.Remove(exe + ".old")
}

// stagedBinaryPath returns where a downloaded update is staged inside dir.
func stagedBinaryPath(dir, version string) string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return filepath.Join(dir, "chop-"+filepath.Base(version)+ext)
}

// CheckForUpdate performs a synchronous, throttled (24h) version check.
// When auto-update is on: checks version + downloads the binary for next-run apply.
// When auto-update is off: checks version + records it so a hint is shown.
// Called only from management commands, never from the wrapped-command or hook paths.
// Silent on errors; the only output is a stderr notice when a download starts.
func CheckForUpdate(currentVersion string) {
	if IsDev(currentVersion) {
		return
	}
	if !shouldCheck() {
		return
	}
	// Touch before the network call so a failure doesn't retry on every run.
	touchLastCheck()

	latest, err := latestVersionWith(lookupClient)
	if err != nil || !isNewer(latest, currentVersion) {
		clearUpdateAvailable()
		return
	}

	// Always record that an update is available (for the hint message)
	recordUpdateAvailable(latest)

	// Only download when auto-update is on
	if !IsAutoUpdateEnabled() {
		return
	}

	dir, err := dataDir()
	if err != nil {
		return
	}

	tmpPath := stagedBinaryPath(dir, latest)
	binaryName := buildBinaryName()
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, latest, binaryName)

	fmt.Fprintf(os.Stderr, "chop: downloading update %s -> %s...\n", currentVersion, latest)
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

	hash, err := hashFile(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return
	}

	content := fmt.Sprintf("%s\n%s\n%s", latest, tmpPath, hash)
	os.WriteFile(pending, []byte(content), 0o600)
}
