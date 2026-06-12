package hooks

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGeminiHookWrapsShellCommand(t *testing.T) {
	input := `{
		"session_id": "test",
		"cwd": "/tmp",
		"hook_event_name": "BeforeTool",
		"tool_name": "run_shell_command",
		"tool_input": {"command": "git status"}
	}`

	output, shouldModify, _ := processGeminiHookInput([]byte(input))
	if !shouldModify {
		t.Fatal("expected command to be wrapped")
	}

	var result geminiHookOutput
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result.Decision != "" {
		t.Errorf("expected no decision override (let platform decide), got %q", result.Decision)
	}
	if !strings.HasPrefix(result.HookSpecificOutput.ToolInput.Command, "chop ") {
		t.Errorf("expected command to start with 'chop ', got %q", result.HookSpecificOutput.ToolInput.Command)
	}
	if !strings.Contains(result.HookSpecificOutput.ToolInput.Command, "git status") {
		t.Errorf("expected command to contain 'git status', got %q", result.HookSpecificOutput.ToolInput.Command)
	}
}

func TestGeminiHookIgnoresNonShellTool(t *testing.T) {
	input := `{
		"session_id": "test",
		"cwd": "/tmp",
		"hook_event_name": "BeforeTool",
		"tool_name": "read_file",
		"tool_input": {"path": "/tmp/foo.txt"}
	}`

	_, shouldModify, _ := processGeminiHookInput([]byte(input))
	if shouldModify {
		t.Fatal("should not modify non-shell tool")
	}
}

func TestGeminiHookPassesThroughUnsupportedCommand(t *testing.T) {
	input := `{
		"session_id": "test",
		"cwd": "/tmp",
		"hook_event_name": "BeforeTool",
		"tool_name": "run_shell_command",
		"tool_input": {"command": "unknown-tool --flag"}
	}`

	_, shouldModify, _ := processGeminiHookInput([]byte(input))
	if shouldModify {
		t.Fatal("should not modify unsupported command")
	}
}

func TestGeminiHookSkipsAlreadyWrapped(t *testing.T) {
	input := `{
		"session_id": "test",
		"cwd": "/tmp",
		"hook_event_name": "BeforeTool",
		"tool_name": "run_shell_command",
		"tool_input": {"command": "chop git status"}
	}`

	_, shouldModify, _ := processGeminiHookInput([]byte(input))
	if shouldModify {
		t.Fatal("should not modify already-wrapped command")
	}
}

func TestGeminiHookCompoundCommand(t *testing.T) {
	input := `{
		"session_id": "test",
		"cwd": "/tmp",
		"hook_event_name": "BeforeTool",
		"tool_name": "run_shell_command",
		"tool_input": {"command": "cd /app && npm test"}
	}`

	output, shouldModify, _ := processGeminiHookInput([]byte(input))
	if !shouldModify {
		t.Fatal("expected compound command to be wrapped")
	}

	var result geminiHookOutput
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	cmd := result.HookSpecificOutput.ToolInput.Command
	if !strings.Contains(cmd, "cd /app") {
		t.Errorf("expected cd to be preserved, got %q", cmd)
	}
	if !strings.Contains(cmd, "chop npm test") {
		t.Errorf("expected npm test to be wrapped, got %q", cmd)
	}
}

func TestGeminiHookOutputFormat(t *testing.T) {
	input := `{
		"session_id": "test",
		"cwd": "/tmp",
		"hook_event_name": "BeforeTool",
		"tool_name": "run_shell_command",
		"tool_input": {"command": "docker ps"}
	}`

	output, shouldModify, _ := processGeminiHookInput([]byte(input))
	if !shouldModify {
		t.Fatal("expected command to be wrapped")
	}

	// Verify the output is valid JSON with the correct Gemini structure
	var raw map[string]interface{}
	if err := json.Unmarshal(output, &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if _, hasDecision := raw["decision"]; hasDecision {
		t.Errorf("expected no 'decision' field (should not override permissions), got %v", raw["decision"])
	}

	hso, ok := raw["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("expected hookSpecificOutput in output")
	}

	ti, ok := hso["tool_input"].(map[string]interface{})
	if !ok {
		t.Fatal("expected tool_input in hookSpecificOutput")
	}

	cmd, ok := ti["command"].(string)
	if !ok || cmd != "chop docker ps" {
		t.Errorf("expected 'chop docker ps', got %q", cmd)
	}
}
