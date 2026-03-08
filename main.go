package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/AgusRdz/chop/cleanup"
	"github.com/AgusRdz/chop/config"
	"github.com/AgusRdz/chop/filters"
	"github.com/AgusRdz/chop/hooks"

	"github.com/AgusRdz/chop/tracking"
	"github.com/AgusRdz/chop/updater"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--help", "help", "-h":
		printHelp()
		return
	case "--version", "version":
		fmt.Printf("chop %s\n", version)
		return
	case "--post-update-check":
		checkInstallDir()
		return
	case "update":
		updater.Run(version)
		return
	case "gain":
		runGain(os.Args[2:])
		return
	case "capture":
		runCapture(os.Args[2:])
		return
	case "config":
		runConfig()
		return
	case "hook":
		hooks.RunHook()
		return
	case "hook-audit":
		runHookAudit(os.Args[2:])
		return
	case "uninstall":
		keepData := false
		for _, a := range os.Args[2:] {
			if a == "--keep-data" {
				keepData = true
			}
		}
		cleanup.Uninstall(keepData)
		return
	case "reset":
		cleanup.Reset()
		return
	case "init":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: chop init <--global|--uninstall|--status>")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "--global", "-g":
			hooks.Install()
		case "--uninstall":
			hooks.Uninstall()
		case "--status":
			installed, path := hooks.IsInstalled()
			if installed {
				fmt.Printf("chop hook is installed (%s)\n", path)
			} else {
				fmt.Printf("chop hook is NOT installed\n")
				fmt.Println("run 'chop init --global' to install")
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown flag %q\nusage: chop init <--global|--uninstall|--status>\n", os.Args[2])
			os.Exit(1)
		}
		return
	}

	cfg := config.Load()

	command := os.Args[1]
	args := os.Args[2:]

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			fmt.Fprintf(os.Stderr, "chop: failed to run %s: %v\n", command, err)
			os.Exit(1)
		}
	}

	raw := string(output)
	fullCmd := command
	if len(args) > 0 {
		fullCmd = command + " " + strings.Join(args, " ")
	}

	// Skip filtering if command is disabled in config
	var finalOutput string
	if cfg.IsDisabled(command) {
		finalOutput = raw
	} else {
		filter := filters.Get(command, args)
		if filter != nil {
			filtered, ferr := filter(raw)
			if ferr != nil {
				finalOutput = raw
			} else {
				finalOutput = filtered
			}
		} else {
			// Auto-detect compression for unrecognized commands
			autoFiltered, aerr := filters.AutoDetect(raw)
			if aerr != nil || autoFiltered == raw {
				finalOutput = raw
			} else {
				finalOutput = autoFiltered
			}
		}
	}

	fmt.Print(finalOutput)
	trackSilent(fullCmd, raw, finalOutput)

	os.Exit(exitCode)
}

func trackSilent(command, raw, filtered string) {
	rawTokens := tracking.CountTokens(raw)
	filteredTokens := tracking.CountTokens(filtered)
	_ = tracking.Track(command, rawTokens, filteredTokens)
}

func runCapture(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: chop capture <command> [args...]")
		os.Exit(1)
	}

	command := args[0]
	cmdArgs := args[1:]

	cmd := exec.Command(command, cmdArgs...)
	cmd.Stdin = os.Stdin

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			fmt.Fprintf(os.Stderr, "chop: failed to run %s: %v\n", command, err)
			os.Exit(1)
		}
	}

	raw := string(output)

	// Build fixture name
	subcommand := ""
	if len(cmdArgs) > 0 {
		subcommand = cmdArgs[0]
	}
	ts := time.Now().Format("20060102-150405")
	var baseName string
	if subcommand != "" {
		baseName = fmt.Sprintf("%s-%s-%s", command, subcommand, ts)
	} else {
		baseName = fmt.Sprintf("%s-%s", command, ts)
	}

	fixtureDir := filepath.Join("tests", "fixtures")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to create fixtures dir: %v\n", err)
		os.Exit(1)
	}

	rawPath := filepath.Join(fixtureDir, baseName+".txt")
	if err := os.WriteFile(rawPath, []byte(raw), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to write raw fixture: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "raw:      %s\n", rawPath)

	// Apply filter if available
	filter := filters.Get(command, cmdArgs)
	if filter != nil {
		filtered, ferr := filter(raw)
		if ferr == nil {
			filteredPath := filepath.Join(fixtureDir, baseName+".filtered.txt")
			if err := os.WriteFile(filteredPath, []byte(filtered), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "chop: failed to write filtered fixture: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "filtered: %s\n", filteredPath)
			}
			fmt.Print(filtered)
		} else {
			fmt.Print(raw)
		}
	} else {
		fmt.Print(raw)
	}

	os.Exit(exitCode)
}

func runConfig() {
	path := config.Path()
	fmt.Printf("config: %s\n", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("(no config file found)")
		} else {
			fmt.Fprintf(os.Stderr, "chop: failed to read config: %v\n", err)
		}
		return
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		fmt.Println("(config file is empty)")
	} else {
		fmt.Println(content)
	}
}

func runGain(args []string) {
	var showHistory, showSummary bool
	for _, a := range args {
		switch a {
		case "--history":
			showHistory = true
		case "--summary":
			showSummary = true
		}
	}

	if showHistory {
		records, err := tracking.GetHistory(20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read history: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(tracking.FormatHistory(records))
		return
	}

	if showSummary {
		summaries, err := tracking.GetCommandSummary()
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read summary: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(tracking.FormatSummary(summaries))
		return
	}

	stats, err := tracking.GetStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to read stats: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(tracking.FormatGain(stats))
}

func runHookAudit(args []string) {
	clearFlag := false
	for _, a := range args {
		if a == "--clear" {
			clearFlag = true
		}
	}

	logPath, err := hooks.AuditLogPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "chop: cannot determine audit log path")
		os.Exit(1)
	}

	if clearFlag {
		if err := os.Truncate(logPath, 0); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("audit log already empty")
				return
			}
			fmt.Fprintf(os.Stderr, "chop: failed to clear audit log: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("audit log cleared")
		return
	}

	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("no hook audit log yet")
			return
		}
		fmt.Fprintf(os.Stderr, "chop: failed to read audit log: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Read all lines, keep last 20
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		fmt.Println("no hook audit log entries")
		return
	}

	start := 0
	if len(lines) > 20 {
		start = len(lines) - 20
	}
	for _, line := range lines[start:] {
		fmt.Println(line)
	}
}


func checkInstallDir() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	oldDir := filepath.Join(home, "bin")
	if !strings.HasPrefix(exe, oldDir+string(filepath.Separator)) && exe != filepath.Join(oldDir, "chop") && exe != filepath.Join(oldDir, "chop.exe") {
		return
	}

	fmt.Println("")
	fmt.Println("note: chop is installed in ~/bin, which is no longer the recommended location.")

	if runtime.GOOS == "windows" {
		fmt.Println("run the migration script to move it to %LOCALAPPDATA%\\Programs\\chop:")
		fmt.Println("")
		fmt.Println("  irm https://raw.githubusercontent.com/AgusRdz/chop/main/migrate.ps1 | iex")
	} else {
		fmt.Println("run the migration script to move it to ~/.local/bin:")
		fmt.Println("")
		fmt.Println("  curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/migrate.sh | sh")
	}
}

func printHelp() {
	fmt.Printf(`chop %s — CLI output compressor for Claude Code

Usage:
  chop <command> [args...]    Run command and compress output
  chop <subcommand>           Run a chop subcommand

Subcommands:
  gain                        Show token savings stats
  gain --history              Recent commands with savings
  gain --summary              Per-command savings breakdown
  config                      Show config file path and contents
  init --global               Install Claude Code hook (~/.claude/settings.json)
  init --uninstall            Remove Claude Code hook
  init --status               Check if hook is installed
  hook-audit                  Show last 20 hook rewrite log entries
  hook-audit --clear          Clear the hook audit log
  uninstall                   Remove everything: hook, data, config, binary
  uninstall --keep-data       Uninstall but preserve tracking history
  reset                       Clear data (tracking, audit log) — keep installation
  update                      Update to the latest version
  --post-update-check         Check install location after an update (called automatically by update)
  help                        Show this help
  version                     Show version

Claude Code integration:
  chop init --global          Register PreToolUse hook for Claude Code
  chop init --uninstall       Remove the hook
  chop init --status          Check hook installation status

Config (%s):
  disabled: [cmd1, cmd2]      Skip filtering, return full output for these commands

Examples:
  chop git status             Compressed git status
  chop docker ps              Compact container list
  chop kubectl get pods       Filtered pod table
  chop curl https://api.io    Auto-compressed JSON response
`, version, config.Path())
}
