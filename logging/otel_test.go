package logging

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTelemetry_InjectTracer(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)
	tracer := provider.Tracer("test/inject")
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })

	ctx := InjectTracer(nil, tracer)
	got := TracerFromContext(ctx)
	if got == nil {
		t.Fatal("TracerFromContext() = nil, want tracer")
	}

	_, span := StartSpan(ctx, "injected-span")
	span.End()

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("span count = %d, want 1", len(spans))
	}
	if got := spans[0].Name(); got != "injected-span" {
		t.Fatalf("span name = %s, want injected-span", got)
	}
}

func TestTelemetry_TracerFromContext_Fallback(t *testing.T) {
	t.Parallel()

	tracer := TracerFromContext(context.Background())
	if tracer == nil {
		t.Fatal("TracerFromContext(background) = nil, want fallback tracer")
	}
}

func TestTelemetry_NewTelemetryResource(t *testing.T) {
	t.Parallel()

	resource, err := newTelemetryResource("service-test")
	if err != nil {
		t.Fatalf("newTelemetryResource() error = %v", err)
	}

	value, ok := resource.Set().Value("service.name")
	if !ok {
		t.Fatal("service.name attribute missing")
	}
	if got := value.AsString(); got != "service-test" {
		t.Fatalf("service.name = %q, want service-test", got)
	}
}

func TestTelemetry_InitTelemetry_EmptyEndpoint(t *testing.T) {
	previousTracerName := defaultTracerName
	defaultTracerName = "hatsukari"
	t.Cleanup(func() {
		defaultTracerName = previousTracerName
	})

	shutdown, err := InitTelemetry(context.Background(), TelemetryConfig{ServiceName: "telemetry-test"})
	if err != nil {
		t.Fatalf("InitTelemetry() error = %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown function is nil")
	}
	if err := shutdown(nil); err != nil {
		t.Fatalf("shutdown(nil) error = %v", err)
	}
	if defaultTracerName != "telemetry-test" {
		t.Fatalf("defaultTracerName = %q, want telemetry-test", defaultTracerName)
	}
}
