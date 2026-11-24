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
	"sync"
	"testing"

	"github.com/shadeform/shadeform-cli/internal/output"
)

// generatedUserAgentSuffix is the end of the string the generated SDK sets.
// Log searches for releases before this header existed match on it.
const generatedUserAgentSuffix = " github.com/shadeform/shadeform-cli/internal/sdk"

// userAgentServer answers `shade ssh-keys list` and records the User-Agent of
// every request it receives.
type userAgentServer struct {
	srv *httptest.Server
	mu  sync.Mutex
	got []string
}

func newUserAgentServer(t *testing.T) *userAgentServer {
	t.Helper()
	s := &userAgentServer{}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.got = append(s.got, r.Header.Get("User-Agent"))
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ssh_keys": []map[string]any{}})
	}))
	t.Cleanup(s.srv.Close)
	return s
}

// only returns the single User-Agent received, failing the test otherwise.
func (s *userAgentServer) only(t *testing.T) string {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.got) != 1 {
		t.Fatalf("server received %d requests, want 1: %q", len(s.got), s.got)
	}
	return s.got[0]
}

// runShadeWithoutAgentFlag is runShade without --agent-mode=false, so the
// flag and the environment decide agent mode the way they do for a user.
func runShadeWithoutAgentFlag(t *testing.T, srv *httptest.Server, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.Getenv("HOME"), ".config"))
	t.Setenv("SHADEFORM_API_KEY", "")
	// Agent mode is process-wide state that earlier tests have set.
	output.ResetAgentMode()
	t.Cleanup(output.ResetAgentMode)

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	full := append([]string{"--server-url", srv.URL, "--api-key", "test-key", "--no-interactive", "--color", "never"}, args...)
	err = ExecuteRoot(context.Background(), root, full)
	return stdout.String(), stderr.String(), err
}

func wantProduct(agent, operation string) string {
	return "shade/" + Version + " (" + runtime.GOOS + "/" + runtime.GOARCH + "; agent=" + agent + "; op=" + operation + ")"
}

// This runs a real command through the real SDK. It fails if the hook is no
// longer registered or internal/cli stops handing over the version, either
// of which would leave the API unable to tell one release from another.
func TestCommandsSendTheCLIVersionAndOperation(t *testing.T) {
	server := newUserAgentServer(t)

	if _, stderr, err := runShade(t, server.srv, "ssh-keys", "list", "-o", "json"); err != nil {
		t.Fatalf("ssh-keys list: %v\n%s", err, stderr)
	}

	got := server.only(t)
	want := wantProduct("off", "listSshKeys")
	if !strings.HasPrefix(got, want+" speakeasy-sdk/go ") {
		t.Errorf("User-Agent = %q\nwant it to start with %q followed by the generated SDK string", got, want)
	}
	if !strings.HasSuffix(got, generatedUserAgentSuffix) {
		t.Errorf("User-Agent lost the generated SDK string: %q", got)
	}
	if strings.Contains(got, "test-key") {
		t.Errorf("User-Agent carries the API key: %q", got)
	}
}

func TestAgentModeIsReportedWhetherSetByFlagOrEnvironment(t *testing.T) {
	// Every variable the CLI treats as "an agent is driving", cleared so the
	// environment running this test does not decide the result.
	agentVariables := []string{
		"CLAUDECODE", "CLAUDE_CODE", "CURSOR_AGENT", "CODEX", "AIDER", "CLINE", "WINDSURF_AGENT",
		"GITHUB_COPILOT", "AMAZON_Q", "GEMINI_CODE_ASSIST", "SRC_CODY", "FORCE_AGENT_MODE",
	}

	for _, tc := range []struct {
		name string
		env  string
		args []string
		want string
	}{
		{name: "nothing set", want: "off"},
		{name: "flag on", args: []string{"--agent-mode"}, want: "on"},
		{name: "environment on", env: "CLAUDECODE", want: "on"},
		{name: "flag off beats the environment", env: "CLAUDECODE", args: []string{"--agent-mode=false"}, want: "off"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range agentVariables {
				t.Setenv(name, "")
			}
			if tc.env != "" {
				t.Setenv(tc.env, "1")
			}
			server := newUserAgentServer(t)

			args := append([]string{"ssh-keys", "list", "-o", "json"}, tc.args...)
			if _, stderr, err := runShadeWithoutAgentFlag(t, server.srv, args...); err != nil {
				t.Fatalf("ssh-keys list: %v\n%s", err, stderr)
			}

			if got := server.only(t); !strings.HasPrefix(got, wantProduct(tc.want, "listSshKeys")+" ") {
				t.Errorf("User-Agent = %q, want agent=%s; requests would be counted under the wrong kind of use", got, tc.want)
			}
		})
	}
}

// --dry-run is how a user checks what shade would send, so it has to show
// the header exactly as a real request carries it.
func TestDryRunShowsTheUserAgent(t *testing.T) {
	server := newUserAgentServer(t)
	want := wantProduct("off", "listSshKeys")

	_, stderr, err := runShade(t, server.srv, "ssh-keys", "list", "--dry-run")
	if err != nil {
		t.Fatalf("dry run: %v\n%s", err, stderr)
	}
	if !strings.Contains(stderr, "User-Agent: "+want+" speakeasy-sdk/go ") {
		t.Errorf("dry run does not show the User-Agent a real request sends:\n%s", stderr)
	}

	stdout, stderr, err := runShade(t, server.srv, "ssh-keys", "list", "--dry-run", "-o", "json")
	if err != nil {
		t.Fatalf("dry run -o json: %v\n%s", err, stderr)
	}
	var preview struct {
		Request struct {
			Headers map[string][]string `json:"headers"`
		} `json:"request"`
	}
	if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
		t.Fatalf("dry run -o json is not JSON: %v\n%s", err, stdout)
	}
	agents := preview.Request.Headers["User-Agent"]
	if len(agents) != 1 || !strings.HasPrefix(agents[0], want+" ") {
		t.Errorf("dry run -o json User-Agent = %q, want one value starting with %q", agents, want)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.got) != 0 {
		t.Errorf("dry run reached the server %d times", len(server.got))
	}
}

// Signing in and out call the API key endpoints directly, not through the
// SDK. Without the header they arrive as Go's default client and the API
// counts a browser sign-in from the CLI as something else.
func TestLoginAndLogoutSendTheCLIUserAgent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	auth := newFakeAuth0(t, 0, false)
	api := newFakeAPI(t)

	if _, stderr, err := runLogin(t, auth, api, "auth", "login", "--no-browser"); err != nil {
		t.Fatalf("login: %v\n%s", err, stderr)
	}
	if _, stderr, err := runLogin(t, auth, api, "auth", "logout"); err != nil {
		t.Fatalf("logout: %v\n%s", err, stderr)
	}

	api.mu.Lock()
	defer api.mu.Unlock()
	want := map[string]string{
		"GET /v1/apikeys":               wantProduct("off", "listApiKeys"),
		"POST /v1/apikeys":              wantProduct("off", "createApiKey"),
		"POST /v1/apikeys/new-a/delete": wantProduct("off", "deleteApiKey"),
	}
	seen := map[string]bool{}
	for _, entry := range api.userAgents {
		method, rest, _ := strings.Cut(entry, " ")
		path, agent, _ := strings.Cut(rest, " ")
		request := method + " " + path
		wantAgent, known := want[request]
		if !known {
			t.Errorf("unexpected request %s", request)
			continue
		}
		seen[request] = true
		if agent != wantAgent {
			t.Errorf("%s sent User-Agent %q, want %q", request, agent, wantAgent)
		}
	}
	for request := range want {
		if !seen[request] {
			t.Errorf("%s was never sent, so its User-Agent was not checked; requests seen: %q", request, api.userAgents)
		}
	}
}
