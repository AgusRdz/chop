package filters

import (
	"strings"
	"testing"
)

func TestFilterRuff(t *testing.T) {
	raw := "src/app.py:1:1: F401 [*] `os` imported but unused\n" +
		"src/app.py:5:1: E302 [*] Expected 2 blank lines, found 1\n" +
		"src/utils.py:10:5: F841 Local variable `x` is assigned to but never used\n" +
		"Found 3 errors.\n" +
		"[*] 2 fixable with the `--fix` option.\n"

	got, err := filterRuff(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression")
	}
	if !strings.Contains(got, "F401") {
		t.Error("expected error code")
	}
	if !strings.Contains(got, "3 problems") {
		t.Error("expected problem count")
	}
}

func TestFilterPylint(t *testing.T) {
	raw := "************* Module app\n" +
		"src/app.py:1:0: C0114: Missing module docstring (missing-module-docstring)\n" +
		"src/app.py:5:0: C0116: Missing function or method docstring (missing-function-docstring)\n"

	got, err := filterPylint(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "C0114") {
		t.Error("expected pylint code")
	}
	if !strings.Contains(got, "2 problems") {
		t.Error("expected problem count")
	}
}

func TestFilterRuff_Empty(t *testing.T) {
	got, err := filterRuff("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "no problems" {
		t.Errorf("expected 'no problems', got %q", got)
	}
}

func TestFilterRuff_LargeCheck(t *testing.T) {
	raw := "src/models/user.py:1:1: D100 Missing docstring in public module\n" +
		"src/models/user.py:5:1: I001 [*] Import block is un-sorted or un-formatted\n" +
		"src/models/user.py:12:80: E501 Line too long (95 > 79)\n" +
		"src/models/user.py:15:5: F841 Local variable `temp` is assigned to but never used\n" +
		"src/services/auth.py:1:1: D100 Missing docstring in public module\n" +
		"src/services/auth.py:8:1: F401 [*] `os.path` imported but unused\n" +
		"src/services/auth.py:22:12: E711 Comparison to `None` (use `is None`)\n" +
		"src/services/auth.py:35:5: B006 Do not use mutable data structures for argument defaults\n" +
		"src/api/views.py:3:1: I001 [*] Import block is un-sorted or un-formatted\n" +
		"src/api/views.py:18:5: E722 Do not use bare `except`\n" +
		"src/api/views.py:25:1: C901 `process_request` is too complex (15 > 10)\n" +
		"src/api/views.py:67:80: E501 Line too long (102 > 79)\n" +
		"src/utils/helpers.py:5:1: F401 [*] `typing.List` imported but unused\n" +
		"src/utils/helpers.py:10:1: D103 Missing docstring in public function\n" +
		"src/utils/helpers.py:22:9: SIM108 [*] Use ternary operator instead of `if`-`else`-block\n" +
		"Found 15 errors.\n" +
		"[*] 4 fixable with the `--fix` option.\n"

	got, err := filterRuff(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, got %d >= %d", len(got), len(raw))
	}
	// Codes grouped alphabetically
	for _, code := range []string{"B006", "C901", "D100", "D103", "E501", "E711", "E722", "F401", "F841", "I001", "SIM108"} {
		if !strings.Contains(got, code) {
			t.Errorf("expected code %s in output", code)
		}
	}
	// D100 appears twice (user.py:1 and auth.py:1)
	if !strings.Contains(got, "D100 (2)") {
		t.Errorf("expected D100 grouped with count 2, got:\n%s", got)
	}
	// E501 appears twice
	if !strings.Contains(got, "E501 (2)") {
		t.Errorf("expected E501 grouped with count 2, got:\n%s", got)
	}
	// F401 appears twice
	if !strings.Contains(got, "F401 (2)") {
		t.Errorf("expected F401 grouped with count 2, got:\n%s", got)
	}
	// I001 appears twice
	if !strings.Contains(got, "I001 (2)") {
		t.Errorf("expected I001 grouped with count 2, got:\n%s", got)
	}
	if !strings.Contains(got, "15 problems") {
		t.Errorf("expected '15 problems' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "fixable") {
		t.Errorf("expected fixable message preserved, got:\n%s", got)
	}
}

func TestFilterRuff_FormatOutput(t *testing.T) {
	// ruff format output has no lint problems — not detected as ruff output
	raw := "4 files reformatted, 12 files left unchanged"

	got, err := filterRuff(raw)
	if err != nil {
		t.Fatal(err)
	}
	// No ruff-like markers (no ": F", ": E", ": W", "Found", "fixable")
	// so looksLikeRuffOutput returns false and raw is returned as-is
	if got != raw {
		t.Errorf("expected raw passthrough for format output, got:\n%s", got)
	}
}
