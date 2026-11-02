package selfupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func allowAll() Gate {
	return Gate{
		CurrentVersion: "0.3.0",
		ReleaseBuild:   true,
		InteractiveTTY: true,
		OutputFormat:   "pretty",
		Getenv:         func(string) string { return "" },
	}
}

func TestShouldNotifyGates(t *testing.T) {
	if !ShouldNotify(allowAll()) {
		t.Fatal("baseline gate should allow the notice")
	}
	cases := map[string]func(*Gate){
		"dev build":        func(g *Gate) { g.ReleaseBuild = false },
		"command failed":   func(g *Gate) { g.CommandFailed = true },
		"not a tty":        func(g *Gate) { g.InteractiveTTY = false },
		"agent mode":       func(g *Gate) { g.AgentMode = true },
		"json output":      func(g *Gate) { g.OutputFormat = "json" },
		"toon output":      func(g *Gate) { g.OutputFormat = "toon" },
		"jq":               func(g *Gate) { g.JQ = true },
		"dry run":          func(g *Gate) { g.DryRun = true },
		"skipped command":  func(g *Gate) { g.SkipCommand = true },
		"CI":               func(g *Gate) { g.Getenv = envOf("CI", "true") },
		"opt-out env":      func(g *Gate) { g.Getenv = envOf(NoCheckEnv, "1") },
		"unparsable build": func(g *Gate) { g.CurrentVersion = "(devel)" },
	}
	for name, mutate := range cases {
		g := allowAll()
		mutate(&g)
		if ShouldNotify(g) {
			t.Errorf("%s: notice should be suppressed", name)
		}
	}
}

func envOf(key, value string) func(string) string {
	return func(k string) string {
		if k == key {
			return value
		}
		return ""
	}
}

// fakeGitHub serves /repos/<repo>/releases/latest and counts requests.
func fakeGitHub(t *testing.T, tag string) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(&hits, 1)
		if r.Header.Get("User-Agent") == "" {
			t.Error("request without User-Agent")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": tag, "assets": []any{}})
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestCheckerCachesForInterval(t *testing.T) {
	srv, hits := fakeGitHub(t, "v0.3.1")
	client := NewClient("0.3.0")
	client.APIBase = srv.URL
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	ch := &Checker{
		Client:    client,
		CachePath: filepath.Join(t.TempDir(), CacheFileName),
		Now:       func() time.Time { return now },
	}
	ctx := context.Background()

	if got := ch.Latest(ctx); got != "v0.3.1" {
		t.Fatalf("first Latest = %q", got)
	}
	if got := ch.Latest(ctx); got != "v0.3.1" {
		t.Fatalf("second Latest = %q", got)
	}
	if n := atomic.LoadInt32(hits); n != 1 {
		t.Fatalf("GitHub was asked %d times inside the interval, want 1", n)
	}

	now = now.Add(DefaultInterval + time.Minute)
	if got := ch.Latest(ctx); got != "v0.3.1" {
		t.Fatalf("Latest after interval = %q", got)
	}
	if n := atomic.LoadInt32(hits); n != 2 {
		t.Fatalf("GitHub was asked %d times after the interval, want 2", n)
	}

	cache := LoadCache(ch.CachePath)
	if cache.Latest != "v0.3.1" || !cache.CheckedAt.Equal(now) {
		t.Fatalf("cache = %+v, want latest v0.3.1 checked at %s", cache, now)
	}
}

func TestCheckerRecordsFailuresWithoutRetrying(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	var hits int32
	client := NewClient("0.3.0")
	client.APIBase = srv.URL
	client.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(&hits, 1)
		return http.DefaultTransport.RoundTrip(r)
	})
	ch := &Checker{Client: client, CachePath: filepath.Join(t.TempDir(), CacheFileName)}
	ctx := context.Background()
	if got := ch.Latest(ctx); got != "" {
		t.Fatalf("Latest on failure = %q, want empty", got)
	}
	if got := ch.Latest(ctx); got != "" {
		t.Fatalf("Latest on failure = %q, want empty", got)
	}
	if n := atomic.LoadInt32(&hits); n != 1 {
		t.Fatalf("a failed check was retried (%d requests); it should wait for the interval", n)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNotifyPrintsOneLineOnlyWhenNewer(t *testing.T) {
	srv, _ := fakeGitHub(t, "v0.3.1")
	client := NewClient("0.3.0")
	client.APIBase = srv.URL
	ch := &Checker{Client: client, CachePath: filepath.Join(t.TempDir(), CacheFileName)}

	var out bytes.Buffer
	ch.Notify(context.Background(), allowAll(), &out)
	want := "A new version of shade is available (v0.3.0 -> v0.3.1). Run: shade update\n"
	if out.String() != want {
		t.Fatalf("notice = %q, want %q", out.String(), want)
	}

	out.Reset()
	g := allowAll()
	g.CurrentVersion = "0.3.1"
	ch.Notify(context.Background(), g, &out)
	if out.Len() != 0 {
		t.Fatalf("up to date, but printed %q", out.String())
	}

	out.Reset()
	g = allowAll()
	g.Getenv = envOf(NoCheckEnv, "1")
	ch.Notify(context.Background(), g, &out)
	if out.Len() != 0 {
		t.Fatalf("opted out, but printed %q", out.String())
	}
}

func TestDetectSource(t *testing.T) {
	devel := func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true }
	installed := func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{Main: debug.Module{Version: "v0.3.1"}}, true }
	cases := []struct {
		name    string
		exe     string
		release bool
		info    func() (*debug.BuildInfo, bool)
		want    Source
	}{
		{"release archive", "/usr/local/bin/shade", true, devel, SourceRelease},
		{"homebrew intel", "/usr/local/Cellar/shade/0.3.1/bin/shade", true, devel, SourceHomebrew},
		{"homebrew arm", "/opt/homebrew/bin/shade", true, devel, SourceHomebrew},
		{"linuxbrew", "/home/linuxbrew/.linuxbrew/bin/shade", true, devel, SourceHomebrew},
		{"go install", "/home/u/go/bin/shade", false, installed, SourceGoInstall},
		{"go build", "/tmp/shade", false, devel, SourceDev},
	}
	for _, c := range cases {
		if got := detectSource(c.exe, c.release, c.info); got != c.want {
			t.Errorf("%s: detectSource = %s, want %s", c.name, got, c.want)
		}
	}
	for _, s := range []Source{SourceHomebrew, SourceGoInstall, SourceDev} {
		if s.Updatable() || s.Advice() == "" {
			t.Errorf("%s should refuse with advice", s)
		}
	}
	if !SourceRelease.Updatable() {
		t.Error("release binaries must be updatable")
	}
}
