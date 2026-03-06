package filters

import (
	"strings"
	"testing"
)

func TestFilterMake(t *testing.T) {
	raw := "make[1]: Entering directory '/home/user/project/src'\n" +
		"gcc -c -Wall -O2 -o main.o main.c\n" +
		"gcc -c -Wall -O2 -o utils.o utils.c\n" +
		"main.c:45:10: warning: unused variable 'temp' [-Wunused-variable]\n" +
		"gcc -o myapp main.o utils.o -lm\n" +
		"make[1]: Leaving directory '/home/user/project/src'\n"

	got, err := filterMake(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression")
	}
	if !strings.Contains(got, "build ok") {
		t.Error("expected build ok")
	}
	if !strings.Contains(got, "warning") {
		t.Error("expected warning preserved")
	}
}

func TestFilterMake_Empty(t *testing.T) {
	got, err := filterMake("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFilterMake_LargeBuild(t *testing.T) {
	raw := "make[1]: Entering directory '/home/user/project'\n" +
		"gcc -c -Wall -Wextra -O2 -I./include -o build/main.o src/main.c\n" +
		"gcc -c -Wall -Wextra -O2 -I./include -o build/utils.o src/utils.c\n" +
		"src/utils.c:45:12: warning: unused variable 'temp' [-Wunused-variable]\n" +
		"   45 |     char *temp = NULL;\n" +
		"      |            ^~~~\n" +
		"gcc -c -Wall -Wextra -O2 -I./include -o build/network.o src/network.c\n" +
		"src/network.c:78:5: warning: implicit declaration of function 'legacy_init' [-Wimplicit-function-declaration]\n" +
		"   78 |     legacy_init();\n" +
		"      |     ^~~~~~~~~~~\n" +
		"src/network.c:112:18: warning: comparison between signed and unsigned integer expressions [-Wsign-compare]\n" +
		"  112 |     if (retval < buffer_size) {\n" +
		"      |                ^\n" +
		"gcc -c -Wall -Wextra -O2 -I./include -o build/database.o src/database.c\n" +
		"gcc -c -Wall -Wextra -O2 -I./include -o build/auth.o src/auth.c\n" +
		"gcc -c -Wall -Wextra -O2 -I./include -o build/logger.o src/logger.c\n" +
		"gcc -o bin/server build/main.o build/utils.o build/network.o build/database.o build/auth.o build/logger.o -lpthread -lssl -lcrypto -lsqlite3\n" +
		"Build complete: bin/server\n" +
		"make[1]: Leaving directory '/home/user/project'\n"

	got, err := filterMake(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, got %d >= %d", len(got), len(raw))
	}
	// 6 gcc -c compile lines collapsed, 3 warnings preserved
	if !strings.Contains(got, "build ok") {
		t.Errorf("expected 'build ok' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "6 files compiled") {
		t.Errorf("expected '6 files compiled', got:\n%s", got)
	}
	if !strings.Contains(got, "3 warnings") {
		t.Errorf("expected '3 warnings', got:\n%s", got)
	}
	// Individual warnings preserved
	if !strings.Contains(got, "unused variable") {
		t.Errorf("expected 'unused variable' warning preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "implicit declaration") {
		t.Errorf("expected 'implicit declaration' warning preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "Wsign-compare") {
		t.Errorf("expected sign-compare warning preserved, got:\n%s", got)
	}
	// Compile commands, link command, enter/leave lines should be stripped
	if strings.Contains(got, "gcc -c") {
		t.Error("expected compile commands to be stripped")
	}
	if strings.Contains(got, "Entering directory") {
		t.Error("expected directory enter/leave to be stripped")
	}
}

func TestFilterMake_WithErrors(t *testing.T) {
	raw := "make[1]: Entering directory '/home/user/project'\n" +
		"gcc -c -Wall -Wextra -O2 -I./include -o build/main.o src/main.c\n" +
		"src/main.c:5:10: error: config.h: No such file or directory\n" +
		"compilation terminated.\n" +
		"make: *** [Makefile:12: build/main.o] Error 1\n"

	got, err := filterMake(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression, got %d >= %d", len(got), len(raw))
	}
	// The error line matches reMakeErr (:5:10: ... error:)
	if !strings.Contains(got, "error:") {
		t.Errorf("expected error line preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "build FAILED") {
		t.Errorf("expected 'build FAILED' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "1 errors") {
		t.Errorf("expected '1 errors' count, got:\n%s", got)
	}
}
