package hooks

import (
	"errors"
	"testing"
)

func TestParseClaudeVersion(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"2.1.286 (Claude Code)", "2.1.286", false},
		{"2.1.286 (Claude Code)\r\n", "2.1.286", false},
		{"2.1.168", "2.1.168", false},
		{"2.1.167", "2.1.167", false},
		{"10.0.0", "10.0.0", false},
		{"garbage", "", true},
		{"", "", true},
		{"2.1", "", true},
	}
	for _, tt := range tests {
		got, err := parseClaudeVersion(tt.in)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("parseClaudeVersion(%q) = %q, %v; want %q, err=%v", tt.in, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestVersionAtLeastThreshold(t *testing.T) {
	tests := []struct {
		v    string
		want bool
	}{
		{"2.1.168", true},
		{"2.1.167", false},
		{"2.1.286", true},
		{"2.2.0", true},
		{"2.0.999", false},
		{"3.0.0", true},
		{"10.0.0", true},
		{"1.99.999", false},
		{"bogus", false},
	}
	for _, tt := range tests {
		if got := versionAtLeast(tt.v, conflictFixedIn); got != tt.want {
			t.Errorf("versionAtLeast(%q) = %v, want %v", tt.v, got, tt.want)
		}
	}
}

func TestConflictBugFixed(t *testing.T) {
	orig := claudeCodeVersion
	defer func() { claudeCodeVersion = orig }()

	tests := []struct {
		name      string
		ver       string
		err       error
		wantVer   string
		wantFixed bool
	}{
		{"new", "2.1.286", nil, "2.1.286", true},
		{"boundary", "2.1.168", nil, "2.1.168", true},
		{"old", "2.1.167", nil, "2.1.167", false},
		{"error", "", errors.New("claude not found"), "", false},
	}
	for _, tt := range tests {
		claudeCodeVersion = func() (string, error) { return tt.ver, tt.err }
		v, fixed := ConflictBugFixed()
		if v != tt.wantVer || fixed != tt.wantFixed {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", tt.name, v, fixed, tt.wantVer, tt.wantFixed)
		}
	}
}
