package server

import (
	"context"
	"testing"

	"github.com/openkcm/checker/internal/config"
)

func TestInitMeters(t *testing.T) {
	// Uses the global (noop) meter provider; must initialise the package-level
	// counter and histogram without error.
	err := initMeters(context.Background(), &config.Config{})
	if err != nil {
		t.Fatalf("initMeters returned error: %v", err)
	}

	if counter == nil {
		t.Error("counter was not initialised")
	}

	if hist == nil {
		t.Error("hist was not initialised")
	}
}
