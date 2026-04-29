package tags

import (
	"reflect"
	"testing"
)

func TestDefinition_ParseRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		relPath     string
		wantHandled bool
		wantTagKey  string
		wantInvalid bool
	}{
		{name: "NotTags", relPath: "posts", wantHandled: false},
		{name: "TagsIndex", relPath: "tags", wantHandled: true, wantTagKey: "", wantInvalid: false},
		{name: "TagDetail", relPath: "tags/go", wantHandled: true, wantTagKey: "go", wantInvalid: false},
		{name: "NestedInvalid", relPath: "tags/go/lang", wantHandled: true, wantTagKey: "", wantInvalid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handled, tagKey, invalid := ParseRoute(tt.relPath)
			if handled != tt.wantHandled || tagKey != tt.wantTagKey || invalid != tt.wantInvalid {
				t.Fatalf("ParseRoute() = (%v, %q, %v), want (%v, %q, %v)", handled, tagKey, invalid, tt.wantHandled, tt.wantTagKey, tt.wantInvalid)
			}
		})
	}
}

func TestDefinition_NormalizeDefinition(t *testing.T) {
	t.Parallel()

	got := NormalizeDefinition(Definition{
		DefaultLang: " ",
		Label:       map[string]string{" JA ": " ラベル "},
		About:       map[string]string{"EN": " about "},
	})

	want := Definition{
		DefaultLang: "ja",
		Label:       map[string]string{"ja": "ラベル"},
		About:       map[string]string{"en": "about"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeDefinition() = %#v, want %#v", got, want)
	}
}

func TestDefinition_BuildLocalizedPublicValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		lang     string
		values   map[string]string
		fallback string
		want     map[string]string
	}{
		{
			name:     "UseFallback",
			lang:     "",
			values:   map[string]string{},
			fallback: "go",
			want:     map[string]string{"default": "go", "ja": "go"},
		},
		{
			name:     "UseLanguageAndJa",
			lang:     "en",
			values:   map[string]string{"en": "Go", "ja": "ゴー"},
			fallback: "",
			want:     map[string]string{"default": "Go", "ja": "ゴー"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := BuildLocalizedPublicValue(tt.lang, tt.values, tt.fallback); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("BuildLocalizedPublicValue() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
