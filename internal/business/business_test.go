package business

import (
	"context"
	"testing"
	"time"

	"github.com/openkcm/checker/internal/config"
)

// TestMainGracefulShutdown verifies Main wires up and starts the HTTP server,
// returning cleanly once the context is cancelled.
func TestMainGracefulShutdown(t *testing.T) {
	cfg := &config.Config{}
	cfg.Application.Name = "checker"
	cfg.Server = config.Server{Address: "127.0.0.1:0", ShutdownTimeout: time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()

	err := Main(ctx, cfg)
	if err != nil {
		t.Fatalf("Main returned error: %v", err)
	}
}

func TestMainListenError(t *testing.T) {
	cfg := &config.Config{}
	cfg.Application.Name = "checker"
	cfg.Server = config.Server{Address: "127.0.0.1:999999", ShutdownTimeout: time.Second}

	err := Main(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid listen address")
	}
}
