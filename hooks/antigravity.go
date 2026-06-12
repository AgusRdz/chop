package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// antigravityHookInput represents the JSON payload from Antigravity IDE's PreToolUse hook.
type antigravityHookInput struct {
	SessionID     string          `json:"session_id"`
	Cwd           string          `json:"cwd"`
	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
}

// antigravityToolInput matches Antigravity IDE's bash tool input.
type antigravityToolInput struct {
	Command string `json:"command"`
}

type antigravityHookOutput struct {
	HookSpecificOutput antigravityHookSpecificOutput `json:"hookSpecificOutput"`
}

type antigravityHookSpecificOutput struct {
	HookEventName      string               `json:"hookEventName"`
	PermissionDecision string               `json:"permissionDecision,omitempty"`
	UpdatedInput       antigravityToolInput `json:"updatedInput"`
}

// RunAntigravityHook reads an Antigravity IDE PreToolUse hook payload from stdin,
// checks if the command should be wrapped with chop, and outputs
// modified JSON on stdout. Always exits 0.
func RunAntigravityHook() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(0)
	}

	output, shouldModify, original := processAntigravityHookInput(input)
	if shouldModify {
		var result antigravityHookOutput
		if err := json.Unmarshal(output, &result); err == nil {
			auditLog(original, result.HookSpecificOutput.UpdatedInput.Command)
		}
		fmt.Print(string(output))
	}
	// If not modifying, output nothing (passthrough)
}

// processAntigravityHookInput parses the Antigravity IDE hook JSON and determines whether
// to wrap the command.
func processAntigravityHookInput(input []byte) ([]byte, bool, string) {
	var h antigravityHookInput
	if err := json.Unmarshal(input, &h); err != nil {
		return nil, false, ""
	}

	// Antigravity uses "bash" or "Bash"
	if h.ToolName != "bash" && h.ToolName != "Bash" {
		return nil, false, ""
	}

	// Global kill switch
	if IsDisabledGlobally() {
		return nil, false, ""
	}

	var ti antigravityToolInput
	if err := json.Unmarshal(h.ToolInput, &ti); err != nil {
		return nil, false, ""
	}

	wrapped, shouldModify, original := rewriteCommand(ti.Command)
	if !shouldModify {
		return nil, false, original
	}

	return buildAntigravityOutput(original, wrapped)
}

// buildAntigravityOutput constructs the Antigravity IDE hook JSON response.
func buildAntigravityOutput(original, wrapped string) ([]byte, bool, string) {
	out := antigravityHookOutput{
		HookSpecificOutput: antigravityHookSpecificOutput{
			HookEventName: "PreToolUse",
			UpdatedInput: antigravityToolInput{
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
