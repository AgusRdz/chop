package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiscoveryInfo contains information for AI agents to discover chop.
type DiscoveryInfo struct {
	Version string `json:"version"`
	Path    string `json:"path"`
}

// ExecutablePath returns a stable path to the running Chop binary when
// possible, falling back to the resolved executable path.
func ExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return preferredExecutablePath(exe, os.Args[0], exec.LookPath)
}

func preferredExecutablePath(executable, invokedAs string, lookPath func(string) (string, error)) (string, error) {
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	if candidate, err := invokedExecutablePath(invokedAs, lookPath); err == nil {
		resolvedInfo, resolvedErr := os.Stat(resolved)
		candidateInfo, candidateErr := os.Stat(candidate)
		if resolvedErr == nil && candidateErr == nil && os.SameFile(resolvedInfo, candidateInfo) {
			return strings.ReplaceAll(candidate, "\\", "/"), nil
		}
	}

	return strings.ReplaceAll(resolved, "\\", "/"), nil
}

func invokedExecutablePath(invokedAs string, lookPath func(string) (string, error)) (string, error) {
	if invokedAs == "" {
		return "", fmt.Errorf("empty executable invocation")
	}

	candidate := invokedAs
	if !filepath.IsAbs(candidate) && filepath.Base(candidate) == candidate {
		var err error
		candidate, err = lookPath(candidate)
		if err != nil {
			return "", err
		}
	}

	return filepath.Abs(candidate)
}

// WriteDiscoveryInfo writes discovery metadata to ~/.chop/path.json.
func WriteDiscoveryInfo(version string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	exe, err := ExecutablePath()
	if err != nil {
		return err
	}

	info := DiscoveryInfo{
		Version: version,
		Path:    exe,
	}

	dir := filepath.Join(home, ".chop")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	path := filepath.Join(dir, "path.json")
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// DiscoveryPath returns the path to the discovery file.
func DiscoveryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".chop", "path.json"), nil
}
