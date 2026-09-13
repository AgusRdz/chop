package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// piHookInput represents the JSON payload sent by the chop pi extension
// (hooks/pi_extension.ts) on each bash tool_call event.
type piHookInput struct {
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

func (h piHookInput) GetToolName() string {
	if h.ToolName != "" {
		return h.ToolName
	}
	return h.ToolNameCamel
}

func (h piHookInput) GetHookEventName() string {
	if h.HookEventName != "" {
		return h.HookEventName
	}
	return h.HookEventNameCamel
}

func (h piHookInput) GetToolInput() json.RawMessage {
	if len(h.ToolInput) > 0 {
		return h.ToolInput
	}
	return h.ToolInputCamel
}

// piToolInput matches Pi's bash tool input.
type piToolInput struct {
	Command string `json:"command"`
}

type piHookOutput struct {
	HookSpecificOutput piHookSpecificOutput `json:"hookSpecificOutput"`
}

type piHookSpecificOutput struct {
	HookEventName      string      `json:"hookEventName"`
	PermissionDecision string      `json:"permissionDecision,omitempty"`
	UpdatedInput       piToolInput `json:"updatedInput"`
}

// RunPiHook reads a payload from stdin (sent by the chop pi extension on a
// bash tool_call event), checks if the command should be wrapped with chop,
// and outputs modified JSON on stdout. Always exits 0.
func RunPiHook() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(0)
	}

	output, shouldModify, original := processPiHookInput(input)
	if shouldModify {
		var result piHookOutput
		if err := json.Unmarshal(output, &result); err == nil {
			auditLog(original, result.HookSpecificOutput.UpdatedInput.Command)
		}
		fmt.Print(string(output))
	}
	// If not modifying, output nothing (passthrough)
}

// processPiHookInput parses the extension's hook JSON and determines whether
// to wrap the command.
func processPiHookInput(input []byte) ([]byte, bool, string) {
	var h piHookInput
	if err := json.Unmarshal(input, &h); err != nil {
		return nil, false, ""
	}

	if h.GetHookEventName() != "PreToolUse" {
		return nil, false, ""
	}

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

	return buildPiOutput(original, wrapped)
}

// buildPiOutput constructs the hook JSON response consumed by the extension.
func buildPiOutput(original, wrapped string) ([]byte, bool, string) {
	out := piHookOutput{
		HookSpecificOutput: piHookSpecificOutput{
			HookEventName: "PreToolUse",
			UpdatedInput: piToolInput{
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
