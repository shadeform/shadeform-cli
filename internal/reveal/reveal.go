// Package reveal carries the per-request opt-in that lets a command receive
// secret material the SDK hooks otherwise strip from API responses.
//
// It lives outside internal/sdk so both the hooks (under internal/sdk) and
// the CLI commands (under internal/cli) can import it.
package reveal

import "context"

type privateKeyKey struct{}

// WithPrivateKey marks ctx so SSH key responses keep their private_key field.
func WithPrivateKey(ctx context.Context) context.Context {
	return context.WithValue(ctx, privateKeyKey{}, true)
}

// PrivateKeyAllowed reports whether ctx carries the WithPrivateKey opt-in.
func PrivateKeyAllowed(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	allowed, _ := ctx.Value(privateKeyKey{}).(bool)
	return allowed
}
