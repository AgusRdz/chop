package updater

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const repo = "AgusRdz/chop"

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// publicKey is the hex-encoded Ed25519 public key used to verify release signatures.
const publicKey = "219d215ae55c0b8401a42cf8e3b84cd22ccdc21f880fb0405cd6daca3c03d94e"

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

	tmpPath := exe + ".tmp"
	if err := download(url, tmpPath); err != nil {
		fmt.Fprintf(os.Stderr, "chop: update failed: %v\n", err)
		os.Remove(tmpPath)
		os.Exit(1)
	}

	// Verify checksum and signature before replacing the binary
	if err := verifyChecksum(tmpPath, latest, binaryName); err != nil {
		fmt.Fprintf(os.Stderr, "chop: verification failed: %v\n", err)
		os.Remove(tmpPath)
		os.Exit(1)
	}

	if err := replaceBinary(exe, tmpPath); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to replace binary: %v\n", err)
		os.Remove(tmpPath)
		os.Exit(1)
	}

	fmt.Printf("updated to %s\n", latest)
	clearUpdateAvailable()

	// Re-exec the new binary for post-update checks.
	// This ensures the check runs with the new version's code regardless
	// of what version performed the update.
	cmd := exec.Command(exe, "--post-update-check")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func latestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := httpClient.Get(url)
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
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %d for %s", resp.StatusCode, url)
	}

	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	// Verify it's not a 404 HTML page
	info, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("failed to verify downloaded file: %w", err)
	}
	if info.Size() < 1024 {
		return fmt.Errorf("downloaded file too small (%d bytes), release may not exist", info.Size())
	}

	return nil
}

// IsDev reports whether the version looks like a dev build.
func IsDev(version string) bool {
	return version == "dev" || strings.Contains(version, "-dirty")
}

// verifyChecksum fetches checksums.txt and checksums.txt.sig from the release,
// verifies the signature of checksums.txt using the embedded public key,
// and then verifies the SHA256 hash of the binary.
func verifyChecksum(binaryPath, version, binaryName string) error {
	checksums, err := fetchReleaseFile(version, "checksums.txt")
	if err != nil {
		return fmt.Errorf("failed to fetch checksums.txt: %w", err)
	}

	signature, err := fetchReleaseFile(version, "checksums.txt.sig")
	if err != nil {
		return fmt.Errorf("failed to fetch checksums.txt.sig: %w", err)
	}

	if err := verifySignature(checksums, signature); err != nil {
		return fmt.Errorf("invalid signature for checksums.txt: %w", err)
	}

	expected, err := parseChecksum(string(checksums), binaryName)
	if err != nil {
		return err
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

func verifySignature(message, signature []byte) error {
	pub, err := hex.DecodeString(publicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	sig, err := hex.DecodeString(strings.TrimSpace(string(signature)))
	if err != nil {
		return fmt.Errorf("invalid signature format: %w", err)
	}

	if !ed25519.Verify(pub, message, sig) {
		return errors.New("ED25519 signature verification failed")
	}

	return nil
}

// fetchReleaseFile downloads a file from the release.
func fetchReleaseFile(version, filename string) ([]byte, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, version, filename)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s returned %d", filename, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
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
