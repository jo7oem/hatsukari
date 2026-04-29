package renderer

import (
	"bytes"
	"context"
	"html/template"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"time"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var wakeUpTime = time.Now()
var markdown = goldmark.New(
	goldmark.WithExtensions(
		meta.Meta,
		extension.GFM,
		extension.CJK,
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
)

const markdownDateLayout = "2006/01/02 15:04:05"

type Option func(*Renderer)

type Renderer struct {
	source            []byte
	contentTemplateFS fs.FS
	contentTemplate   string
	siteTemplateFS    fs.FS
	siteTemplate      string
	contentsVariables map[string]any
	contentsPosts     any
	siteVariables     map[string]any
	logger            *logging.Logger
}

func WithTemplateFS(templateFS fs.FS, templateName string) Option {
	return func(r *Renderer) {
		r.contentTemplateFS = templateFS
		r.contentTemplate = templateName
	}
}

func WithSiteTemplateFS(templateFS fs.FS, templateName string) Option {
	return func(r *Renderer) {
		r.siteTemplateFS = templateFS
		r.siteTemplate = templateName
	}
}

func WithContentsVariables(contentsVariables map[string]any) Option {
	return func(r *Renderer) {
		r.contentsVariables = contentsVariables
	}
}

func WithContentsPosts(contentsPosts any) Option {
	return func(r *Renderer) {
		r.contentsPosts = contentsPosts
	}
}

func WithSiteVariables(siteVariables map[string]any) Option {
	return func(r *Renderer) {
		r.siteVariables = siteVariables
	}
}

func WithLogger(logger *logging.Logger) Option {
	return func(r *Renderer) {
		r.logger = logger
	}
}

func NewRenderer(b []byte, opts ...Option) *Renderer {
	r := &Renderer{source: b}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Renderer) Render() ([]byte, error) {
	return r.render(context.Background())
}

func (r *Renderer) render(ctx context.Context) ([]byte, error) {
	spanCtx, span := logging.StartSpan(ctx, "renderer.Render",
		trace.WithAttributes(
			attribute.Int("renderer.source.bytes", len(r.source)),
			attribute.String("renderer.content_template", r.contentTemplate),
			attribute.String("renderer.site_template", r.siteTemplate),
		),
	)
	defer span.End()

	_, extractMetaSpan := logging.StartSpan(spanCtx, "renderer.extractMeta")
	metaData, err := ExtractMeta(r.source)
	extractMetaSpan.End()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	normalizedMeta := make(map[string]any, len(metaData))
	for key, value := range metaData {
		normalizedMeta[key] = normalizeMetaValue(value)
	}

	contentsData := map[string]any{
		"body":      template.HTML(""),
		"variables": r.contentsVariables,
		"posts":     r.contentsPosts,
	}
	pageData := map[string]any{
		"contents": contentsData,
		"site":     r.siteVariables,
		"page": map[string]any{
			"meta": normalizedMeta,
		},
	}

	markdownTemplateData := make(map[string]any, len(normalizedMeta)+len(pageData))
	maps.Copy(markdownTemplateData, normalizedMeta)
	// 本文テンプレートではシステム変数を優先する
	maps.Copy(markdownTemplateData, pageData)

	_, expandMarkdownSpan := logging.StartSpan(spanCtx, "renderer.expandMarkdownTemplate")
	expandedMarkdown, err := renderMarkdownWithMetaTemplate(r.source, markdownTemplateData)
	expandMarkdownSpan.End()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	htmlBuffer := bytes.NewBuffer(nil)
	parserContext := parser.NewContext()
	_, markdownConvertSpan := logging.StartSpan(spanCtx, "renderer.markdownConvert")
	if err := markdown.Convert(expandedMarkdown, htmlBuffer, parser.WithContext(parserContext)); err != nil {
		markdownConvertSpan.RecordError(err)
		markdownConvertSpan.End()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	markdownConvertSpan.End()

	contentsData["body"] = template.HTML(htmlBuffer.String())
	pageMeta, _ := pageData["page"].(map[string]any)
	pageMeta["meta"] = meta.Get(parserContext)

	out := htmlBuffer.Bytes()
	if r.contentTemplateFS != nil && r.contentTemplate != "" {
		_, contentTemplateSpan := logging.StartSpan(spanCtx, "renderer.applyContentTemplate",
			trace.WithAttributes(attribute.String("renderer.content_template", r.contentTemplate)),
		)
		out, err = renderPageTemplate(r.contentTemplateFS, r.contentTemplate, pageData, out)
		contentTemplateSpan.End()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	contentsData["body"] = template.HTML(string(out))
	if r.siteTemplateFS != nil && r.siteTemplate != "" {
		_, siteTemplateSpan := logging.StartSpan(spanCtx, "renderer.applySiteTemplate",
			trace.WithAttributes(attribute.String("renderer.site_template", r.siteTemplate)),
		)
		out, err = renderPageTemplate(r.siteTemplateFS, r.siteTemplate, pageData, out)
		siteTemplateSpan.End()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	return out, nil
}

func (r *Renderer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, span := logging.StartSpan(req.Context(), "renderer.ServeHTTP",
		trace.WithAttributes(attribute.String("http.path", req.URL.Path)),
	)
	defer span.End()

	b, err := r.render(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if r.logger != nil {
			r.logger.Error("failed to render", err, slog.String("path", req.URL.Path))
		}
		http.Error(w, "failed to render", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, req, "a.html", wakeUpTime, bytes.NewReader(b))
}
