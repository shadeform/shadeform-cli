package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const testPrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nabc123\n-----END OPENSSH PRIVATE KEY-----"

func newSSHKeyServer(t *testing.T, privateKey string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/sshkeys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ssh_keys": []map[string]any{
				{"id": "key-1", "name": "My Laptop Key", "public_key": "ssh-ed25519 AAA", "private_key": privateKey, "is_default": true},
			},
		})
	})
	mux.HandleFunc("/sshkeys/key-1/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "key-1", "name": "My Laptop Key", "public_key": "ssh-ed25519 AAA", "private_key": privateKey, "is_default": true,
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func runShade(t *testing.T, srv *httptest.Server, args ...string) (string, string, error) {
	t.Helper()
	// Keep config/keyring lookups away from the developer's real home directory.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.Getenv("HOME"), ".config"))
	t.Setenv("SHADEFORM_API_KEY", "")

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	full := append([]string{"--server-url", srv.URL, "--api-key", "test-key", "--agent-mode=false", "--no-interactive", "--color", "never"}, args...)
	err = ExecuteRoot(context.Background(), root, full)
	return stdout.String(), stderr.String(), err
}

func TestSSHKeysListNeverPrintsPrivateKey(t *testing.T) {
	srv := newSSHKeyServer(t, testPrivateKey)
	for _, format := range []string{"json", "pretty", "yaml", "toon"} {
		stdout, stderr, err := runShade(t, srv, "ssh-keys", "list", "-o", format)
		if err != nil {
			t.Fatalf("[%s] list failed: %v\nstderr: %s", format, err, stderr)
		}
		if strings.Contains(stdout+stderr, "abc123") || strings.Contains(stdout+stderr, "private_key") {
			t.Fatalf("[%s] private key leaked in output:\n%s\n%s", format, stdout, stderr)
		}
		if !strings.Contains(stdout, "ssh-ed25519 AAA") {
			t.Fatalf("[%s] expected public key in output:\n%s", format, stdout)
		}
	}

	stdout, _, err := runShade(t, srv, "ssh-keys", "list", "--jq", ".ssh_keys[0] | keys")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout, "private_key") {
		t.Fatalf("--jq passthrough leaked private_key: %s", stdout)
	}
}

func TestSSHKeysGetNeverPrintsPrivateKey(t *testing.T) {
	srv := newSSHKeyServer(t, testPrivateKey)
	stdout, stderr, err := runShade(t, srv, "ssh-keys", "get", "key-1", "-o", "json")
	if err != nil {
		t.Fatalf("get failed: %v\n%s", err, stderr)
	}
	if strings.Contains(stdout, "abc123") {
		t.Fatalf("get leaked private key: %s", stdout)
	}
}

func TestSSHKeysDownloadWritesFile(t *testing.T) {
	srv := newSSHKeyServer(t, testPrivateKey)
	out := filepath.Join(t.TempDir(), "nested", "key.pem")

	stdout, stderr, err := runShade(t, srv, "ssh-keys", "download", "key-1", "--out", out)
	if err != nil {
		t.Fatalf("download failed: %v\n%s", err, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != testPrivateKey+"\n" {
		t.Fatalf("unexpected file contents: %q", data)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(out)
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("expected mode 0600, got %o", info.Mode().Perm())
		}
	}
	if !strings.Contains(stdout, out) || !strings.Contains(stdout, "ssh -i") {
		t.Fatalf("expected path and ssh hint in output: %s", stdout)
	}
	if strings.Contains(stdout, "abc123") {
		t.Fatalf("download printed the key material: %s", stdout)
	}

	// Second run must refuse to overwrite.
	_, stderr, err = runShade(t, srv, "ssh-keys", "download", "key-1", "--out", out)
	if err == nil {
		t.Fatal("expected error when file exists without --force")
	}
	if !strings.Contains(err.Error()+stderr, "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v / %s", err, stderr)
	}

	// --force overwrites and re-applies 0600.
	_ = os.Chmod(out, 0o644)
	_, stderr, err = runShade(t, srv, "ssh-keys", "download", "key-1", "--out", out, "--force")
	if err != nil {
		t.Fatalf("download --force failed: %v\n%s", err, stderr)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(out)
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("expected --force to restore mode 0600, got %o", info.Mode().Perm())
		}
	}
}

func TestSSHKeysDownloadDefaultPath(t *testing.T) {
	srv := newSSHKeyServer(t, testPrivateKey)
	stdout, stderr, err := runShade(t, srv, "ssh-keys", "download", "--id", "key-1", "-o", "json")
	if err != nil {
		t.Fatalf("download failed: %v\n%s", err, stderr)
	}
	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("expected JSON result, got %q: %v", stdout, err)
	}
	wantSuffix := filepath.Join(".ssh", "shadeform-my-laptop-key.pem")
	if !strings.HasSuffix(result.Path, wantSuffix) || !strings.HasPrefix(result.Path, os.Getenv("HOME")) {
		t.Fatalf("unexpected default path %q", result.Path)
	}
	if result.ID != "key-1" || result.Name != "My Laptop Key" {
		t.Fatalf("unexpected result %+v", result)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout, "abc123") {
		t.Fatalf("JSON result leaked key material: %s", stdout)
	}
}

func TestSSHKeysDownloadStdout(t *testing.T) {
	srv := newSSHKeyServer(t, testPrivateKey)
	stdout, stderr, err := runShade(t, srv, "ssh-keys", "download", "key-1", "--stdout")
	if err != nil {
		t.Fatalf("download --stdout failed: %v\n%s", err, stderr)
	}
	if stdout != testPrivateKey+"\n" {
		t.Fatalf("expected raw key on stdout, got %q", stdout)
	}
}

func TestSSHKeysDownloadUserUploadedKey(t *testing.T) {
	srv := newSSHKeyServer(t, "")
	out := filepath.Join(t.TempDir(), "key.pem")
	_, stderr, err := runShade(t, srv, "ssh-keys", "download", "key-1", "--out", out)
	if err == nil {
		t.Fatal("expected error for key without stored private half")
	}
	if !strings.Contains(err.Error()+stderr, "no private key stored") {
		t.Fatalf("unexpected error: %v / %s", err, stderr)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Fatal("no file should be written when there is no private key")
	}
}

func TestSSHKeysDownloadAppearsInUsageSchemas(t *testing.T) {
	srv := newSSHKeyServer(t, testPrivateKey)
	for _, args := range [][]string{
		{"--usage"},
		{"ssh-keys", "--usage"},
		{"sk", "--usage"},
		{"ssh-keys", "download", "--usage"},
	} {
		stdout, stderr, err := runShade(t, srv, args...)
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, stderr)
		}
		if strings.Count(stdout, `cmd "download"`) != 1 {
			t.Fatalf("%v: expected exactly one download entry in schema, got %d:\n%s", args, strings.Count(stdout, `cmd "download"`), stdout)
		}
		if !strings.Contains(stdout, `flag "--stdout"`) {
			t.Fatalf("%v: expected download flags in schema:\n%s", args, stdout)
		}
	}
	// The root schema must nest download under ssh-keys, not another group.
	stdout, _, _ := runShade(t, srv, "--usage")
	sshStart := strings.Index(stdout, `cmd "ssh-keys"`)
	sshEnd := sshStart + strings.Index(stdout[sshStart:], "\n}\n")
	if !strings.Contains(stdout[sshStart:sshEnd], `  cmd "download"`) {
		t.Fatalf("download not nested under ssh-keys in root schema:\n%s", stdout[sshStart:sshEnd])
	}
}

func TestSSHKeysDownloadDryRunMakesNoRequest(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	t.Cleanup(srv.Close)
	out := filepath.Join(t.TempDir(), "key.pem")
	_, stderr, err := runShade(t, srv, "ssh-keys", "download", "key-1", "--out", out, "--dry-run")
	if err != nil {
		t.Fatalf("dry-run failed: %v\n%s", err, stderr)
	}
	if hit {
		t.Fatal("dry-run must not contact the server")
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Fatal("dry-run must not write a file")
	}
	if !strings.Contains(stderr, "/sshkeys/key-1/info") {
		t.Fatalf("expected request preview on stderr, got: %s", stderr)
	}
}
