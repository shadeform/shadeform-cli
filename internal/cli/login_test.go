package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shadeform/shadeform-cli/internal/auth0device"
	"github.com/shadeform/shadeform-cli/internal/loginstate"
)

// fakeAuth0 answers the two device-flow endpoints. pendingPolls is how many
// token requests report authorization_pending before succeeding; one of them
// is a slow_down so the back-off path runs too.
type fakeAuth0 struct {
	srv          *httptest.Server
	mu           sync.Mutex
	pendingPolls int
	polls        int
	deny         bool
	codeRequests []map[string]string
}

func newFakeAuth0(t *testing.T, pendingPolls int, deny bool) *fakeAuth0 {
	t.Helper()
	f := &fakeAuth0{pendingPolls: pendingPolls, deny: deny}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/device/code", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		f.codeRequests = append(f.codeRequests, map[string]string{
			"client_id": r.Form.Get("client_id"), "audience": r.Form.Get("audience"), "scope": r.Form.Get("scope"),
		})
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"device_code": "dev-123", "user_code": "ABCD-EFGH",
			"verification_uri": f.srv.URL + "/activate", "verification_uri_complete": f.srv.URL + "/activate?user_code=ABCD-EFGH",
			"expires_in": 900, "interval": 1,
		})
	})
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" || r.Form.Get("device_code") != "dev-123" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		f.mu.Lock()
		f.polls++
		n := f.polls
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case f.deny:
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "access_denied", "error_description": "User cancelled"})
		case n == 1 && f.pendingPolls > 0:
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "slow_down"})
		case n <= f.pendingPolls:
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "jwt-abc", "token_type": "Bearer", "expires_in": 86400})
		}
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

// fakeAPI is the apikeys surface of shade-service: Bearer or X-API-KEY,
// list/create/delete, the duplicate-name and ten-key rules.
type fakeAPI struct {
	srv     *httptest.Server
	mu      sync.Mutex
	keys    []map[string]any
	secrets map[string]string // id -> plaintext key
	deleted []string
	created []map[string]any
}

func newFakeAPI(t *testing.T, existingNames ...string) *fakeAPI {
	t.Helper()
	f := &fakeAPI{secrets: map[string]string{}}
	for i, n := range existingNames {
		id := "existing-" + string(rune('a'+i))
		f.keys = append(f.keys, map[string]any{"id": id, "name": n, "permissions": []string{"platform_read"}, "created_at": "2026-01-01T00:00:00Z"})
		f.secrets[id] = "secret-" + id
	}
	authed := func(r *http.Request) bool {
		if r.Header.Get("Authorization") == "Bearer jwt-abc" {
			return true
		}
		if k := r.Header.Get("X-API-KEY"); k != "" {
			f.mu.Lock()
			defer f.mu.Unlock()
			for _, s := range f.secrets {
				if s == k {
					return true
				}
			}
		}
		return false
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/apikeys", func(w http.ResponseWriter, r *http.Request) {
		if !authed(r) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"title":"Unauthorized","status":401,"detail":"The bearer token is missing, malformed, expired, or was not issued for this API."}`))
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"api_keys": f.keys})
		case http.MethodPost:
			var req struct {
				Name        string   `json:"name"`
				Permissions []string `json:"permissions"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			for _, k := range f.keys {
				if k["name"] == req.Name {
					http.Error(w, "API key with name "+req.Name+" already exists for this account", http.StatusBadRequest)
					return
				}
			}
			if len(f.keys) >= 10 {
				http.Error(w, "Maximum number of API keys (10) reached for this account", http.StatusBadRequest)
				return
			}
			id := "new-" + string(rune('a'+len(f.created)))
			secret := "sk_" + id
			f.keys = append(f.keys, map[string]any{"id": id, "name": req.Name, "permissions": req.Permissions, "created_at": time.Now().UTC().Format(time.RFC3339)})
			f.secrets[id] = secret
			rec := map[string]any{"id": id, "name": req.Name, "permissions": req.Permissions}
			f.created = append(f.created, rec)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "api_key": secret, "name": req.Name, "permissions": req.Permissions})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/v1/apikeys/", func(w http.ResponseWriter, r *http.Request) {
		if !authed(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/apikeys/"), "/delete")
		f.mu.Lock()
		defer f.mu.Unlock()
		kept := f.keys[:0]
		found := false
		for _, k := range f.keys {
			if k["id"] == id {
				found = true
				continue
			}
			kept = append(kept, k)
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		f.keys = kept
		delete(f.secrets, id)
		f.deleted = append(f.deleted, id)
		w.WriteHeader(http.StatusOK)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

// runLogin runs the real root command with a temp HOME, the keychain off,
// and the fake tenant/API wired in through the documented env overrides.
// Each test sets HOME to a fresh temp dir first.
func runLogin(t *testing.T, auth *fakeAuth0, api *fakeAPI, args ...string) (string, string, error) {
	t.Helper()
	return runLoginAs(t, auth, api, "cli-client-id", args...)
}

func runLoginAs(t *testing.T, auth *fakeAuth0, api *fakeAPI, clientID string, args ...string) (string, string, error) {
	t.Helper()
	if !strings.HasPrefix(os.Getenv("HOME"), os.TempDir()) {
		t.Fatalf("test must set HOME to a temp dir before calling runLogin")
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.Getenv("HOME"), ".config"))
	t.Setenv("SHADEFORM_API_KEY", "")
	t.Setenv(auth0device.EnvDomain, auth.srv.URL)
	t.Setenv(auth0device.EnvClientID, clientID)
	t.Setenv(auth0device.EnvAudience, "https://api.example.test")
	realSleep := auth0device.Sleep
	auth0device.Sleep = func(ctx context.Context, d time.Duration) error { return nil }
	t.Cleanup(func() { auth0device.Sleep = realSleep })

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	full := append([]string{"--server-url", api.srv.URL, "--no-keyring", "--agent-mode=false", "--no-interactive", "--color", "never"}, args...)
	err = ExecuteRoot(context.Background(), root, full)
	return stdout.String(), stderr.String(), err
}

func readConfigFile(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".config", "shade", "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml: %v", err)
	}
	return string(b)
}

func TestLoginMintsAKeyAndStoresIt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 2, false)
	api := newFakeAPI(t, "my laptop")

	stdout, stderr, err := runLogin(t, auth, api, "auth", "login", "--no-browser", "-o", "json")
	if err != nil {
		t.Fatalf("login: %v\nstderr: %s", err, stderr)
	}

	// The device flow asked for the right things and was polled through
	// slow_down and pending to success.
	if len(auth.codeRequests) != 1 || auth.codeRequests[0]["client_id"] != "cli-client-id" || auth.codeRequests[0]["audience"] != "https://api.example.test" {
		t.Fatalf("device code request: %+v", auth.codeRequests)
	}
	if auth.polls != 3 {
		t.Errorf("polled %d times, want 3 (slow_down, pending, success)", auth.polls)
	}
	if !strings.Contains(stderr, "ABCD-EFGH") || !strings.Contains(stderr, "/activate?user_code=ABCD-EFGH") {
		t.Errorf("stderr should show the code and URL:\n%s", stderr)
	}

	// A key named after this machine was minted with the default permissions.
	if len(api.created) != 1 {
		t.Fatalf("created %d keys, want 1", len(api.created))
	}
	name := api.created[0]["name"].(string)
	if !strings.HasPrefix(name, "shade CLI (") {
		t.Errorf("key name %q, want the shade CLI (<host>) form", name)
	}
	if perms := api.created[0]["permissions"].([]string); strings.Join(perms, ",") != "platform_read,platform_write" {
		t.Errorf("permissions %v", perms)
	}

	// The secret is stored (config file, since --no-keyring) and nothing
	// about the JWT is.
	cfgText := readConfigFile(t)
	if !strings.Contains(cfgText, "sk_new-a") {
		t.Errorf("config.yaml does not hold the minted key:\n%s", cfgText)
	}
	if strings.Contains(cfgText, "jwt-abc") || strings.Contains(stdout, "jwt-abc") {
		t.Errorf("the access token leaked into config or output")
	}
	if strings.Contains(stdout, "sk_new-a") {
		t.Errorf("the key secret was printed:\n%s", stdout)
	}

	var res loginResult
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("stdout is not the JSON result: %v\n%s", err, stdout)
	}
	if res.Method != "browser" || res.APIKeyID != "new-a" || res.StoredIn != "config" {
		t.Errorf("result %+v", res)
	}

	// Login state records the key for logout.
	state, err := loginstate.Load(filepath.Join(os.Getenv("HOME"), ".config", "shade", "config.yaml"))
	if err != nil || state == nil || state.APIKeyID != "new-a" || state.ServerURL != api.srv.URL {
		t.Fatalf("login state: %+v, %v", state, err)
	}
}

func TestLoginAgainReplacesThePreviousKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t)

	if _, stderr, err := runLogin(t, auth, api, "auth", "login", "--no-browser"); err != nil {
		t.Fatalf("first login: %v\n%s", err, stderr)
	}
	stdout, stderr, err := runLogin(t, auth, api, "auth", "login", "--no-browser")
	if err != nil {
		t.Fatalf("second login: %v\n%s", err, stderr)
	}
	if len(api.deleted) != 1 || api.deleted[0] != "new-a" {
		t.Errorf("deleted %v, want the first key revoked", api.deleted)
	}
	if len(api.keys) != 1 || api.keys[0]["id"] != "new-b" {
		t.Errorf("server keys %v, want only the new one", api.keys)
	}
	if !strings.Contains(stdout, "Revoked the previous key new-a") {
		t.Errorf("stdout:\n%s", stdout)
	}
	if !strings.Contains(readConfigFile(t), "sk_new-b") {
		t.Errorf("config does not hold the new key")
	}
}

func TestLoginAvoidsADuplicateName(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t, "build box")

	_, stderr, err := runLogin(t, auth, api, "auth", "login", "--no-browser", "--name", "build box", "--permissions", "platform_read")
	if err != nil {
		t.Fatalf("login: %v\n%s", err, stderr)
	}
	name := api.created[0]["name"].(string)
	if name == "build box" || !strings.HasPrefix(name, "build box 20") {
		t.Errorf("name %q, want 'build box <timestamp>'", name)
	}
	if perms := api.created[0]["permissions"].([]string); strings.Join(perms, ",") != "platform_read" {
		t.Errorf("permissions %v", perms)
	}
}

func TestLoginExplainsTheKeyCap(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	names := make([]string, 10)
	for i := range names {
		names[i] = "key " + string(rune('0'+i))
	}
	api := newFakeAPI(t, names...)

	_, _, err := runLogin(t, auth, api, "auth", "login", "--no-browser")
	if err == nil || !strings.Contains(err.Error(), "already has 10 API keys") || !strings.Contains(err.Error(), "key 3") {
		t.Fatalf("want a cap error naming the existing keys, got %v", err)
	}
	if len(api.created) != 0 {
		t.Errorf("a key was created despite the cap")
	}
}

func TestLoginDeclinedInBrowser(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, true)
	api := newFakeAPI(t)

	_, _, err := runLogin(t, auth, api, "auth", "login", "--no-browser")
	if err == nil || !strings.Contains(err.Error(), "declined") {
		t.Fatalf("want a declined error, got %v", err)
	}
	if len(api.created) != 0 {
		t.Errorf("a key was created after a declined sign-in")
	}
	if _, statErr := os.Stat(filepath.Join(os.Getenv("HOME"), ".config", "shade", "config.yaml")); statErr == nil {
		t.Errorf("config.yaml was written after a failed login")
	}
}

func TestLoginRejectsUnknownPermission(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t)

	_, _, err := runLogin(t, auth, api, "auth", "login", "--no-browser", "--permissions", "owner")
	if err == nil || !strings.Contains(err.Error(), `unknown permission "owner"`) {
		t.Fatalf("want a validation error, got %v", err)
	}
	if len(auth.codeRequests) != 0 {
		t.Errorf("a device code was requested before validating flags")
	}
}

func TestLoginWithPastedKeyDoesNotOpenAnything(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t)

	stdout, _, err := runLogin(t, auth, api, "auth", "login", "--api-key", "pasted-key")
	if err != nil {
		t.Fatalf("login --api-key: %v", err)
	}
	if len(auth.codeRequests) != 0 || len(api.created) != 0 {
		t.Errorf("the paste path contacted Auth0 or minted a key")
	}
	if !strings.Contains(readConfigFile(t), "pasted-key") {
		t.Errorf("pasted key not stored")
	}
	if !strings.Contains(stdout, "Configuration saved to") {
		t.Errorf("stdout:\n%s", stdout)
	}
}

func TestLoginDryRunIsANoop(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t)

	_, stderr, err := runLogin(t, auth, api, "auth", "login", "--dry-run")
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if !strings.Contains(stderr, "[DRY-RUN]") || len(auth.codeRequests) != 0 {
		t.Errorf("dry run made requests or printed nothing: %s", stderr)
	}
}

func TestLoginWithoutAClientIDPointsAtApiKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t)
	if auth0device.DefaultClientID != "" {
		t.Skip("a client ID is compiled in; the not-configured path cannot be reached")
	}
	_, _, err := runLoginAs(t, auth, api, "", "auth", "login", "--no-browser")
	if err == nil || !strings.Contains(err.Error(), "--api-key") {
		t.Fatalf("want an error pointing at --api-key, got %v", err)
	}
}

func TestLogoutRevokesTheMintedKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t)

	if _, stderr, err := runLogin(t, auth, api, "auth", "login", "--no-browser"); err != nil {
		t.Fatalf("login: %v\n%s", err, stderr)
	}
	stdout, stderr, err := runLogin(t, auth, api, "auth", "logout")
	if err != nil {
		t.Fatalf("logout: %v\n%s", err, stderr)
	}
	if len(api.deleted) != 1 || api.deleted[0] != "new-a" {
		t.Errorf("deleted %v, want new-a revoked", api.deleted)
	}
	if !strings.Contains(stdout, "Revoked API key new-a") {
		t.Errorf("stdout:\n%s", stdout)
	}
	if strings.Contains(readConfigFile(t), "sk_new-a") {
		t.Errorf("key still in config after logout")
	}
	if state, _ := loginstate.Load(filepath.Join(os.Getenv("HOME"), ".config", "shade", "config.yaml")); state != nil {
		t.Errorf("login state survived logout: %+v", state)
	}
}

func TestLogoutKeepRemoteLeavesTheKeyOnTheServer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t)

	if _, stderr, err := runLogin(t, auth, api, "auth", "login", "--no-browser"); err != nil {
		t.Fatalf("login: %v\n%s", err, stderr)
	}
	stdout, _, err := runLogin(t, auth, api, "auth", "logout", "--keep-remote")
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	if len(api.deleted) != 0 {
		t.Errorf("--keep-remote revoked %v", api.deleted)
	}
	if !strings.Contains(stdout, "was not revoked (--keep-remote)") {
		t.Errorf("stdout:\n%s", stdout)
	}
	if strings.Contains(readConfigFile(t), "sk_new-a") {
		t.Errorf("key still in config after logout")
	}
}

func TestLogoutAfterPastedKeyOnlyClearsLocally(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t, "hand made")

	if _, _, err := runLogin(t, auth, api, "auth", "login", "--api-key", "secret-existing-a"); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runLogin(t, auth, api, "auth", "logout")
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	if len(api.deleted) != 0 {
		t.Errorf("logout revoked a key shade did not create: %v", api.deleted)
	}
	if strings.Contains(stdout, "Revoked") {
		t.Errorf("stdout claims a revocation:\n%s", stdout)
	}
}
