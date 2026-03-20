package renderer

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
)

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
