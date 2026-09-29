package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openkcm/checker/internal/config"
)

func testConfigWithHandlers() *config.Config {
	cfg := &config.Config{}
	cfg.Application.Name = "checker"
	cfg.Server = config.Server{Address: "127.0.0.1:0", ShutdownTimeout: time.Second}
	cfg.Healthcheck = config.Healthcheck{
		Enabled:         true,
		Name:            "healthcheck",
		Endpoint:        "/healthz",
		RefreshDuration: time.Hour,
	}
	cfg.Healthchecks = []config.Healthcheck{
		{Enabled: true, Endpoint: "/healthz2", RefreshDuration: time.Hour},
		{Enabled: false, Endpoint: "/disabled", RefreshDuration: time.Hour},
	}
	cfg.Versions = config.Versions{Enabled: true, Endpoint: "/versions"}

	return cfg
}

func TestCreateHTTPServer(t *testing.T) {
	srv := createHTTPServer(t.Context(), testConfigWithHandlers())

	if srv.Addr != "127.0.0.1:0" {
		t.Errorf("addr = %q, want 127.0.0.1:0", srv.Addr)
	}

	if srv.Handler == nil {
		t.Fatal("expected a handler to be configured")
	}
}

func TestRegisterHandlers(t *testing.T) {
	mux := http.NewServeMux()

	// registerHandlers spins up background refresh goroutines; t.Context() is
	// cancelled at cleanup so they stop.
	registerHandlers(t.Context(), mux, testConfigWithHandlers())

	// Enabled endpoints must resolve to a registered pattern. Checking the
	// matched pattern (rather than executing the handler) keeps this test free
	// of the async cache-refresh race.
	for _, path := range []string{"/healthz", "/healthz2", "/versions"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)

		if _, pattern := mux.Handler(req); pattern == "" {
			t.Errorf("endpoint %q not registered", path)
		}
	}

	// A disabled healthcheck endpoint must not be registered.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/disabled", nil)

	if _, pattern := mux.Handler(req); pattern != "" {
		t.Errorf("disabled endpoint should not be registered, got pattern %q", pattern)
	}
}

func TestStartHTTPServerGracefulShutdown(t *testing.T) {
	cfg := testConfigWithHandlers()

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()

	// StartHTTPServer blocks until the context is cancelled, then shuts down.
	err := StartHTTPServer(ctx, cfg)
	if err != nil {
		t.Fatalf("StartHTTPServer returned error: %v", err)
	}
}

func TestStartHTTPServerListenError(t *testing.T) {
	cfg := &config.Config{}
	cfg.Application.Name = "checker"
	// Invalid port forces the listener creation to fail.
	cfg.Server = config.Server{Address: "127.0.0.1:999999", ShutdownTimeout: time.Second}

	err := StartHTTPServer(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid listen address")
	}
}
