package filters

import (
	"strings"
	"testing"
)

func TestFilterMypy(t *testing.T) {
	raw := `src/app.py:12: error: Incompatible types in assignment (expression has type "str", variable has type "int")  [assignment]
src/app.py:25: error: "Dict[str, Any]" has no attribute "missing"  [attr-defined]
src/utils.py:8: note: Revealed type is "builtins.str"
Found 2 errors in 2 files (checked 10 source files)`

	got, err := filterMypy(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "[assignment]") {
		t.Error("expected error code group")
	}
	if !strings.Contains(got, "Found 2 errors") {
		t.Error("expected summary")
	}
	if strings.Contains(got, "note:") {
		t.Error("notes should be skipped")
	}
}

func TestFilterMypy_Success(t *testing.T) {
	raw := "Success: no issues found in 10 source files"
	got, err := filterMypy(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Errorf("expected passthrough for success, got: %s", got)
	}
}

func TestFilterMypy_Empty(t *testing.T) {
	got, err := filterMypy("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFilterMypy_LargeMultiFile(t *testing.T) {
	raw := `src/models/user.py:15: error: Incompatible types in assignment (expression has type "str", variable has type "int")  [assignment]
src/models/user.py:23: error: Argument 1 to "process" has incompatible type "Optional[str]"; expected "str"  [arg-type]
src/models/user.py:45: note: Revealed type is "builtins.str"
src/services/auth.py:8: error: Module "jwt" has no attribute "encode_token"  [attr-defined]
src/services/auth.py:12: error: Missing return statement  [return]
src/services/auth.py:30: error: Incompatible return value type (got "None", expected "Dict[str, Any]")  [return-value]
src/api/views.py:22: error: "HttpRequest" has no attribute "user_id"  [attr-defined]
src/api/views.py:55: error: Unsupported operand types for + ("int" and "str")  [operator]
src/api/views.py:78: error: Item "None" of "Optional[User]" has no attribute "email"  [union-attr]
src/utils/helpers.py:10: error: Function is missing a return type annotation  [no-untyped-def]
src/utils/helpers.py:15: error: Need type annotation for "cache" (hint: "cache: Dict[str, Any] = ...")  [var-annotated]
src/utils/helpers.py:33: error: Incompatible types in assignment (expression has type "List[str]", variable has type "Tuple[str, ...]")  [assignment]
Found 12 errors in 4 files (checked 28 source files)`

	got, err := filterMypy(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, raw=%d got=%d", len(raw), len(got))
	}
	// Notes should be filtered out
	if strings.Contains(got, "Revealed type") {
		t.Error("notes should be skipped")
	}
	// Should group by error code
	for _, code := range []string{"[assignment]", "[arg-type]", "[attr-defined]", "[return]", "[return-value]", "[operator]", "[union-attr]", "[no-untyped-def]", "[var-annotated]"} {
		if !strings.Contains(got, code) {
			t.Errorf("expected error code %s in output, got: %s", code, got)
		}
	}
	// attr-defined has 2 occurrences, should show count
	if !strings.Contains(got, "[attr-defined] (2)") {
		t.Errorf("expected attr-defined grouped with count 2, got: %s", got)
	}
	// assignment has 2 occurrences
	if !strings.Contains(got, "[assignment] (2)") {
		t.Errorf("expected assignment grouped with count 2, got: %s", got)
	}
	// Summary line should be preserved
	if !strings.Contains(got, "Found 12 errors in 4 files") {
		t.Errorf("expected summary line, got: %s", got)
	}
}

func TestFilterMypy_SuccessLarge(t *testing.T) {
	raw := "Success: no issues found in 28 source files"
	got, err := filterMypy(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Errorf("expected passthrough for success, got: %s", got)
	}
}

func TestFilterMypy_WarningsOnly(t *testing.T) {
	raw := `src/app.py:10: warning: Unused "type: ignore" comment  [unused-ignore]
src/app.py:20: warning: Unused "type: ignore" comment  [unused-ignore]
Found 2 errors in 1 file (checked 5 source files)`

	got, err := filterMypy(raw)
	if err != nil {
		t.Fatal(err)
	}
	// Warnings are treated like errors (not notes)
	if !strings.Contains(got, "[unused-ignore]") {
		t.Errorf("expected warning code in output, got: %s", got)
	}
	if !strings.Contains(got, "Found 2 errors") {
		t.Errorf("expected summary, got: %s", got)
	}
}

func TestFilterMypy_SingleError(t *testing.T) {
	raw := `src/main.py:5: error: Function is missing a return type annotation  [no-untyped-def]
Found 1 error in 1 file (checked 1 source file)`

	got, err := filterMypy(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "[no-untyped-def] (1)") {
		t.Errorf("expected grouped error, got: %s", got)
	}
	if !strings.Contains(got, "src/main.py:5") {
		t.Errorf("expected location, got: %s", got)
	}
	if !strings.Contains(got, "Found 1 error") {
		t.Errorf("expected summary, got: %s", got)
	}
}
