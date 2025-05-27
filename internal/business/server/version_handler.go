package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/openkcm/checker/internal/version"
	"github.com/openkcm/common-sdk/pkg/commoncfg"
	"github.com/openkcm/common-sdk/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	slogctx "github.com/veqryn/slog-context"

	"github.com/openkcm/checker/internal/config"
)

func versionsHandlerFunc(cfg *config.Config) func(http.ResponseWriter, *http.Request) {
	traceAttrs := otlp.CreateAttributesFrom(cfg.Application,
		attribute.String(commoncfg.AttrOperation, "versions"),
	)

	tracer := otel.Tracer("VersionsHandler", trace.WithInstrumentationAttributes(traceAttrs...))

	return func(w http.ResponseWriter, req *http.Request) {
		// Request Id will be propagated through all method calls propagated of this HTTP handler
		ctx := slogctx.With(req.Context(),
			commoncfg.AttrRequestID, uuid.New().String(),
			commoncfg.AttrOperation, "versions",
		)

		// Manual OTEL Tracing
		parentCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))

		ctx, span := tracer.Start(
			parentCtx,
			"versions-span",
			trace.WithAttributes(traceAttrs...),
		)
		defer span.End()

		// Metrics
		requestStartTime := time.Now()
		defer func() {
			elapsedTime := float64(time.Since(requestStartTime)) / float64(time.Millisecond)

			// Metrics logic
			attrs := metric.WithAttributes(
				otlp.CreateAttributesFrom(cfg.Application,
					attribute.String("userAgent", req.UserAgent()),
					attribute.String(commoncfg.AttrOperation, "versions"),
				)...,
			)

			counter.Add(ctx, 1, attrs)
			hist.Record(ctx, elapsedTime, attrs)
		}()

		// Business Logic
		slogctx.Info(ctx, "Starting versions request")

		w.Header().Set("Content-Type", "application/json")

		status := http.StatusOK
		response := map[string]any{}

		defer func() {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(response)
		}()

		response, status = version.Do(ctx, &cfg.Versions)
		response[cfg.Application.Name] = json.RawMessage(cfg.Application.BuildInfo.String())

		slogctx.Info(ctx, "Finished versions request",
			"durationMs", time.Since(requestStartTime)/time.Millisecond)
		// End Business Logic
	}
}
