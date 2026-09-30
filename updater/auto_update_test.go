package updater

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestShouldCheck_NeverChecked(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if !shouldCheck() {
		t.Error("should return true when never checked")
	}
}

func TestShouldCheck_RecentlyChecked(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	touchLastCheck()

	if shouldCheck() {
		t.Error("should return false when recently checked")
	}
}

func TestShouldCheck_StaleCheck(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	path, _ := lastCheckPath()
	os.MkdirAll(filepath.Dir(path), 0o700)
	os.WriteFile(path, []byte("old"), 0o644)
	stale := time.Now().Add(-25 * time.Hour)
	os.Chtimes(path, stale, stale)

	if !shouldCheck() {
		t.Error("should return true when check is stale (>24h)")
	}
}

func TestApplyPendingUpdate_DevVersion(t *testing.T) {
	// Should be a no-op for dev builds — just verify no panic
	ApplyPendingUpdate("dev")
}

func TestApplyPendingUpdate_NoPending(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Should return silently when no pending update exists
	ApplyPendingUpdate("v1.0.0")
}

func TestApplyPendingUpdate_InvalidMarker(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	path, _ := pendingUpdatePath()
	os.MkdirAll(filepath.Dir(path), 0o700)
	os.WriteFile(path, []byte("v2.0.0"), 0o644) // missing binary path

	ApplyPendingUpdate("v1.0.0")

	// Marker should be cleaned up
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("should clean up invalid marker file")
	}
}

func TestApplyPendingUpdate_MissingBinary(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	path, _ := pendingUpdatePath()
	os.MkdirAll(filepath.Dir(path), 0o700)
	os.WriteFile(path, []byte("v2.0.0\n/nonexistent/chop.new"), 0o644)

	ApplyPendingUpdate("v1.0.0")

	// Marker should be cleaned up
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("should clean up marker when binary is missing")
	}
}

// failTransport fails the test if any network request is attempted.
type failTransport struct{ t *testing.T }

func (f failTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.t.Errorf("unexpected network request to %s", r.URL)
	return nil, errors.New("network disabled in test")
}

func stubLookupClient(t *testing.T) {
	orig := lookupClient
	lookupClient = &http.Client{Transport: failTransport{t}}
	t.Cleanup(func() { lookupClient = orig })
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubLookupClient(t)

	CheckForUpdate("dev")
	CheckForUpdate("v1.0.0-dirty")

	path, _ := lastCheckPath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("dev versions must not touch the last-check file")
	}
}

func TestCheckForUpdate_Throttled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubLookupClient(t)
	touchLastCheck()

	CheckForUpdate("v1.0.0") // would fail the test via failTransport if it hit the network
}

func TestCheckForUpdate_TouchesBeforeLookup(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	orig := lookupClient
	lookupClient = &http.Client{Transport: failTransportSilent{}}
	t.Cleanup(func() { lookupClient = orig })

	CheckForUpdate("v1.0.0")

	if shouldCheck() {
		t.Error("failed lookup should still count as a check so it is not retried every run")
	}
}

type failTransportSilent struct{}

func (failTransportSilent) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline")
}

func TestStagedBinaryPath(t *testing.T) {
	dir := t.TempDir()
	got := stagedBinaryPath(dir, "v1.2.3")
	want := filepath.Join(dir, "chop-v1.2.3")
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if got != want {
		t.Errorf("stagedBinaryPath = %q, want %q", got, want)
	}
	if filepath.Dir(stagedBinaryPath(dir, "../../evil")) != dir {
		t.Error("version must not escape the data dir")
	}
}

func TestRemoveOldBinary(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "chop")
	old := exe + ".old"
	os.WriteFile(exe, []byte("current"), 0o700)
	os.WriteFile(old, []byte("stale"), 0o700)

	removeOldBinary(exe)

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error(".old should be removed")
	}
	if _, err := os.Stat(exe); err != nil {
		t.Error("current binary must be untouched")
	}
	removeOldBinary(exe) // missing .old is not an error
}

func TestTouchLastCheck(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

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

	os.WriteFile(dest, []byte("old"), 0o700)
	os.WriteFile(src, []byte("new"), 0o700)

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
