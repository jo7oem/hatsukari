package renderer

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/jo7oem/hatsukari/logging"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestRenderer_ServeHTTP_SpanHierarchy(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)
	tracer := provider.Tracer("test/renderer")
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })

	r := NewRenderer(
		[]byte("---\ntitle: order\n---\ncontent\n"),
		WithTemplateFS(fstest.MapFS{"template.md": &fstest.MapFile{Data: []byte("CONTENT|BODY={{.contents.body}}")}}, "template.md"),
		WithSiteTemplateFS(fstest.MapFS{"site-template.md": &fstest.MapFile{Data: []byte("SITE|BODY={{.contents.body}}")}}, "site-template.md"),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(logging.InjectTracer(req.Context(), tracer))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)
	if got := rec.Code; got != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", got, http.StatusOK)
	}

	spans := spansByName(recorder.Ended())
	assertSpanParent(t, spans, "renderer.Render", "renderer.ServeHTTP")
	assertSpanParent(t, spans, "renderer.extractMeta", "renderer.Render")
	assertSpanParent(t, spans, "renderer.expandMarkdownTemplate", "renderer.Render")
	assertSpanParent(t, spans, "renderer.markdownConvert", "renderer.Render")
	assertSpanParent(t, spans, "renderer.applyContentTemplate", "renderer.Render")
	assertSpanParent(t, spans, "renderer.applySiteTemplate", "renderer.Render")
}

func TestRenderer_ServeHTTP_ErrorRecordsSpan(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)
	tracer := provider.Tracer("test/renderer")
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })

	logger := logging.NewLogger(slog.NewTextHandler(io.Discard, nil), "test")
	r := NewRenderer(
		[]byte("plain\n"),
		WithTemplateFS(fstest.MapFS{"template.md": &fstest.MapFile{Data: []byte("{{")}}, "template.md"),
		WithLogger(logger),
	)

	req := httptest.NewRequest(http.MethodGet, "/broken", nil)
	req = req.WithContext(logging.InjectTracer(req.Context(), tracer))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)
	if got := rec.Code; got != http.StatusInternalServerError {
		t.Fatalf("ServeHTTP() status = %d, want %d", got, http.StatusInternalServerError)
	}

	spans := spansByName(recorder.Ended())
	if len(spans["renderer.Render"]) != 1 {
		t.Fatalf("renderer.Render span count = %d, want 1", len(spans["renderer.Render"]))
	}
	if len(spans["renderer.applyContentTemplate"]) != 1 {
		t.Fatalf("renderer.applyContentTemplate span count = %d, want 1", len(spans["renderer.applyContentTemplate"]))
	}
	if got := spans["renderer.Render"][0].Status().Code; got != codes.Error {
		t.Fatalf("renderer.Render status code = %v, want %v", got, codes.Error)
	}
}

func spansByName(spans []sdktrace.ReadOnlySpan) map[string][]sdktrace.ReadOnlySpan {
	indexed := make(map[string][]sdktrace.ReadOnlySpan)
	for _, span := range spans {
		indexed[span.Name()] = append(indexed[span.Name()], span)
	}
	return indexed
}

func assertSpanParent(t *testing.T, spans map[string][]sdktrace.ReadOnlySpan, childName, parentName string) {
	t.Helper()
	children := spans[childName]
	parents := spans[parentName]
	if len(children) != 1 {
		t.Fatalf("%s span count = %d, want 1", childName, len(children))
	}
	if len(parents) != 1 {
		t.Fatalf("%s span count = %d, want 1", parentName, len(parents))
	}
	if got, want := children[0].Parent().SpanID(), parents[0].SpanContext().SpanID(); got != want {
		t.Fatalf("%s parent span id = %s, want %s", childName, got, want)
	}
}

func TestRenderer_StartSpan_UsesInjectedTracer(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)
	tracer := provider.Tracer("test/injected")
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })

	ctx, span := logging.StartSpan(logging.InjectTracer(t.Context(), tracer), "custom")
	span.End()
	_ = ctx

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("span count = %d, want 1", len(spans))
	}
	if got := spans[0].Name(); got != "custom" {
		t.Fatalf("span name = %s, want custom", got)
	}
	if got := spans[0].SpanContext().TraceID(); got == (trace.TraceID{}) {
		t.Fatal("trace id should not be zero")
	}
}
