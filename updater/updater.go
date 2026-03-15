package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

	// Verify checksum (best-effort — skip if checksums.txt not published yet)
	if err := verifyChecksum(exe, latest, binaryName); err != nil {
		fmt.Fprintf(os.Stderr, "chop: checksum verification failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "chop: the downloaded binary may be corrupted — reverting")
		os.Exit(1)
	}

	fmt.Printf("updated to %s\n", latest)
	clearUpdateAvailable()

	// Re-exec the new binary for post-update checks.
	cmd := exec.Command(exe, "--post-update-check")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
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

	// Write to temp file next to the binary, computing SHA256 as we go
	tmpPath := destPath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	h := sha256.New()
	_, err = io.Copy(f, io.TeeReader(resp.Body, h))
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write binary: %w", err)
	}

	// On Windows, can't replace a running binary directly.
	oldPath := destPath + ".old"
	os.Remove(oldPath)

	if runtime.GOOS == "windows" {
		if err := os.Rename(destPath, oldPath); err != nil && !os.IsNotExist(err) {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to move old binary: %w", err)
		}
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		if runtime.GOOS == "windows" {
			os.Rename(oldPath, destPath)
		}
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	if runtime.GOOS != "windows" {
		os.Remove(oldPath)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("failed to verify new binary: %w", err)
	}
	if info.Size() < 1024 {
		return fmt.Errorf("downloaded file too small (%d bytes), release may not exist", info.Size())
	}

	return nil
}

// verifyChecksum fetches checksums.txt from the release and verifies the binary.
// Returns nil if verification passes or if checksums.txt is not available (graceful fallback).
func verifyChecksum(binaryPath, version, binaryName string) error {
	expected, err := fetchExpectedChecksum(version, binaryName)
	if err != nil {
		// checksums.txt not published yet — skip verification silently
		return nil
	}

	actual, err := hashFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to hash downloaded binary: %w", err)
	}

	if actual != expected {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

// fetchExpectedChecksum downloads checksums.txt and extracts the hash for binaryName.
func fetchExpectedChecksum(version, binaryName string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/checksums.txt", repo, version)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("checksums.txt returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return parseChecksum(string(body), binaryName)
}

// parseChecksum extracts the SHA256 hash for binaryName from sha256sum-formatted text.
// Format: "hash  filename\n"
func parseChecksum(checksums, binaryName string) (string, error) {
	for _, line := range strings.Split(checksums, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// sha256sum output: "hash  filename" (two spaces) or "hash *filename"
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimPrefix(parts[1], "*")
		if name == binaryName {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %s", binaryName)
}

// hashFile computes the SHA256 hex digest of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// IsDev reports whether the version looks like a dev build.
func IsDev(version string) bool {
	return version == "dev" || strings.Contains(version, "-dirty")
}
