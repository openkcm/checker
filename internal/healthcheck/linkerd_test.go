package healthcheck

import (
	"context"
	"net/http"
	"testing"

	"github.com/openkcm/checker/internal/config"
)

// TestVerifyLinkerdInitFailure forces InitializeKubeAPIClient to fail by pointing
// KUBECONFIG at a non-existent file, exercising the error path.
func TestVerifyLinkerdInitFailure(t *testing.T) {
	t.Setenv("KUBECONFIG", "/nonexistent/kubeconfig.yaml")

	cfg := &config.Linkerd{
		Tag:                   "linkerd",
		ControlPlaneNamespace: "linkerd",
		DataPlaneNamespace:    "linkerd",
		CNINamespace:          "linkerd-cni",
		Checks:                []string{"linkerd-existence"},
	}

	resp, status := verifyLinkerd(context.Background(), cfg)
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", status)
	}
	if resp.Status != NOTOK || len(resp.Errors) == 0 {
		t.Errorf("expected NOTOK with errors, got %+v", resp)
	}
}
