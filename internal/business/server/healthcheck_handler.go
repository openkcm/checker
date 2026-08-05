package server

import (
	"encoding/json"
	"fmt"
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
	"github.com/openkcm/checker/internal/healthcheck"
)

func maskResponse(r *healthcheck.Response) *healthcheck.Response {
	cloned := *r
	cloned.URL = "****"

	if len(r.Errors) > 0 {
		maskedErrors := make([]healthcheck.ErrorResponse, len(r.Errors))

		for i, e := range r.Errors {
			maskedErrors[i] = e
			if maskedErrors[i].Message != "" {
				maskedErrors[i].Message = "****"
			}
		}

		cloned.Errors = maskedErrors
	}

	return &cloned
}

func maskURLs(response map[string]any) map[string]any {
	masked := make(map[string]any, len(response))

	for k, v := range response {
		switch val := v.(type) {
		case []*healthcheck.Response:
			maskedList := make([]*healthcheck.Response, len(val))

			for i, r := range val {
				maskedList[i] = maskResponse(r)
			}

			masked[k] = maskedList
		case *healthcheck.Response:
			masked[k] = maskResponse(val)
		default:
			masked[k] = v
		}
	}

	return masked
}

func healthcheckHandlerFunc(operation string, cfg *config.Config, ch *healthcheck.CachedResponses) func(http.ResponseWriter, *http.Request) {
	traceAttrs := otlp.CreateAttributesFrom(cfg.Application,
		attribute.String(commoncfg.AttrOperation, operation),
	)

	tracer := otel.Tracer("HealthCheckerHandler", trace.WithInstrumentationAttributes(traceAttrs...))

	return func(w http.ResponseWriter, req *http.Request) {
		// Request Id will be propagated through all method calls propagated of this HTTP handler
		ctx := slogctx.With(req.Context(),
			commoncfg.AttrRequestID, uuid.New().String(),
			commoncfg.AttrOperation, operation,
		)

		// Manual OTEL Tracing
		parentCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))

		ctx, span := tracer.Start(
			parentCtx,
			operation+"-span",
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
					attribute.String(commoncfg.AttrOperation, operation),
				)...,
			)

			counter.Add(ctx, 1, attrs)
			hist.Record(ctx, elapsedTime, attrs)
		}()

		// Business Logic
		slogctx.Info(ctx, "Starting request", "operation", operation)

		w.Header().Set("Content-Type", "application/json")

		rawResponse := ch.Response()
		response := rawResponse

		if cfg.Healthcheck.MaskURLs {
			response = maskURLs(rawResponse)
		}

		w.WriteHeader(ch.Status())
		_ = json.NewEncoder(w).Encode(response)

		slogctx.Info(ctx, fmt.Sprintf("Finished %s request", operation),
			"durationMs", time.Since(requestStartTime)/time.Millisecond, "response", rawResponse, "status", ch.Status())
		// End Business Logic
	}
}
