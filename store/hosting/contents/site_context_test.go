package contents

import "testing"

func TestSiteContext_VariablesMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		context    SiteContext
		posts      map[string]any
		wantHasKey map[string]bool
	}{
		{
			name:       "EmptyContext",
			context:    SiteContext{},
			posts:      nil,
			wantHasKey: map[string]bool{"title": true, "variables": true, "indexes": true, "posts": false},
		},
		{
			name: "WithValues",
			context: SiteContext{
				Title:     "site-title",
				Variables: map[string]any{"theme": "dark"},
				Indexes:   []map[string]any{{"url": "/", "title": "top"}},
			},
			posts:      map[string]any{"latest": []string{"a"}},
			wantHasKey: map[string]bool{"title": true, "variables": true, "indexes": true, "posts": true},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.context.VariablesMap(tt.posts)
			for key, want := range tt.wantHasKey {
				_, exists := got[key]
				if exists != want {
					t.Fatalf("VariablesMap() key=%q exists=%v, want %v", key, exists, want)
				}
			}
		})
	}
}

func TestSiteContext_VariablesMap_CloneSafety(t *testing.T) {
	t.Parallel()

	ctx := SiteContext{
		Title:     "site-title",
		Variables: map[string]any{"theme": "dark"},
		Indexes:   []map[string]any{{"url": "/", "title": "top"}},
	}
	got := ctx.VariablesMap(nil)

	variables, ok := got["variables"].(map[string]any)
	if !ok {
		t.Fatalf("variables type = %T, want map[string]any", got["variables"])
	}
	variables["theme"] = "light"
	if ctx.Variables["theme"] != "dark" {
		t.Fatalf("context variables mutated: %#v", ctx.Variables)
	}

	indexes, ok := got["indexes"].([]map[string]any)
	if !ok {
		t.Fatalf("indexes type = %T, want []map[string]any", got["indexes"])
	}
	indexes[0]["title"] = "changed"
	if ctx.Indexes[0]["title"] != "top" {
		t.Fatalf("context indexes mutated: %#v", ctx.Indexes)
	}
}

func TestSiteContext_VariablesMapWithLocale(t *testing.T) {
	t.Parallel()

	ctx := SiteContext{Title: "site-title"}
	got := ctx.VariablesMapWithLocale(nil, "en")
	if got["currentLocale"] != "en" {
		t.Fatalf("currentLocale = %#v, want en", got["currentLocale"])
	}
}
