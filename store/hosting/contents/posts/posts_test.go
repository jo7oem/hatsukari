package posts

import (
	"reflect"
	"testing"
	"time"
)

func TestEntry_IsDirectVisible(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name  string
		entry Entry
		want  bool
	}{
		{name: "Public", entry: Entry{Visibility: VisibilityPublic}, want: true},
		{name: "Private", entry: Entry{Visibility: VisibilityPrivate}, want: false},
		{name: "FuturePublish", entry: Entry{Visibility: VisibilityPublic, PublishAt: &future}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.entry.IsDirectVisible(now); got != tt.want {
				t.Fatalf("IsDirectVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPosts_ParseMetaTime(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("JST", 9*60*60)
	got, ok := ParseMetaTime("2026-04-29", loc)
	if !ok {
		t.Fatal("ParseMetaTime() failed")
	}
	if got.Location().String() != loc.String() {
		t.Fatalf("location = %s, want %s", got.Location().String(), loc.String())
	}
}

func TestPosts_ParseVisibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  Visibility
	}{
		{name: "Default", input: nil, want: VisibilityPublic},
		{name: "Known", input: "unlisted", want: VisibilityUnlisted},
		{name: "Unknown", input: "foo", want: VisibilityPrivate},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseVisibility(tt.input); got != tt.want {
				t.Fatalf("ParseVisibility() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPosts_ParseTagKeys(t *testing.T) {
	t.Parallel()

	got := ParseTagKeys([]any{"go", " go ", "", nil, "web"})
	want := []string{"go", "web"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseTagKeys() = %#v, want %#v", got, want)
	}
}

func TestPosts_BuildTagMap(t *testing.T) {
	t.Parallel()

	entries := []Entry{{
		Title: "first",
		Tags: []Tag{
			{Key: "go", URL: "/tags/go/", Label: map[string]string{"ja": "Go"}, About: map[string]string{"ja": "about"}},
		},
	}}
	got := BuildTagMap(entries)
	if len(got) != 1 {
		t.Fatalf("BuildTagMap() length = %d, want 1", len(got))
	}
	feed, ok := got["go"]
	if !ok {
		t.Fatal("BuildTagMap() missing go")
	}
	if feed.Count != 1 || len(feed.Posts) != 1 {
		t.Fatalf("feed = %#v, want count=1 posts=1", feed)
	}
}
