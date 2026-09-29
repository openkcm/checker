package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openkcm/checker/internal/config"
	"github.com/openkcm/checker/internal/healthcheck"
)

// TestMain initialises the package-level meters once so handler tests that
// record metrics do not hit a nil counter/histogram.
func TestMain(m *testing.M) {
	if err := initMeters(context.Background(), &config.Config{}); err != nil {
		panic(err)
	}

	m.Run()
}

// newPopulatedCache spins up a backend and returns a cache that has completed
// at least one refresh cycle against it.
func newPopulatedCache(t *testing.T, backendBody string) *healthcheck.CachedResponses {
	t.Helper()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(backendBody))
	}))
	t.Cleanup(backend.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cfg := &config.Healthcheck{
		Name:            "healthcheck",
		RefreshDuration: time.Hour, // avoid repeated refreshes; the first one runs immediately
		Cluster: config.Domain{
			Enabled:   true,
			Tag:       "cluster",
			Resources: []config.Resource{{Name: "svc", URL: backend.URL}},
		},
	}

	cache := healthcheck.NewCachedResponses(ctx, cfg)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := cache.Response()["cluster"]; ok {
			return cache
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("cache was not populated in time")

	return nil
}

func TestHealthcheckHandlerFunc(t *testing.T) {
	cache := newPopulatedCache(t, "healthy")

	cfg := &config.Config{}
	handler := healthcheckHandlerFunc("healthcheck", cfg, cache)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response not valid JSON: %v", err)
	}
	if _, ok := body["cluster"]; !ok {
		t.Error("expected cluster key in response body")
	}
}

func TestHealthcheckHandlerFuncMasksURLs(t *testing.T) {
	cache := newPopulatedCache(t, "healthy")

	cfg := &config.Config{}
	cfg.Healthcheck.MaskURLs = true

	handler := healthcheckHandlerFunc("healthcheck", cfg, cache)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	var body map[string][]*healthcheck.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response not valid JSON: %v", err)
	}
	for _, r := range body["cluster"] {
		if r.URL != "****" {
			t.Errorf("expected masked URL, got %q", r.URL)
		}
	}
}

func TestMaskResponse(t *testing.T) {
	in := &healthcheck.Response{
		URL: "http://secret",
		Errors: []healthcheck.ErrorResponse{
			{Error: "boom", Message: "sensitive detail"},
			{Error: "nomsg"},
		},
	}

	out := maskResponse(in)

	if out.URL != "****" {
		t.Errorf("URL = %q, want ****", out.URL)
	}
	if out.Errors[0].Message != "****" {
		t.Errorf("error message not masked: %q", out.Errors[0].Message)
	}
	if out.Errors[1].Message != "" {
		t.Errorf("empty message should stay empty, got %q", out.Errors[1].Message)
	}
	// Original must be untouched.
	if in.URL != "http://secret" {
		t.Error("maskResponse mutated the input")
	}
}

func TestMaskURLs(t *testing.T) {
	response := map[string]any{
		"list":   []*healthcheck.Response{{URL: "http://a"}},
		"single": &healthcheck.Response{URL: "http://b"},
		"other":  "left-alone",
	}

	masked := maskURLs(response)

	if masked["list"].([]*healthcheck.Response)[0].URL != "****" {
		t.Error("list response URL not masked")
	}
	if masked["single"].(*healthcheck.Response).URL != "****" {
		t.Error("single response URL not masked")
	}
	if masked["other"] != "left-alone" {
		t.Error("non-response value should be passed through unchanged")
	}
}
