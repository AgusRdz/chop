package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/AgusRdz/chop/cleanup"
	"github.com/AgusRdz/chop/config"
	"github.com/AgusRdz/chop/filters"
	"github.com/AgusRdz/chop/hooks"
	"github.com/AgusRdz/chop/tracking"
	"github.com/AgusRdz/chop/updater"
)

//go:embed CHANGELOG.md
var changelog string

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	// Apply any pending auto-update from a previous run
	updater.ApplyPendingUpdate(version)

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
	case "changelog", "--changelog":
		runChangelog(os.Args[2:])
		return
	case "--post-update-check":
		checkInstallDir()
		return
	case "--_bg-update":
		if len(os.Args) >= 3 {
			updater.RunBackgroundUpdate(os.Args[2])
		}
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
		runConfig(os.Args[2:])
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
	case "disable":
		if hooks.IsDisabledGlobally() {
			fmt.Println("chop is already disabled")
			return
		}
		if err := hooks.Disable(); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to disable: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("chop disabled — hook will pass through all commands")
		fmt.Println("run 'chop enable' to resume")
		return
	case "enable":
		if !hooks.IsDisabledGlobally() {
			fmt.Println("chop is already enabled")
			return
		}
		if err := hooks.Enable(); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to enable: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("chop enabled — hook will compress commands again")
		return
	case "doctor":
		runDoctor()
		return
	case "filter":
		runFilter(os.Args[2:])
		return
	case "local":
		runLocal(os.Args[2:])
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

	command := os.Args[1]
	args := os.Args[2:]

	// Load config: global + local overlay from cwd
	cwd, _ := os.Getwd()
	cfg := config.LoadWithLocal(cwd)

	// Load user-defined custom filters
	filters.SetUserFilters(config.LoadCustomFiltersWithLocal(cwd))

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
	subCmd := ""
	if len(args) > 0 {
		subCmd = args[0]
	}
	var finalOutput string
	if cfg.IsDisabled(command, subCmd) {
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

	// Check for updates in background (every 24h, downloads silently)
	updater.BackgroundCheck(version)

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

func runConfig(args []string) {
	if len(args) > 0 && args[0] == "init" {
		initConfig()
		return
	}

	path := config.Path()
	fmt.Printf("config: %s\n", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("(no config file found)")
			fmt.Println("\nrun 'chop config init' to create a starter config")
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

func initConfig() {
	path := config.Path()

	if _, err := os.Stat(path); err == nil {
		fmt.Printf("config already exists: %s\n", path)
		return
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to create config dir: %v\n", err)
		os.Exit(1)
	}

	starter := `# Global chop config
# Docs: https://github.com/AgusRdz/chop#configuration
#
# Disable built-in filters for specific commands globally.
# Use "chop local add" to disable per-project instead.
#
# disabled: ["git diff", "docker ps"]

disabled: []
`

	if err := os.WriteFile(path, []byte(starter), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to write config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("created: %s\n", path)
	fmt.Println("edit this file to disable built-in filters globally")
}

func runGain(args []string) {
	var showHistory, showSummary, showUnchopped, verbose bool
	var skipCmd, unskipCmd, deleteCmd, noTrackCmd, resumeTrackCmd string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--history":
			showHistory = true
		case "--summary":
			showSummary = true
		case "--unchopped":
			showUnchopped = true
		case "--verbose", "-v":
			verbose = true
			showUnchopped = true
		case "--skip":
			if i+1 < len(args) {
				i++
				skipCmd = args[i]
				showUnchopped = true
			}
		case "--unskip":
			if i+1 < len(args) {
				i++
				unskipCmd = args[i]
				showUnchopped = true
			}
		case "--delete":
			if i+1 < len(args) {
				i++
				deleteCmd = args[i]
			}
		case "--no-track":
			if i+1 < len(args) {
				i++
				noTrackCmd = args[i]
			}
		case "--resume-track":
			if i+1 < len(args) {
				i++
				resumeTrackCmd = args[i]
			}
		}
	}

	if deleteCmd != "" {
		if err := tracking.DeleteCommand(deleteCmd); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to delete command: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("deleted all records for %q\n", deleteCmd)
		return
	}

	if noTrackCmd != "" {
		if err := tracking.DeleteCommand(noTrackCmd); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to delete records: %v\n", err)
			os.Exit(1)
		}
		if err := tracking.AddTrackingSkip(noTrackCmd); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to add no-track entry: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%q removed from history and will no longer be tracked\n", noTrackCmd)
		return
	}

	if resumeTrackCmd != "" {
		if err := tracking.RemoveTrackingSkip(resumeTrackCmd); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to remove no-track entry: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%q will be tracked again\n", resumeTrackCmd)
		return
	}

	if showUnchopped {
		if skipCmd != "" {
			if err := tracking.SkipUnchopped(skipCmd); err != nil {
				fmt.Fprintf(os.Stderr, "chop: failed to skip command: %v\n", err)
				os.Exit(1)
			}
		}
		if unskipCmd != "" {
			if err := tracking.UnskipUnchopped(unskipCmd); err != nil {
				fmt.Fprintf(os.Stderr, "chop: failed to unskip command: %v\n", err)
				os.Exit(1)
			}
		}
		summaries, err := tracking.GetUnchopped()
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read unchopped: %v\n", err)
			os.Exit(1)
		}
		// Auto-exclude commands that already have a registered filter -
		// they're just not compressing for this specific invocation (stale data).
		var candidates, filteredCmds []tracking.UnchoppedSummary
		for _, s := range summaries {
			parts := strings.Fields(s.Command)
			if len(parts) > 0 && filters.HasFilter(parts[0], parts[1:]) {
				filteredCmds = append(filteredCmds, s)
				continue
			}
			candidates = append(candidates, s)
		}
		skipped, err := tracking.GetSkippedCommands()
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read skip list: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(tracking.FormatUnchopped(candidates, skipped, filteredCmds, verbose))
		return
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

const localConfigFile = ".chop.yml"

func runLocal(args []string) {
	if len(args) == 0 {
		showLocalConfig()
		return
	}

	switch args[0] {
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop local add <command>")
			os.Exit(1)
		}
		localAdd(args[1:])
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop local remove <command>")
			os.Exit(1)
		}
		localRemove(args[1:])
	case "clear":
		localClear()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\nusage: chop local [add|remove|clear]\n", args[0])
		os.Exit(1)
	}
}

func showLocalConfig() {
	data, err := os.ReadFile(localConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("no local config (.chop.yml)")
			return
		}
		fmt.Fprintf(os.Stderr, "chop: failed to read %s: %v\n", localConfigFile, err)
		os.Exit(1)
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		fmt.Println("local config is empty")
	} else {
		fmt.Println(content)
	}
}

func localAdd(commands []string) {
	cfg := config.LoadFrom(localConfigFile)

	for _, cmd := range commands {
		// Skip duplicates
		found := false
		for _, d := range cfg.Disabled {
			if strings.EqualFold(d, cmd) {
				found = true
				break
			}
		}
		if !found {
			cfg.Disabled = append(cfg.Disabled, cmd)
		}
	}

	writeLocalConfig(cfg.Disabled)
	ensureGitignore()

	for _, cmd := range commands {
		fmt.Printf("disabled: %s\n", cmd)
	}
}

func localRemove(commands []string) {
	cfg := config.LoadFrom(localConfigFile)

	for _, cmd := range commands {
		for i, d := range cfg.Disabled {
			if strings.EqualFold(d, cmd) {
				cfg.Disabled = append(cfg.Disabled[:i], cfg.Disabled[i+1:]...)
				break
			}
		}
	}

	if len(cfg.Disabled) == 0 {
		localClear()
		return
	}

	writeLocalConfig(cfg.Disabled)
	for _, cmd := range commands {
		fmt.Printf("enabled: %s\n", cmd)
	}
}

func localClear() {
	if err := os.Remove(localConfigFile); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("no local config to clear")
			return
		}
		fmt.Fprintf(os.Stderr, "chop: failed to remove %s: %v\n", localConfigFile, err)
		os.Exit(1)
	}
	fmt.Println("local config removed")
}

func writeLocalConfig(disabled []string) {
	var b strings.Builder
	b.WriteString("disabled: [")
	for i, d := range disabled {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%q", d))
	}
	b.WriteString("]\n")

	if err := os.WriteFile(localConfigFile, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to write %s: %v\n", localConfigFile, err)
		os.Exit(1)
	}
}

func ensureGitignore() {
	const gitignorePath = ".gitignore"
	entries := []string{localConfigFile}

	data, _ := os.ReadFile(gitignorePath)
	content := string(data)

	var toAdd []string
	for _, entry := range entries {
		if !strings.Contains(content, entry) {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) == 0 {
		return
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return // silent - don't break the command for .gitignore issues
	}
	defer f.Close()

	// Add spacing before the chop section
	if len(content) > 0 && content[len(content)-1] != '\n' {
		fmt.Fprintln(f)
	}
	if len(content) > 0 {
		fmt.Fprintln(f)
	}
	fmt.Fprintln(f, "# chop")
	for _, entry := range toAdd {
		fmt.Fprintln(f, entry)
	}
	fmt.Printf("added %s to .gitignore\n", strings.Join(toAdd, ", "))
}

func runFilter(args []string) {
	if len(args) == 0 {
		showFilters()
		return
	}

	switch args[0] {
	case "path":
		fmt.Println(config.FiltersConfigPath())
	case "init":
		local := len(args) > 1 && args[1] == "--local"
		initFiltersConfig(local)
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop filter add <command> [--keep p1,p2] [--drop p1,p2] [--head N] [--tail N] [--exec script] [--local]")
			os.Exit(1)
		}
		filterAdd(args[1:])
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop filter remove <command> [--local]")
			os.Exit(1)
		}
		filterRemove(args[1:])
	case "test":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop filter test <command> [subcommand]")
			os.Exit(1)
		}
		testFilter(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\nusage: chop filter [path|init|add|remove|test]\n", args[0])
		os.Exit(1)
	}
}

func showFilters() {
	path := config.FiltersConfigPath()
	fmt.Printf("config: %s\n\n", path)

	filters := config.LoadCustomFilters()
	if len(filters) == 0 {
		fmt.Println("no custom filters defined")
		fmt.Println("\nrun 'chop filter init' to create a global config")
		fmt.Println("run 'chop filter init --local' to create a project-level config")
		return
	}

	for cmd, cf := range filters {
		fmt.Printf("  %s\n", cmd)
		if len(cf.Keep) > 0 {
			fmt.Printf("    keep: %v\n", cf.Keep)
		}
		if len(cf.Drop) > 0 {
			fmt.Printf("    drop: %v\n", cf.Drop)
		}
		if cf.Head > 0 {
			fmt.Printf("    head: %d\n", cf.Head)
		}
		if cf.Tail > 0 {
			fmt.Printf("    tail: %d\n", cf.Tail)
		}
		if cf.Exec != "" {
			fmt.Printf("    exec: %s\n", cf.Exec)
		}
	}
}

func initFiltersConfig(local bool) {
	var path string
	if local {
		cwd, _ := os.Getwd()
		path = filepath.Join(cwd, ".chop-filters.yml")
	} else {
		path = config.FiltersConfigPath()
	}

	// Don't overwrite existing config
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("config already exists: %s\n", path)
		return
	}

	if !local {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to create config dir: %v\n", err)
			os.Exit(1)
		}
	}

	starter := `# Custom chop filters - user-defined output compression rules
# Docs: https://github.com/AgusRdz/chop#custom-filters
#
# Each filter matches a command (or "command subcommand") and applies rules:
#   keep: [regex...]   - only keep lines matching at least one pattern
#   drop: [regex...]   - remove lines matching any pattern
#   head: N            - keep first N lines (after keep/drop)
#   tail: N            - keep last N lines (after keep/drop)
#   exec: script       - pipe output through an external script
#
# Examples:
#
# filters:
#   "myctl deploy":
#     keep: ["ERROR", "WARN", "deployed", "^="]
#     drop: ["DEBUG", "^\\s*$"]
#
#   "ansible-playbook":
#     keep: ["^PLAY", "^TASK", "fatal", "changed", "^\\s+ok="]
#     tail: 20
#
#   "custom-tool":
#     exec: "~/.config/chop/scripts/custom-tool.sh"

filters: {}
`

	if err := os.WriteFile(path, []byte(starter), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to write config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("created: %s\n", path)
	fmt.Println("edit this file to add your custom filters")
}

// filterConfigPath returns the target filters file path based on the --local flag.
func filterConfigPath(local bool) string {
	if local {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, ".chop-filters.yml")
	}
	return config.FiltersConfigPath()
}

// writeFilters writes the filters map to path in a clean, human-readable YAML format.
// Uses inline arrays and omits zero/empty fields.
func writeFilters(path string, filters map[string]config.CustomFilter) {
	var sb strings.Builder
	sb.WriteString("filters:\n")

	// Sort keys for stable output
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, cmd := range keys {
		cf := filters[cmd]
		sb.WriteString(fmt.Sprintf("  %q:\n", cmd))
		if len(cf.Keep) > 0 {
			sb.WriteString(fmt.Sprintf("    keep: %s\n", inlineStringSlice(cf.Keep)))
		}
		if len(cf.Drop) > 0 {
			sb.WriteString(fmt.Sprintf("    drop: %s\n", inlineStringSlice(cf.Drop)))
		}
		if cf.Head > 0 {
			sb.WriteString(fmt.Sprintf("    head: %d\n", cf.Head))
		}
		if cf.Tail > 0 {
			sb.WriteString(fmt.Sprintf("    tail: %d\n", cf.Tail))
		}
		if cf.Exec != "" {
			sb.WriteString(fmt.Sprintf("    exec: %q\n", cf.Exec))
		}
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to write %s: %v\n", path, err)
		os.Exit(1)
	}
}

// inlineStringSlice formats a string slice as a YAML inline array: ["a", "b", "c"]
func inlineStringSlice(s []string) string {
	quoted := make([]string, len(s))
	for i, v := range s {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func filterAdd(args []string) {
	// First arg is the command name; rest are flags
	cmdName := args[0]
	rest := args[1:]

	var keep, drop []string
	var head, tail int
	var exec string
	local := false

	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--keep":
			if i+1 >= len(rest) {
				fmt.Fprintln(os.Stderr, "chop: --keep requires a value")
				os.Exit(1)
			}
			i++
			keep = strings.Split(rest[i], ",")
		case "--drop":
			if i+1 >= len(rest) {
				fmt.Fprintln(os.Stderr, "chop: --drop requires a value")
				os.Exit(1)
			}
			i++
			drop = strings.Split(rest[i], ",")
		case "--head":
			if i+1 >= len(rest) {
				fmt.Fprintln(os.Stderr, "chop: --head requires a value")
				os.Exit(1)
			}
			i++
			fmt.Sscanf(rest[i], "%d", &head)
		case "--tail":
			if i+1 >= len(rest) {
				fmt.Fprintln(os.Stderr, "chop: --tail requires a value")
				os.Exit(1)
			}
			i++
			fmt.Sscanf(rest[i], "%d", &tail)
		case "--exec":
			if i+1 >= len(rest) {
				fmt.Fprintln(os.Stderr, "chop: --exec requires a value")
				os.Exit(1)
			}
			i++
			exec = rest[i]
		case "--local":
			local = true
		default:
			fmt.Fprintf(os.Stderr, "chop: unknown flag %q\n", rest[i])
			os.Exit(1)
		}
	}

	if len(keep) == 0 && len(drop) == 0 && head == 0 && tail == 0 && exec == "" {
		fmt.Fprintln(os.Stderr, "chop: filter add requires at least one rule (--keep, --drop, --head, --tail, --exec)")
		os.Exit(1)
	}

	path := filterConfigPath(local)

	if !local {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to create config dir: %v\n", err)
			os.Exit(1)
		}
	}

	existing := config.LoadCustomFiltersFrom(path)
	if existing == nil {
		existing = make(map[string]config.CustomFilter)
	}

	existing[cmdName] = config.CustomFilter{
		Keep: keep,
		Drop: drop,
		Head: head,
		Tail: tail,
		Exec: exec,
	}

	writeFilters(path, existing)
	fmt.Printf("filter added: %s (%s)\n", cmdName, path)
}

func filterRemove(args []string) {
	cmdName := args[0]
	local := len(args) > 1 && args[1] == "--local"

	path := filterConfigPath(local)
	existing := config.LoadCustomFiltersFrom(path)
	if existing == nil {
		fmt.Fprintf(os.Stderr, "no filters found in %s\n", path)
		os.Exit(1)
	}

	if _, ok := existing[cmdName]; !ok {
		fmt.Fprintf(os.Stderr, "filter not found: %s\n", cmdName)
		os.Exit(1)
	}

	delete(existing, cmdName)
	writeFilters(path, existing)
	fmt.Printf("filter removed: %s\n", cmdName)
}

func testFilter(args []string) {
	command := args[0]
	subArgs := args[1:]

	cwd, _ := os.Getwd()
	customFilters := config.LoadCustomFiltersWithLocal(cwd)
	cf := config.LookupCustomFilter(customFilters, command, subArgs)

	if cf == nil {
		fmt.Fprintf(os.Stderr, "no custom filter found for %q\n", strings.Join(args, " "))
		os.Exit(1)
	}

	fn := filters.BuildUserFilter(cf)
	if fn == nil {
		fmt.Fprintf(os.Stderr, "filter definition is empty for %q\n", strings.Join(args, " "))
		os.Exit(1)
	}

	// Read stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to read stdin: %v\n", err)
		os.Exit(1)
	}

	result, err := fn(string(input))
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: filter error: %v\n", err)
		fmt.Print(string(input))
		os.Exit(1)
	}

	fmt.Print(result)
}

func runChangelog(args []string) {
	if changelog == "" {
		fmt.Println("no changelog available")
		return
	}

	// --full: show entire history
	if len(args) > 0 && (args[0] == "--full" || args[0] == "-f") {
		fmt.Print(changelog)
		return
	}

	// Default: show only the latest version's section
	fmt.Print(extractLatestVersion(changelog))
}

// extractLatestVersion extracts the first version section from the changelog.
func extractLatestVersion(cl string) string {
	lines := strings.Split(cl, "\n")
	var result []string
	inSection := false

	for _, line := range lines {
		if strings.HasPrefix(line, "## [") {
			if inSection {
				break // hit the next version, stop
			}
			if strings.HasPrefix(line, "## [Unreleased]") {
				continue // skip unreleased section
			}
			inSection = true
		}
		if inSection {
			result = append(result, line)
		}
	}

	if len(result) == 0 {
		return cl
	}
	return strings.Join(result, "\n") + "\n"
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

func runDoctor() {
	issues := 0

	// 1. Check if hook is installed
	installed, _ := hooks.IsInstalled()
	if !installed {
		fmt.Println("[!] hook is not installed")
		fmt.Println("    fix: chop init --global")
		issues++
	} else {
		// 2. Check if hook path matches current binary
		hookCmd := hooks.GetHookCommand()
		expectedCmd, err := buildExpectedHookCmd()
		if err == nil && hookCmd != expectedCmd {
			fmt.Println("[!] hook points to wrong binary")
			fmt.Printf("    current: %s\n", hookCmd)
			fmt.Printf("    expected: %s\n", expectedCmd)
			fmt.Println("    fixing...")
			hooks.Install()
			issues++
		} else {
			fmt.Println("[ok] hook is installed and path is correct")
		}
	}

	// 3. Check if binary is in legacy ~/bin
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		home, herr := os.UserHomeDir()
		if herr == nil {
			oldDir := filepath.Join(home, "bin")
			if strings.HasPrefix(exe, oldDir+string(filepath.Separator)) {
				fmt.Println("[!] binary is in legacy ~/bin location")
				if runtime.GOOS == "windows" {
					fmt.Println("    fix: irm https://raw.githubusercontent.com/AgusRdz/chop/main/migrate.ps1 | iex")
				} else {
					fmt.Println("    fix: curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/migrate.sh | sh")
				}
				issues++
			}
		}
	}

	// 4. Check if chop is disabled
	if hooks.IsDisabledGlobally() {
		fmt.Println("[!] chop is disabled — hook is passing through all commands")
		fmt.Println("    fix: chop enable")
		issues++
	}

	if issues == 0 {
		fmt.Println("\nall good!")
	} else {
		fmt.Printf("\n%d issue(s) found\n", issues)
	}
}

func buildExpectedHookCmd() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	exe = strings.ReplaceAll(exe, "\\", "/")
	return fmt.Sprintf(`"%s" hook`, exe), nil
}

func printHelp() {
	fmt.Printf(`chop %s - CLI output compressor for Claude Code

Usage:
  chop <command> [args...]    Run command and compress output
  chop <subcommand>           Run a chop subcommand

Subcommands:
  gain                        Show token savings stats
  gain --history              Recent commands with savings
  gain --summary              Per-command savings breakdown
  gain --unchopped            Commands never compressed (new filter candidates)
  gain --unchopped --skip X   Mark command X as intentionally not needing a filter
  gain --unchopped --unskip X Remove command X from the skip list
  gain --delete X             Permanently delete all tracking records for command X
  gain --no-track X           Delete records for X and never track it again
  gain --resume-track X       Re-enable tracking for a previously ignored command
  config                      Show global config path and contents
  config init                 Create a starter global config.yml
  init --global               Install Claude Code hook (~/.claude/settings.json)
  init --uninstall            Remove Claude Code hook
  init --status               Check if hook is installed
  hook-audit                  Show last 20 hook rewrite log entries
  hook-audit --clear          Clear the hook audit log
  uninstall                   Remove everything: hook, data, config, binary
  uninstall --keep-data       Uninstall but preserve tracking history
  reset                       Clear data (tracking, audit log) - keep installation
  filter                      List custom user-defined filters
  filter path                 Show filters config file path
  filter init                 Create a starter ~/.config/chop/filters.yml
  filter init --local         Create a starter .chop-filters.yml in current dir
  filter add <cmd> [flags]    Add or update a filter (--keep, --drop, --head, --tail, --exec, --local)
  filter remove <cmd>         Remove a filter (--local for project-level)
  filter test <cmd>           Test a custom filter (reads stdin)
  local                       Show local project config (.chop.yml)
  local add "git diff"        Disable a command in this project
  local remove "git diff"     Re-enable a command in this project
  local clear                 Remove local config
  disable                     Bypass chop — hook passes through all commands
  enable                      Resume chop — hook compresses commands again
  doctor                      Check and fix common issues (hook path, install location)
  changelog                   Show changes in the current version
  changelog --full            Show full changelog history
  update                      Update to the latest version
  --post-update-check         Check install location after an update (called automatically by update)
  help                        Show this help
  version                     Show version

Claude Code integration:
  chop init --global          Register PreToolUse hook for Claude Code
  chop init --uninstall       Remove the hook
  chop init --status          Check hook installation status

Config (%s):
  disabled: [cmd1, "git diff"]  Skip filtering for commands (supports subcommands)

Local config (.chop.yml in project dir - managed via chop local):
  disabled: ["git diff"]        Overrides global disabled list for this project

Custom filters (%s):
  Define your own output compression rules for any command.
  Run 'chop filter init' to create a starter config with examples.

  Rules (applied in order):
    keep: [regex...]   Only keep lines matching at least one pattern
    drop: [regex...]   Remove lines matching any pattern
    head: N            Keep first N lines (after keep/drop)
    tail: N            Keep last N lines (after keep/drop)
    exec: script       Pipe output through an external script

  Test with: echo "sample output" | chop filter test <command>

Examples:
  chop git status             Compressed git status
  chop docker ps              Compact container list
  chop kubectl get pods       Filtered pod table
  chop curl https://api.io    Auto-compressed JSON response
  chop cat app.log            Pattern-grouped log lines with repeat counts
  chop tail -f app.log        Same, for streaming log files
`, version, config.Path(), config.FiltersConfigPath())
}
