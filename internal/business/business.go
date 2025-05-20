package business

import (
	"context"

	slogctx "github.com/veqryn/slog-context"

	"github.com/openkcm/checker/internal/business/server"
	"github.com/openkcm/checker/internal/config"
)

func Main(ctx context.Context, cfg *config.Config) error {
	ctx = slogctx.WithGroup(ctx, "checker")
	return server.StartHTTPServer(ctx, cfg)
}
