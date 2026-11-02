package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeRelease is a GitHub-shaped release server: the API endpoints plus the
// asset downloads, all on one httptest server.
type fakeRelease struct {
	srv       *httptest.Server
	tag       string
	assets    map[string][]byte
	checksums []byte
}

func newFakeRelease(t *testing.T, tag string, assets map[string][]byte, checksums []byte) *fakeRelease {
	t.Helper()
	f := &fakeRelease{tag: tag, assets: assets, checksums: checksums}
	mux := http.NewServeMux()
	release := func(w http.ResponseWriter, r *http.Request) {
		names := make([]map[string]string, 0, len(f.assets)+1)
		for name := range f.assets {
			names = append(names, map[string]string{"name": name, "browser_download_url": f.srv.URL + "/download/" + name})
		}
		if f.checksums != nil {
			names = append(names, map[string]string{"name": ChecksumsFile, "browser_download_url": f.srv.URL + "/download/" + ChecksumsFile})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": f.tag, "assets": names})
	}
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", release)
	mux.HandleFunc("/repos/"+Repo+"/releases/tags/"+tag, release)
	mux.HandleFunc("/repos/"+Repo+"/releases/tags/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if name == ChecksumsFile && f.checksums != nil {
			_, _ = w.Write(f.checksums)
			return
		}
		data, ok := f.assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeRelease) updater(t *testing.T, goos, goarch string) *Updater {
	t.Helper()
	c := NewClient("0.3.0")
	c.APIBase = f.srv.URL
	return &Updater{Client: c, GOOS: goos, GOARCH: goarch}
}

func tarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, f := range []struct {
		name string
		data []byte
	}{{"README.md", []byte("readme")}, {name, content}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(f.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipped(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sumLine(name string, data []byte) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:]) + "  " + name + "\n"
}

// A tiny "binary": a shell script that prints its version, so the swapped
// file can prove it is the new one by being executed.
func scriptBinary(version string) []byte {
	return []byte("#!/bin/sh\necho shade " + version + "\n")
}

func TestResolveAndFetchTarGz(t *testing.T) {
	bin := scriptBinary("0.3.1")
	archive := tarGz(t, "shade", bin)
	f := newFakeRelease(t, "v0.3.1", map[string][]byte{"shade_Linux_x86_64.tar.gz": archive}, []byte(sumLine("shade_Linux_x86_64.tar.gz", archive)))
	up := f.updater(t, "linux", "amd64")

	plan, err := up.Resolve(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Release.Tag != "v0.3.1" || plan.AssetName != "shade_Linux_x86_64.tar.gz" {
		t.Fatalf("plan = %+v", plan)
	}
	got, err := up.Fetch(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bin) {
		t.Fatalf("extracted %q, want the binary", got)
	}

	// Pinned tag goes through the tags endpoint; unknown tag is ErrNoRelease.
	if _, err := up.Resolve(context.Background(), "0.3.1"); err != nil {
		t.Fatalf("pinned resolve: %v", err)
	}
	if _, err := up.Resolve(context.Background(), "v9.9.9"); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("unknown tag err = %v, want ErrNoRelease", err)
	}
}

func TestFetchZipForWindows(t *testing.T) {
	bin := []byte("MZ fake exe")
	archive := zipped(t, "shade.exe", bin)
	f := newFakeRelease(t, "v0.3.1", map[string][]byte{"shade_Windows_x86_64.zip": archive}, []byte(sumLine("shade_Windows_x86_64.zip", archive)))
	up := f.updater(t, "windows", "amd64")
	plan, err := up.Resolve(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := up.Fetch(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bin) {
		t.Fatalf("extracted %q", got)
	}
}

func TestFetchRefusesBadChecksum(t *testing.T) {
	archive := tarGz(t, "shade", scriptBinary("0.3.1"))
	f := newFakeRelease(t, "v0.3.1", map[string][]byte{"shade_Darwin_arm64.tar.gz": archive},
		[]byte(sumLine("shade_Darwin_arm64.tar.gz", []byte("something else"))))
	up := f.updater(t, "darwin", "arm64")
	plan, err := up.Resolve(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := up.Fetch(context.Background(), plan); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("err = %v, want ErrChecksumMismatch", err)
	}
}

func TestResolveRequiresChecksumsAndAsset(t *testing.T) {
	archive := tarGz(t, "shade", scriptBinary("0.3.1"))
	noSums := newFakeRelease(t, "v0.3.1", map[string][]byte{"shade_Linux_arm64.tar.gz": archive}, nil)
	if _, err := noSums.updater(t, "linux", "arm64").Resolve(context.Background(), ""); err == nil {
		t.Fatal("a release without checksums.txt must be refused")
	}
	wrongPlatform := newFakeRelease(t, "v0.3.1", map[string][]byte{"shade_Linux_arm64.tar.gz": archive}, []byte(sumLine("shade_Linux_arm64.tar.gz", archive)))
	if _, err := wrongPlatform.updater(t, "darwin", "arm64").Resolve(context.Background(), ""); err == nil {
		t.Fatal("a release without this platform's archive must be refused")
	}
	unlisted := newFakeRelease(t, "v0.3.1", map[string][]byte{"shade_Linux_arm64.tar.gz": archive}, []byte(sumLine("other.tar.gz", archive)))
	plan, err := unlisted.updater(t, "linux", "arm64").Resolve(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unlisted.updater(t, "linux", "arm64").Fetch(context.Background(), plan); err == nil {
		t.Fatal("an archive missing from checksums.txt must be refused")
	}
}

func TestReplaceSwapsAtomically(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("script binaries are not executable on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "shade")
	if err := os.WriteFile(target, scriptBinary("0.3.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Replace(target, scriptBinary("0.3.1")); err != nil {
		t.Fatal(err)
	}
	out, err := runScript(target)
	if err != nil {
		t.Fatal(err)
	}
	if out != "shade 0.3.1\n" {
		t.Fatalf("new binary printed %q", out)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("replaced binary is not executable: %v", info.Mode())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestReplaceReportsPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a non-root POSIX user")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "shade")
	if err := os.WriteFile(target, scriptBinary("0.3.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := Replace(target, scriptBinary("0.3.1"))
	var perm *PermissionError
	if !errors.As(err, &perm) {
		t.Fatalf("err = %v, want PermissionError", err)
	}
	out, _ := runScript(target)
	if out != "shade 0.3.0\n" {
		t.Fatalf("original binary was touched: %q", out)
	}
}

func runScript(path string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.Command(path)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running %s: %w", path, err)
	}
	return buf.String(), nil
}
