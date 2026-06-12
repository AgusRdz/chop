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
const publicKey = "f2bbc7ef2c427df9963a15f403ea17ff08ed5ee2b6d6e6ac49920e19be255d5f"

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

	// Verify checksum and signature before replacing the binary.
	// verifyChecksum returns the verified hash so we can re-check it right
	// before the rename, closing the TOCTOU window (mirrors ApplyPendingUpdate).
	verifiedHash, err := verifyChecksumAndHash(tmpPath, latest, binaryName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: verification failed: %v\n", err)
		os.Remove(tmpPath)
		os.Exit(1)
	}

	// Re-hash immediately before rename to close the TOCTOU window.
	preInstallHash, err := hashFile(tmpPath)
	if err != nil || preInstallHash != verifiedHash {
		fmt.Fprintf(os.Stderr, "chop: binary hash changed before install — aborting\n")
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
		return "", fmt.Errorf("could not reach GitHub (check your internet connection or firewall): %w", err)
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

	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	// Verify it's not a 404 HTML page or otherwise invalid binary
	info, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("failed to verify downloaded file: %w", err)
	}
	if info.Size() < 1024 {
		return fmt.Errorf("downloaded file too small (%d bytes), release may not exist", info.Size())
	}
	if err := checkBinaryMagic(destPath); err != nil {
		return err
	}

	return nil
}

// checkBinaryMagic verifies the file at path starts with the expected magic bytes
// for the current platform (ELF on Linux, Mach-O on macOS, PE on Windows).
func checkBinaryMagic(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open binary for validation: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 4)
	if _, err := io.ReadFull(f, buf); err != nil {
		return fmt.Errorf("binary too small to read magic bytes: %w", err)
	}

	switch runtime.GOOS {
	case "linux":
		if buf[0] != 0x7f || buf[1] != 'E' || buf[2] != 'L' || buf[3] != 'F' {
			return fmt.Errorf("downloaded file is not a valid ELF binary")
		}
	case "darwin":
		valid := (buf[0] == 0xca && buf[1] == 0xfe && buf[2] == 0xba && buf[3] == 0xbe) || // fat binary
			(buf[0] == 0xcf && buf[1] == 0xfa && buf[2] == 0xed && buf[3] == 0xfe) || // 64-bit LE
			(buf[0] == 0xce && buf[1] == 0xfa && buf[2] == 0xed && buf[3] == 0xfe) || // 32-bit LE
			(buf[0] == 0xfe && buf[1] == 0xed && buf[2] == 0xfa && buf[3] == 0xcf) || // 64-bit BE
			(buf[0] == 0xfe && buf[1] == 0xed && buf[2] == 0xfa && buf[3] == 0xce) // 32-bit BE
		if !valid {
			return fmt.Errorf("downloaded file is not a valid Mach-O binary")
		}
	case "windows":
		if buf[0] != 'M' || buf[1] != 'Z' {
			return fmt.Errorf("downloaded file is not a valid PE binary")
		}
	}
	return nil
}

// IsDev reports whether the version looks like a dev build.
func IsDev(version string) bool {
	return version == "dev" || strings.Contains(version, "-dirty")
}

// isNewer reports whether version a is strictly newer than version b.
// Expects semver tags in the form "vX.Y.Z". Returns false for malformed input.
func isNewer(a, b string) bool {
	pa, ok1 := parseSemver(a)
	pb, ok2 := parseSemver(b)
	if !ok1 || !ok2 {
		return false
	}
	if pa[0] != pb[0] {
		return pa[0] > pb[0]
	}
	if pa[1] != pb[1] {
		return pa[1] > pb[1]
	}
	return pa[2] > pb[2]
}

// parseSemver parses a "vX.Y.Z" tag into [major, minor, patch].
func parseSemver(v string) ([3]int, bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var nums [3]int
	for i, p := range parts {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return [3]int{}, false
			}
			n = n*10 + int(c-'0')
		}
		nums[i] = n
	}
	return nums, true
}

// verifyChecksum fetches checksums.txt and checksums.txt.sig from the release,
// verifies the signature of checksums.txt using the embedded public key,
// and then verifies the SHA256 hash of the binary.
func verifyChecksum(binaryPath, version, binaryName string) error {
	_, err := verifyChecksumAndHash(binaryPath, version, binaryName)
	return err
}

// verifyChecksumAndHash is like verifyChecksum but also returns the verified SHA256
// hex digest so callers can re-check the file right before installation.
func verifyChecksumAndHash(binaryPath, version, binaryName string) (string, error) {
	checksums, err := fetchReleaseFile(version, "checksums.txt")
	if err != nil {
		return "", fmt.Errorf("failed to fetch checksums.txt: %w", err)
	}

	signature, err := fetchReleaseFile(version, "checksums.txt.sig")
	if err != nil {
		return "", fmt.Errorf("failed to fetch checksums.txt.sig: %w", err)
	}

	if err := verifySignature(checksums, signature); err != nil {
		return "", fmt.Errorf("invalid signature for checksums.txt: %w", err)
	}

	expected, err := parseChecksum(string(checksums), binaryName)
	if err != nil {
		return "", err
	}

	actual, err := hashFile(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to hash downloaded binary: %w", err)
	}

	if actual != expected {
		return "", fmt.Errorf("SHA256 mismatch: expected %s, got %s", expected, actual)
	}
	return actual, nil
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

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("%s not found (404)", filename)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s returned %d", filename, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// parseChecksum extracts the SHA256 hash for binaryName from sha256sum-formatted text.
// Format: "hash  filename\n"
func parseChecksum(checksums, binaryName string) (string, error) {
	remaining := checksums
	for len(remaining) > 0 {
		var line string
		if i := strings.IndexByte(remaining, '\n'); i >= 0 {
			line = remaining[:i]
			remaining = remaining[i+1:]
		} else {
			line = remaining
			remaining = ""
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// sha256sum output: "hash  filename" (two spaces) or "hash *filename"
		// Find first space (end of hash)
		i := strings.IndexByte(line, ' ')
		if i == -1 {
			continue
		}
		hash := line[:i]

		// Find start of filename (first non-space after hash)
		rest := line[i+1:]
		j := 0
		for j < len(rest) && rest[j] == ' ' {
			j++
		}
		if j == len(rest) {
			continue
		}
		filenamePart := rest[j:]

		name := strings.TrimPrefix(filenamePart, "*")
		if name == binaryName {
			return hash, nil
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
