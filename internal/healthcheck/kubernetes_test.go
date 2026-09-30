package healthcheck

import (
	"context"
	"net/http"
	"testing"

	"github.com/openkcm/checker/internal/config"
)

// TestVerifyK8SResourceBadKubeconfig forces the KUBECONFIG branch to fail by
// pointing at a non-existent kubeconfig file.
func TestVerifyK8SResourceBadKubeconfig(t *testing.T) {
	t.Setenv("KUBECONFIG", "/nonexistent/kubeconfig.yaml")

	rc := &config.Resource{Name: "k8s", URL: "/healthz"}

	resp, status := verifyK8SResource(context.Background(), rc)
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", status)
	}

	if resp.Status != NOTOK || len(resp.Errors) == 0 {
		t.Errorf("expected NOTOK with errors, got %+v", resp)
	}

	if resp.Errors[0].Error != ERR_BAD_REQUEST {
		t.Errorf("expected bad request error, got %q", resp.Errors[0].Error)
	}
}
