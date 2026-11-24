package cli

import (
	"github.com/shadeform/shadeform-cli/internal/output"
	"github.com/shadeform/shadeform-cli/internal/useragent"
)

// The User-Agent needs the CLI version and whether agent mode is on. Both
// live in packages that import the SDK, so the SDK's hook cannot import them
// back; this hands them over instead. They are passed as functions because
// main sets Version, and flag parsing sets agent mode, after this runs.
func init() {
	useragent.Configure(func() string { return Version }, output.IsAgentMode)
}
