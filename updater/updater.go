package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const repo = "AgusRdz/chop"

type ghRelease struct {
	TagName string `json:"tag_name"`
}

// Run checks for the latest version and updates the binary if needed.
func Run(currentVersion string) {
	fmt.Println("checking for updates...")

	latest, err := latestVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to check for updates: %v\n", err)
		os.Exit(1)
	}

	if latest == currentVersion {
		fmt.Printf("already up to date (%s)\n", currentVersion)
		return
	}

	fmt.Printf("updating %s -> %s\n", currentVersion, latest)

	binaryName := buildBinaryName()
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, latest, binaryName)

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to find current binary: %v\n", err)
		os.Exit(1)
	}

	if err := download(url, exe); err != nil {
		fmt.Fprintf(os.Stderr, "chop: update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("updated to %s\n", latest)
}

func latestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func buildBinaryName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("chop-%s-%s%s", goos, goarch, ext)
}

func download(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %d for %s", resp.StatusCode, url)
	}

	// Write to temp file next to the binary, then rename
	tmpPath := destPath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write binary: %w", err)
	}

	// On Windows, can't replace a running binary directly.
	// Rename current to .old, rename .tmp to current.
	oldPath := destPath + ".old"
	os.Remove(oldPath)

	if runtime.GOOS == "windows" {
		if err := os.Rename(destPath, oldPath); err != nil && !os.IsNotExist(err) {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to move old binary: %w", err)
		}
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		// Try to restore old binary on failure
		if runtime.GOOS == "windows" {
			os.Rename(oldPath, destPath)
		}
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Clean up old binary (best-effort, may fail on Windows if still running)
	if runtime.GOOS != "windows" {
		os.Remove(oldPath)
	}

	// Verify it's not a 404 HTML page
	info, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("failed to verify new binary: %w", err)
	}
	if info.Size() < 1024 {
		return fmt.Errorf("downloaded file too small (%d bytes), release may not exist", info.Size())
	}

	return nil
}

// CheckReminder prints a reminder if the current version looks like a dev build.
func IsDev(version string) bool {
	return version == "dev" || strings.Contains(version, "-dirty")
}
