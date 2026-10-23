package main

import "testing"

func TestQuoteWindowsArg(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"status", "status"},
		{"package-lock.json", "package-lock.json"},
		{"C:/repo/file.go", "C:/repo/file.go"},
		{"--format=%H", "--format=%H"},
		{"", `""`},
		{`"engines"`, `"\"engines\""`},
		{`engines"`, `"engines\""`},
		{`exec.Command\|newCommand(`, `"exec.Command\|newCommand("`},
		{"a b", `"a b"`},
		{`dir\`, `"dir\\"`},
		{`a\"b`, `"a\\\"b"`},
		{"*.go", `"*.go"`},
	}
	for _, tt := range tests {
		if got := quoteWindowsArg(tt.in); got != tt.want {
			t.Errorf("quoteWindowsArg(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWindowsCmdLine(t *testing.T) {
	if got := windowsCmdLine("git", []string{"status", "--short"}); got != "" {
		t.Errorf("plain args should keep default quoting, got %q", got)
	}
	got := windowsCmdLine("grep", []string{"-n", `"engines"`, "package-lock.json"})
	want := `grep -n "\"engines\"" package-lock.json`
	if got != want {
		t.Errorf("windowsCmdLine = %q, want %q", got, want)
	}
}
