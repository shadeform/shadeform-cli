package hooks

import (
	"net/http"

	"github.com/shadeform/shadeform-cli/internal/useragent"
)

// UserAgentHook puts the CLI's own product token in front of the User-Agent
// the generated SDK sets, so the API can tell which shade release made a
// request, on which platform, and whether an agent was driving it. See
// internal/useragent for the format and what it is used for.
//
// It runs as a hook, not as an SDK option, because the operation name is only
// known per request and the generated code sets the header itself on every
// call.
type UserAgentHook struct{}

var _ beforeRequestHook = (*UserAgentHook)(nil)

func (UserAgentHook) BeforeRequest(hookCtx BeforeRequestContext, req *http.Request) (*http.Request, error) {
	if req == nil {
		return req, nil
	}
	req.Header.Set("User-Agent", useragent.With(hookCtx.OperationID, req.Header.Get("User-Agent")))
	return req, nil
}
