package auth0device

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func noSleep(t *testing.T) {
	t.Helper()
	real := Sleep
	Sleep = func(ctx context.Context, d time.Duration) error { return nil }
	t.Cleanup(func() { Sleep = real })
}

func tokenServer(t *testing.T, status int, body map[string]any) Config {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	return Config{BaseURL: srv.URL, ClientID: "c", Audience: "a", Scope: "openid", HTTP: srv.Client()}
}

func TestPollExpiredToken(t *testing.T) {
	noSleep(t)
	cfg := tokenServer(t, http.StatusForbidden, map[string]any{"error": "expired_token"})
	_, err := cfg.Poll(context.Background(), &DeviceCode{DeviceCode: "d", ExpiresIn: 900, Interval: 1})
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("want ErrExpired, got %v", err)
	}
}

func TestPollGivesUpAtTheDeadline(t *testing.T) {
	noSleep(t)
	cfg := tokenServer(t, http.StatusForbidden, map[string]any{"error": "authorization_pending"})
	// A code whose lifetime is already over must end the loop, pending or not.
	_, err := cfg.Poll(context.Background(), &DeviceCode{DeviceCode: "d", ExpiresIn: -1, Interval: 1})
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("want ErrExpired at the deadline, got %v", err)
	}
}

func TestPollSurfacesUnknownOAuthErrors(t *testing.T) {
	noSleep(t)
	cfg := tokenServer(t, http.StatusBadRequest, map[string]any{"error": "invalid_grant", "error_description": "Unknown device code"})
	_, err := cfg.Poll(context.Background(), &DeviceCode{DeviceCode: "d", ExpiresIn: 900, Interval: 1})
	if err == nil || err.Error() != "waiting for sign-in: invalid_grant: Unknown device code" {
		t.Fatalf("got %v", err)
	}
}

func TestPollHonoursContextCancellation(t *testing.T) {
	cfg := tokenServer(t, http.StatusForbidden, map[string]any{"error": "authorization_pending"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := cfg.Poll(ctx, &DeviceCode{DeviceCode: "d", ExpiresIn: 900, Interval: 1})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestRequestDeviceCodeNeedsAClientID(t *testing.T) {
	cfg := Config{BaseURL: "http://127.0.0.1:1"}
	if _, err := cfg.RequestDeviceCode(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}

func TestDefaultConfigTargetsProduction(t *testing.T) {
	t.Setenv(EnvDomain, "")
	t.Setenv(EnvClientID, "")
	t.Setenv(EnvAudience, "")
	cfg := DefaultConfig()
	if cfg.BaseURL != "https://auth.shadeform.ai" || cfg.ClientID != DefaultClientID || cfg.Audience != "https://api.shadeform.ai" {
		t.Fatalf("%+v", cfg)
	}
	if DefaultClientID == "" {
		t.Fatal("DefaultClientID must be set for a release build")
	}
}

func TestDefaultConfigEnvOverrides(t *testing.T) {
	t.Setenv(EnvDomain, "tenant.example.test")
	t.Setenv(EnvClientID, "id")
	t.Setenv(EnvAudience, "aud")
	cfg := DefaultConfig()
	if cfg.BaseURL != "https://tenant.example.test" || cfg.ClientID != "id" || cfg.Audience != "aud" {
		t.Fatalf("%+v", cfg)
	}
	t.Setenv(EnvDomain, "http://localhost:9999/")
	if cfg := DefaultConfig(); cfg.BaseURL != "http://localhost:9999" {
		t.Fatalf("a full URL should be kept as given: %q", cfg.BaseURL)
	}
	t.Setenv(EnvDomain, "")
	if cfg := DefaultConfig(); cfg.BaseURL != "https://"+DefaultDomain {
		t.Fatalf("default: %q", cfg.BaseURL)
	}
}
