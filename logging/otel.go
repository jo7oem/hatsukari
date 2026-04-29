package logging

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	telemetryInitMaxAttempts = 3
	telemetryInitBackoffBase = 200 * time.Millisecond
	defaultTracerName        = "hatsukari"
)

type TelemetryConfig struct {
	ServiceName      string
	ExporterEndpoint string
	Insecure         bool
}

type tracerContextKey struct{}

func InitTelemetry(ctx context.Context, conf TelemetryConfig) (func(context.Context) error, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	serviceName := strings.TrimSpace(conf.ServiceName)
	if serviceName == "" {
		serviceName = defaultTracerName
	}
	defaultTracerName = serviceName

	endpoint := strings.TrimSpace(conf.ExporterEndpoint)
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exp, err := newOTLPExporterWithRetry(ctx, endpoint, conf.Insecure)
	if err != nil {
		return nil, err
	}

	tp, err := newTracerProvider(serviceName, exp)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func Tracer(name string) trace.Tracer {
	tracerName := strings.TrimSpace(name)
	if tracerName == "" {
		tracerName = defaultTracerName
	}
	return otel.Tracer(tracerName)
}

func InjectTracer(ctx context.Context, tr trace.Tracer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if tr == nil {
		tr = Tracer("")
	}
	return context.WithValue(ctx, tracerContextKey{}, tr)
}

func TracerFromContext(ctx context.Context) trace.Tracer {
	if ctx != nil {
		if tr, ok := ctx.Value(tracerContextKey{}).(trace.Tracer); ok && tr != nil {
			return tr
		}
	}
	return Tracer("")
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	return TracerFromContext(ctx).Start(ctx, name, opts...)
}

func newOTLPExporterWithRetry(ctx context.Context, endpoint string, insecure bool) (sdktrace.SpanExporter, error) {
	attempts := max(telemetryInitMaxAttempts, 1)
	backoff := telemetryInitBackoffBase
	if backoff <= 0 {
		backoff = 200 * time.Millisecond
	}

	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	var lastErr error
	for i := 1; i <= attempts; i++ {
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err == nil {
			return exp, nil
		}
		lastErr = err

		if i == attempts {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}

	return nil, fmt.Errorf("failed to initialize otlp exporter after %d attempts: %w", attempts, lastErr)
}

func newTracerProvider(serviceName string, exp sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	), nil
}
