package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/samber/oops"

	slogctx "github.com/veqryn/slog-context"

	"github.com/openkcm/checker/internal/config"
	"github.com/openkcm/checker/internal/healthcheck"
)

// registerHandlers registers the default http handlers for the status server
func registerHandlers(ctx context.Context, mux *http.ServeMux, cfg *config.Config) {
	if cfg.Healthcheck.Enabled {
		cache := healthcheck.NewCachedResponses(ctx, &cfg.Healthcheck)
		mux.HandleFunc(cfg.Healthcheck.Endpoint,
			healthcheckHandlerFunc(cfg.Healthcheck.Name, cfg, cache),
		)
	}

	for idx, hc := range cfg.Healthchecks {
		if hc.Enabled {
			mux.HandleFunc(hc.Endpoint,
				healthcheckHandlerFunc(
					fmt.Sprintf("%s-%d", cfg.Healthcheck.Name, idx),
					cfg,
					healthcheck.NewCachedResponses(ctx, &hc),
				),
			)
		}
	}

	if cfg.Versions.Enabled {
		mux.HandleFunc(cfg.Versions.Endpoint, versionsHandlerFunc(cfg))
	}
}

// createStatusServer creates a status http server using the given config
func createHTTPServer(ctx context.Context, cfg *config.Config) *http.Server {
	mux := http.NewServeMux()

	registerHandlers(ctx, mux, cfg)

	slogctx.Info(ctx, "Creating HTTP server", "address", cfg.Server.Address)

	return &http.Server{
		Addr:    cfg.Server.Address,
		Handler: mux,
	}
}

// StartHTTPServer starts the gRPC server using the given config.
func StartHTTPServer(ctx context.Context, cfg *config.Config) error {
	err := initMeters(ctx, cfg)
	if err != nil {
		return err
	}

	server := createHTTPServer(ctx, cfg)

	slogctx.Info(ctx, "Starting HTTP listener", "address", server.Addr)

	var lc net.ListenConfig

	listener, err := lc.Listen(ctx, "tcp", server.Addr)
	if err != nil {
		return oops.In("HTTP Server").
			WithContext(ctx).
			Wrapf(err, "Failed creating HTTP listener")
	}

	go func() {
		slogctx.Info(ctx, "Starting HTTP server", "address", server.Addr)

		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slogctx.Error(ctx, "ErrorField serving HTTP endpoint", "error", err)
		}

		slogctx.Info(ctx, "Stopped HTTP server")
	}()

	<-ctx.Done()

	shutdownCtx, shutdownRelease := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
	defer shutdownRelease()

	err = server.Shutdown(shutdownCtx)
	if err != nil {
		return oops.In("HTTP Server").
			WithContext(ctx).
			Wrapf(err, "Failed shutting down HTTP server")
	}

	slogctx.Info(ctx, "Completed graceful shutdown of HTTP server")

	return nil
}
