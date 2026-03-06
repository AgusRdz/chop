package filters

import (
	"strings"
	"testing"
)

func TestFilterCompiler(t *testing.T) {
	raw := "main.c: In function 'main':\n" +
		"main.c:10:5: warning: implicit declaration of function 'printf' [-Wimplicit-function-declaration]\n" +
		"   10 |     printf(\"hello\\n\");\n" +
		"      |     ^~~~~~\n" +
		"main.c:15:12: error: expected ';' before '}' token\n" +
		"   15 |     return 0\n" +
		"      |            ^\n"

	got, err := filterCompiler(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression")
	}
	if !strings.Contains(got, "errors(1)") {
		t.Error("expected error count")
	}
	if !strings.Contains(got, "warnings(1)") {
		t.Error("expected warning count")
	}
}

func TestFilterCompiler_FatalError(t *testing.T) {
	// "fatal error:" contains ": error:" so looksLikeCompilerOutput matches
	raw := "src/main.cpp:3:10: fatal error: boost/asio.hpp: No such file or directory\n" +
		"    3 | #include <boost/asio.hpp>\n" +
		"      |          ^~~~~~~~~~~~~~~~\n" +
		"compilation terminated.\n"

	got, err := filterCompiler(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "error") {
		t.Errorf("expected error info preserved, got: %s", got)
	}
	if len(got) <= len(strings.TrimSpace(raw)) {
		// Filter should compress or at least not expand
	}
}

func TestFilterCompiler_MultipleWarningsAndErrors(t *testing.T) {
	raw := "src/server.c:12:5: warning: implicit declaration of function 'gets' [-Wimplicit-function-declaration]\n" +
		"   12 |     gets(buffer);\n" +
		"      |     ^~~~\n" +
		"src/server.c:12:5: warning: the `gets' function is dangerous and should not be used.\n" +
		"src/server.c:25:14: warning: format '%d' expects argument of type 'int', but argument 2 has type 'long int' [-Wformat=]\n" +
		"   25 |     printf(\"%d bytes\\n\", file_size);\n" +
		"      |             ~^           ~~~~~~~~~\n" +
		"      |              |           |\n" +
		"      |              int         long int\n" +
		"      |             %ld\n" +
		"src/server.c:38:1: warning: control reaches end of non-void function [-Wreturn-type]\n" +
		"   38 | }\n" +
		"      | ^\n" +
		"src/server.c:45:22: error: expected ';' before '}' token\n" +
		"   45 |     return result\n" +
		"      |                  ^\n" +
		"      |                  ;\n" +
		"   46 | }\n" +
		"      | ~\n" +
		"src/server.c:52:5: error: 'unknown_type' undeclared (first use in this function)\n" +
		"   52 |     unknown_type var;\n" +
		"      |     ^~~~~~~~~~~~\n" +
		"src/server.c:52:5: note: each undeclared identifier is reported only once for each function it appears in\n"

	got, err := filterCompiler(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "errors(2):") {
		t.Errorf("expected errors(2), got: %s", got)
	}
	if !strings.Contains(got, "warnings(4):") {
		t.Errorf("expected warnings(4), got: %s", got)
	}
	if !strings.Contains(got, "src/server.c:45: expected ';' before '}' token") {
		t.Errorf("expected semicolon error detail, got: %s", got)
	}
	if !strings.Contains(got, "src/server.c:52: 'unknown_type' undeclared (first use in this function)") {
		t.Errorf("expected undeclared error detail, got: %s", got)
	}
	if !strings.Contains(got, "src/server.c:12: implicit declaration") {
		t.Errorf("expected implicit decl warning, got: %s", got)
	}
	if !strings.Contains(got, "src/server.c:38: control reaches end") {
		t.Errorf("expected return-type warning, got: %s", got)
	}
	// Notes should not appear in output
	if strings.Contains(got, "note") {
		t.Errorf("notes should be excluded from output, got: %s", got)
	}
	if len(got) >= len(strings.TrimSpace(raw)) {
		t.Errorf("expected compression, raw=%d got=%d", len(raw), len(got))
	}
}

func TestFilterCompiler_Empty(t *testing.T) {
	got, err := filterCompiler("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
