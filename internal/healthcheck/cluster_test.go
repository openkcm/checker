package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openkcm/checker/internal/config"
)

func TestVerifyServiceResourceSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("healthy"))
	}))
	defer srv.Close()

	rc := &config.Resource{
		Name:   "svc",
		URL:    srv.URL,
		Checks: []config.Check{{Type: config.ContainsCheckType, Value: "healthy"}},
	}

	resp, status := verifyServiceResource(context.Background(), rc)
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if resp.Status != OK {
		t.Errorf("resp.Status = %q, want OK", resp.Status)
	}
}

func TestVerifyServiceResourceCheckFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("degraded"))
	}))
	defer srv.Close()

	rc := &config.Resource{
		Name:   "svc",
		URL:    srv.URL,
		Checks: []config.Check{{Type: config.ContainsCheckType, Value: "healthy"}},
	}

	resp, status := verifyServiceResource(context.Background(), rc)
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", status)
	}
	if resp.Status != NOTOK || len(resp.Errors) == 0 {
		t.Errorf("expected NOTOK with errors, got %+v", resp)
	}
}

func TestVerifyServiceResourceBadRequest(t *testing.T) {
	rc := &config.Resource{Name: "svc", URL: "://bad-url"}

	resp, status := verifyServiceResource(context.Background(), rc)
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", status)
	}
	if resp.Errors[0].Error != ERR_BAD_REQUEST {
		t.Errorf("expected bad request error, got %+v", resp.Errors)
	}
}

func TestVerifyServiceResourceDoError(t *testing.T) {
	// Server that is closed immediately so the connection is refused.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	rc := &config.Resource{Name: "svc", URL: url}

	resp, status := verifyServiceResource(context.Background(), rc)
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", status)
	}
	if resp.Status != NOTOK {
		t.Errorf("resp.Status = %q, want NOT OK", resp.Status)
	}
}
