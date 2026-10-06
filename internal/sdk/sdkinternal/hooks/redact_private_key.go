package hooks

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/shadeform/shadeform-cli/internal/reveal"
)

// privateKeyField is returned by the SSH key endpoints for Shadeform-managed
// keys but is not part of the public OpenAPI document. Printing it from a
// list command would push every private key in the account into shell
// history, CI logs, and agent context, so it is removed here, upstream of
// both typed deserialization and the CLI's raw JSON passthrough.
const privateKeyField = "private_key"

// privateKeyOperations are the operation IDs whose responses carry the field.
var privateKeyOperations = map[string]bool{
	"SshKeys":     true,
	"SshKeysInfo": true,
}

// RedactPrivateKeyHook strips private_key from SSH key responses unless the
// request context carries the reveal.WithPrivateKey opt-in.
type RedactPrivateKeyHook struct{}

var _ afterSuccessHook = (*RedactPrivateKeyHook)(nil)

func (RedactPrivateKeyHook) AfterSuccess(hookCtx AfterSuccessContext, res *http.Response) (*http.Response, error) {
	if !privateKeyOperations[hookCtx.OperationID] || reveal.PrivateKeyAllowed(hookCtx.Context) {
		return res, nil
	}
	if res == nil || res.Body == nil || !isJSONResponse(res) {
		return res, nil
	}

	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return res, err
	}

	redacted, changed := stripPrivateKey(body)
	if !changed {
		redacted = body
	}
	res.Body = io.NopCloser(bytes.NewReader(redacted))
	res.ContentLength = int64(len(redacted))
	if res.Header.Get("Content-Length") != "" {
		res.Header.Set("Content-Length", strconv.Itoa(len(redacted)))
	}
	return res, nil
}

func isJSONResponse(res *http.Response) bool {
	mediaType, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(strings.SplitN(res.Header.Get("Content-Type"), ";", 2)[0]))
	}
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

// stripPrivateKey removes private_key from a single key object or from each
// element of a top-level ssh_keys array. Non-JSON or unexpected shapes are
// returned unchanged so the caller can fall back to the original body.
func stripPrivateKey(body []byte) ([]byte, bool) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var parsed interface{}
	if err := dec.Decode(&parsed); err != nil {
		return nil, false
	}

	obj, ok := parsed.(map[string]interface{})
	if !ok {
		return nil, false
	}

	changed := false
	if _, present := obj[privateKeyField]; present {
		delete(obj, privateKeyField)
		changed = true
	}
	if keys, ok := obj["ssh_keys"].([]interface{}); ok {
		for _, item := range keys {
			if key, ok := item.(map[string]interface{}); ok {
				if _, present := key[privateKeyField]; present {
					delete(key, privateKeyField)
					changed = true
				}
			}
		}
	}
	if !changed {
		return nil, false
	}

	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(obj); err != nil {
		return nil, false
	}
	return bytes.TrimRight(out.Bytes(), "\n"), true
}
