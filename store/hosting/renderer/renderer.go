package renderer

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	texttemplate "text/template"
	"time"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	gtext "github.com/yuin/goldmark/text"
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
	siteVariables     map[string]any
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

func WithSiteVariables(siteVariables map[string]any) Option {
	return func(r *Renderer) {
		r.siteVariables = siteVariables
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
	metaData, err := extractMeta(r.source)
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
		},
		"site": r.siteVariables,
		"page": map[string]any{
			"meta": meta.Get(context),
		},
	}

	return renderPageTemplate(r.templateFS, r.templateName, pageData, htmlBuffer.Bytes())
}

func extractMeta(source []byte) (map[string]any, error) {
	context := parser.NewContext()
	_ = markdown.Parser().Parse(gtext.NewReader(source), parser.WithContext(context))
	metaData := meta.Get(context)
	if metaData == nil {
		return map[string]any{}, nil
	}
	return metaData, nil
}

func renderMarkdownWithMetaTemplate(source []byte, metaData map[string]any) ([]byte, error) {
	tmpl, err := texttemplate.New("markdown").Option("missingkey=zero").Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown template: %w", err)
	}

	data := make(map[string]any, len(metaData))
	for key, value := range metaData {
		data[key] = normalizeMetaValue(value)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return nil, fmt.Errorf("failed to execute markdown template: %w", err)
	}
	return out.Bytes(), nil
}

func normalizeMetaValue(value any) any {
	switch v := value.(type) {
	case time.Time:
		return v.Format(markdownDateLayout)
	case string:
		if formatted, ok := normalizeDateString(v); ok {
			return formatted
		}
		return v
	case []any:
		normalized := make([]any, 0, len(v))
		for _, item := range v {
			normalized = append(normalized, normalizeMetaValue(item))
		}
		return normalized
	case map[string]any:
		normalized := make(map[string]any, len(v))
		for key, item := range v {
			normalized[key] = normalizeMetaValue(item)
		}
		return normalized
	case map[any]any:
		normalized := make(map[string]any, len(v))
		for key, item := range v {
			normalized[fmt.Sprint(key)] = normalizeMetaValue(item)
		}
		return normalized
	default:
		return value
	}
}

func normalizeDateString(value string) (string, bool) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t.Format(markdownDateLayout), true
		}
	}

	return "", false
}

func renderPageTemplate(templateFS fs.FS, templateName string, data map[string]any, fallback []byte) ([]byte, error) {
	entries, err := fs.ReadDir(templateFS, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read templates dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}

	tmpl, err := template.New("templates").Option("missingkey=zero").ParseFS(templateFS, files...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	if tmpl.Lookup(templateName) == nil {
		return fallback, nil
	}

	out := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(out, templateName, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %q: %w", templateName, err)
	}
	return out.Bytes(), nil
}

func (r *Renderer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	b, err := r.Render()
	if err != nil {
		http.Error(w, "failed to render", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, req, "a.html", wakeUpTime, bytes.NewReader(b))
}
