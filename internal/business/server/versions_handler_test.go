package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openkcm/checker/internal/config"
)

func TestVersionsHandlerFunc(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"9.9.9"}`))
	}))
	defer backend.Close()

	cfg := &config.Config{}
	cfg.Application.Name = "checker"
	cfg.Versions = config.Versions{
		Enabled:   true,
		Timeout:   2 * time.Second,
		Resources: []*config.ServiceResource{{Name: "svc", URL: backend.URL}},
	}

	handler := versionsHandlerFunc(cfg)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/versions", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var body map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("response not valid JSON: %v", err)
	}

	if _, ok := body["svc"]; !ok {
		t.Error("expected svc key in versions response")
	}

	if _, ok := body["checker"]; !ok {
		t.Error("expected application build info under its name")
	}
}

func TestVersionsHandlerFuncNoResources(t *testing.T) {
	cfg := &config.Config{}
	cfg.Application.Name = "checker"
	cfg.Versions = config.Versions{Enabled: true}

	handler := versionsHandlerFunc(cfg)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/versions", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
