package healthcheck

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/linkerd/linkerd2/pkg/healthcheck"

	slogctx "github.com/veqryn/slog-context"

	"github.com/openkcm/checker/internal/config"
)

func verifyLinkerd(ctx context.Context, cfg *config.Linkerd) (*Response, int) {
	errors := make([]ErrorResponse, 0)

	response := &Response{
		Status: OK,
	}

	checks := make([]healthcheck.CategoryID, 0)

	for _, c := range cfg.Checks {
		checks = append(checks, healthcheck.CategoryID(c))
	}

	crdManifest := bytes.Buffer{}
	hc := healthcheck.NewHealthChecker(checks, &healthcheck.Options{
		IsMainCheckCommand:    true,
		ControlPlaneNamespace: cfg.ControlPlaneNamespace,
		CNINamespace:          cfg.CNINamespace,
		DataPlaneNamespace:    cfg.DataPlaneNamespace,
		KubeConfig:            "",
		KubeContext:           "",
		Impersonate:           "",
		ImpersonateGroup:      []string{},
		APIAddr:               "",
		VersionOverride:       "",
		RetryDeadline:         time.Now().Add(time.Duration(cfg.RetryDeadline) * time.Second),
		CNIEnabled:            cfg.Enabled,
		InstallManifest:       "",
		CRDManifest:           crdManifest.String(),
	})

	// Run the healthchecks using the new API
	success, warning := hc.RunChecks(func(result *healthcheck.CheckResult) {
		if result.Err != nil && result.Err.Error() != "" {
			errors = append(errors, ErrorResponse{
				Error:   result.Err.Error(),
				Message: result.Description,
			})
		}
	})

	if warning {
		slogctx.Warn(ctx, "Linkerd check has some warnings", "errors", errors)
	}

	status := http.StatusOK

	if !success {
		slogctx.Warn(ctx, "Linkerd check failed", "errors", errors)

		if len(errors) > 0 {
			status = http.StatusServiceUnavailable
			response.Errors = errors
			response.Status = NOTOK
		}
	}

	return response, status
}
