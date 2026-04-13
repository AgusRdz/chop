package main

import (
	"bufio"
	"encoding/json"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	// Show hint if a newer version is available and auto-update is off
	updater.NotifyIfUpdateAvailable(version)

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
	case "--agent-info", "agent-info":
		runAgentInfo()
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
		_ = config.WriteDiscoveryInfo(version)
		return
	case "auto-update":
		runAutoUpdate(os.Args[2:])
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
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "--gemini":
				hooks.RunGeminiHook()
			case "--codex":
				hooks.RunCodexHook()
			case "--antigravity":
				hooks.RunAntigravityHook()
			default:
				hooks.RunHook()
			}
		} else {
			hooks.RunHook()
		}
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
	case "fix-hooks":
		runFixHooks()
		return
	case "unwrap-hooks":
		runUnwrapHooks()
		return
	case "filter":
		runFilter(os.Args[2:])
		return
	case "local":
		runLocal(os.Args[2:])
		return
	case "global":
		runGlobal(os.Args[2:])
		return
	case "list":
		runList()
		return
	case "diff":
		runDiff(os.Args[2:])
		return
	case "completion":
		runCompletion(os.Args[2:])
		return
	case "init", "setup":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: chop init <--global|--gemini|--codex|--antigravity|--uninstall|--status|--agent-handshake>")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "--agent-handshake":
			exe, err := os.Executable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "chop: failed to get executable path: %v\n", err)
				os.Exit(1)
			}
			exe, err = filepath.EvalSymlinks(exe)
			if err != nil {
				fmt.Fprintf(os.Stderr, "chop: failed to resolve symlinks: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("AGENT DISCOVERY: chop is installed at %s. I have saved this to my persistent memory. I will now use this path for all CLI compression tasks.\n", exe)
		case "--global", "-g":
			hooks.Install(version)
		case "--gemini":
			if len(os.Args) > 3 {
				switch os.Args[3] {
				case "--uninstall":
					hooks.GeminiUninstall()
				case "--status":
					installed, path := hooks.GeminiIsInstalled()
					if installed {
						fmt.Printf("chop Gemini CLI hook is installed (%s)\n", path)
					} else {
						fmt.Printf("chop Gemini CLI hook is NOT installed\n")
						fmt.Println("run 'chop init --gemini' to install")
					}
				default:
					fmt.Fprintf(os.Stderr, "unknown flag %q\nusage: chop init --gemini [--uninstall|--status]\n", os.Args[3])
					os.Exit(1)
				}
			} else {
				hooks.GeminiInstall(version)
			}
		case "--codex":
			if len(os.Args) > 3 {
				switch os.Args[3] {
				case "--uninstall":
					hooks.CodexUninstall()
				case "--status":
					installed, path := hooks.CodexIsInstalled()
					if installed {
						fmt.Printf("chop Codex CLI hook is installed (%s)\n", path)
					} else {
						fmt.Printf("chop Codex CLI hook is NOT installed\n")
						fmt.Println("run 'chop init --codex' to install")
					}
				default:
					fmt.Fprintf(os.Stderr, "unknown flag %q\nusage: chop init --codex [--uninstall|--status]\n", os.Args[3])
					os.Exit(1)
				}
			} else {
				hooks.CodexInstall(version)
			}
		case "--antigravity":
			if len(os.Args) > 3 {
				switch os.Args[3] {
				case "--uninstall":
					hooks.AntigravityUninstall()
				case "--status":
					installed, path := hooks.AntigravityIsInstalled()
					if installed {
						fmt.Printf("chop Antigravity IDE hook is installed (%s)\n", path)
					} else {
						fmt.Printf("chop Antigravity IDE hook is NOT installed\n")
						fmt.Println("run 'chop init --antigravity' to install")
					}
				default:
					fmt.Fprintf(os.Stderr, "unknown flag %q\nusage: chop init --antigravity [--uninstall|--status]\n", os.Args[3])
					os.Exit(1)
				}
			} else {
				hooks.AntigravityInstall(version)
			}
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
			gInstalled, gPath := hooks.GeminiIsInstalled()
			if gInstalled {
				fmt.Printf("chop Gemini CLI hook is installed (%s)\n", gPath)
			}
			cInstalled, cPath := hooks.CodexIsInstalled()
			if cInstalled {
				fmt.Printf("chop Codex CLI hook is installed (%s)\n", cPath)
			}
			aInstalled, aPath := hooks.AntigravityIsInstalled()
			if aInstalled {
				fmt.Printf("chop Antigravity IDE hook is installed (%s)\n", aPath)
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown flag %q\nusage: chop init <--global|--gemini|--codex|--antigravity|--uninstall|--status>\n", os.Args[2])
			os.Exit(1)
		}
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	validateCommand(command)

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

	var finalOutput string
	// Never compress failed command output — error messages must be preserved in full.
	if exitCode != 0 {
		finalOutput = raw
	} else if cfg.IsDisabled(command, args...) {
		finalOutput = raw
	} else {
		filter := filters.Get(command, args)
		if filter != nil {
			filtered, ferr := filter(raw)
			if ferr != nil || len(filtered) > len(raw) {
				finalOutput = raw
			} else {
				finalOutput = filtered
			}
		} else {
			// Auto-detect compression for unrecognized commands
			autoFiltered, aerr := filters.AutoDetect(raw)
			if aerr != nil || autoFiltered == raw || len(autoFiltered) > len(raw) {
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
	_ = tracking.Track(normalizeCommandForTracking(command), rawTokens, filteredTokens)
}

// normalizeCommandForTracking strips flags that carry no semantic meaning for
// tracking purposes. Currently handles git's -C <path> flag, which only sets
// the working directory and is transparent to what the command actually does.
func normalizeCommandForTracking(command string) string {
	fields := strings.Fields(command)
	if len(fields) < 2 || fields[0] != "git" {
		return command
	}
	out := fields[:1:1] // keep "git"
	for i := 1; i < len(fields); i++ {
		if fields[i] == "-C" {
			i++ // skip the path argument too
			continue
		}
		out = append(out, fields[i])
	}
	return strings.Join(out, " ")
}

func runCapture(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: chop capture <command> [args...]")
		os.Exit(1)
	}

	command := args[0]
	cmdArgs := args[1:]

	validateCommand(command)

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

	// Reject path separators in command/subcommand to prevent path traversal in fixture filenames.
	if strings.ContainsAny(command, "/\\") || strings.ContainsAny(subcommand, "/\\") {
		fmt.Fprintln(os.Stderr, "chop: capture: command name must not contain path separators")
		os.Exit(1)
	}
	ts := time.Now().Format("20060102-150405")
	var baseName string
	safeCommand := sanitizeFilename(command)
	if subcommand != "" {
		safeSubcommand := sanitizeFilename(subcommand)
		baseName = fmt.Sprintf("%s-%s-%s", safeCommand, safeSubcommand, ts)
	} else {
		baseName = fmt.Sprintf("%s-%s", safeCommand, ts)
	}

	fixtureDir := filepath.Join("tests", "fixtures")
	if err := os.MkdirAll(fixtureDir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to create fixtures dir: %v\n", err)
		os.Exit(1)
	}

	rawPath := filepath.Join(fixtureDir, baseName+".txt")
	if err := os.WriteFile(rawPath, []byte(raw), 0o600); err != nil {
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
			if err := os.WriteFile(filteredPath, []byte(filtered), 0o600); err != nil {
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
	if len(args) == 0 {
		showConfig()
		return
	}
	switch args[0] {
	case "init":
		initConfig()
	case "edit":
		path := config.Path()
		if _, err := os.Stat(path); os.IsNotExist(err) {
			initConfig()
		}
		openInEditor(path)
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: chop config set <key> <value>")
			fmt.Fprintln(os.Stderr, "       chop config set editor vim")
			fmt.Fprintln(os.Stderr, "       chop config set history_compressed_only true")
			os.Exit(1)
		}
		configSet(args[1], args[2])
	case "export":
		configExport()
	case "import":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop config import <file>")
			os.Exit(1)
		}
		configImport(args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\nusage: chop config [init|edit|set|export|import]\n", args[0])
		os.Exit(1)
	}
}

func showConfig() {
	path := config.Path()
	fmt.Printf("%s %s\n", bold("config:"), cyan(path))

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("(no config file found)")
		fmt.Println("\nrun 'chop config init' to create a starter config")
		return
	}

	cfg := config.Load()
	if cfg.Editor != "" {
		fmt.Printf("%s %s\n", bold("editor:"), cyan(cfg.Editor))
	}
	if cfg.HistoryCompressedOnly {
		fmt.Printf("%s %s\n", bold("history_compressed_only:"), cyan("true"))
	}
	if len(cfg.Disabled) == 0 {
		fmt.Printf("%s %s\n", bold("disabled:"), dim("(none)"))
		return
	}
	fmt.Println(bold("disabled:"))
	for _, d := range cfg.Disabled {
		fmt.Printf("  %s %s\n", dim("-"), yellow(d))
	}
}

// configExport writes config.yml and filters.yml to stdout as a portable archive.
func configExport() {
	cfgPath := config.Path()
	filterPath := config.FiltersConfigPath()

	exported := false
	for _, p := range []string{cfgPath, filterPath} {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		fmt.Printf("# --- %s ---\n", filepath.Base(p))
		fmt.Println(strings.TrimSpace(string(data)))
		fmt.Println()
		exported = true
	}
	if !exported {
		fmt.Fprintln(os.Stderr, "chop: no config files found — run 'chop config init' first")
		os.Exit(1)
	}
}

// configImport reads a config file exported by configExport and writes each
// section to the appropriate destination (config.yml or filters.yml).
func configImport(src string) {
	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: cannot read %s: %v\n", src, err)
		os.Exit(1)
	}

	type section struct {
		name    string
		content strings.Builder
	}

	var sections []section
	var cur *section

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "# --- ") && strings.HasSuffix(line, " ---") {
			name := strings.TrimSuffix(strings.TrimPrefix(line, "# --- "), " ---")
			sections = append(sections, section{name: name})
			cur = &sections[len(sections)-1]
			continue
		}
		if cur != nil {
			cur.content.WriteString(line)
			cur.content.WriteByte('\n')
		}
	}

	if len(sections) == 0 {
		fmt.Fprintln(os.Stderr, "chop: no sections found — file must be created with 'chop config export'")
		os.Exit(1)
	}

	destFor := map[string]string{
		"config.yml":  config.Path(),
		"filters.yml": config.FiltersConfigPath(),
	}

	for _, s := range sections {
		dest, ok := destFor[s.name]
		if !ok {
			fmt.Fprintf(os.Stderr, "chop: unknown section %q — skipping\n", s.name)
			continue
		}
		content := strings.TrimSpace(s.content.String()) + "\n"
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to create dir for %s: %v\n", dest, err)
			os.Exit(1)
		}
		if err := os.WriteFile(dest, []byte(content), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to write %s: %v\n", dest, err)
			os.Exit(1)
		}
		fmt.Printf("imported: %s\n", dest)
	}
}

// openInEditor opens the given file in the user's preferred editor.
// Editor resolution order: config.editor → $VISUAL → $EDITOR → common editors (code, vim, nano, notepad).
func openInEditor(path string) {
	editor := config.Load().Editor
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		candidates := []string{"code", "vim", "nano", "notepad"}
		for _, e := range candidates {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		fmt.Fprintln(os.Stderr, "chop: no editor found — set $EDITOR or $VISUAL")
		fmt.Fprintf(os.Stderr, "      file is at: %s\n", path)
		os.Exit(1)
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "chop: editor exited with error: %v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	path := config.Path()

	if _, err := os.Stat(path); err == nil {
		fmt.Printf("config already exists: %s\n", path)
		return
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to create config dir: %v\n", err)
		os.Exit(1)
	}

	starter := `# Global chop config
# Docs: https://github.com/AgusRdz/chop#configuration

# Preferred editor for "chop config edit", "chop filter edit", "chop local edit".
# Falls back to $VISUAL, $EDITOR, then auto-detects (code, vim, nano, notepad).
# Run: chop config set editor <vim|code|nano|...>
# editor: code

# Only show commands that were actually compressed in "chop gain --history".
# When true, commands with 0% savings are hidden from the history view.
# history_compressed_only: false

# Disable built-in filters for specific commands globally.
# Use "chop local add" to disable per-project instead.
# Both formats are supported:
#   disabled: ["git diff", "docker ps"]
#   disabled:
#     - git diff

disabled:
  - git diff
`

	if err := os.WriteFile(path, []byte(starter), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to write config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("created: %s\n", path)
	fmt.Println("edit this file to disable built-in filters globally")
}

// configSet writes a single key-value pair to the global config file.
// If the key already exists it is updated in-place; otherwise it is appended.
// Supported keys: editor, history_compressed_only
func configSet(key, value string) {
	allowed := map[string]bool{"editor": true, "history_compressed_only": true}
	if !allowed[strings.ToLower(key)] {
		fmt.Fprintf(os.Stderr, "chop: unknown config key %q\n", key)
		fmt.Fprintln(os.Stderr, "supported keys: editor, history_compressed_only")
		os.Exit(1)
	}
	if key == "history_compressed_only" && value != "true" && value != "false" {
		fmt.Fprintf(os.Stderr, "chop: history_compressed_only must be true or false\n")
		os.Exit(1)
	}

	path := config.Path()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		initConfig()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to read config: %v\n", err)
		os.Exit(1)
	}

	newLine := key + ": " + value
	lines := strings.Split(string(data), "\n")
	updated := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		k, _, ok := strings.Cut(trimmed, ":")
		if ok && strings.TrimSpace(k) == key {
			lines[i] = newLine
			updated = true
			break
		}
	}
	if !updated {
		lines = append(lines, newLine)
	}

	out := strings.Join(lines, "\n")
	// Ensure single trailing newline
	out = strings.TrimRight(out, "\n") + "\n"

	if err := os.WriteFile(path, []byte(out), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to write config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s: %s\n", key, value)
}

func runGain(args []string) {
	var showHistory, showSummary, showUnchopped, verbose, showAll, showProjects bool
	var skipCmd, unskipCmd, deleteCmd, noTrackCmd, resumeTrackCmd, exportFormat, sinceStr, projectFilter string
	historyLimit := 20
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--history":
			showHistory = true
		case "--summary":
			showSummary = true
		case "--projects":
			showProjects = true
		case "--project":
			if i+1 < len(args) {
				i++
				projectFilter = args[i]
			}
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
		case "--export":
			if i+1 < len(args) {
				i++
				exportFormat = args[i]
			}
		case "--since":
			if i+1 < len(args) {
				i++
				sinceStr = args[i]
			}
		case "--limit":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil && n > 0 {
					historyLimit = n
				}
			}
		case "--all":
			showAll = true
		}
	}

	var sinceDuration time.Duration
	if sinceStr != "" {
		d, err := tracking.ParseSinceDuration(sinceStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: invalid duration %q — use formats like 7d, 2w, 24h, 30m\n", sinceStr)
			os.Exit(1)
		}
		sinceDuration = d
	}

	if exportFormat != "" {
		if exportFormat != "json" && exportFormat != "csv" {
			fmt.Fprintf(os.Stderr, "chop: unknown export format %q — use json or csv\n", exportFormat)
			os.Exit(1)
		}
		var records []tracking.Record
		var stats tracking.Stats
		var err error
		if sinceDuration > 0 {
			records, err = tracking.GetHistorySince(10000, sinceDuration)
		} else {
			records, err = tracking.GetHistory(10000)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read history: %v\n", err)
			os.Exit(1)
		}
		if sinceDuration > 0 {
			stats, err = tracking.GetStatsSince(sinceDuration)
		} else {
			stats, err = tracking.GetStats()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read stats: %v\n", err)
			os.Exit(1)
		}
		if exportFormat == "json" {
			if err := tracking.ExportJSON(os.Stdout, records, stats); err != nil {
				fmt.Fprintf(os.Stderr, "chop: export failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := tracking.ExportCSV(os.Stdout, records); err != nil {
				fmt.Fprintf(os.Stderr, "chop: export failed: %v\n", err)
				os.Exit(1)
			}
		}
		return
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

	if showProjects {
		summaries, err := tracking.GetProjectSummary()
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read project summary: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(tracking.FormatProjectSummary(summaries))
		return
	}

	if showHistory {
		var records []tracking.Record
		var err error
		limit := historyLimit
		if showAll {
			limit = 10000
		}
		if projectFilter != "" {
			records, err = tracking.GetHistoryByProject(projectFilter, limit)
		} else if sinceDuration > 0 {
			records, err = tracking.GetHistorySince(limit, sinceDuration)
		} else {
			records, err = tracking.GetHistory(limit)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read history: %v\n", err)
			os.Exit(1)
		}
		if config.Load().HistoryCompressedOnly {
			records = tracking.FilterCompressed(records)
		}
		fmt.Print(tracking.FormatHistory(records, verbose, tracking.IsColorEnabled()))
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

	if sinceDuration > 0 {
		stats, err := tracking.GetStatsSince(sinceDuration)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read stats: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(tracking.FormatGainSince(stats, sinceStr))
		return
	}
	stats, err := tracking.GetStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to read stats: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(tracking.FormatGain(stats))
}

func runAutoUpdate(args []string) {
	if len(args) == 0 {
		if updater.IsAutoUpdateEnabled() {
			fmt.Println("auto-update: on")
		} else {
			fmt.Println("auto-update: off")
			fmt.Println("chop will notify you when updates are available")
			fmt.Println("run 'chop auto-update on' to enable automatic updates")
		}
		return
	}

	switch args[0] {
	case "on":
		if updater.IsAutoUpdateEnabled() {
			fmt.Println("auto-update is already on")
			return
		}
		if err := updater.SetAutoUpdate(true); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to enable auto-update: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("auto-update enabled — chop will update itself in the background")
	case "off":
		if !updater.IsAutoUpdateEnabled() {
			fmt.Println("auto-update is already off")
			return
		}
		if err := updater.SetAutoUpdate(false); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to disable auto-update: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("auto-update disabled — run 'chop update' to update manually")
	default:
		fmt.Fprintf(os.Stderr, "usage: chop auto-update [on|off]\n")
		os.Exit(1)
	}
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
	case "edit":
		path := localConfigFile
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte("disabled: []\n"), 0o600); err != nil {
				fmt.Fprintf(os.Stderr, "chop: failed to create %s: %v\n", path, err)
				os.Exit(1)
			}
			fmt.Printf("created: %s\n", path)
		}
		openInEditor(path)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\nusage: chop local [add|remove|clear|edit]\n", args[0])
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

	if err := os.WriteFile(localConfigFile, []byte(b.String()), 0o600); err != nil {
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

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
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

func runGlobal(args []string) {
	if len(args) == 0 {
		showGlobalConfig()
		return
	}

	switch args[0] {
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop global add <command>")
			os.Exit(1)
		}
		globalAdd(args[1:])
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop global remove <command>")
			os.Exit(1)
		}
		globalRemove(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\nusage: chop global [add|remove]\n", args[0])
		os.Exit(1)
	}
}

func showGlobalConfig() {
	path := config.Path()
	fmt.Printf("%s %s\n", bold("config:"), cyan(path))

	cfg := config.Load()
	if len(cfg.Disabled) == 0 {
		fmt.Printf("%s %s\n", bold("disabled:"), dim("(none)"))
		return
	}
	fmt.Println(bold("disabled:"))
	for _, d := range cfg.Disabled {
		fmt.Printf("  %s %s\n", dim("-"), yellow(d))
	}
}

func globalAdd(commands []string) {
	path := config.Path()
	cfg := config.Load()

	// Ensure config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to create config dir: %v\n", err)
			os.Exit(1)
		}
	}

	for _, cmd := range commands {
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

	writeGlobalConfig(cfg.Disabled)
	for _, cmd := range commands {
		fmt.Printf("disabled globally: %s\n", cmd)
	}
}

func globalRemove(commands []string) {
	cfg := config.Load()

	for _, cmd := range commands {
		for i, d := range cfg.Disabled {
			if strings.EqualFold(d, cmd) {
				cfg.Disabled = append(cfg.Disabled[:i], cfg.Disabled[i+1:]...)
				break
			}
		}
	}

	writeGlobalConfig(cfg.Disabled)
	for _, cmd := range commands {
		fmt.Printf("enabled globally: %s\n", cmd)
	}
}

func writeGlobalConfig(disabled []string) {
	var b strings.Builder
	b.WriteString("disabled: [")
	for i, d := range disabled {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%q", d))
	}
	b.WriteString("]\n")

	path := config.Path()
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to write %s: %v\n", path, err)
		os.Exit(1)
	}
}

func runList() {
	builtins := filters.ListBuiltins()
	seen := make(map[string][]string)
	for _, b := range builtins {
		seen[b.Command] = append(seen[b.Command], b.Subcommand)
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Println("built-in filters:")
	for _, cmd := range keys {
		subs := seen[cmd]
		if len(subs) == 1 && subs[0] == "" {
			fmt.Printf("  %s\n", cmd)
		} else {
			var nonEmpty []string
			for _, s := range subs {
				if s != "" {
					nonEmpty = append(nonEmpty, s)
				}
			}
			fmt.Printf("  %s (%s)\n", cmd, strings.Join(nonEmpty, ", "))
		}
	}
}

func runDiff(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: chop diff <command> [args...]")
		fmt.Fprintln(os.Stderr, "       echo 'output' | chop diff --stdin <command> [subcommand]")
		os.Exit(1)
	}

	var raw string
	var command string
	var cmdArgs []string

	if args[0] == "--stdin" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop diff --stdin <command> [subcommand]")
			os.Exit(1)
		}
		command = args[1]
		cmdArgs = args[2:]
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to read stdin: %v\n", err)
			os.Exit(1)
		}
		raw = string(input)
	} else {
		command = args[0]
		cmdArgs = args[1:]

		validateCommand(command)

		cmd := exec.Command(command, cmdArgs...)
		cmd.Stdin = os.Stdin
		output, err := cmd.CombinedOutput()
		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				fmt.Fprintf(os.Stderr, "chop: failed to run %s: %v\n", command, err)
				os.Exit(1)
			}
		}
		raw = string(output)
	}

	if raw == "" {
		fmt.Println("(no output)")
		return
	}

	filter := filters.Get(command, cmdArgs)
	var filtered string
	var filterName string

	if filter != nil {
		result, err := filter(raw)
		if err != nil || result == raw {
			filtered = raw
			filterName = "(no compression)"
		} else {
			filtered = result
			filterName = "built-in"
		}
	} else {
		result, err := filters.AutoDetect(raw)
		if err != nil || result == raw {
			filtered = raw
			filterName = "(no filter matched)"
		} else {
			filtered = result
			filterName = "auto-detect"
		}
	}

	rawLines := strings.Count(raw, "\n")
	filteredLines := strings.Count(filtered, "\n")
	rawTokens := tracking.CountTokens(raw)
	filteredTokens := tracking.CountTokens(filtered)

	savings := 0.0
	if rawTokens > 0 {
		savings = 100.0 - (float64(filteredTokens)/float64(rawTokens))*100.0
	}

	fmt.Println("=== RAW ===")
	rawPreview := raw
	rawTruncated := false
	if rawLines > 30 {
		rawPreview = firstNLines(raw, 30)
		rawTruncated = true
	}
	fmt.Print(rawPreview)
	if rawTruncated {
		fmt.Printf("\n... (%d more lines)\n", rawLines-30)
	}
	fmt.Println()

	fmt.Printf("=== FILTERED (%s) ===\n", filterName)
	fmt.Print(filtered)
	if !strings.HasSuffix(filtered, "\n") {
		fmt.Println()
	}
	fmt.Println()

	fmt.Println("=== STATS ===")
	fmt.Printf("lines:   %d -> %d\n", rawLines, filteredLines)
	fmt.Printf("tokens:  %d -> %d\n", rawTokens, filteredTokens)
	if savings > 0 {
		fmt.Printf("savings: %.1f%%\n", savings)
	} else {
		fmt.Println("savings: 0% (no compression)")
	}
}

// firstNLines returns the first n lines of s.
func firstNLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
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
	case "new":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: chop filter new <command>")
			os.Exit(1)
		}
		filterNew(strings.Join(args[1:], " "))
	case "edit":
		local := len(args) > 1 && args[1] == "--local"
		if local {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "chop: %v\n", err)
				os.Exit(1)
			}
			path := filepath.Join(cwd, ".chop-filters.yml")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				initFiltersConfig(true)
			}
			openInEditor(path)
		} else {
			path := config.FiltersConfigPath()
			if _, err := os.Stat(path); os.IsNotExist(err) {
				initFiltersConfig(false)
			}
			openInEditor(path)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\nusage: chop filter [path|init|add|remove|test|new|edit]\n", args[0])
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
		if err := os.MkdirAll(dir, 0o700); err != nil {
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

	if err := os.WriteFile(path, []byte(starter), 0o600); err != nil {
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

	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
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
		if err := os.MkdirAll(dir, 0o700); err != nil {
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

// filterNew scaffolds a new custom filter entry for the given command and
// guides the user through the capture → diff → tune workflow.
func filterNew(command string) {
	filterPath := config.FiltersConfigPath()
	cwd, _ := os.Getwd()

	// Check if a filter already exists for this command
	existing := config.LoadCustomFiltersWithLocal(cwd)
	if _, ok := existing[command]; ok {
		fmt.Printf("a filter for %q already exists in %s\n", command, filterPath)
		fmt.Println("use 'chop filter add' to modify it")
		return
	}

	// Ensure filters.yml exists
	if _, err := os.Stat(filterPath); os.IsNotExist(err) {
		initFiltersConfig(false)
	}

	// Build the scaffold entry
	scaffold := fmt.Sprintf(`
  # Filter for: %s
  # Tune with: chop capture %s  (saves raw fixture)
  #            chop diff %s     (shows compression preview)
  "%s":
    # keep: ["pattern1", "pattern2"]   # only lines matching these are kept
    # drop: ["pattern1", "pattern2"]   # lines matching these are removed
    # head: 50                         # keep first N lines (after keep/drop)
    # tail: 20                         # keep last N lines (after keep/drop)
    # exec: "~/.config/chop/scripts/%s.sh"  # pipe through external script
`, command, command, command, command, strings.ReplaceAll(command, " ", "-"))

	// Append to filters.yml
	f, err := os.OpenFile(filterPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to open filters file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	if _, err := f.WriteString(scaffold); err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to write scaffold: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("scaffolded filter for %q in %s\n", command, filterPath)
	fmt.Println()
	fmt.Println("next steps:")
	fmt.Printf("  1. chop capture %s          — capture real output as fixture\n", command)
	fmt.Printf("  2. edit %s              — uncomment and tune the rules\n", filepath.Base(filterPath))
	fmt.Printf("  3. chop diff %s             — preview compression before enabling\n", command)
	fmt.Printf("  4. chop filter test %s      — verify against stdin\n", command)
}

func runAgentInfo() {
	exe, _ := os.Executable()
	exe, _ = filepath.EvalSymlinks(exe)

	type hookInfo struct {
		Name      string `json:"name"`
		Installed bool   `json:"installed"`
		Path      string `json:"path,omitempty"`
	}

	var hooksList []hookInfo

	// Claude
	cInstalled, cPath := hooks.IsInstalled()
	hooksList = append(hooksList, hookInfo{Name: "claude", Installed: cInstalled, Path: cPath})

	// Gemini
	gInstalled, gPath := hooks.GeminiIsInstalled()
	hooksList = append(hooksList, hookInfo{Name: "gemini", Installed: gInstalled, Path: gPath})

	// Codex
	cxInstalled, cxPath := hooks.CodexIsInstalled()
	hooksList = append(hooksList, hookInfo{Name: "codex", Installed: cxInstalled, Path: cxPath})

	// Antigravity
	aInstalled, aPath := hooks.AntigravityIsInstalled()
	hooksList = append(hooksList, hookInfo{Name: "antigravity", Installed: aInstalled, Path: aPath})

	info := struct {
		Version string     `json:"version"`
		Path    string     `json:"path"`
		Hooks   []hookInfo `json:"hooks"`
	}{
		Version: version,
		Path:    exe,
		Hooks:   hooksList,
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to marshal agent info: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))

	// Ensure discovery file is also up to date
	_ = config.WriteDiscoveryInfo(version)
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
		if hooks.HasChopAwareHook() {
			// User has a wrapper script (e.g. chop-verify.sh) that invokes chop internally.
			// This is a valid non-standard install — do not report as an error.
			fmt.Println("[ok] chop wrapper script detected (non-standard install)")
		} else {
			fmt.Println("[!] hook is not installed")
			fmt.Println("    fix: chop init --global")
			issues++
		}
	} else {
		// 2. Check if hook path matches current binary
		hookCmd := hooks.GetHookCommand()
		expectedCmd, err := buildExpectedHookCmd()
		if err == nil && hookCmd != expectedCmd {
			fmt.Println("[!] hook points to wrong binary")
			fmt.Printf("    current: %s\n", hookCmd)
			fmt.Printf("    expected: %s\n", expectedCmd)
			fmt.Println("    fixing...")
			hooks.Install(version)
			issues++
		} else {
			fmt.Println("[ok] hook is installed and path is correct")
		}
	}

	// 3. Check for competing Bash PreToolUse hooks (Claude Code bug #15897)
	conflicts, err := hooks.FindConflictingBashHooks()
	if err == nil && conflicts.HasConflict() {
		fmt.Println("[!] competing Bash PreToolUse hooks detected — chop compression will be silently disabled")
		fmt.Println("    cause: https://github.com/anthropics/claude-code/issues/15897")
		fmt.Println("    when multiple Bash PreToolUse hooks are active, Claude Code drops updatedInput from all of them")
		if len(conflicts.SettingsConflicts) > 0 {
			fmt.Println("    competing hooks in ~/.claude/settings.json:")
			for _, cmd := range conflicts.SettingsConflicts {
				fmt.Printf("      - %s\n", cmd)
			}
		}
		if len(conflicts.PluginConflicts) > 0 {
			fmt.Println("    competing hooks from plugins:")
			for _, path := range conflicts.PluginConflicts {
				fmt.Printf("      - %s\n", path)
			}
		}
		fmt.Println("    fix: run `chop fix-hooks` to generate a combined wrapper script automatically")
		issues++
	} else if err == nil {
		fmt.Println("[ok] no competing Bash PreToolUse hooks")
	}

	// 4. Check if binary is in legacy ~/bin
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

	// 5. Check tracking DB health
	if err := tracking.Init(); err != nil {
		fmt.Printf("[!] tracking database unavailable: %v\n", err)
		fmt.Printf("    path: %s\n", tracking.DBPath())
		fmt.Println("    fix: chop gain --delete (resets DB) or check file permissions")
		issues++
	} else {
		fmt.Println("[ok] tracking database is healthy")
	}

	// 6. Check global config syntax
	cfgPath := config.Path()
	if _, err := os.Stat(cfgPath); err == nil {
		if errs := config.Validate(cfgPath); len(errs) > 0 {
			fmt.Printf("[!] config file has %d issue(s): %s\n", len(errs), cfgPath)
			for _, e := range errs {
				fmt.Printf("    - %s\n", e)
			}
			issues++
		} else {
			fmt.Println("[ok] global config is valid")
		}
	}

	// 7. Check custom filters syntax (regex patterns + exec scripts)
	filterPath := config.FiltersConfigPath()
	if _, err := os.Stat(filterPath); err == nil {
		if errs := config.ValidateFilters(filterPath); len(errs) > 0 {
			fmt.Printf("[!] filters config has %d issue(s): %s\n", len(errs), filterPath)
			for _, e := range errs {
				fmt.Printf("    - %s\n", e)
			}
			issues++
		} else {
			fmt.Println("[ok] custom filters are valid")
		}
	}

	if issues == 0 {
		fmt.Println("\nall good!")
	} else {
		fmt.Printf("\n%d issue(s) found\n", issues)
	}
}

func runFixHooks() {
	conflicts, err := hooks.FindConflictingBashHooks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: failed to check for conflicts: %v\n", err)
		os.Exit(1)
	}

	if !conflicts.HasConflict() {
		fmt.Println("no hook conflicts detected — nothing to fix")
		return
	}

	fmt.Println("Competing Bash PreToolUse hooks detected:")
	fmt.Println("  cause: https://github.com/anthropics/claude-code/issues/15897")
	for _, cmd := range conflicts.SettingsConflicts {
		fmt.Printf("  [settings.json] %s\n", cmd)
	}
	for _, path := range conflicts.PluginConflicts {
		fmt.Printf("  [plugin]        %s\n", path)
	}
	fmt.Println()

	if len(conflicts.PluginConflicts) > 0 && len(conflicts.SettingsConflicts) == 0 {
		fmt.Println("Plugin conflicts cannot be auto-fixed — chop cannot modify plugin files.")
		fmt.Println("Options:")
		fmt.Println("  [1] Disable the plugin's Bash PreToolUse hook manually (set PreToolUse: [] in the plugin hooks.json)")
		fmt.Println("  [2] Use a wrapper script: merge the plugin's hook logic into ~/.claude/hooks/chop-wrapper.sh manually")
		return
	}

	fmt.Println("How would you like to fix this?")
	fmt.Println("  [1] Standard — install chop directly and let you merge hooks manually")
	fmt.Println("  [2] Wrapper  — generate ~/.claude/hooks/chop-wrapper.sh combining all hooks (recommended)")
	if len(conflicts.PluginConflicts) > 0 {
		fmt.Println("  Note: plugin conflicts must still be fixed manually (disable PreToolUse in plugin hooks.json)")
	}
	fmt.Print("\nEnter choice [1/2]: ")

	var choice string
	fmt.Scan(&choice) //nolint:errcheck

	switch strings.TrimSpace(choice) {
	case "1":
		fmt.Println()
		fmt.Println("Standard install selected.")
		fmt.Println("chop is already installed. Remove or merge the competing hooks manually, then run `chop doctor` to verify.")
	case "2":
		chopBin, err := buildExpectedHookCmd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to determine binary path: %v\n", err)
			os.Exit(1)
		}
		// Extract just the binary path from `"<path>" hook`
		chopBinPath := strings.TrimSuffix(strings.Trim(chopBin, `"`), `" hook`)
		chopBinPath = strings.TrimSuffix(chopBin, " hook")
		chopBinPath = strings.Trim(chopBinPath, `"`)

		scriptPath, err := hooks.GenerateConflictFixScript(conflicts, chopBinPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to generate wrapper: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("generated: %s\n", scriptPath)

		if err := hooks.ApplyConflictFix(scriptPath); err != nil {
			fmt.Fprintf(os.Stderr, "chop: failed to update settings.json: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("updated:   ~/.claude/settings.json")
		fmt.Println()
		fmt.Println("Done. Run `chop doctor` to verify the fix.")
	default:
		fmt.Println("Invalid choice. Run `chop fix-hooks` again.")
		os.Exit(1)
	}
}

func runUnwrapHooks() {
	installed, _ := hooks.IsInstalled()
	if !installed && !hooks.HasChopAwareHook() {
		fmt.Println("no chop hook detected — nothing to unwrap")
		return
	}

	wrapped, err := hooks.UnwrapHooks(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chop: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("chop hook restored to standard install")
	fmt.Println("chop-wrapper.sh removed")

	if len(wrapped) > 0 {
		fmt.Println()
		fmt.Println("The following hooks were inside the wrapper — re-enable them manually in their plugin hooks.json:")
		for _, cmd := range wrapped {
			fmt.Printf("  - %s\n", cmd)
		}
	}

	fmt.Println()
	fmt.Println("Run `chop doctor` to verify.")
}

func runCompletion(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: chop completion <bash|zsh|fish|powershell>")
		fmt.Fprintln(os.Stderr, "\nAdd to your shell profile:")
		fmt.Fprintln(os.Stderr, "  bash:        source <(chop completion bash)")
		fmt.Fprintln(os.Stderr, "  zsh:         source <(chop completion zsh)")
		fmt.Fprintln(os.Stderr, "  fish:        chop completion fish | source")
		fmt.Fprintln(os.Stderr, "  powershell:  chop completion powershell | Invoke-Expression")
		os.Exit(1)
	}
	switch args[0] {
	case "bash":
		fmt.Print(completionBash)
	case "zsh":
		fmt.Print(completionZsh)
	case "fish":
		fmt.Print(completionFish)
	case "powershell":
		fmt.Print(completionPowerShell)
	default:
		fmt.Fprintf(os.Stderr, "unknown shell %q — supported: bash, zsh, fish, powershell\n", args[0])
		os.Exit(1)
	}
}

const completionBash = `# chop bash completion
# Add to ~/.bashrc: source <(chop completion bash)
_chop_completion() {
    local cur prev
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    local cmd="${COMP_WORDS[1]}"

    local top_cmds="help version gain capture config filter local list diff init update auto-update enable disable doctor uninstall reset completion"

    case "$cmd" in
        gain)
            case "$prev" in
                --export) COMPREPLY=($(compgen -W "json csv" -- "$cur")) ;;
                --since)  COMPREPLY=($(compgen -W "1h 6h 24h 7d 30d" -- "$cur")) ;;
                --limit)  COMPREPLY=() ;;
                *)        COMPREPLY=($(compgen -W "--history --summary --unchopped --verbose --all --since --limit --export --skip --unskip --delete --no-track --resume-track" -- "$cur")) ;;
            esac ;;
        filter)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=($(compgen -W "path init edit add remove test new" -- "$cur"))
            else
                case "${COMP_WORDS[2]}" in
                    add)  COMPREPLY=($(compgen -W "--keep --drop --head --tail --exec --local" -- "$cur")) ;;
                    init) COMPREPLY=($(compgen -W "--local" -- "$cur")) ;;
                    edit) COMPREPLY=($(compgen -W "--local" -- "$cur")) ;;
                esac
            fi ;;
        config)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=($(compgen -W "init edit set export import" -- "$cur"))
            elif [[ ${COMP_CWORD} -eq 3 && "${COMP_WORDS[2]}" == "set" ]]; then
                COMPREPLY=($(compgen -W "editor" -- "$cur"))
            fi ;;
        local)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=($(compgen -W "add remove clear edit" -- "$cur"))
            fi ;;
        init|setup)
            COMPREPLY=($(compgen -W "--global --gemini --codex --antigravity --uninstall --status" -- "$cur")) ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh fish powershell" -- "$cur")) ;;
        diff)
            COMPREPLY=($(compgen -W "--stdin" -- "$cur")) ;;
        uninstall)
            COMPREPLY=($(compgen -W "--keep-data" -- "$cur")) ;;
        *)
            if [[ ${COMP_CWORD} -eq 1 ]]; then
                COMPREPLY=($(compgen -W "$top_cmds" -- "$cur"))
            fi ;;
    esac
}
complete -F _chop_completion chop
`

const completionZsh = `# chop zsh completion
# Add to ~/.zshrc: source <(chop completion zsh)
_chop() {
    local state

    _arguments \
        '1: :->cmd' \
        '*: :->args'

    case $state in
        cmd)
            _values 'command' \
                'help' 'version' 'gain' 'capture' 'config' 'filter' 'local' \
                'list' 'diff' 'init' 'update' 'auto-update' 'enable' 'disable' \
                'doctor' 'uninstall' 'reset' 'completion'
            ;;
        args)
            case ${words[2]} in
                gain)
                    _values 'option' \
                        '--history' '--summary' '--unchopped' '--verbose' '--all' \
                        '--since' '--limit' '--export' '--skip' '--unskip' \
                        '--delete' '--no-track' '--resume-track'
                    ;;
                filter)
                    if [[ ${#words} -eq 3 ]]; then
                        _values 'subcommand' 'path' 'init' 'edit' 'add' 'remove' 'test' 'new'
                    else
                        case ${words[3]} in
                            add)  _values 'option' '--keep' '--drop' '--head' '--tail' '--exec' '--local' ;;
                            init) _values 'option' '--local' ;;
                            edit) _values 'option' '--local' ;;
                        esac
                    fi ;;
                config)
                    if [[ ${#words} -eq 3 ]]; then
                        _values 'subcommand' 'init' 'edit' 'set' 'export' 'import'
                    elif [[ ${#words} -eq 4 && ${words[3]} == 'set' ]]; then
                        _values 'key' 'editor'
                    fi ;;
                local)
                    [[ ${#words} -eq 3 ]] && _values 'subcommand' 'add' 'remove' 'clear' 'edit' ;;
                init|setup)
                    _values 'option' '--global' '--gemini' '--codex' '--antigravity' '--uninstall' '--status' ;;
                completion)
                    _values 'shell' 'bash' 'zsh' 'fish' 'powershell' ;;
                diff)
                    _values 'option' '--stdin' ;;
                uninstall)
                    _values 'option' '--keep-data' ;;
            esac ;;
    esac
}
compdef _chop chop
`

const completionFish = `# chop fish completion
# Add to fish config: chop completion fish | source

# Disable file completion by default
complete -c chop -f

# Top-level commands
set -l cmds help version gain capture config filter local list diff init update auto-update enable disable doctor uninstall reset completion
complete -c chop -n "not __fish_seen_subcommand_from $cmds" -a "$cmds"

# gain
complete -c chop -n "__fish_seen_subcommand_from gain" -l history    -d "Show recent command history"
complete -c chop -n "__fish_seen_subcommand_from gain" -l summary    -d "Per-command savings summary"
complete -c chop -n "__fish_seen_subcommand_from gain" -l unchopped  -d "Commands with no filter"
complete -c chop -n "__fish_seen_subcommand_from gain" -l verbose -s v -d "Verbose output"
complete -c chop -n "__fish_seen_subcommand_from gain" -l all        -d "Show all records"
complete -c chop -n "__fish_seen_subcommand_from gain" -l since      -d "Since duration (e.g. 24h)"  -r
complete -c chop -n "__fish_seen_subcommand_from gain" -l limit      -d "Limit records"              -r
complete -c chop -n "__fish_seen_subcommand_from gain" -l export     -d "Export format (json|csv)"   -r -a "json csv"
complete -c chop -n "__fish_seen_subcommand_from gain" -l skip       -d "Skip command from unchopped" -r
complete -c chop -n "__fish_seen_subcommand_from gain" -l unskip     -d "Remove command from skip list" -r
complete -c chop -n "__fish_seen_subcommand_from gain" -l delete     -d "Delete history for command" -r
complete -c chop -n "__fish_seen_subcommand_from gain" -l no-track   -d "Stop tracking command"      -r
complete -c chop -n "__fish_seen_subcommand_from gain" -l resume-track -d "Resume tracking command"  -r

# filter subcommands
complete -c chop -n "__fish_seen_subcommand_from filter; and not __fish_seen_subcommand_from path init edit add remove test new" \
    -a "path init edit add remove test new"
complete -c chop -n "__fish_seen_subcommand_from filter; and __fish_seen_subcommand_from add" \
    -l keep -l drop -l head -l tail -l exec -l local
complete -c chop -n "__fish_seen_subcommand_from filter; and __fish_seen_subcommand_from edit" \
    -l local -d "Edit local project filters"

# config subcommands
complete -c chop -n "__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from init edit set export import" \
    -a "init edit set export import"
complete -c chop -n "__fish_seen_subcommand_from config; and __fish_seen_subcommand_from set" \
    -a "editor history_compressed_only" -d "Config key to set"

# local subcommands
complete -c chop -n "__fish_seen_subcommand_from local; and not __fish_seen_subcommand_from add remove clear edit" \
    -a "add remove clear edit"

# init flags
complete -c chop -n "__fish_seen_subcommand_from init setup" \
    -a "--global --gemini --codex --antigravity --uninstall --status"

# completion shells
complete -c chop -n "__fish_seen_subcommand_from completion" -a "bash zsh fish powershell"

# diff
complete -c chop -n "__fish_seen_subcommand_from diff" -l stdin -d "Read from stdin"

# uninstall
complete -c chop -n "__fish_seen_subcommand_from uninstall" -l keep-data -d "Keep tracking data"
`

const completionPowerShell = `# chop PowerShell completion
# Add to $PROFILE: chop completion powershell | Invoke-Expression

Register-ArgumentCompleter -Native -CommandName chop -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $words = $commandAst.CommandElements
    $cmd   = if ($words.Count -ge 2) { $words[1].Value } else { "" }

    $topCmds = @(
        'help','version','gain','capture','config','filter','local',
        'list','diff','init','update','auto-update','enable','disable',
        'doctor','uninstall','reset','completion'
    )

    $complete = {
        param($list)
        $list | Where-Object { $_ -like "$wordToComplete*" } |
            ForEach-Object { [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_) }
    }

    if ($words.Count -le 2) {
        & $complete $topCmds
        return
    }

    switch ($cmd) {
        'gain' {
            & $complete @('--history','--summary','--unchopped','--verbose','--all',
                          '--since','--limit','--export','--skip','--unskip',
                          '--delete','--no-track','--resume-track')
        }
        'filter' {
            if ($words.Count -eq 3) {
                & $complete @('path','init','edit','add','remove','test','new')
            } elseif ($words[2].Value -eq 'add') {
                & $complete @('--keep','--drop','--head','--tail','--exec','--local')
            } elseif ($words[2].Value -in @('init','edit')) {
                & $complete @('--local')
            }
        }
        'config' {
            if ($words.Count -eq 3) { & $complete @('init','edit','set','export','import') }
            elseif ($words.Count -eq 4 -and $words[2].Value -eq 'set') { & $complete @('editor') }
        }
        'local' {
            if ($words.Count -eq 3) { & $complete @('add','remove','clear','edit') }
        }
        'init' { & $complete @('--global','--gemini','--codex','--antigravity','--uninstall','--status') }
        'completion' { & $complete @('bash','zsh','fish','powershell') }
        'diff' { & $complete @('--stdin') }
        'uninstall' { & $complete @('--keep-data') }
    }
}
`

// validateCommand checks if a command name is safe to execute and exits if not.
// Blocks shell metacharacters to prevent confusion and protect against
// potential shell-based wrappers that might be used to invoke chop.
func validateCommand(cmd string) {
	if strings.ContainsAny(cmd, ";|&><`$()\n\r") {
		fmt.Fprintf(os.Stderr, "chop: invalid command name %q\n", cmd)
		os.Exit(1)
	}
}

// sanitizeFilename replaces path separators and traversal sequences to prevent
// writing files outside the intended directory.
func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "..", "__")
	return s
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
	useColor := tracking.IsColorEnabled()

	// Color helpers — no-ops when color is off or output is piped.
	bold := func(s string) string {
		if useColor {
			return "\033[1m" + s + "\033[0m"
		}
		return s
	}
	dim := func(s string) string {
		if useColor {
			return "\033[2m" + s + "\033[0m"
		}
		return s
	}
	cyan := func(s string) string {
		if useColor {
			return "\033[36m" + s + "\033[0m"
		}
		return s
	}
	yellow := func(s string) string {
		if useColor {
			return "\033[33m" + s + "\033[0m"
		}
		return s
	}

	// section prints a coloured group header.
	section := func(name string) string {
		return bold(cyan(name)) + "\n"
	}

	// row formats one command entry: left-col plain text, right-col dimmed.
	const colW = 40
	row := func(cmd, desc string) string {
		return fmt.Sprintf("  %-*s%s\n", colW, cmd, dim(desc))
	}

	// flag highlights flag names in yellow within a description string.
	flag := func(f string) string { return yellow(f) }

	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("%s %s — CLI output compressor for AI agents\n\n", bold("chop"), version))

	// Usage
	b.WriteString(bold("Usage") + "\n")
	b.WriteString(row("chop <command> [args...]", "Wrap and compress a command's output"))
	b.WriteString(row("chop <subcommand>", "Run a chop management subcommand"))
	b.WriteString("\n")

	// Analytics
	b.WriteString(section("Analytics"))
	b.WriteString(row("gain", "Overall token savings summary"))
	b.WriteString(row("gain "+flag("--history"), "Recent commands with savings (last 20)"))
	b.WriteString(row("gain --history "+flag("--limit")+" N", "Show last N commands"))
	b.WriteString(row("gain --history "+flag("--all"), "Show all records"))
	b.WriteString(row("gain --history "+flag("--since")+" <d>", "Filter to time window (e.g. 7d, 2w, 24h)"))
	b.WriteString(row("gain --history "+flag("--verbose"), "Full command strings + project groups"))
	b.WriteString(row("gain --history "+flag("--project")+" <path>", "Filter history to a specific project"))
	b.WriteString(row("gain "+flag("--summary"), "Per-command savings breakdown"))
	b.WriteString(row("gain "+flag("--projects"), "Per-project savings breakdown"))
	b.WriteString(row("gain "+flag("--unchopped"), "Commands with no filter (new candidates)"))
	b.WriteString(row("gain --unchopped "+flag("--skip")+" X", "Mark X as intentionally unfiltered"))
	b.WriteString(row("gain --unchopped "+flag("--unskip")+" X", "Remove X from the skip list"))
	b.WriteString(row("gain "+flag("--delete")+" X", "Delete all tracking records for X"))
	b.WriteString(row("gain "+flag("--no-track")+" X", "Delete records for X and stop tracking"))
	b.WriteString(row("gain "+flag("--resume-track")+" X", "Re-enable tracking for X"))
	b.WriteString(row("gain "+flag("--export")+" json|csv", "Export history to stdout"))
	b.WriteString("\n")

	// Filters
	b.WriteString(section("Filters"))
	b.WriteString(row("list", "List all built-in filters"))
	b.WriteString(row("filter", "List custom user-defined filters"))
	b.WriteString(row("filter new <cmd>", "Scaffold a new filter + guided workflow"))
	b.WriteString(row("filter add <cmd> [flags]", flag("--keep")+" "+flag("--drop")+" "+flag("--head")+" "+flag("--tail")+" "+flag("--exec")+" "+flag("--local")))
	b.WriteString(row("filter remove <cmd>", "Remove a filter ("+flag("--local")+" for project-level)"))
	b.WriteString(row("filter test <cmd>", "Test a custom filter against stdin"))
	b.WriteString(row("filter init", "Create starter global filters.yml"))
	b.WriteString(row("filter init "+flag("--local"), "Create starter .chop-filters.yml in cwd"))
	b.WriteString(row("filter edit", "Open global filters.yml in $EDITOR"))
	b.WriteString(row("filter edit "+flag("--local"), "Open local .chop-filters.yml in $EDITOR"))
	b.WriteString(row("filter path", "Show filters config file path"))
	b.WriteString(row("diff <cmd> [args...]", "Show raw vs filtered output side-by-side"))
	b.WriteString(row("diff "+flag("--stdin")+" <cmd>", "Diff using stdin instead of running command"))
	b.WriteString("\n")

	// Config
	b.WriteString(section("Config"))
	b.WriteString(row("config", "Show global config path and contents"))
	b.WriteString(row("config init", "Create a starter global config.yml"))
	b.WriteString(row("config set editor <vim|code|...>", "Set preferred editor for chop edit commands"))
	b.WriteString(row("config set history_compressed_only <true|false>", "Only show compressed commands in history"))
	b.WriteString(row("config edit", "Open global config.yml in preferred editor"))
	b.WriteString(row("config export", "Export config.yml + filters.yml to stdout"))
	b.WriteString(row("config import <file>", "Import config from a file (created by export)"))
	b.WriteString(row("global", "Show global config"))
	b.WriteString(row("global add \"git diff --cached\"", "Disable a command globally"))
	b.WriteString(row("global remove \"git diff --cached\"", "Re-enable a command globally"))
	b.WriteString(row("local", "Show local project config (.chop.yml)"))
	b.WriteString(row("local add \"git diff\"", "Disable a command in this project"))
	b.WriteString(row("local remove \"git diff\"", "Re-enable a command in this project"))
	b.WriteString(row("local clear", "Remove local project config"))
	b.WriteString(row("local edit", "Open local .chop.yml in $EDITOR"))
	b.WriteString("\n")

	// Integrations
	b.WriteString(section("Integrations"))
	b.WriteString(row("init "+flag("--global"), "Install Claude Code hook (~/.claude/settings.json)"))
	b.WriteString(row("init "+flag("--gemini"), "Install Gemini CLI hook (~/.gemini/settings.json)"))
	b.WriteString(row("init "+flag("--codex"), "Install Codex CLI hook (~/.codex/settings.json)"))
	b.WriteString(row("init "+flag("--antigravity"), "Install Antigravity IDE hook"))
	b.WriteString(row("init --<platform> "+flag("--uninstall"), "Remove a platform hook"))
	b.WriteString(row("init --<platform> "+flag("--status"), "Check a platform hook status"))
	b.WriteString(row("init "+flag("--agent-handshake"), "Emit discovery message for AI agents"))
	b.WriteString("\n")

	// Shell
	b.WriteString(section("Shell"))
	b.WriteString(row("completion bash", "Print bash completion script"))
	b.WriteString(row("completion zsh", "Print zsh completion script"))
	b.WriteString(row("completion fish", "Print fish completion script"))
	b.WriteString(row("completion powershell", "Print PowerShell completion script"))
	b.WriteString("\n")

	// Maintenance
	b.WriteString(section("Maintenance"))
	b.WriteString(row("doctor", "Check hook, DB health, config, filter syntax"))
	b.WriteString(row("update", "Update to the latest version"))
	b.WriteString(row("auto-update", "Show auto-update status"))
	b.WriteString(row("auto-update on|off", "Enable or disable automatic background updates"))
	b.WriteString(row("enable / disable", "Resume or bypass chop globally"))
	b.WriteString(row("uninstall", "Remove hook, data, config, and binary"))
	b.WriteString(row("uninstall "+flag("--keep-data"), "Uninstall but preserve tracking history"))
	b.WriteString(row("reset", "Clear tracking data and audit log"))
	b.WriteString(row("hook-audit", "Show last 20 hook rewrite log entries"))
	b.WriteString(row("hook-audit "+flag("--clear"), "Clear the hook audit log"))
	b.WriteString("\n")

	// Other
	b.WriteString(section("Other"))
	b.WriteString(row("version", "Show version"))
	b.WriteString(row("agent-info", "Show JSON info for AI agents"))
	b.WriteString(row("changelog", "Show changes in the current version"))
	b.WriteString(row("changelog "+flag("--full"), "Show full changelog history"))
	b.WriteString(row("help", "Show this help"))
	b.WriteString("\n")

	// Config reference
	b.WriteString(bold("Config") + dim(" ("+config.Path()+")") + "\n")
	b.WriteString(fmt.Sprintf("  %s %s  %s\n",
		yellow("editor:"), dim("vim"),
		dim("Preferred editor for all 'chop * edit' commands")))
	b.WriteString(fmt.Sprintf("  %s %s  %s\n",
		yellow("history_compressed_only:"), dim("true"),
		dim("Hide 0% savings entries from 'chop gain --history'")))
	b.WriteString(fmt.Sprintf("  %s %s  %s\n",
		yellow("disabled:"), dim("- git diff"),
		dim("Prefix-based: 'git diff' disables all git diff variants")))
	b.WriteString("\n")

	// Custom filters reference
	b.WriteString(bold("Custom filters") + dim(" ("+config.FiltersConfigPath()+")") + "\n")
	b.WriteString(dim("  Run 'chop filter init' to create a starter config with examples.\n"))
	b.WriteString(fmt.Sprintf("  %s [regex...]  %s\n", yellow("keep:"), dim("Only keep lines matching at least one pattern")))
	b.WriteString(fmt.Sprintf("  %s [regex...]  %s\n", yellow("drop:"), dim("Remove lines matching any pattern")))
	b.WriteString(fmt.Sprintf("  %s N           %s\n", yellow("head:"), dim("Keep first N lines (after keep/drop)")))
	b.WriteString(fmt.Sprintf("  %s N           %s\n", yellow("tail:"), dim("Keep last N lines (after keep/drop)")))
	b.WriteString(fmt.Sprintf("  %s script      %s\n", yellow("exec:"), dim("Pipe output through an external script (global config only)")))
	b.WriteString("\n")

	// Examples
	b.WriteString(bold("Examples") + "\n")
	b.WriteString(row("chop git status", "Compressed git status"))
	b.WriteString(row("chop docker ps", "Compact container list"))
	b.WriteString(row("chop kubectl get pods", "Filtered pod table"))
	b.WriteString(row("chop curl https://api.io", "Auto-compressed JSON response"))
	b.WriteString(row("chop cat app.log", "Pattern-grouped log lines with repeat counts"))

	fmt.Print(b.String())
}
