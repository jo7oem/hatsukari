package site

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jo7oem/hatsukari/store/hosting/contents"
)

func TestSite_Setup(t *testing.T) {
	t.Parallel()

	s, err := OpenSiteDir("./assets_test/indexing_site")
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	indexes, ok := s.Variables()["indexes"].([]map[string]any)
	if !ok {
		t.Fatalf("site indexes type mismatch: %T", s.Variables()["indexes"])
	}

	if got, want := len(indexes), 4; got != want {
		t.Fatalf("indexes length = %d, want %d", got, want)
	}

	wantURLs := []string{"/c/", "/b/", "/a/", "/"}
	for i, want := range wantURLs {
		if got := indexes[i]["url"]; got != want {
			t.Fatalf("indexes[%d].url = %v, want %s", i, got, want)
		}
	}

	bTitle, ok := indexes[1]["title"].(map[string]string)
	if !ok {
		t.Fatalf("indexes[1].title type mismatch: %T", indexes[1]["title"])
	}
	if got, want := bTitle["default"], "Bee"; got != want {
		t.Fatalf("indexes[1].title.default = %s, want %s", got, want)
	}
	if got, want := bTitle["ja"], "/b/"; got != want {
		t.Fatalf("indexes[1].title.ja = %s, want %s", got, want)
	}

	server := httptest.NewServer(s)
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET / error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("GET / status = %d, want %d", got, want)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, fragment := range []string{"NAV=4", "FIRST=/c/", "FIRST_JA=シー", "CURRENT=", "FIRST_SELECTED=シー"} {
		if !strings.Contains(bodyStr, fragment) {
			t.Fatalf("GET / body does not contain %q\nbody=%s", fragment, bodyStr)
		}
	}

	respEN, err := server.Client().Get(server.URL + "/?lang=en")
	if err != nil {
		t.Fatalf("GET /?lang=en error = %v", err)
	}
	defer func() { _ = respEN.Body.Close() }()

	if got, want := respEN.StatusCode, http.StatusOK; got != want {
		t.Fatalf("GET /?lang=en status = %d, want %d", got, want)
	}

	bodyEN, _ := io.ReadAll(respEN.Body)
	bodyENStr := string(bodyEN)
	for _, fragment := range []string{"CURRENT=en", "FIRST_SELECTED=シー"} {
		if !strings.Contains(bodyENStr, fragment) {
			t.Fatalf("GET /?lang=en body does not contain %q\nbody=%s", fragment, bodyENStr)
		}
	}
}

func TestOpenSiteDir_Timezone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		timezoneValue  string
		wantLocation   string
		wantErrContain string
	}{
		{
			name:          "DefaultUTC",
			timezoneValue: "",
			wantLocation:  "UTC",
		},
		{
			name:          "TrimmedTimezone",
			timezoneValue: "  Asia/Tokyo  ",
			wantLocation:  "Asia/Tokyo",
		},
		{
			name:           "InvalidTimezone",
			timezoneValue:  "  Invalid/Zone  ",
			wantErrContain: `invalid timezone "Invalid/Zone"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			siteDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(siteDir, ".site.yml"), []byte(siteYAMLForTest(tt.timezoneValue)), 0o600); err != nil {
				t.Fatalf("failed to write .site.yml: %v", err)
			}

			contentDir := filepath.Join(siteDir, "content")
			if err := os.MkdirAll(contentDir, 0o700); err != nil {
				t.Fatalf("failed to create content dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(contentDir, ".content.yaml"), []byte("registerIndexing: true\ncontentsDir: []\n"), 0o600); err != nil {
				t.Fatalf("failed to write .content.yaml: %v", err)
			}
			if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte("index\n"), 0o600); err != nil {
				t.Fatalf("failed to write index.md: %v", err)
			}

			s, err := OpenSiteDir(siteDir)
			if tt.wantErrContain != "" {
				if err == nil {
					t.Fatalf("OpenSiteDir() error = nil, want contains %q", tt.wantErrContain)
				}
				if !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Fatalf("OpenSiteDir() error = %v, want contains %q", err, tt.wantErrContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("OpenSiteDir() error = %v", err)
			}
			t.Cleanup(func() { _ = s.Close() })

			if got := s.location.String(); got != tt.wantLocation {
				t.Fatalf("location = %s, want %s", got, tt.wantLocation)
			}
		})
	}
}

func siteYAMLForTest(timezone string) string {
	if timezone == "" {
		return "title: \"test\"\nrootContentDir: \"content\"\n"
	}
	return "title: \"test\"\ntimezone: \"" + timezone + "\"\nrootContentDir: \"content\"\n"
}

func TestSite_Posts(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(siteDir, ".site.yml"), []byte("title: \"posts\"\ntimezone: \"UTC\"\nlatest: 2\nrootContentDir: \"content\"\n"), 0o600); err != nil {
		t.Fatalf("failed to write site config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(siteDir, "content", "templates"), 0o700); err != nil {
		t.Fatalf("failed to create root templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "content", ".content.yaml"), []byte("registerIndexing: true\ncontentsDir:\n  - posts\n"), 0o600); err != nil {
		t.Fatalf("failed to write root content config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "content", "index.md"), []byte("root\n"), 0o600); err != nil {
		t.Fatalf("failed to write root index: %v", err)
	}

	postsDir := filepath.Join(siteDir, "content", "posts")
	if err := os.MkdirAll(filepath.Join(postsDir, "templates"), 0o700); err != nil {
		t.Fatalf("failed to create posts templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(postsDir, ".content.yaml"), []byte("contentType: posts\nlatest: 1\ntemplatesDir: templates\ncontentsDir: []\n"), 0o600); err != nil {
		t.Fatalf("failed to write posts content config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(postsDir, "templates", "template.md"), []byte("SITE_LATEST={{len .site.posts.latest}}|CONTENT_LATEST={{len .contents.posts.latest}}|CONTENT_ALL={{len .contents.posts.all}}|BODY={{.contents.body}}\n"), 0o600); err != nil {
		t.Fatalf("failed to write posts template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(postsDir, "index.md"), []byte("posts-index\n"), 0o600); err != nil {
		t.Fatalf("failed to write posts index: %v", err)
	}

	files := map[string]string{
		"a.md":      "---\ntitle: A\npostedAt: 2026-03-01T00:00:00Z\nvisibility: public\nsummary: A\n---\nA\n",
		"e.md":      "---\ntitle: E\npostedAt: 2026-03-05T00:00:00Z\nvisibility: public\nsummary: E\n---\nE\n",
		"b.md":      "---\ntitle: B\npostedAt: 2026-03-06T00:00:00Z\nvisibility: unlisted\nsummary: B\n---\nB\n",
		"c.md":      "---\ntitle: C\npostedAt: 2026-03-07T00:00:00Z\nvisibility: directOnly\nsummary: C\n---\nC\n",
		"d.md":      "---\ntitle: D\npostedAt: 2026-03-08T00:00:00Z\nvisibility: private\nsummary: D\n---\nD\n",
		"future.md": "---\ntitle: F\npostedAt: 2026-03-09T00:00:00Z\npublishAt: 2999-01-01T00:00:00Z\nvisibility: public\nsummary: F\n---\nF\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(postsDir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	s, err := OpenSiteDir(siteDir)
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	sitePosts, ok := s.Variables()["posts"].(map[string]any)
	if !ok {
		t.Fatalf("site posts type mismatch: %T", s.Variables()["posts"])
	}

	allPosts, ok := sitePosts["all"].([]contents.PostEntry)
	if !ok {
		t.Fatalf("site posts all type mismatch: %T", sitePosts["all"])
	}
	if got, want := len(allPosts), 2; got != want {
		t.Fatalf("site posts all length = %d, want %d", got, want)
	}

	latestPosts, ok := sitePosts["latest"].([]contents.PostEntry)
	if !ok {
		t.Fatalf("site posts latest type mismatch: %T", sitePosts["latest"])
	}
	if got, want := len(latestPosts), 2; got != want {
		t.Fatalf("site posts latest length = %d, want %d", got, want)
	}

	server := httptest.NewServer(s)
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + "/posts/")
	if err != nil {
		t.Fatalf("GET /posts/ error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("GET /posts/ status = %d, want %d", got, want)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, fragment := range []string{"SITE_LATEST=2", "CONTENT_LATEST=1", "CONTENT_ALL=2"} {
		if !strings.Contains(bodyStr, fragment) {
			t.Fatalf("GET /posts/ body does not contain %q\nbody=%s", fragment, bodyStr)
		}
	}

	for pathName, wantStatus := range map[string]int{
		"/posts/e":      http.StatusOK,
		"/posts/b":      http.StatusOK,
		"/posts/c":      http.StatusOK,
		"/posts/d":      http.StatusNotFound,
		"/posts/future": http.StatusNotFound,
		"/posts/b.md":   http.StatusNotFound,
	} {
		resp, err := server.Client().Get(server.URL + pathName)
		if err != nil {
			t.Fatalf("GET %s error = %v", pathName, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != wantStatus {
			t.Fatalf("GET %s status = %d, want %d", pathName, resp.StatusCode, wantStatus)
		}
	}
}
