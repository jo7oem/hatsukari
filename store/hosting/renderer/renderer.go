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
	"github.com/yuin/goldmark/parser"
)

var wakeUpTime = time.Now()
var markdown = goldmark.New(
	goldmark.WithExtensions(
		meta.Meta,
	),
)

const markdownDateLayout = "2006/01/02 15:04:05"

type Option func(*Renderer)

type Renderer struct {
	source            []byte
	templateFS        fs.FS
	templateName      string
	contentsVariables map[string]any
	contentsPosts     any
	siteVariables     map[string]any
	logger            *logging.Logger
}

func WithTemplateFS(templateFS fs.FS, templateName string) Option {
	return func(r *Renderer) {
		r.templateFS = templateFS
		r.templateName = templateName
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

	expandedMarkdown, err := renderMarkdownWithMetaTemplate(r.source, metaData)
	if err != nil {
		return nil, err
	}

	htmlBuffer := bytes.NewBuffer(nil)
	context := parser.NewContext()
	if err := markdown.Convert(expandedMarkdown, htmlBuffer, parser.WithContext(context)); err != nil {
		return nil, err
	}

	if r.templateFS == nil || r.templateName == "" {
		return htmlBuffer.Bytes(), nil
	}

	pageData := map[string]any{
		"contents": map[string]any{
			"body":      template.HTML(htmlBuffer.String()),
			"variables": r.contentsVariables,
			"posts":     r.contentsPosts,
		},
		"site": r.siteVariables,
		"page": map[string]any{
			"meta": meta.Get(context),
		},
	}

	return renderPageTemplate(r.templateFS, r.templateName, pageData, htmlBuffer.Bytes())
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
