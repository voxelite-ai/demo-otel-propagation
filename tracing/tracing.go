package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type tracing struct {
	TracerProvider *sdktrace.TracerProvider
	spanExporter   sdktrace.SpanExporter
	resources      *resource.Resource
}

func Start(ctx context.Context) (*tracing, error) {
	resources, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", "demoservice"),
			attribute.String("telemetry.sdk.language", "GO"),
		),
	)
	if err != nil {
		return nil, err
	}

	tp, traceExporter, err := newTracer(ctx, resources)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tp)

	// See: https://opentelemetry.io/docs/languages/go/instrumentation/#propagators-and-context
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &tracing{
		TracerProvider: tp,
		spanExporter:   traceExporter,

		resources: resources,
	}, nil
}

func (t *tracing) Shutdown(ctx context.Context) error {
	if t.spanExporter != nil {
		if err := t.spanExporter.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}

// The provided context is used for exporter initialization, and the resource parameter defines the trace resource attributes.
func newTracer(ctx context.Context, resource *resource.Resource) (tracer *sdktrace.TracerProvider, exporter sdktrace.SpanExporter, err error) {
	exporter, err = otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint("localhost:4317"),
			otlptracegrpc.WithInsecure(),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
			sdktrace.WithBatchTimeout(sdktrace.DefaultScheduleDelay*time.Millisecond),
			sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
		),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource),
	)

	return tp, exporter, err
}
