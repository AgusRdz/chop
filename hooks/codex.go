package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// codexHookInput represents the JSON payload from Codex CLI's PreToolUse hook.
// Based on typical PreToolUse implementations.
type codexHookInput struct {
	SessionID          string          `json:"session_id"`
	SessionIDCamel     string          `json:"sessionId"`
	Cwd                string          `json:"cwd"`
	HookEventName      string          `json:"hook_event_name"`
	HookEventNameCamel string          `json:"hookEventName"`
	ToolName           string          `json:"tool_name"`
	ToolNameCamel      string          `json:"toolName"`
	ToolInput          json.RawMessage `json:"tool_input"`
	ToolInputCamel     json.RawMessage `json:"toolInput"`
}

func (h codexHookInput) GetToolName() string {
	if h.ToolName != "" {
		return h.ToolName
	}
	return h.ToolNameCamel
}

func (h codexHookInput) GetHookEventName() string {
	if h.HookEventName != "" {
		return h.HookEventName
	}
	return h.HookEventNameCamel
}

func (h codexHookInput) GetToolInput() json.RawMessage {
	if len(h.ToolInput) > 0 {
		return h.ToolInput
	}
	return h.ToolInputCamel
}

// codexToolInput matches Codex CLI's bash tool input.
type codexToolInput struct {
	Command string `json:"command"`
}

type codexHookOutput struct {
	HookSpecificOutput codexHookSpecificOutput `json:"hookSpecificOutput"`
}

type codexHookSpecificOutput struct {
	HookEventName      string         `json:"hookEventName"`
	PermissionDecision string         `json:"permissionDecision,omitempty"`
	UpdatedInput       codexToolInput `json:"updatedInput"`
}

// RunCodexHook reads a Codex CLI PreToolUse hook payload from stdin,
// checks if the command should be wrapped with chop, and outputs
// modified JSON on stdout. Always exits 0.
func RunCodexHook() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(0)
	}

	output, shouldModify, original := processCodexHookInput(input)
	if shouldModify {
		var result codexHookOutput
		if err := json.Unmarshal(output, &result); err == nil {
			auditLog(original, result.HookSpecificOutput.UpdatedInput.Command)
		}
		fmt.Print(string(output))
	}
	// If not modifying, output nothing (passthrough)
}

// processCodexHookInput parses the Codex CLI hook JSON and determines whether
// to wrap the command.
func processCodexHookInput(input []byte) ([]byte, bool, string) {
	var h codexHookInput
	if err := json.Unmarshal(input, &h); err != nil {
		return nil, false, ""
	}

	if h.GetHookEventName() != "PreToolUse" {
		return nil, false, ""
	}

	// Codex CLI uses "bash" or "Bash"
	toolName := h.GetToolName()
	if toolName != "bash" && toolName != "Bash" {
		return nil, false, ""
	}

	// Global kill switch
	if IsDisabledGlobally() {
		return nil, false, ""
	}

	var rti resilientToolInput
	if err := json.Unmarshal(h.GetToolInput(), &rti); err != nil {
		return nil, false, ""
	}

	cmd := rti.Command
	if cmd == "" {
		if rti.CmdUpper != "" {
			cmd = rti.CmdUpper
		} else {
			cmd = rti.CmdLower
		}
	}

	wrapped, shouldModify, original := rewriteCommand(cmd)
	if !shouldModify {
		return nil, false, original
	}

	return buildCodexOutput(original, wrapped)
}

// buildCodexOutput constructs the Codex CLI hook JSON response.
func buildCodexOutput(original, wrapped string) ([]byte, bool, string) {
	out := codexHookOutput{
		HookSpecificOutput: codexHookSpecificOutput{
			HookEventName: "PreToolUse",
			UpdatedInput: codexToolInput{
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
