package hooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/shadeform/shadeform-cli/internal/reveal"
)

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Type": []string{"application/json"}, "Content-Length": []string{"999"}},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: 999,
	}
}

func runHook(t *testing.T, ctx context.Context, operationID, body string) (map[string]interface{}, []byte, *http.Response) {
	t.Helper()
	res, err := (RedactPrivateKeyHook{}).AfterSuccess(AfterSuccessContext{HookContext: HookContext{Context: ctx, OperationID: operationID}}, jsonResponse(body))
	if err != nil {
		t.Fatalf("AfterSuccess returned error: %v", err)
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("body is not JSON after hook: %v\n%s", err, raw)
	}
	return parsed, raw, res
}

const listBody = `{"ssh_keys":[{"id":"a","name":"managed","public_key":"ssh-ed25519 AAA","private_key":"-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----","is_default":true},{"id":"b","name":"mine","public_key":"ssh-rsa BBB","private_key":"","is_default":false}]}`

const infoBody = `{"id":"a","name":"managed","public_key":"ssh-ed25519 AAA","private_key":"-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----","is_default":true}`

func TestRedactsListResponse(t *testing.T) {
	parsed, raw, res := runHook(t, context.Background(), "SshKeys", listBody)
	if strings.Contains(string(raw), "private_key") {
		t.Fatalf("private_key still present in list body: %s", raw)
	}
	keys := parsed["ssh_keys"].([]interface{})
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	first := keys[0].(map[string]interface{})
	if first["public_key"] != "ssh-ed25519 AAA" || first["is_default"] != true {
		t.Fatalf("other fields were altered: %v", first)
	}
	if res.ContentLength != int64(len(raw)) || res.Header.Get("Content-Length") != strconv.Itoa(len(raw)) {
		t.Fatalf("content length not updated: %d / %s vs %d", res.ContentLength, res.Header.Get("Content-Length"), len(raw))
	}
}

func TestRedactsInfoResponse(t *testing.T) {
	parsed, raw, _ := runHook(t, context.Background(), "SshKeysInfo", infoBody)
	if strings.Contains(string(raw), "private_key") {
		t.Fatalf("private_key still present in info body: %s", raw)
	}
	if parsed["id"] != "a" || parsed["name"] != "managed" {
		t.Fatalf("other fields were altered: %v", parsed)
	}
}

func TestRevealOptInKeepsPrivateKey(t *testing.T) {
	_, raw, _ := runHook(t, reveal.WithPrivateKey(context.Background()), "SshKeysInfo", infoBody)
	if !strings.Contains(string(raw), "BEGIN OPENSSH PRIVATE KEY") {
		t.Fatalf("opt-in context should keep private_key: %s", raw)
	}
}

func TestOtherOperationsUntouched(t *testing.T) {
	body := `{"private_key":"should-stay","id":"x"}`
	_, raw, _ := runHook(t, context.Background(), "InstancesInfo", body)
	if string(raw) != body {
		t.Fatalf("non-SSH operation body was modified: %s", raw)
	}
}

func TestNonJSONPassesThrough(t *testing.T) {
	res := &http.Response{
		Header: http.Header{"Content-Type": []string{"text/plain"}},
		Body:   io.NopCloser(strings.NewReader("private_key=stays")),
	}
	out, err := (RedactPrivateKeyHook{}).AfterSuccess(AfterSuccessContext{HookContext: HookContext{Context: context.Background(), OperationID: "SshKeys"}}, res)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(out.Body)
	if string(raw) != "private_key=stays" {
		t.Fatalf("text body modified: %s", raw)
	}
}

func TestBodyWithoutPrivateKeyIsByteIdentical(t *testing.T) {
	body := `{"ssh_keys":[{"id":"a","name":"n","public_key":"p","is_default":false}]}`
	_, raw, _ := runHook(t, context.Background(), "SshKeys", body)
	if string(raw) != body {
		t.Fatalf("body without private_key should be untouched, got %s", raw)
	}
}