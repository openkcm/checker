package main

import (
	"context"
	"errors"
	"testing"
)

func TestRunFuncWithSignalHandlingSuccess(t *testing.T) {
	// Avoid the real graceful-shutdown sleep.
	orig := *gracefulShutdownSec
	*gracefulShutdownSec = 0
	defer func() { *gracefulShutdownSec = orig }()

	called := false
	code := runFuncWithSignalHandling(func(ctx context.Context) error {
		called = true
		if ctx == nil {
			t.Error("expected a non-nil context")
		}
		return nil
	})

	if !called {
		t.Error("target function was not invoked")
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestRunFuncWithSignalHandlingError(t *testing.T) {
	orig := *gracefulShutdownSec
	*gracefulShutdownSec = 0
	defer func() { *gracefulShutdownSec = orig }()

	code := runFuncWithSignalHandling(func(context.Context) error {
		return errors.New("boom")
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
