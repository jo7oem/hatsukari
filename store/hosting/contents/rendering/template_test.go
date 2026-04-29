package rendering

import (
	"reflect"
	"testing"
)

func TestTemplate_ResolveContentTemplateName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "Default", raw: "", want: "template.html"},
		{name: "Trimmed", raw: "  template.md  ", want: "template.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveContentTemplateName(tt.raw); got != tt.want {
				t.Fatalf("ResolveContentTemplateName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplate_ResolveSiteTemplateCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		siteTemplate string
		resolvedPath string
		want         []string
	}{
		{name: "NoExt", siteTemplate: "", resolvedPath: "index", want: nil},
		{name: "Configured", siteTemplate: "custom.md", resolvedPath: "index.md", want: []string{"custom.md"}},
		{name: "HTML", siteTemplate: "", resolvedPath: "index.html", want: []string{"site-template.html"}},
		{name: "Markdown", siteTemplate: "", resolvedPath: "index.md", want: []string{"site-template.md", "site-template.html"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveSiteTemplateCandidates(tt.siteTemplate, tt.resolvedPath); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ResolveSiteTemplateCandidates() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
