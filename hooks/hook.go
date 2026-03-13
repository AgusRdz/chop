package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
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
}

// shellBuiltins are commands that should never be wrapped.
var shellBuiltins = []string{
	"cd ", "export ", "source ", "echo ", "printf ", "set ", "unset ", "alias ", "eval ",
}

// pipeRedirectOperators make wrapping ambiguous — skip the entire command.
// File redirects always have surrounding spaces ("> file", ">> file", "< file").
// fd-style redirects like "2>&1" intentionally do not match so they can be wrapped.
var pipeRedirectOperators = []string{" | ", " > ", " >> ", " < "}

// logicalSeparators chain independent commands — split and wrap each segment.
var logicalSeparators = []string{" && ", " || ", " ; "}

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
	PermissionDecision string    `json:"permissionDecision"`
	UpdatedInput       toolInput `json:"updatedInput"`
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

// processHookInput parses the hook JSON and determines whether to wrap the command.
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

	command := strings.TrimSpace(ti.Command)
	if command == "" {
		return nil, false, ""
	}

	// Already wrapped with chop
	if strings.HasPrefix(command, "chop ") {
		return nil, false, command
	}

	// Starts with a dot (source shorthand)
	if strings.HasPrefix(command, ". ") {
		return nil, false, command
	}

	// Logical chaining operators — split first, then evaluate each segment independently.
	// Must run before the shell-builtin and redirect checks so that commands like
	// `cd /path && npm test 2>&1` are split into segments rather than rejected wholesale.
	for _, op := range logicalSeparators {
		if containsOutsideQuotes(command, op) {
			return wrapCompound(command)
		}
	}

	// Shell builtins
	for _, prefix := range shellBuiltins {
		if strings.HasPrefix(command, prefix) {
			return nil, false, command
		}
	}

	// Pipe and redirect operators — can't wrap safely, pass through unchanged.
	for _, op := range pipeRedirectOperators {
		if containsOutsideQuotes(command, op) {
			return nil, false, command
		}
	}

	// Single command — wrap if supported.
	if !shouldWrap(command) {
		return nil, false, command
	}

	return buildOutput(command, "chop "+command)
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
	baseCmd := command

	// Find the end of the base command, respecting quotes
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
	baseCmd = command[:endIdx]

	if len(baseCmd) >= 2 && ((baseCmd[0] == '"' && baseCmd[len(baseCmd)-1] == '"') || (baseCmd[0] == '\'' && baseCmd[len(baseCmd)-1] == '\'')) {
		baseCmd = baseCmd[1 : len(baseCmd)-1]
	}
	if lastSlash := strings.LastIndexAny(baseCmd, `/\`); lastSlash != -1 {
		baseCmd = baseCmd[lastSlash+1:]
	}
	if strings.HasSuffix(strings.ToLower(baseCmd), ".exe") {
		baseCmd = baseCmd[:len(baseCmd)-4]
	}
	return supportedCommands[baseCmd]
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

// wrapCompound splits a compound command on logical operators and wraps each
// supported segment with chop, reassembling the result.
func wrapCompound(command string) ([]byte, bool, string) {
	segments, operators := splitLogical(command)
	modified := false
	result := make([]string, len(segments))
	for i, seg := range segments {
		seg = strings.TrimSpace(seg)
		if shouldWrap(seg) {
			result[i] = "chop " + seg
			modified = true
		} else {
			result[i] = seg
		}
	}
	if !modified {
		return nil, false, command
	}
	var sb strings.Builder
	for i, seg := range result {
		if i > 0 {
			sb.WriteString(operators[i-1])
		}
		sb.WriteString(seg)
	}
	return buildOutput(command, sb.String())
}

// buildOutput constructs the hook JSON response for a rewritten command.
func buildOutput(original, wrapped string) ([]byte, bool, string) {
	out := hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName:      "PreToolUse",
			PermissionDecision: "allow",
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
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".local", "share", "chop")
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "hook-audit.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "%s  rewrite  %s -> %s\n", ts, original, rewritten)
}

// AuditLogPath returns the path to the hook audit log file.
func AuditLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "chop", "hook-audit.log"), nil
}
