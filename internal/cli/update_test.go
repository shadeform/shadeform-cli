package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shadeform/shadeform-cli/internal/selfupdate"
)

// fakeReleases is a minimal GitHub: latest/tags endpoints and asset downloads.
type fakeReleases struct {
	srv      *httptest.Server
	tag      string
	archive  []byte
	sums     []byte
	asset    string
	requests int32
}

func newFakeReleases(t *testing.T, tag string, binary []byte) *fakeReleases {
	t.Helper()
	asset, err := selfupdate.AssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "shade", Mode: 0o755, Size: int64(len(binary)), Typeflag: tar.TypeReg})
	_, _ = tw.Write(binary)
	_ = tw.Close()
	_ = gz.Close()
	sum := sha256.Sum256(buf.Bytes())
	f := &fakeReleases{tag: tag, archive: buf.Bytes(), asset: asset, sums: []byte(hex.EncodeToString(sum[:]) + "  " + asset + "\n")}
	mux := http.NewServeMux()
	release := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.requests, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": f.tag, "assets": []map[string]string{
			{"name": asset, "browser_download_url": f.srv.URL + "/dl/" + asset},
			{"name": selfupdate.ChecksumsFile, "browser_download_url": f.srv.URL + "/dl/" + selfupdate.ChecksumsFile},
		}})
	}
	mux.HandleFunc("/repos/"+selfupdate.Repo+"/releases/latest", release)
	mux.HandleFunc("/repos/"+selfupdate.Repo+"/releases/tags/"+tag, release)
	mux.HandleFunc("/dl/"+asset, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(f.archive) })
	mux.HandleFunc("/dl/"+selfupdate.ChecksumsFile, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(f.sums) })
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

// withUpdateFixture points the command at the fake GitHub and a scratch
// "binary", with the version, build and terminal state the test wants.
func withUpdateFixture(t *testing.T, f *fakeReleases, current, buildTime string, source selfupdate.Source, tty bool) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.Getenv("HOME"), ".config"))
	t.Setenv("CI", "")
	t.Setenv(selfupdate.NoCheckEnv, "")
	target := filepath.Join(t.TempDir(), "shade")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho shade "+current+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	prevVersion, prevBuild := Version, BuildTime
	prevClient, prevTarget, prevSource, prevTTY := updateClient, updateTarget, updateSource, interactiveTerminal
	Version, BuildTime = current, buildTime
	updateClient = func(timeout time.Duration) *selfupdate.Client {
		c := selfupdate.NewClient(current)
		c.APIBase = f.srv.URL
		c.HTTP.Timeout = timeout
		return c
	}
	updateTarget = func() (string, error) { return target, nil }
	updateSource = func(string, bool) selfupdate.Source { return source }
	interactiveTerminal = func() bool { return tty }
	t.Cleanup(func() {
		Version, BuildTime = prevVersion, prevBuild
		updateClient, updateTarget, updateSource, interactiveTerminal = prevClient, prevTarget, prevSource, prevTTY
	})
	return target
}

func runUpdateCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	full := append([]string{"--no-keyring", "--agent-mode=false", "--no-interactive", "--color", "never"}, args...)
	err = ExecuteRoot(context.Background(), root, full)
	return stdout.String(), stderr.String(), err
}

func TestUpdateCheckReportsNewerRelease(t *testing.T) {
	f := newFakeReleases(t, "v0.3.1", []byte("#!/bin/sh\necho shade 0.3.1\n"))
	withUpdateFixture(t, f, "0.3.0", "2026-11-02T00:00:00Z", selfupdate.SourceRelease, false)

	stdout, _, err := runUpdateCLI(t, "update", "--check")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "v0.3.0 -> v0.3.1") || !strings.Contains(stdout, "shade update") {
		t.Fatalf("stdout = %q", stdout)
	}

	stdout, _, err = runUpdateCLI(t, "update", "--check", "-o", "json")
	if err != nil {
		t.Fatal(err)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("json: %v in %q", err, stdout)
	}
	if res["current_version"] != "v0.3.0" || res["latest_version"] != "v0.3.1" || res["updated"] != false {
		t.Fatalf("json result = %v", res)
	}
}

func TestUpdateReplacesBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("script binaries are not executable on Windows")
	}
	f := newFakeReleases(t, "v0.3.1", []byte("#!/bin/sh\necho shade 0.3.1\n"))
	target := withUpdateFixture(t, f, "0.3.0", "2026-11-02T00:00:00Z", selfupdate.SourceRelease, false)

	stdout, stderr, err := runUpdateCLI(t, "update")
	if err != nil {
		t.Fatalf("update: %v (stderr %q)", err, stderr)
	}
	if !strings.Contains(stdout, "Updated shade v0.3.0 -> v0.3.1") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr, "Downloading shade v0.3.1") {
		t.Fatalf("stderr = %q", stderr)
	}
	b, _ := os.ReadFile(target)
	if !strings.Contains(string(b), "echo shade 0.3.1") {
		t.Fatalf("binary was not replaced: %q", b)
	}
	cache := selfupdate.LoadCache(filepath.Join(os.Getenv("HOME"), ".config", "shade", selfupdate.CacheFileName))
	if cache.Latest != "v0.3.1" {
		t.Fatalf("cache after update = %+v", cache)
	}

	// Already current: nothing downloaded, nothing changed.
	Version = "0.3.1"
	stdout, _, err = runUpdateCLI(t, "update")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "already the latest release") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestUpdateRefusesForeignInstalls(t *testing.T) {
	f := newFakeReleases(t, "v0.3.1", []byte("x"))
	target := withUpdateFixture(t, f, "0.3.0", "", selfupdate.SourceHomebrew, false)
	before, _ := os.ReadFile(target)

	_, _, err := runUpdateCLI(t, "update")
	if err == nil || !strings.Contains(err.Error(), "brew upgrade shade") {
		t.Fatalf("err = %v, want Homebrew advice", err)
	}
	after, _ := os.ReadFile(target)
	if !bytes.Equal(before, after) {
		t.Fatal("binary changed despite refusal")
	}

	// --check still answers for a Homebrew install.
	stdout, _, err := runUpdateCLI(t, "update", "--check")
	if err != nil || !strings.Contains(stdout, "v0.3.1") {
		t.Fatalf("--check: %v %q", err, stdout)
	}
}

func TestUpdateDryRunMakesNoRequest(t *testing.T) {
	f := newFakeReleases(t, "v0.3.1", []byte("x"))
	withUpdateFixture(t, f, "0.3.0", "2026-11-02T00:00:00Z", selfupdate.SourceRelease, false)

	_, stderr, err := runUpdateCLI(t, "update", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "[DRY-RUN]") {
		t.Fatalf("stderr = %q", stderr)
	}
	if n := atomic.LoadInt32(&f.requests); n != 0 {
		t.Fatalf("dry run made %d requests", n)
	}
}

func TestUpdateNoticeAfterCommand(t *testing.T) {
	f := newFakeReleases(t, "v0.3.1", []byte("x"))
	withUpdateFixture(t, f, "0.3.0", "2026-11-02T00:00:00Z", selfupdate.SourceRelease, true)

	// `shade version` is a local command that always succeeds.
	_, stderr, err := runUpdateCLI(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	want := "A new version of shade is available (v0.3.0 -> v0.3.1). Run: shade update"
	if !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want notice", stderr)
	}
	if n := atomic.LoadInt32(&f.requests); n != 1 {
		t.Fatalf("notice made %d requests, want 1", n)
	}

	// Second run inside 24h: answered from the cache, still shown.
	_, stderr, _ = runUpdateCLI(t, "version")
	if !strings.Contains(stderr, want) {
		t.Fatalf("second run stderr = %q", stderr)
	}
	if n := atomic.LoadInt32(&f.requests); n != 1 {
		t.Fatalf("cache missed: %d requests", n)
	}

	// Machine output and the opt-out keep stderr clean.
	_, stderr, _ = runUpdateCLI(t, "version", "-o", "json")
	if strings.Contains(stderr, "new version") {
		t.Fatalf("-o json printed the notice: %q", stderr)
	}
	t.Setenv(selfupdate.NoCheckEnv, "1")
	_, stderr, _ = runUpdateCLI(t, "version")
	if strings.Contains(stderr, "new version") {
		t.Fatalf("opt-out printed the notice: %q", stderr)
	}
	t.Setenv(selfupdate.NoCheckEnv, "")
	t.Setenv("CI", "true")
	_, stderr, _ = runUpdateCLI(t, "version")
	if strings.Contains(stderr, "new version") {
		t.Fatalf("CI printed the notice: %q", stderr)
	}
}

func TestUpdateNoticeSkipsDevBuildsAndPipes(t *testing.T) {
	f := newFakeReleases(t, "v0.3.1", []byte("x"))

	// Dev build (no BuildTime), terminal attached.
	withUpdateFixture(t, f, "0.3.0", "", selfupdate.SourceDev, true)
	_, stderr, _ := runUpdateCLI(t, "version")
	if strings.Contains(stderr, "new version") || atomic.LoadInt32(&f.requests) != 0 {
		t.Fatalf("dev build checked for updates: %q", stderr)
	}

	// Release build, output piped.
	withUpdateFixture(t, f, "0.3.0", "2026-11-02T00:00:00Z", selfupdate.SourceRelease, false)
	_, stderr, _ = runUpdateCLI(t, "version")
	if strings.Contains(stderr, "new version") || atomic.LoadInt32(&f.requests) != 0 {
		t.Fatalf("piped run checked for updates: %q", stderr)
	}

	// `shade update --check` itself never gets the trailing notice.
	withUpdateFixture(t, f, "0.3.0", "2026-11-02T00:00:00Z", selfupdate.SourceRelease, true)
	_, stderr, _ = runUpdateCLI(t, "update", "--check")
	if strings.Contains(stderr, "new version") {
		t.Fatalf("update --check printed the notice: %q", stderr)
	}
}
