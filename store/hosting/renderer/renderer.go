package renderer

import (
	"bytes"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
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
	metaData, err := ExtractMeta(r.source)
	if err != nil {
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
	for key, value := range normalizedMeta {
		markdownTemplateData[key] = value
	}
	// 本文テンプレートではシステム変数を優先する
	for key, value := range pageData {
		markdownTemplateData[key] = value
	}

	expandedMarkdown, err := renderMarkdownWithMetaTemplate(r.source, markdownTemplateData)
	if err != nil {
		return nil, err
	}

	htmlBuffer := bytes.NewBuffer(nil)
	context := parser.NewContext()
	if err := markdown.Convert(expandedMarkdown, htmlBuffer, parser.WithContext(context)); err != nil {
		return nil, err
	}

	contentsData["body"] = template.HTML(htmlBuffer.String())
	pageMeta, _ := pageData["page"].(map[string]any)
	pageMeta["meta"] = meta.Get(context)

	out := htmlBuffer.Bytes()
	if r.contentTemplateFS != nil && r.contentTemplate != "" {
		out, err = renderPageTemplate(r.contentTemplateFS, r.contentTemplate, pageData, out)
		if err != nil {
			return nil, err
		}
	}

	contentsData["body"] = template.HTML(string(out))
	if r.siteTemplateFS != nil && r.siteTemplate != "" {
		out, err = renderPageTemplate(r.siteTemplateFS, r.siteTemplate, pageData, out)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (r *Renderer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	b, err := r.Render()
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to render", err, slog.String("path", req.URL.Path))
		}
		http.Error(w, "failed to render", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, req, "a.html", wakeUpTime, bytes.NewReader(b))
}
