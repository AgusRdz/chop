package cleanup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AgusRdz/chop/hooks"
)

// Uninstall removes everything: hook, data, config, and the binary itself.
// It prints what it removes. The binary self-deletion happens last.
func Uninstall(keepData bool) {
	// 1. Hooks — remove all supported integrations
	if installed, _ := hooks.IsInstalled(); installed {
		hooks.Uninstall()
	}
	if installed, _ := hooks.CodexIsInstalled(); installed {
		hooks.CodexUninstall()
	}
	if installed, _ := hooks.GeminiIsInstalled(); installed {
		hooks.GeminiUninstall()
	}
	if installed, _ := hooks.AntigravityIsInstalled(); installed {
		hooks.AntigravityUninstall()
	}

	// 2. Data
	if !keepData {
		dir := dataDir()
		if removed := removeDir(dir); removed {
			fmt.Printf("removed data (%s)\n", dir)
		}
	}

	// 3. Config
	dir := configDir()
	if removed := removeDir(dir); removed {
		fmt.Printf("removed config (%s)\n", dir)
	}

	// 4. Binary
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		if err := os.Remove(exe); err == nil {
			fmt.Printf("removed binary (%s)\n", exe)
		}
	}

	fmt.Println("chop uninstalled")
}

// Reset clears data (tracking DB, audit log, tee files) but keeps config, hook, and binary.
func Reset() {
	dir := dataDir()

	if removeFile(filepath.Join(dir, "tracking.db")) {
		fmt.Println("cleared tracking database")
	}

	if removeFile(filepath.Join(dir, "hook-audit.log")) {
		fmt.Println("cleared audit log")
	}
}

// dataDir returns the chop data directory (~/.local/share/chop/).
func dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "chop")
}

// configDir returns the chop config directory, respecting XDG_CONFIG_HOME.
func configDir() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "chop")
}

// removeDir removes a directory and returns true if it existed.
func removeDir(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return os.RemoveAll(path) == nil
}

// removeFile removes a single file and returns true if it existed.
func removeFile(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return os.Remove(path) == nil
}
