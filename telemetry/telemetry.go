package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	telemetryInitMaxAttempts = 3
	telemetryInitBackoffBase = 200 * time.Millisecond
	metricExportInterval     = 10 * time.Second
	defaultTracerName        = "hatsukari"
)

type Config struct {
	ServiceName      string
	ExporterEndpoint string
	Insecure         bool
}

type tracerContextKey struct{}

func Init(ctx context.Context, conf Config) (func(context.Context) error, error) {
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

	traceExp, err := newOTLPTraceExporterWithRetry(ctx, endpoint, conf.Insecure)
	if err != nil {
		return nil, err
	}

	tp, err := newTracerProvider(serviceName, traceExp)
	if err != nil {
		return nil, err
	}

	metricExp, err := newOTLPMetricExporterWithRetry(ctx, endpoint, conf.Insecure)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, err
	}
	mp, err := newMeterProvider(serviceName, metricExp)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, err
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	return func(shutdownCtx context.Context) error {
		if shutdownCtx == nil {
			shutdownCtx = context.Background()
		}
		metricErr := mp.Shutdown(shutdownCtx)
		traceErr := tp.Shutdown(shutdownCtx)
		if metricErr != nil {
			return metricErr
		}
		return traceErr
	}, nil
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

func newOTLPTraceExporterWithRetry(ctx context.Context, endpoint string, insecure bool) (sdktrace.SpanExporter, error) {
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

func newOTLPMetricExporterWithRetry(ctx context.Context, endpoint string, insecure bool) (sdkmetric.Exporter, error) {
	attempts := max(telemetryInitMaxAttempts, 1)
	backoff := telemetryInitBackoffBase
	if backoff <= 0 {
		backoff = 200 * time.Millisecond
	}

	opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
	if insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	var lastErr error
	for i := 1; i <= attempts; i++ {
		exp, err := otlpmetricgrpc.New(ctx, opts...)
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

	return nil, fmt.Errorf("failed to initialize otlp metric exporter after %d attempts: %w", attempts, lastErr)
}

func newTracerProvider(serviceName string, exp sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {
	r, err := newTelemetryResource(serviceName)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	), nil
}

func newMeterProvider(serviceName string, exp sdkmetric.Exporter) (*sdkmetric.MeterProvider, error) {
	r, err := newTelemetryResource(serviceName)
	if err != nil {
		return nil, err
	}

	interval := metricExportInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(r),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(interval))),
	), nil
}

func newTelemetryResource(serviceName string) (*resource.Resource, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}
