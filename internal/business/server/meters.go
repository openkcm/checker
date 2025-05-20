package server

import (
	"context"

	"github.com/openkcm/common-sdk/pkg/otlp"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/openkcm/checker/internal/config"
)

var (
	counter metric.Int64Counter
	hist    metric.Float64Histogram
)

func initMeters(ctx context.Context, cfg *config.Config) error {
	meter := otel.Meter(
		cfg.Application.Name,
		metric.WithInstrumentationVersion(otel.Version()),
		metric.WithInstrumentationAttributes(otlp.CreateAttributesFrom(cfg.Application)...),
	)

	var err error

	counter, err = meter.Int64Counter(
		"http.request_count",
		metric.WithDescription("Incoming request count"),
		metric.WithUnit("requests"),
	)
	if err != nil {
		return oops.In("HTTP Server").
			WithContext(ctx).
			Wrapf(err, "creating request_count meter")
	}

	hist, err = meter.Float64Histogram(
		"http.duration",
		metric.WithDescription("Incoming end to end duration"),
		metric.WithUnit("milliseconds"),
	)
	if err != nil {
		return oops.In("HTTP Server").
			WithContext(ctx).
			Wrapf(err, "creating duration meter")
	}

	return nil
}
