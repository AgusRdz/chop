package cleanup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AgusRdz/chop/config"
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

func dataDir() string {
	return config.DataDir()
}

func configDir() string {
	return config.ConfigDir()
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
