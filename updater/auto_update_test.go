package updater

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldCheck_NeverChecked(t *testing.T) {
	// Override dataDir for test isolation
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	if !shouldCheck() {
		t.Error("should return true when never checked")
	}
}

func TestShouldCheck_RecentlyChecked(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create a recent check file
	touchLastCheck()

	if shouldCheck() {
		t.Error("should return false when recently checked")
	}
}

func TestShouldCheck_StaleCheck(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create a stale check file
	path, _ := lastCheckPath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("old"), 0o644)
	// Backdate the file
	stale := time.Now().Add(-25 * time.Hour)
	os.Chtimes(path, stale, stale)

	if !shouldCheck() {
		t.Error("should return true when check is stale (>24h)")
	}
}

func TestApplyPendingUpdate_DevVersion(t *testing.T) {
	result := ApplyPendingUpdate("dev")
	if result {
		t.Error("should not apply updates for dev builds")
	}
}

func TestApplyPendingUpdate_NoPending(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	result := ApplyPendingUpdate("v1.0.0")
	if result {
		t.Error("should return false when no pending update exists")
	}
}

func TestApplyPendingUpdate_InvalidMarker(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Write invalid marker (missing binary path)
	path, _ := pendingUpdatePath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("v2.0.0"), 0o644)

	result := ApplyPendingUpdate("v1.0.0")
	if result {
		t.Error("should return false for invalid marker format")
	}

	// Marker should be cleaned up
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("should clean up invalid marker file")
	}
}

func TestApplyPendingUpdate_MissingBinary(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Write marker pointing to non-existent binary
	path, _ := pendingUpdatePath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("v2.0.0\n/nonexistent/chop.new"), 0o644)

	result := ApplyPendingUpdate("v1.0.0")
	if result {
		t.Error("should return false when temp binary doesn't exist")
	}

	// Marker should be cleaned up
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("should clean up marker when binary is missing")
	}
}

func TestBackgroundCheck_DevVersion(t *testing.T) {
	// Should be a no-op for dev builds - just verify no panic
	BackgroundCheck("dev")
	BackgroundCheck("v1.0.0-dirty")
}

func TestTouchLastCheck(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	touchLastCheck()

	path, err := lastCheckPath()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("touch should create the check file")
	}
}

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()

	dest := filepath.Join(dir, "chop")
	src := filepath.Join(dir, "chop.new")

	os.WriteFile(dest, []byte("old"), 0o755)
	os.WriteFile(src, []byte("new"), 0o755)

	if err := replaceBinary(dest, src); err != nil {
		t.Fatalf("replaceBinary failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Errorf("expected 'new', got %q", string(data))
	}

	// Source should no longer exist (renamed)
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source file should be removed after rename")
	}
}
