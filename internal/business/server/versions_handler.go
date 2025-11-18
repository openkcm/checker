package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/openkcm/common-sdk/pkg/commoncfg"
	"github.com/openkcm/common-sdk/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	slogctx "github.com/veqryn/slog-context"

	"github.com/openkcm/checker/internal/config"
	"github.com/openkcm/checker/internal/versions"
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

		response := versions.Query(ctx, &cfg.Versions)
		response[cfg.Application.Name] = cfg.Application.BuildInfo

		jsonStr, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slogctx.Error(ctx, "Error encoding response", "error", err)

			return
		}

		w.WriteHeader(http.StatusOK)

		_, err = w.Write(jsonStr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slogctx.Error(ctx, "Error encoding response", "error", err)

			return
		}

		slogctx.Info(ctx, "Finished versions request",
			"durationMs", time.Since(requestStartTime)/time.Millisecond, "response", string(jsonStr))

		// End Business Logic
	}
}
