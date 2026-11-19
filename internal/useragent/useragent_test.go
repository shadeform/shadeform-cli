package useragent

import (
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// generated is the User-Agent the Speakeasy SDK sets before the hook runs.
const generated = "speakeasy-sdk/go 0.3.1 2.948.4 1.0.0 github.com/shadeform/shadeform-cli/internal/sdk"

// configure sets the source for one test and restores "not configured"
// afterwards, so tests do not leak into each other.
func configure(t *testing.T, version string, agent bool) {
	t.Helper()
	Configure(func() string { return version }, func() bool { return agent })
	t.Cleanup(func() { current.Store(nil) })
}

func TestProductNamesTheReleasePlatformAgentModeAndOperation(t *testing.T) {
	configure(t, "0.4.0", false)

	want := "shade/0.4.0 (" + runtime.GOOS + "/" + runtime.GOARCH + "; agent=off; op=listInstances)"
	if got := Product("listInstances"); got != want {
		t.Fatalf("Product = %q, want %q", got, want)
	}

	configure(t, "0.4.0", true)
	if got := Product("listInstances"); !strings.Contains(got, "; agent=on; ") {
		t.Fatalf("agent mode is on but the header says %q; agent-driven use would be counted as a person's", got)
	}
}

// The API splits the header on whitespace, takes a first field starting with
// "shade/" as the CLI, and reads the version from the start of what follows.
// These are the same steps, so a format change that would stop releases
// being counted fails here and not silently in production.
func TestTheServerCanReadTheClientAndVersion(t *testing.T) {
	serverVersion := regexp.MustCompile(`^\d{1,4}\.\d{1,4}\.\d{1,4}`)

	for _, tc := range []struct {
		version     string
		wantVersion string
	}{
		{"0.4.0", "0.4.0"},
		{"v0.4.0", "0.4.0"},
		{"0.4.1-next+20260101", "0.4.1"},
		{"12.345.6789", "12.345.6789"},
		{"dev", ""},
		{"", ""},
	} {
		configure(t, tc.version, false)
		header := With("createInstance", generated)

		fields := strings.Fields(header)
		rest, isCLI := strings.CutPrefix(fields[0], "shade/")
		if !isCLI {
			t.Fatalf("version %q: first field is %q; the API would not count this request as the CLI", tc.version, fields[0])
		}
		if got := serverVersion.FindString(rest); got != tc.wantVersion {
			t.Errorf("version %q: the API would store version %q, want %q", tc.version, got, tc.wantVersion)
		}
	}
}

func TestWithKeepsTheGeneratedStringAfterTheProduct(t *testing.T) {
	configure(t, "0.4.0", false)

	got := With("listInstances", generated)
	if !strings.HasPrefix(got, "shade/0.4.0 (") {
		t.Errorf("header does not start with the product: %q", got)
	}
	if !strings.HasSuffix(got, ") "+generated) {
		t.Errorf("header lost the generated SDK string, which log searches for older releases match on: %q", got)
	}
}

func TestWithDoesNotPrefixTwice(t *testing.T) {
	configure(t, "0.4.0", false)

	once := With("listInstances", generated)
	if twice := With("listInstances", once); twice != once {
		t.Fatalf("a request sent through the hook twice grew a second product:\n once: %q\ntwice: %q", once, twice)
	}
}

func TestWithNothingToKeepSendsOnlyTheProduct(t *testing.T) {
	configure(t, "0.4.0", false)

	for _, existing := range []string{"", "   "} {
		if got := With("createApiKey", existing); got != Product("createApiKey") {
			t.Errorf("With(%q) = %q, want only the product", existing, got)
		}
	}
}

func TestBeforeConfigureTheVersionIsUnknownAndAgentModeOff(t *testing.T) {
	current.Store(nil)

	got := Product("listInstances")
	if !strings.HasPrefix(got, "shade/unknown (") || !strings.Contains(got, "agent=off") {
		t.Fatalf("unconfigured Product = %q", got)
	}
}

// A version comes from a linker flag, so a local or forked build can put
// anything in it. Go refuses to send a header containing a control
// character, which would break every command in that build.
func TestOddVersionsAndOperationsCannotBreakTheHeader(t *testing.T) {
	for _, tc := range []struct {
		name      string
		version   string
		operation string
	}{
		{"newline in version", "0.4.0\r\nX-Injected: 1", "listInstances"},
		{"separators in version", "0.4.0 (dirty); agent=on", "listInstances"},
		{"non-ASCII version", "0.4.0-süß", "listInstances"},
		{"control characters in operation", "0.4.0", "list\x00Instances\n"},
		{"very long version", strings.Repeat("9", 5000), "listInstances"},
		{"nothing usable", "()", ";;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configure(t, tc.version, false)
			header := With(tc.operation, generated)

			if len(header) > 300 {
				t.Errorf("header is %d bytes long", len(header))
			}
			for _, r := range header {
				if r < ' ' || r > '~' {
					t.Fatalf("header contains %q, which Go will not send: %q", r, header)
				}
			}
			// One "(" and one ";agent=" mean the build's version did not
			// add fields of its own.
			product := strings.TrimSuffix(header, " "+generated)
			if strings.Count(product, "(") != 1 || strings.Count(product, "agent=") != 1 || strings.Count(product, " ") != 3 {
				t.Errorf("the version or operation changed the shape of the header: %q", product)
			}

			// The real check: net/http accepts it.
			req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:1/", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("User-Agent", header)
			if _, err := http.DefaultTransport.RoundTrip(req); err != nil && strings.Contains(err.Error(), "invalid header") {
				t.Fatalf("net/http refused the header: %v", err)
			}
		})
	}
}
