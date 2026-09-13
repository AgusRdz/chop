package hooks

import (
	"encoding/json"
	"testing"
)

func TestPiHookWrapsBash(t *testing.T) {
	input := `{
		"session_id": "test",
		"cwd": "/tmp",
		"hook_event_name": "PreToolUse",
		"tool_name": "bash",
		"tool_input": {
			"command": "git status"
		}
	}`

	output, shouldModify, original := processPiHookInput([]byte(input))
	if !shouldModify {
		t.Fatal("expected shouldModify to be true")
	}
	if original != "git status" {
		t.Errorf("expected original 'git status', got %q", original)
	}

	var out piHookOutput
	if err := json.Unmarshal(output, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if out.HookSpecificOutput.UpdatedInput.Command != "chop git status" {
		t.Errorf("expected 'chop git status', got %q", out.HookSpecificOutput.UpdatedInput.Command)
	}
}

func TestPiHookIgnoresNonBashTool(t *testing.T) {
	input := `{
		"session_id": "test",
		"cwd": "/tmp",
		"hook_event_name": "PreToolUse",
		"tool_name": "read",
		"tool_input": {
			"path": "test.txt"
		}
	}`

	_, shouldModify, _ := processPiHookInput([]byte(input))
	if shouldModify {
		t.Error("expected shouldModify to be false for non-bash tool")
	}
}

func TestPiHookSkipsAlreadyWrapped(t *testing.T) {
	input := `{
		"session_id": "test",
		"cwd": "/tmp",
		"hook_event_name": "PreToolUse",
		"tool_name": "bash",
		"tool_input": {
			"command": "chop git status"
		}
	}`

	_, shouldModify, _ := processPiHookInput([]byte(input))
	if shouldModify {
		t.Error("expected shouldModify to be false for already wrapped command")
	}
}
