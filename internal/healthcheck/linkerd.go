package healthcheck

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/linkerd/linkerd2/pkg/healthcheck"
	"github.com/openkcm/checker/internal/config"
	slogctx "github.com/veqryn/slog-context"
)

func verifyLinkerd(ctx context.Context, cfg *config.Linkerd) (ret Response, status int) {
	status = http.StatusOK
	errors := make([]ErrorResponse, 0)

	ret = Response{
		Status: "OK",
	}

	defer func() {
		if len(errors) > 0 {
			status = http.StatusServiceUnavailable
			ret.Errors = errors
			ret.Status = "NOT OK"
		}
	}()

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

	output := &bytes.Buffer{}
	outerr := &bytes.Buffer{}
	success, _ := healthcheck.RunChecks(output, outerr, hc, cfg.Output)

	errMsg := outerr.String()
	outMsg := output.String()
	if !success && len(errMsg) > 0 {
		errors = append(errors, ErrorResponse{
			Error:   errMsg,
			Message: outMsg,
		})
	}
	if !success {
		slogctx.Warn(ctx, "Linkerd check", "output", outMsg, "error", errMsg)
	}

	return
}
