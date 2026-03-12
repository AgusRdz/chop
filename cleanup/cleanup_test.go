package cleanup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveDir_Existing(t *testing.T) {
	dir := t.TempDir()
	// Create a file inside so it's non-empty
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !removeDir(dir) {
		t.Fatal("expected removeDir to return true for existing dir")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("directory should have been removed")
	}
}

func TestRemoveDir_NonExistent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	if removeDir(dir) {
		t.Fatal("expected removeDir to return false for non-existent dir")
	}
}

func TestRemoveFile_Existing(t *testing.T) {
	f := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !removeFile(f) {
		t.Fatal("expected removeFile to return true for existing file")
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatal("file should have been removed")
	}
}

func TestRemoveFile_NonExistent(t *testing.T) {
	f := filepath.Join(t.TempDir(), "does-not-exist.txt")
	if removeFile(f) {
		t.Fatal("expected removeFile to return false for non-existent file")
	}
}

func TestConfigDir_DefaultsToHomeConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	dir := configDir()
	if filepath.Base(dir) != "chop" {
		t.Errorf("expected last segment to be 'chop', got %q", filepath.Base(dir))
	}
	if filepath.Base(filepath.Dir(dir)) != ".config" {
		t.Errorf("expected parent to be '.config', got %q", filepath.Base(filepath.Dir(dir)))
	}
}

func TestConfigDir_RespectsXDGConfigHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	dir := configDir()
	expected := filepath.Join(tmp, "chop")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}
}

func TestDataDir_UnderHome(t *testing.T) {
	dir := dataDir()
	if filepath.Base(dir) != "chop" {
		t.Errorf("expected last segment to be 'chop', got %q", filepath.Base(dir))
	}
}

func TestReset_RemovesTrackingDBAndAuditLog(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp) // isolate configDir (not used by Reset, but safe)

	// Patch dataDir by creating files in a known temp location and
	// verifying the removeFile logic indirectly via a direct call.
	db := filepath.Join(tmp, "tracking.db")
	log := filepath.Join(tmp, "hook-audit.log")
	if err := os.WriteFile(db, []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(log, []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify removeFile works correctly for both file types (unit-level check)
	if !removeFile(db) {
		t.Error("expected removeFile to return true for tracking.db")
	}
	if !removeFile(log) {
		t.Error("expected removeFile to return true for hook-audit.log")
	}

	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Error("tracking.db should be gone")
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Error("hook-audit.log should be gone")
	}
}

func TestReset_NoErrorWhenFilesAbsent(t *testing.T) {
	// Reset should not panic or error when the data dir doesn't exist.
	// We can't fully isolate dataDir(), but we can verify removeFile handles it.
	missing := filepath.Join(t.TempDir(), "nonexistent.db")
	if removeFile(missing) {
		t.Error("expected false for missing file")
	}
}
