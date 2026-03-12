package filters

import (
	"strings"
	"testing"

	"github.com/AgusRdz/chop/config"
)

func TestBuildUserFilter_Nil(t *testing.T) {
	fn := BuildUserFilter(nil)
	if fn != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestBuildUserFilter_Empty(t *testing.T) {
	cf := &config.CustomFilter{}
	fn := BuildUserFilter(cf)
	if fn != nil {
		t.Fatal("expected nil for empty filter")
	}
}

func TestBuildUserFilter_KeepOnly(t *testing.T) {
	cf := &config.CustomFilter{
		Keep: []string{"ERROR", "WARN"},
	}
	fn := BuildUserFilter(cf)
	if fn == nil {
		t.Fatal("expected non-nil filter")
	}

	input := "INFO starting\nERROR failed\nDEBUG trace\nWARNING slow\nINFO done"
	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "ERROR failed") {
		t.Error("expected ERROR line to be kept")
	}
	if !strings.Contains(result, "WARNING slow") {
		t.Error("expected WARNING line to be kept")
	}
	if strings.Contains(result, "INFO starting") {
		t.Error("expected INFO line to be dropped")
	}
	if strings.Contains(result, "DEBUG trace") {
		t.Error("expected DEBUG line to be dropped")
	}
}

func TestBuildUserFilter_DropOnly(t *testing.T) {
	cf := &config.CustomFilter{
		Drop: []string{"DEBUG", "TRACE"},
	}
	fn := BuildUserFilter(cf)
	if fn == nil {
		t.Fatal("expected non-nil filter")
	}

	input := "INFO starting\nDEBUG init\nTRACE details\nERROR fail\nDEBUG cleanup"
	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(result, "DEBUG") {
		t.Error("expected DEBUG lines to be dropped")
	}
	if strings.Contains(result, "TRACE") {
		t.Error("expected TRACE lines to be dropped")
	}
	if !strings.Contains(result, "INFO starting") {
		t.Error("expected INFO line to be kept")
	}
	if !strings.Contains(result, "ERROR fail") {
		t.Error("expected ERROR line to be kept")
	}
}

func TestBuildUserFilter_KeepAndDrop(t *testing.T) {
	cf := &config.CustomFilter{
		Keep: []string{"ERROR", "WARN", "INFO"},
		Drop: []string{"DEBUG"},
	}
	fn := BuildUserFilter(cf)
	if fn == nil {
		t.Fatal("expected non-nil filter")
	}

	input := "DEBUG init\nINFO starting\nDEBUG trace\nERROR fail\nINFO done"
	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}

	// DEBUG should be dropped first, then keep applied to remaining
	if strings.Contains(result, "DEBUG") {
		t.Error("expected DEBUG lines to be dropped")
	}
	if !strings.Contains(result, "INFO starting") {
		t.Error("expected INFO line to be kept")
	}
	if !strings.Contains(result, "ERROR fail") {
		t.Error("expected ERROR line to be kept")
	}
}

func TestBuildUserFilter_HeadOnly(t *testing.T) {
	cf := &config.CustomFilter{
		Head: 3,
	}
	fn := BuildUserFilter(cf)

	lines := make([]string, 20)
	for i := range lines {
		lines[i] = strings.Repeat("x", 10)
	}
	input := strings.Join(lines, "\n")

	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "... (17 more lines)") {
		t.Errorf("expected truncation message, got:\n%s", result)
	}

	resultLines := strings.Split(result, "\n")
	// 3 content lines + 1 truncation line
	if len(resultLines) != 4 {
		t.Errorf("expected 4 lines, got %d", len(resultLines))
	}
}

func TestBuildUserFilter_TailOnly(t *testing.T) {
	cf := &config.CustomFilter{
		Tail: 3,
	}
	fn := BuildUserFilter(cf)

	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, strings.Repeat("x", 10))
	}
	input := strings.Join(lines, "\n")

	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "... (17 lines skipped)") {
		t.Errorf("expected skip message, got:\n%s", result)
	}
}

func TestBuildUserFilter_HeadAndTail(t *testing.T) {
	cf := &config.CustomFilter{
		Head: 2,
		Tail: 2,
	}
	fn := BuildUserFilter(cf)

	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "line"+strings.Repeat(" ", i))
	}
	input := strings.Join(lines, "\n")

	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "... (6 lines hidden)") {
		t.Errorf("expected hidden message, got:\n%s", result)
	}
}

func TestBuildUserFilter_EmptyInput(t *testing.T) {
	cf := &config.CustomFilter{
		Keep: []string{"ERROR"},
	}
	fn := BuildUserFilter(cf)

	result, err := fn("")
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestBuildUserFilter_RegexPatterns(t *testing.T) {
	cf := &config.CustomFilter{
		Keep: []string{`^\[.*\]`, `^=+`},
	}
	fn := BuildUserFilter(cf)

	input := "[2026-01-01] Event occurred\nrandom noise\n=== Section ===\nmore noise"
	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "[2026-01-01] Event occurred") {
		t.Error("expected bracketed line to match")
	}
	if !strings.Contains(result, "=== Section ===") {
		t.Error("expected section line to match")
	}
	if strings.Contains(result, "random noise") {
		t.Error("expected noise to be filtered")
	}
}

func TestBuildUserFilter_InvalidRegexSkipped(t *testing.T) {
	cf := &config.CustomFilter{
		Keep: []string{"[invalid", "ERROR"},
	}
	fn := BuildUserFilter(cf)
	if fn == nil {
		t.Fatal("expected non-nil filter even with invalid regex")
	}

	input := "ERROR something\nINFO other"
	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "ERROR") {
		t.Error("expected valid pattern to still work")
	}
}

func TestBuildUserFilter_ShortInputNoTruncation(t *testing.T) {
	cf := &config.CustomFilter{
		Head: 100,
	}
	fn := BuildUserFilter(cf)

	input := "line1\nline2\nline3"
	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}
	if result != input {
		t.Errorf("expected passthrough for short input, got:\n%s", result)
	}
}

func TestBuildUserFilter_DropWithRegex(t *testing.T) {
	cf := &config.CustomFilter{
		Drop: []string{`^\s*$`, `^#`},
	}
	fn := BuildUserFilter(cf)

	input := "data1\n\n# comment\ndata2\n   \ndata3"
	result, err := fn(input)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(result, "# comment") {
		t.Error("expected comment to be dropped")
	}
	if !strings.Contains(result, "data1") || !strings.Contains(result, "data2") || !strings.Contains(result, "data3") {
		t.Error("expected data lines to be kept")
	}
}

func TestExpandHome(t *testing.T) {
	result := expandHome("/absolute/path")
	if result != "/absolute/path" {
		t.Errorf("absolute path should be unchanged, got %q", result)
	}

	result = expandHome("relative/path")
	if result != "relative/path" {
		t.Errorf("relative path should be unchanged, got %q", result)
	}

	result = expandHome("~/something")
	if strings.HasPrefix(result, "~") {
		t.Errorf("tilde should be expanded, got %q", result)
	}
}

// TestGetAndHasFilterWithUserFilters verifies that SetUserFilters wires into
// Get() and HasFilter() correctly.
func TestGetAndHasFilterWithUserFilters(t *testing.T) {
	// Reset global state after test
	defer SetUserFilters(nil)

	SetUserFilters(map[string]config.CustomFilter{
		"mytool deploy": {Keep: []string{"ERROR", "WARN"}},
		"mytool":        {Drop: []string{"DEBUG"}},
	})

	t.Run("Get returns user filter for subcommand", func(t *testing.T) {
		fn := Get("mytool", []string{"deploy"})
		if fn == nil {
			t.Fatal("expected filter, got nil")
		}
		result, err := fn("ERROR bad\nINFO ok\nDEBUG trace")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result, "ERROR bad") {
			t.Error("expected ERROR line in output")
		}
		if strings.Contains(result, "INFO ok") {
			t.Error("expected INFO line to be filtered out")
		}
	})

	t.Run("Get falls back to base command", func(t *testing.T) {
		fn := Get("mytool", []string{"status"})
		if fn == nil {
			t.Fatal("expected filter, got nil")
		}
		result, err := fn("DEBUG noise\nINFO data")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(result, "DEBUG") {
			t.Error("expected DEBUG line to be dropped")
		}
	})

	t.Run("HasFilter true for registered command", func(t *testing.T) {
		if !HasFilter("mytool", []string{"deploy"}) {
			t.Error("expected HasFilter true for mytool deploy")
		}
	})

	t.Run("HasFilter false for unknown command", func(t *testing.T) {
		// "unknowntool" has no user filter and no built-in filter
		if HasFilter("unknowntool", nil) {
			t.Error("expected HasFilter false for unknown command")
		}
	})
}
