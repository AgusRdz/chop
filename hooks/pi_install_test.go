package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPiInstallWritesExtension(t *testing.T) {
	tmpDir := t.TempDir()
	extPath := filepath.Join(tmpDir, "agent", "extensions", "chop.ts")

	if err := piInstallTo(extPath, "/usr/local/bin/chop"); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if !piIsInstalledAt(extPath) {
		t.Fatal("expected extension to be detected as installed")
	}

	data, err := os.ReadFile(extPath)
	if err != nil {
		t.Fatalf("failed to read extension: %v", err)
	}
	if !strings.Contains(string(data), "/usr/local/bin/chop") {
		t.Error("extension file missing chop binary path")
	}
	if strings.Contains(string(data), "__CHOP_BIN__") {
		t.Error("extension file still contains unresolved placeholder")
	}
}

func TestPiUninstallRemovesGeneratedFile(t *testing.T) {
	tmpDir := t.TempDir()
	extPath := filepath.Join(tmpDir, "chop.ts")

	if err := piInstallTo(extPath, "/usr/local/bin/chop"); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if err := piUninstallFrom(extPath); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	if _, err := os.Stat(extPath); !os.IsNotExist(err) {
		t.Error("expected extension file to be removed")
	}
}

func TestPiUninstallRefusesForeignFile(t *testing.T) {
	tmpDir := t.TempDir()
	extPath := filepath.Join(tmpDir, "chop.ts")

	if err := os.WriteFile(extPath, []byte("// hand-written extension\n"), 0o644); err != nil {
		t.Fatalf("failed to write extension: %v", err)
	}

	if err := piUninstallFrom(extPath); err == nil {
		t.Error("expected uninstall to refuse a file it did not generate")
	}

	if _, err := os.Stat(extPath); err != nil {
		t.Error("expected foreign file to remain in place")
	}
}

func TestPiUninstallNoopWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	extPath := filepath.Join(tmpDir, "chop.ts")

	if err := piUninstallFrom(extPath); err != nil {
		t.Errorf("expected no error when file does not exist, got %v", err)
	}
}
