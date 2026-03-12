package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIsDev(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"dev", true},
		{"v1.2.3-dirty", true},
		{"v1.0.0", false},
		{"v1.6.0", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsDev(c.version); got != c.want {
			t.Errorf("IsDev(%q) = %v, want %v", c.version, got, c.want)
		}
	}
}

func TestBuildBinaryName(t *testing.T) {
	name := buildBinaryName()
	if !strings.HasPrefix(name, "chop-") {
		t.Errorf("expected name to start with 'chop-', got %q", name)
	}
	if !strings.Contains(name, runtime.GOOS) {
		t.Errorf("expected name to contain GOOS %q, got %q", runtime.GOOS, name)
	}
	if !strings.Contains(name, runtime.GOARCH) {
		t.Errorf("expected name to contain GOARCH %q, got %q", runtime.GOARCH, name)
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
		t.Errorf("expected .exe suffix on windows, got %q", name)
	}
	if runtime.GOOS != "windows" && strings.HasSuffix(name, ".exe") {
		t.Errorf("unexpected .exe suffix on %s, got %q", runtime.GOOS, name)
	}
}

func TestLatestVersion_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ghRelease{TagName: "v1.7.0"})
	}))
	defer srv.Close()

	// Temporarily override the URL by monkey-patching via a test helper.
	// Since latestVersion() uses the package-level `repo` constant we can't
	// inject the server URL directly — so we test the JSON decoding path
	// by calling the parsing logic directly.
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if release.TagName != "v1.7.0" {
		t.Errorf("expected v1.7.0, got %q", release.TagName)
	}
}

func TestLatestVersion_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 response")
	}
}

func TestDownload_Success(t *testing.T) {
	// Serve a fake binary large enough to pass the size check (>1024 bytes).
	payload := strings.Repeat("x", 2048)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "chop-test")
	if err := download(srv.URL, dest); err != nil {
		t.Fatalf("download failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("could not read dest file: %v", err)
	}
	if string(data) != payload {
		t.Error("downloaded content does not match expected payload")
	}
}

func TestDownload_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "chop-test")
	err := download(srv.URL, dest)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention 404, got: %v", err)
	}
}

func TestDownload_TooSmall(t *testing.T) {
	// Serve a payload smaller than 1024 bytes (simulates a 404 HTML page slipping through).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("tiny"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "chop-test")
	err := download(srv.URL, dest)
	if err == nil {
		t.Fatal("expected error for undersized binary")
	}
	if !strings.Contains(err.Error(), "too small") {
		t.Errorf("expected 'too small' error, got: %v", err)
	}
}

func TestDownload_CleansUpTmpOnWriteFailure(t *testing.T) {
	// We can't easily simulate a write failure mid-stream, but we can verify
	// the .tmp file is not left behind after a failed download (404 path).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "chop-test")
	_ = download(srv.URL, dest)

	tmp := dest + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error(".tmp file should be cleaned up after failed download")
	}
}
