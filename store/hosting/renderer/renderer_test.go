package renderer

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestRenderer_Render(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		render   func() *Renderer
		want     string
		contains []string
		wantErr  bool
	}{
		{
			name: "ExpandMarkdownMeta",
			render: func() *Renderer {
				source := []byte(`---
title: sample-title
Suuji: [7, 8]
date: 2026-03-10T22:10:11Z
---
{{.title}}/{{index .Suuji 0}}/{{.date}}
`)
				return NewRenderer(source)
			},
			want: "<p>sample-title/7/2026/03/10 22:10:11</p>\n",
		},
		{
			name: "ApplyTemplateWithPartials",
			render: func() *Renderer {
				source := []byte(`---
title: demo
---
hello {{.title}}
`)
				templateFS := fstest.MapFS{
					"template.md": &fstest.MapFile{Data: []byte(`{{template "parts.md" .}}BODY={{.contents.body}}|TITLE={{.page.meta.title}}|VAR={{index .contents.variables "label"}}|SITE={{index (index .site.indexes 0) "url"}}`)},
					"parts.md":    &fstest.MapFile{Data: []byte(`{{define "parts.md"}}HEAD|{{end}}`)},
				}
				return NewRenderer(
					source,
					WithTemplateFS(templateFS, "template.md"),
					WithContentsVariables(map[string]any{"label": "v1"}),
					WithSiteVariables(map[string]any{"indexes": []map[string]any{{"url": "/"}}}),
				)
			},
			contains: []string{"HEAD|", "BODY=<p>hello demo</p>\n", "TITLE=demo", "VAR=v1", "SITE=/"},
		},
		{
			name: "TemplateMissingFallback",
			render: func() *Renderer {
				source := []byte("plain\n")
				templateFS := fstest.MapFS{
					"parts.md": &fstest.MapFile{Data: []byte(`{{define "parts.md"}}X{{end}}`)},
				}
				return NewRenderer(source, WithTemplateFS(templateFS, "template.md"))
			},
			want: "<p>plain</p>\n",
		},
		{
			name: "ApplyContentThenSiteTemplate",
			render: func() *Renderer {
				source := []byte("---\ntitle: order\n---\ncontent\n")
				contentTemplateFS := fstest.MapFS{
					"template.md": &fstest.MapFile{Data: []byte("CONTENT|BODY={{.contents.body}}|META={{.page.meta.title}}")},
				}
				siteTemplateFS := fstest.MapFS{
					"site-template.md": &fstest.MapFile{Data: []byte("SITE|BODY={{.contents.body}}|META={{.page.meta.title}}")},
				}
				return NewRenderer(
					source,
					WithTemplateFS(contentTemplateFS, "template.md"),
					WithSiteTemplateFS(siteTemplateFS, "site-template.md"),
				)
			},
			contains: []string{"SITE|BODY=CONTENT|BODY=<p>content</p>", "META=order"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.render().Render()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Render() error = %v, wantErr %v", err, tt.wantErr)
			}

			gotBody := string(got)
			if tt.want != "" && gotBody != tt.want {
				t.Fatalf("Render() mismatch\nwant: %q\ngot : %q", tt.want, gotBody)
			}

			for _, fragment := range tt.contains {
				if !strings.Contains(gotBody, fragment) {
					t.Fatalf("Render() body does not contain %q\nbody=%s", fragment, gotBody)
				}
			}
		})
	}
}
