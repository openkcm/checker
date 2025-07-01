package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/openkcm/checker/internal/healthcheck"
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

func healthcheckHandlerFunc(cfg *config.Config, ch *healthcheck.CachedResponses) func(http.ResponseWriter, *http.Request) {
	traceAttrs := otlp.CreateAttributesFrom(cfg.Application,
		attribute.String(commoncfg.AttrOperation, "healthcheck"),
	)

	tracer := otel.Tracer("HealthCheckerHandler", trace.WithInstrumentationAttributes(traceAttrs...))

	return func(w http.ResponseWriter, req *http.Request) {
		// Request Id will be propagated through all method calls propagated of this HTTP handler
		ctx := slogctx.With(req.Context(),
			commoncfg.AttrRequestID, uuid.New().String(),
			commoncfg.AttrOperation, "healthcheck",
		)

		// Manual OTEL Tracing
		parentCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))

		ctx, span := tracer.Start(
			parentCtx,
			"healthcheck-span",
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
					attribute.String(commoncfg.AttrOperation, "healthcheck"),
				)...,
			)

			counter.Add(ctx, 1, attrs)
			hist.Record(ctx, elapsedTime, attrs)
		}()

		// Business Logic
		slogctx.Info(ctx, "Starting healthcheck request")

		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(ch.Status())
		_ = json.NewEncoder(w).Encode(ch.Response())

		slogctx.Info(ctx, "Finished healthcheck request",
			"durationMs", time.Since(requestStartTime)/time.Millisecond, "response", ch.Response(), "status", ch.Status())
		// End Business Logic
	}
}
