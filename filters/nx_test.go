package filters

import (
	"strings"
	"testing"
)

func TestFilterNxBuild(t *testing.T) {
	raw := "> nx run my-app:build\n\n> my-app@1.0.0 build\n> ng build\n\n" +
		"✔ Browser application bundle generation complete.\n\n" +
		"Build at: 2024-01-15 - Time: 8000ms\n\n" +
		"----------------------------------------------\n\n" +
		">  NX   Successfully ran target build for project my-app (10s)\n"

	got, err := filterNxBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "nx build my-app ok") {
		t.Errorf("expected nx build result, got: %s", got)
	}
}

func TestFilterNxTest(t *testing.T) {
	raw := "> nx run my-app:test\n\n" +
		"PASS src/app/app.component.spec.ts\n" +
		"Test Suites: 1 passed, 1 total\n" +
		"Tests:       5 passed, 5 total\n" +
		"Time:        3.5 s\n\n" +
		">  NX   Successfully ran target test for project my-app (5s)\n"

	got, err := filterNxTest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= len(raw) {
		t.Errorf("expected compression")
	}
}

func TestFilterNxBuild_Empty(t *testing.T) {
	got, err := filterNxBuild("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
