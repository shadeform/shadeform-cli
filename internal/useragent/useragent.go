// Package useragent builds the User-Agent that shade sends to the Shadeform
// API, for example:
//
//	shade/0.4.0 (darwin/arm64; agent=off; op=listInstances) speakeasy-sdk/go 0.4.0 2.948.4 1.0.0 github.com/shadeform/shadeform-cli/internal/sdk
//
// The first part is what this package adds. The rest is the string the
// generated SDK already sent, kept so anything matching on it keeps working.
//
// Shadeform's API counts requests by client and version from this header, to
// see which releases are in use and whether the CLI is driven by people or by
// coding agents. It carries nothing about the machine beyond OS and CPU
// architecture, and nothing about the user. The README's "What shade sends"
// section is the public statement of that; change both together.
//
// The package lives outside internal/sdk and internal/cli so the SDK hook and
// the hand-written API callers can both import it without a cycle.
package useragent

import (
	"runtime"
	"strings"
	"sync/atomic"
)

// product is the User-Agent product name. Shadeform's API recognises the CLI
// by a first token starting with "shade/" and reads the version after it, so
// renaming this stops new releases being counted as the CLI.
const product = "shade"

// source is where the CLI version and agent mode come from. Both are
// functions because neither is known when packages initialise: the version is
// set by main from a linker flag, and agent mode is decided when flags are
// parsed.
type source struct {
	version   func() string
	agentMode func() bool
}

var current atomic.Pointer[source]

// Configure tells the package where to read the CLI version and agent mode.
// internal/cli calls it once from an init function. Until it is called the
// version is reported as "unknown" and agent mode as off, which is what a
// program embedding the SDK without the CLI would send.
func Configure(version func() string, agentMode func() bool) {
	current.Store(&source{version: version, agentMode: agentMode})
}

// Product returns the part this package adds, for the given operation:
//
//	shade/0.4.0 (darwin/arm64; agent=off; op=listInstances)
//
// operationID names the API operation in the API document's terms. It lets
// the server tell commands apart that share a route.
func Product(operationID string) string {
	version, agent := "", false
	if src := current.Load(); src != nil {
		if src.version != nil {
			version = src.version()
		}
		if src.agentMode != nil {
			agent = src.agentMode()
		}
	}

	agentValue := "off"
	if agent {
		agentValue = "on"
	}

	return product + "/" + token(strings.TrimPrefix(version, "v")) +
		" (" + runtime.GOOS + "/" + runtime.GOARCH +
		"; agent=" + agentValue +
		"; op=" + token(operationID) + ")"
}

// With puts Product in front of the User-Agent a request already carries.
// A value that already starts with the product is returned unchanged, so a
// request passed through twice (a retry, say) is not prefixed twice.
func With(operationID, existing string) string {
	existing = strings.TrimSpace(existing)
	if strings.HasPrefix(existing, product+"/") {
		return existing
	}
	if existing == "" {
		return Product(operationID)
	}
	return Product(operationID) + " " + existing
}

// maxTokenLength bounds a version or operation name. Real ones are far
// shorter; the cap only stops an odd build from sending a huge header.
const maxTokenLength = 64

// token reduces s to characters that are safe inside the header and cannot
// be mistaken for its separators. A version comes from a linker flag, so a
// local or forked build can set it to anything. Go refuses to send a header
// value containing a control character, which would fail every request, and
// a space or parenthesis would shift the fields the server reads.
func token(s string) string {
	var b strings.Builder
	for _, r := range s {
		if b.Len() >= maxTokenLength {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '-', r == '_', r == '+':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}
