package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AgusRdz/chop/config"
)

var supportedCommands = map[string]bool{
	"git": true, "npm": true, "npx": true, "pnpm": true, "yarn": true, "bun": true,
	"docker": true, "docker-compose": true, "dotnet": true, "kubectl": true, "helm": true, "terraform": true,
	"ansible-playbook": true,
	"cargo": true, "go": true, "tsc": true, "eslint": true, "biome": true,
	"gh": true, "grep": true, "rg": true, "curl": true, "http": true,
	"aws": true, "az": true, "gcloud": true, "mvn": true, "gradle": true, "gradlew": true,
	"ng": true, "nx": true, "pytest": true, "pip": true, "pip3": true, "uv": true,
	"mypy": true, "ruff": true, "flake8": true, "pylint": true,
	"bundle": true, "bundler": true, "rspec": true, "rubocop": true,
	"composer": true, "make": true, "cmake": true,
	"gcc": true, "g++": true, "cc": true, "c++": true, "clang": true, "clang++": true,
	"ping": true, "ps": true, "ss": true, "netstat": true, "df": true, "du": true,
	"cat": true, "tail": true, "less": true, "more": true,
	"find": true, "node": true, "node16": true, "node18": true, "node20": true, "node22": true,
	"acli": true,
	// filters added in v1.38.2 — wired to registry but missing from hook interception
	"golangci-lint": true, "turbo": true, "poetry": true, "conda": true, "pipenv": true,
	"snyk": true, "trivy": true, "kustomize": true, "argocd": true, "flux": true,
	"stern": true, "vault": true, "psql": true, "mysql": true, "sqlite3": true,
	"glab": true, "mvnw": true,
}

// shellBuiltins are commands that should never be wrapped.
var shellBuiltins = []string{
	"cd ", "export ", "source ", "echo ", "printf ", "set ", "unset ", "alias ", "eval ",
}

// pipeRedirectOperators make wrapping ambiguous — skip the entire command.
// File redirects always have surrounding spaces ("> file", ">> file", "< file").
// fd-style redirects like "2>&1" intentionally do not match so they can be wrapped (and later stripped).
var pipeRedirectOperators = []string{" | ", " > ", " >> ", " < "}

// logicalSeparators chain independent commands — split and wrap each segment.
var logicalSeparators = []string{" && ", " || ", " ; "}

// isStreamingInvocation returns true for commands that produce continuous output
// and must not be wrapped (chop buffers everything and would return nothing on timeout).
func isStreamingInvocation(command string) bool {
	baseCmd, args := splitHookInvocation(command)
	baseCmd = strings.ToLower(baseCmd)
	args = normalizeHookArgs(args)

	// Watch modes are streaming regardless of the runner. Match complete flags
	// so finite options such as --watchAll=false and --watchman are not caught.
	if hasEnabledWatchFlag(args) {
		return true
	}

	// stern follows by default
	if baseCmd == "stern" {
		return true
	}

	// -f and --follow are overloaded by many finite commands (for example
	// mvn -f, git fetch -f, git log --follow, and rg --follow). They are
	// streaming only for commands whose output-following mode chop must avoid.
	if hasEnabledFollowFlag(args) {
		switch baseCmd {
		case "tail":
			return true
		case "kubectl":
			return hasHookArg(args, "logs")
		case "docker", "podman", "docker-compose":
			return hasHookArg(args, "logs")
		case "aws":
			// aws logs tail --follow streams CloudWatch logs; other aws -f uses are finite.
			return hasHookArgSequence(args, "logs", "tail")
		case "argocd", "flux":
			return hasHookArg(args, "logs")
		}
	}

	// docker compose up without -d/--detach
	composeUp := (baseCmd == "docker" || baseCmd == "podman") && hasHookArgSequence(args, "compose", "up")
	composeUp = composeUp || (baseCmd == "docker-compose" && len(args) > 0 && args[0] == "up")
	if composeUp && !hasEnabledDetachFlag(args) {
		return true
	}

	// ping without -c/--count (infinite on Linux/macOS)
	if baseCmd == "ping" && !hasPingCountFlag(args) {
		return true
	}

	return false
}

func normalizeHookArgs(args []string) []string {
	normalized := make([]string, len(args))
	for i, arg := range args {
		normalized[i] = strings.ToLower(strings.Trim(arg, `"'`))
	}
	return normalized
}

func hasHookArg(args []string, targets ...string) bool {
	for _, arg := range args {
		for _, target := range targets {
			if arg == target {
				return true
			}
		}
	}
	return false
}

func hasHookArgSequence(args []string, first, second string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == first && args[i+1] == second {
			return true
		}
	}
	return false
}

func hasEnabledFollowFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-f" || arg == "--follow" {
			return true
		}
		if strings.HasPrefix(arg, "--follow=") {
			value := strings.TrimPrefix(arg, "--follow=")
			if value != "false" && value != "0" {
				return true
			}
		}
	}
	return false
}

func hasEnabledWatchFlag(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--watch", "--watch=true", "--watch=1",
			"--watchall", "--watchall=true", "--watchall=1":
			return true
		}
	}
	return false
}

func hasEnabledDetachFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-d" || arg == "--detach" || arg == "--detach=true" || arg == "--detach=1" {
			return true
		}
	}
	return false
}

func hasPingCountFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-c" || arg == "--count" || strings.HasPrefix(arg, "--count=") {
			return true
		}
		if strings.HasPrefix(arg, "-c") && len(arg) > 2 {
			return true
		}
	}
	return false
}

// quoteState tracks parser position relative to shell quoting.
type quoteState int

const (
	quoteNone   quoteState = iota
	quoteSingle            // inside '...'
	quoteDouble            // inside "..."
)

// scanQuoteState advances through s character by character, returning the
// quote state and index of the first occurrence of needle that appears
// outside of any quoted region. Returns -1 if not found.
func indexOutsideQuotes(s, needle string) int {
	state := quoteNone
	for i := 0; i < len(s); {
		ch := s[i]
		switch state {
		case quoteNone:
			if ch == '\'' {
				state = quoteSingle
				i++
			} else if ch == '"' {
				state = quoteDouble
				i++
			} else if strings.HasPrefix(s[i:], needle) {
				return i
			} else {
				i++
			}
		case quoteSingle:
			// No escaping inside single quotes — only ' ends it.
			if ch == '\'' {
				state = quoteNone
			}
			i++
		case quoteDouble:
			if ch == '\\' && i+1 < len(s) {
				i += 2 // skip escaped char
			} else if ch == '"' {
				state = quoteNone
				i++
			} else {
				i++
			}
		}
	}
	return -1
}

// containsOutsideQuotes reports whether needle appears in s outside quotes.
func containsOutsideQuotes(s, needle string) bool {
	return indexOutsideQuotes(s, needle) != -1
}

// hookInput represents the JSON payload received from Claude Code's PreToolUse hook.
type hookInput struct {
	SessionID     string          `json:"session_id"`
	Cwd           string          `json:"cwd"`
	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
}

type toolInput struct {
	Command string `json:"command"`
}

type hookOutput struct {
	HookSpecificOutput hookSpecificOutput `json:"hookSpecificOutput"`
}

type hookSpecificOutput struct {
	HookEventName      string    `json:"hookEventName"`
	PermissionDecision string    `json:"permissionDecision,omitempty"`
	UpdatedInput       toolInput `json:"updatedInput"`
}

// resilientToolInput is used to gracefully handle input field name variations.
type resilientToolInput struct {
	Command  string `json:"command"`
	CmdUpper string `json:"Cmd"`
	CmdLower string `json:"cmd"`
}

// RunHook reads a Claude Code PreToolUse hook payload from stdin,
// checks if the command should be wrapped with chop, and outputs
// modified JSON on stdout. Always exits 0.
func RunHook() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(0)
	}

	output, shouldModify, original := processHookInput(input)
	if shouldModify {
		var result hookOutput
		if err := json.Unmarshal(output, &result); err == nil {
			auditLog(original, result.HookSpecificOutput.UpdatedInput.Command)
		}
		fmt.Print(string(output))
	}
	// If not modifying, output nothing (passthrough)
}

// processHookInput parses the Claude Code hook JSON and determines whether to wrap the command.
// Returns (outputJSON, shouldModify, originalCommand).
func processHookInput(input []byte) ([]byte, bool, string) {
	var h hookInput
	if err := json.Unmarshal(input, &h); err != nil {
		return nil, false, ""
	}

	if h.ToolName != "Bash" {
		return nil, false, ""
	}

	// Global kill switch — pass through everything when disabled.
	if IsDisabledGlobally() {
		return nil, false, ""
	}

	var ti toolInput
	if err := json.Unmarshal(h.ToolInput, &ti); err != nil {
		return nil, false, ""
	}

	wrapped, shouldModify, original := rewriteCommand(ti.Command)
	if !shouldModify {
		return nil, false, original
	}

	return buildOutput(original, wrapped)
}

// rewriteCommand takes a raw shell command and returns the wrapped version.
// Returns (wrappedCommand, shouldModify, originalCommand).
// This is the shared logic used by both Claude Code and Gemini CLI hooks.
func rewriteCommand(command string) (string, bool, string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", false, ""
	}

	// Already wrapped with chop
	if strings.HasPrefix(command, "chop ") {
		return "", false, command
	}

	// Starts with a dot (source shorthand)
	if strings.HasPrefix(command, ". ") {
		return "", false, command
	}

	// Logical chaining operators — split first, then evaluate each segment independently.
	for _, op := range logicalSeparators {
		if containsOutsideQuotes(command, op) {
			segments, operators := splitLogical(command)
			modified := false
			result := make([]string, len(segments))
			for i, seg := range segments {
				seg = strings.TrimSpace(seg)
				if shouldWrap(seg) {
					result[i] = "chop " + stripStderrRedirect(seg)
					modified = true
				} else {
					result[i] = seg
				}
			}
			if !modified {
				return "", false, command
			}
			var sb strings.Builder
			for i, seg := range result {
				if i > 0 {
					sb.WriteString(operators[i-1])
				}
				sb.WriteString(seg)
			}
			return sb.String(), true, command
		}
	}

	// Shell builtins
	for _, prefix := range shellBuiltins {
		if strings.HasPrefix(command, prefix) {
			return "", false, command
		}
	}

	// Pipe and redirect operators — can't wrap safely
	for _, op := range pipeRedirectOperators {
		if containsOutsideQuotes(command, op) {
			return "", false, command
		}
	}

	// Single command — wrap if supported
	if !shouldWrap(command) {
		return "", false, command
	}

	return "chop " + stripStderrRedirect(command), true, command
}

// stripStderrRedirect removes trailing "2>&1" from a command — redundant when chop uses CombinedOutput.
func stripStderrRedirect(cmd string) string {
	cmd = strings.TrimRight(cmd, " ")
	cmd = strings.TrimSuffix(cmd, "2>&1")
	return strings.TrimRight(cmd, " ")
}

// shouldWrap returns true if a single (non-compound) command should be wrapped with chop.
func shouldWrap(command string) bool {
	if strings.HasPrefix(command, "chop ") || strings.HasPrefix(command, ". ") {
		return false
	}
	for _, prefix := range shellBuiltins {
		if strings.HasPrefix(command, prefix) {
			return false
		}
	}
	// Streaming/follow invocations buffer indefinitely — never wrap them.
	if isStreamingInvocation(command) {
		return false
	}

	// Pipe and redirect operators — can't wrap safely (guard here covers both
	// single-command and individual compound segments).
	for _, op := range pipeRedirectOperators {
		if containsOutsideQuotes(command, op) {
			return false
		}
	}
	baseCmd, _ := splitHookInvocation(command)
	return supportedCommands[baseCmd]
}

// splitHookInvocation returns the normalized executable name and its raw
// whitespace-delimited arguments. The executable scan respects quoted paths,
// matching the hook's existing path and .exe handling.
func splitHookInvocation(command string) (string, []string) {
	// Find the end of the base command, respecting quotes.
	state := quoteNone
	endIdx := len(command)
	for i := 0; i < len(command); i++ {
		ch := command[i]
		switch state {
		case quoteNone:
			if ch == '\'' {
				state = quoteSingle
			} else if ch == '"' {
				state = quoteDouble
			} else if ch == ' ' {
				endIdx = i
				goto foundEnd
			}
		case quoteSingle:
			if ch == '\'' {
				state = quoteNone
			}
		case quoteDouble:
			if ch == '\\' && i+1 < len(command) {
				i++ // skip escaped char
			} else if ch == '"' {
				state = quoteNone
			}
		}
	}
foundEnd:
	baseCmd := command[:endIdx]

	if len(baseCmd) >= 2 && ((baseCmd[0] == '"' && baseCmd[len(baseCmd)-1] == '"') || (baseCmd[0] == '\'' && baseCmd[len(baseCmd)-1] == '\'')) {
		baseCmd = baseCmd[1 : len(baseCmd)-1]
	}
	if lastSlash := strings.LastIndexAny(baseCmd, `/\`); lastSlash != -1 {
		baseCmd = baseCmd[lastSlash+1:]
	}
	if strings.HasSuffix(strings.ToLower(baseCmd), ".exe") {
		baseCmd = baseCmd[:len(baseCmd)-4]
	}
	return baseCmd, strings.Fields(command[endIdx:])
}

// splitLogical splits a command on logical separators (" && ", " || ", " ; "),
// returning the segments and the operators between them.
// Only splits on operators that appear outside of quoted strings.
func splitLogical(command string) (segments []string, operators []string) {
	rest := command
	for {
		earliest := -1
		earliestOp := ""
		for _, op := range logicalSeparators {
			if idx := indexOutsideQuotes(rest, op); idx != -1 && (earliest == -1 || idx < earliest) {
				earliest = idx
				earliestOp = op
			}
		}
		if earliest == -1 {
			segments = append(segments, rest)
			break
		}
		segments = append(segments, rest[:earliest])
		operators = append(operators, earliestOp)
		rest = rest[earliest+len(earliestOp):]
	}
	return
}

// buildOutput constructs the hook JSON response for a rewritten command.
func buildOutput(original, wrapped string) ([]byte, bool, string) {
	out := hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName: "PreToolUse",
			UpdatedInput: toolInput{
				Command: wrapped,
			},
		},
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, false, original
	}
	return data, true, original
}

// auditLog appends a rewrite entry to the hook audit log.
// Silent on all errors - never slows down or breaks the hook.
func auditLog(original, rewritten string) {
	dir := config.DataDir()
	os.MkdirAll(dir, 0o700)
	path := filepath.Join(dir, "hook-audit.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "%s  rewrite  %s -> %s\n", ts, original, rewritten)
}

// AuditLogPath returns the path to the hook audit log file.
func AuditLogPath() (string, error) {
	return filepath.Join(config.DataDir(), "hook-audit.log"), nil
}
