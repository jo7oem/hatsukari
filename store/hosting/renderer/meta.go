package renderer

import (
	"bytes"
	"fmt"
	texttemplate "text/template"
	"time"

	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	gtext "github.com/yuin/goldmark/text"
)

func ExtractMeta(source []byte) (map[string]any, error) {
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
