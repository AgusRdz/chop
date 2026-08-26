package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeInput(command string) []byte {
	input := map[string]interface{}{
		"session_id":      "test-session",
		"cwd":             "/tmp",
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
		"tool_input": map[string]string{
			"command": command,
		},
	}
	data, _ := json.Marshal(input)
	return data
}

func TestSupportedCommandGetsPrepended(t *testing.T) {
	tests := []struct {
		cmd      string
		expected string
	}{
		{"npm test", "chop npm test"},
		{"git status", "chop git status"},
		{"docker ps", "chop docker ps"},
		{"kubectl get pods", "chop kubectl get pods"},
		{"cargo build", "chop cargo build"},
		{"go test ./...", "chop go test ./..."},
		{"curl https://api.io", "chop curl https://api.io"},
		{"dotnet build", "chop dotnet build"},
		{"cat /var/log/syslog", "chop cat /var/log/syslog"},
		{"tail -n 100 /var/log/app.log", "chop tail -n 100 /var/log/app.log"},
		{"find . -name '*.go'", "chop find . -name '*.go'"},
		{`"/usr/bin/npm" install`, `chop "/usr/bin/npm" install`},
		{`'/usr/bin/npm' install`, `chop '/usr/bin/npm' install`},
		{`"C:\Program Files\nodejs\npm.exe" install`, `chop "C:\Program Files\nodejs\npm.exe" install`},
		{`'C:\Program Files\nodejs\npm.exe' install`, `chop 'C:\Program Files\nodejs\npm.exe' install`},
		{`npm.exe install`, `chop npm.exe install`},
		{`"npm.exe" install`, `chop "npm.exe" install`},
		{`"npm" install`, `chop "npm" install`},
		{`'npm' install`, `chop 'npm' install`},
		// fd-style redirects must not block wrapping; 2>&1 is stripped (redundant with CombinedOutput)
		{`npm run test 2>&1`, `chop npm run test`},
		{`npm test -- --watch=false 2>&1`, `chop npm test -- --watch=false`},
		{`go test ./... 2>&1`, `chop go test ./...`},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			output, shouldModify, _ := processHookInput(makeInput(tt.cmd))
			if !shouldModify {
				t.Fatalf("expected command to be modified: %s", tt.cmd)
			}

			var result hookOutput
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("failed to parse output JSON: %v", err)
			}

			if result.HookSpecificOutput.UpdatedInput.Command != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.HookSpecificOutput.UpdatedInput.Command)
			}
			if result.HookSpecificOutput.PermissionDecision != "" {
				t.Errorf("expected no permissionDecision (let platform decide), got %q", result.HookSpecificOutput.PermissionDecision)
			}
			if result.HookSpecificOutput.HookEventName != "PreToolUse" {
				t.Errorf("expected hookEventName 'PreToolUse', got %q", result.HookSpecificOutput.HookEventName)
			}
		})
	}
}

func TestAlreadyChoppedPassthrough(t *testing.T) {
	_, shouldModify, _ := processHookInput(makeInput("chop git status"))
	if shouldModify {
		t.Error("should not modify already-chopped command")
	}
}

func TestPipePassthrough(t *testing.T) {
	tests := []string{
		"git log | head -10",
		"docker ps | grep running",
		"cat file.txt | wc -l",
	}
	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			_, shouldModify, _ := processHookInput(makeInput(cmd))
			if shouldModify {
				t.Errorf("should not modify pipe command: %s", cmd)
			}
		})
	}
}

func TestStreamingPassthrough(t *testing.T) {
	tests := []string{
		"tail -f /var/log/app.log",
		"tail --follow=name /var/log/app.log",
		"kubectl logs -f pod/my-pod",
		"kubectl logs --follow pod/my-pod",
		"docker logs -f my-container",
		"docker logs --follow=true my-container",
		"npm test -- --watchAll",
		"docker compose up",
		"stern my-pod",
		"aws logs tail /aws/lambda/my-fn --follow",
		"argocd app logs my-app --follow",
		"flux logs --follow",
		"ping google.com",
		"ping -n google.com",
	}
	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			_, shouldModify, _ := processHookInput(makeInput(cmd))
			if shouldModify {
				t.Errorf("should not modify streaming command: %s", cmd)
			}
		})
	}
}

func TestFiniteCommandsWithFollowLikeFlagsAreWrapped(t *testing.T) {
	tests := []struct {
		cmd      string
		expected string
	}{
		{"mvn -f pom.xml test", "chop mvn -f pom.xml test"},
		{"git fetch -f", "chop git fetch -f"},
		{"git log --follow README.md", "chop git log --follow README.md"},
		{"rg --follow pattern .", "chop rg --follow pattern ."},
		{"npm test -- --watchAll=false", "chop npm test -- --watchAll=false"},
		{"npm test -- --watchman", "chop npm test -- --watchman"},
		{"kubectl apply -f manifest.yaml", "chop kubectl apply -f manifest.yaml"},
		{"aws s3 cp . s3://bucket --recursive --follow-symlinks", "chop aws s3 cp . s3://bucket --recursive --follow-symlinks"},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			output, shouldModify, _ := processHookInput(makeInput(tt.cmd))
			if !shouldModify {
				t.Fatalf("expected finite command to be modified: %s", tt.cmd)
			}

			var result hookOutput
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("failed to parse output JSON: %v", err)
			}
			if result.HookSpecificOutput.UpdatedInput.Command != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.HookSpecificOutput.UpdatedInput.Command)
			}
		})
	}
}

func TestRedirectPassthrough(t *testing.T) {
	tests := []string{
		"git diff > output.txt",
		"echo hello >> log.txt",
		"docker run < input.txt",
	}
	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			_, shouldModify, _ := processHookInput(makeInput(cmd))
			if shouldModify {
				t.Errorf("should not modify redirect command: %s", cmd)
			}
		})
	}
}

func TestCompoundWithRedirectPassthrough(t *testing.T) {
	// A segment containing a redirect or pipe must not be wrapped (HIGH-002).
	// Each case lists the bad wrapping that must NOT appear in the output.
	tests := []struct {
		cmd         string
		mustNotWrap string
	}{
		{"git fetch && git log > full.txt", "chop git log > full.txt"},
		{"git diff > output.txt && git status", "chop git diff"},
		{"a && b | grep X", "chop b | grep"},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			output, shouldModify, _ := processHookInput(makeInput(tt.cmd))
			if shouldModify {
				var result hookOutput
				if err := json.Unmarshal(output, &result); err == nil {
					wrapped := result.HookSpecificOutput.UpdatedInput.Command
					if strings.Contains(wrapped, tt.mustNotWrap) {
						t.Errorf("redirect/pipe-containing segment was wrapped: %q in %q", tt.mustNotWrap, wrapped)
					}
				}
			}
		})
	}
}

func TestCompoundCommandPassthrough(t *testing.T) {
	// Commands where no segment is supported — should pass through unchanged.
	tests := []string{
		"cd /tmp; ls",
		"mkdir foo && cd foo",
	}
	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			_, shouldModify, _ := processHookInput(makeInput(cmd))
			if shouldModify {
				t.Errorf("should not modify compound command: %s", cmd)
			}
		})
	}
}

func TestCompoundCommandWrapping(t *testing.T) {
	tests := []struct {
		cmd      string
		expected string
	}{
		{
			"git add . && git commit -m 'test'",
			"chop git add . && chop git commit -m 'test'",
		},
		{
			"npm install || echo failed",
			"chop npm install || echo failed",
		},
		{
			"go build ./... && go test ./...",
			"chop go build ./... && chop go test ./...",
		},
		{
			"docker build . && docker ps",
			"chop docker build . && chop docker ps",
		},
		{
			`cd /path/to/project && npm run test 2>&1`,
			`cd /path/to/project && chop npm run test`,
		},
		{
			`cd "/c/Users/user/repos/project" && npm run test`,
			`cd "/c/Users/user/repos/project" && chop npm run test`,
		},
		{
			`cd /app && docker compose up -d && git status`,
			`cd /app && chop docker compose up -d && chop git status`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			output, shouldModify, _ := processHookInput(makeInput(tt.cmd))
			if !shouldModify {
				t.Fatalf("expected compound command to be modified: %s", tt.cmd)
			}
			var result hookOutput
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("failed to parse output JSON: %v", err)
			}
			if result.HookSpecificOutput.UpdatedInput.Command != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.HookSpecificOutput.UpdatedInput.Command)
			}
		})
	}
}

func TestQuotedOperatorsNotSplit(t *testing.T) {
	// Operators inside quotes must NOT be treated as logical separators.
	tests := []struct {
		cmd      string
		expected string
	}{
		// && inside double quotes — single grep, wrapped
		{`grep "foo && bar" file.txt`, `chop grep "foo && bar" file.txt`},
		// || inside double quotes — single grep, wrapped
		{`grep "feat || fix" logs.txt`, `chop grep "feat || fix" logs.txt`},
		// && inside single quotes — single grep, wrapped
		{`grep '$1 > 0 && $2 < 100' data.csv`, `chop grep '$1 > 0 && $2 < 100' data.csv`},
		// || inside --grep= value
		{`git log --grep="feat || fix" --oneline`, `chop git log --grep="feat || fix" --oneline`},
		// real && after quoted section — only the real operator splits
		{`grep "a && b" file.txt && echo done`, `chop grep "a && b" file.txt && echo done`},
		// escaped quote inside double-quoted string
		{`grep "say \"hello && bye\"" file.txt`, `chop grep "say \"hello && bye\"" file.txt`},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			output, shouldModify, _ := processHookInput(makeInput(tt.cmd))
			if !shouldModify {
				t.Fatalf("expected command to be modified: %s", tt.cmd)
			}
			var result hookOutput
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("failed to parse output JSON: %v", err)
			}
			if result.HookSpecificOutput.UpdatedInput.Command != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.HookSpecificOutput.UpdatedInput.Command)
			}
		})
	}
}

func TestQuotedRedirectNotSkipped(t *testing.T) {
	// > and < inside quotes must NOT trigger the redirect passthrough.
	tests := []struct {
		cmd      string
		expected string
	}{
		{`grep '$1 > 0' data.csv`, `chop grep '$1 > 0' data.csv`},
		{`grep "a > b" file.txt`, `chop grep "a > b" file.txt`},
		{`grep "x < y" file.txt`, `chop grep "x < y" file.txt`},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			output, shouldModify, _ := processHookInput(makeInput(tt.cmd))
			if !shouldModify {
				t.Fatalf("expected command to be modified: %s", tt.cmd)
			}
			var result hookOutput
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("failed to parse output JSON: %v", err)
			}
			if result.HookSpecificOutput.UpdatedInput.Command != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.HookSpecificOutput.UpdatedInput.Command)
			}
		})
	}
}

func TestUnsupportedCommandPassthrough(t *testing.T) {
	tests := []string{
		"vim file.txt",
		"nano config.yml",
		"ls -la",
		"mkdir newdir",
		"rm -rf temp",
	}
	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			_, shouldModify, _ := processHookInput(makeInput(cmd))
			if shouldModify {
				t.Errorf("should not modify unsupported command: %s", cmd)
			}
		})
	}
}

func TestEmptyCommandPassthrough(t *testing.T) {
	_, shouldModify, _ := processHookInput(makeInput(""))
	if shouldModify {
		t.Error("should not modify empty command")
	}
}

func TestShellBuiltinPassthrough(t *testing.T) {
	tests := []string{
		"cd /tmp",
		"export FOO=bar",
		"source ~/.bashrc",
		". ~/.bashrc",
		"echo hello world",
		"printf '%s\\n' hello",
		"set -e",
		"unset FOO",
		"alias ll='ls -la'",
		"eval some_command",
	}
	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			_, shouldModify, _ := processHookInput(makeInput(cmd))
			if shouldModify {
				t.Errorf("should not modify shell builtin: %s", cmd)
			}
		})
	}
}

func TestNonBashToolPassthrough(t *testing.T) {
	input := map[string]interface{}{
		"session_id":      "test-session",
		"cwd":             "/tmp",
		"hook_event_name": "PreToolUse",
		"tool_name":       "Read",
		"tool_input": map[string]string{
			"file_path": "/some/file.txt",
		},
	}
	data, _ := json.Marshal(input)
	_, shouldModify, _ := processHookInput(data)
	if shouldModify {
		t.Error("should not modify non-Bash tool")
	}
}

func TestInvalidJSONPassthrough(t *testing.T) {
	_, shouldModify, _ := processHookInput([]byte("not json"))
	if shouldModify {
		t.Error("should not modify invalid JSON")
	}
}

func TestAuditLogWritesToFile(t *testing.T) {
	// Use a temp directory to avoid polluting the real audit log
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "hook-audit.log")

	// Manually write audit entry (same logic as auditLog but to temp path)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("failed to open audit log: %v", err)
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	original := "git status"
	rewritten := "chop git status"
	fmt.Fprintf(f, "%s  rewrite  %s -> %s\n", ts, original, rewritten)
	f.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "rewrite") {
		t.Error("audit log should contain 'rewrite'")
	}
	if !strings.Contains(content, "git status -> chop git status") {
		t.Errorf("audit log should contain rewrite entry, got: %s", content)
	}

	// Verify it's a single line
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}
}

func TestAuditLogAppendsMultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "hook-audit.log")

	for i := 0; i < 3; i++ {
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatalf("failed to open audit log: %v", err)
		}
		ts := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(f, "%s  rewrite  cmd%d -> chop cmd%d\n", ts, i, i)
		f.Close()
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}
