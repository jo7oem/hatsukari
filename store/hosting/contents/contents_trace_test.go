package contents_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/contents"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestContent_ServeHTTP_SpanHierarchy(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeContentTestFile(t, filepath.Join(siteDir, ".content.yaml"), "templatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeContentTestFile(t, filepath.Join(siteDir, "templates", "template.md"), "BODY={{.contents.body}}\n")
	writeContentTestFile(t, filepath.Join(siteDir, "index.md"), "hello\n")

	root, err := os.OpenRoot(siteDir)
	if err != nil {
		t.Fatalf("os.OpenRoot() error = %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })

	content, err := contents.OpenContentDir(root, ".", newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenContentDir() error = %v", err)
	}

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)
	tracer := provider.Tracer("test/contents")
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(logging.InjectTracer(req.Context(), tracer))
	rec := httptest.NewRecorder()

	content.ServeHTTP(rec, req)
	if got := rec.Code; got != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", got, http.StatusOK)
	}

	spans := contentSpansByName(recorder.Ended())
	assertContentSpanParent(t, spans, "contents.customRoutingHandler", "contents.ServeHTTP")
	assertContentSpanParent(t, spans, "contents.resolveContentPath", "contents.customRoutingHandler")
	assertContentSpanParent(t, spans, "contents.readMarkdown", "contents.customRoutingHandler")
	assertContentSpanParent(t, spans, "contents.resolveTemplates", "contents.customRoutingHandler")
	assertContentSpanParent(t, spans, "renderer.ServeHTTP", "contents.customRoutingHandler")
	assertContentSpanParent(t, spans, "renderer.Render", "renderer.ServeHTTP")
}

func TestContent_ServeHTTP_RenderTagsPageSpan(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeContentTestFile(t, filepath.Join(siteDir, ".content.yaml"), "contentType: posts\ntemplatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeContentTestFile(t, filepath.Join(siteDir, ".tag.yaml"), "known:\n  defaultLang: ja\n  label:\n    ja: \"既知\"\n")
	writeContentTestFile(t, filepath.Join(siteDir, "templates", "template.md"), "BODY={{.contents.body}}\n")
	writeContentTestFile(t, filepath.Join(siteDir, "templates", "tags.md"), "TAG={{.contents.posts.currentTag.Key}}\n")
	writeContentTestFile(t, filepath.Join(siteDir, "index.md"), "---\ntitle: posts\n---\nposts\n")
	writeContentTestFile(t, filepath.Join(siteDir, "entry.md"), "---\ntitle: entry\npostedAt: 2026-03-01T00:00:00Z\nvisibility: public\ntags: [\"known\"]\n---\nentry\n")

	root, err := os.OpenRoot(siteDir)
	if err != nil {
		t.Fatalf("os.OpenRoot() error = %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })

	content, err := contents.OpenContentDir(root, ".", newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenContentDir() error = %v", err)
	}
	content.SetPostsContext(nil, 10)

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)
	tracer := provider.Tracer("test/contents")
	t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })

	req := httptest.NewRequest(http.MethodGet, "/tags/known", nil)
	req = req.WithContext(logging.InjectTracer(req.Context(), tracer))
	rec := httptest.NewRecorder()

	content.ServeHTTP(rec, req)
	if got := rec.Code; got != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", got, http.StatusOK)
	}

	spans := contentSpansByName(recorder.Ended())
	assertContentSpanParent(t, spans, "contents.renderTagsPage", "contents.customRoutingHandler")
	assertContentSpanParent(t, spans, "renderer.ServeHTTP", "contents.renderTagsPage")
}

func contentSpansByName(spans []sdktrace.ReadOnlySpan) map[string][]sdktrace.ReadOnlySpan {
	indexed := make(map[string][]sdktrace.ReadOnlySpan)
	for _, span := range spans {
		indexed[span.Name()] = append(indexed[span.Name()], span)
	}
	return indexed
}

func assertContentSpanParent(t *testing.T, spans map[string][]sdktrace.ReadOnlySpan, childName, parentName string) {
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
