package site

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/contents"
)

func TestSite_Setup(t *testing.T) {
	t.Parallel()

	s, err := OpenSiteDir("./assets_test/indexing_site", newDiscardLogger())
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
	if got, want := s.Variables()["title"], "Indexing Test"; got != want {
		t.Fatalf("site title = %v, want %s", got, want)
	}
	siteVariables, ok := s.Variables()["variables"].(map[string]any)
	if !ok {
		t.Fatalf("site variables type mismatch: %T", s.Variables()["variables"])
	}
	if got := len(siteVariables); got != 0 {
		t.Fatalf("site variables length = %d, want 0", got)
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

func TestSite_OpenDir_LoggerRequired(t *testing.T) {
	t.Parallel()

	_, err := OpenSiteDir("./assets_test/indexing_site", nil)
	if err == nil {
		t.Fatal("OpenSiteDir() error = nil, want logger required error")
	}
	if !strings.Contains(err.Error(), "logger must not be nil") {
		t.Fatalf("OpenSiteDir() error = %v, want contains logger must not be nil", err)
	}
}

func TestSite_SiteVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		siteYAML     string
		contains     []string
		notContains  []string
		assertConfig func(t *testing.T, s *Site)
	}{
		{
			name:     "WithConfiguredVariables",
			siteYAML: "title: \"site-title\"\ntimezone: \"UTC\"\nrootContentDir: \"content\"\nvariables:\n  title: \"var-title\"\n  indexes: \"shadow\"\n  currentLocale: \"shadow-locale\"\n",
			contains: []string{
				"SITE_TITLE=site-title",
				"VAR_TITLE=var-title",
				"INDEX_COUNT=1",
				"CURRENT=",
				"VAR_INDEXES=shadow",
				"VAR_CURRENT=shadow-locale",
			},
			assertConfig: func(t *testing.T, s *Site) {
				t.Helper()
				if got, want := s.Variables()["title"], "site-title"; got != want {
					t.Fatalf("site title = %v, want %s", got, want)
				}
				siteVariables, ok := s.Variables()["variables"].(map[string]any)
				if !ok {
					t.Fatalf("site variables type mismatch: %T", s.Variables()["variables"])
				}
				if got, want := siteVariables["title"], "var-title"; got != want {
					t.Fatalf("site.variables.title = %v, want %s", got, want)
				}
				if got := siteVariables["indexes"]; got != "shadow" {
					t.Fatalf("site.variables.indexes = %v, want shadow", got)
				}
				indexes, ok := s.Variables()["indexes"].([]map[string]any)
				if !ok {
					t.Fatalf("site indexes type mismatch: %T", s.Variables()["indexes"])
				}
				if got, want := len(indexes), 1; got != want {
					t.Fatalf("site indexes length = %d, want %d", got, want)
				}
				if _, exists := s.Variables()["currentLocale"]; exists {
					t.Fatal("site currentLocale should not exist before request")
				}
			},
		},
		{
			name:     "WithoutConfiguredVariables",
			siteYAML: "title: \"site-title\"\ntimezone: \"UTC\"\nrootContentDir: \"content\"\n",
			contains: []string{
				"SITE_TITLE=site-title",
				"VAR_COUNT=0",
			},
			notContains: []string{"var-title"},
			assertConfig: func(t *testing.T, s *Site) {
				t.Helper()
				siteVariables, ok := s.Variables()["variables"].(map[string]any)
				if !ok {
					t.Fatalf("site variables type mismatch: %T", s.Variables()["variables"])
				}
				if got := len(siteVariables); got != 0 {
					t.Fatalf("site variables length = %d, want 0", got)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			siteDir := t.TempDir()
			writeTestFile(t, filepath.Join(siteDir, ".site.yml"), tt.siteYAML)
			writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "registerIndexing: true\ncontentsDir: []\n")
			writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "root\n")
			writeTestFile(t, filepath.Join(siteDir, "content", "templates", "template.md"), "SITE_TITLE={{.site.title}}|VAR_TITLE={{index .site.variables \"title\"}}|INDEX_COUNT={{len .site.indexes}}|CURRENT={{index .site \"currentLocale\"}}|VAR_INDEXES={{index .site.variables \"indexes\"}}|VAR_CURRENT={{index .site.variables \"currentLocale\"}}|VAR_COUNT={{len .site.variables}}\n")
			writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "registerIndexing: true\ntemplatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")

			s, err := OpenSiteDir(siteDir, newDiscardLogger())
			if err != nil {
				t.Fatalf("OpenSiteDir() error = %v", err)
			}
			t.Cleanup(func() { _ = s.Close() })

			if tt.assertConfig != nil {
				tt.assertConfig(t, s)
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
			for _, fragment := range tt.contains {
				if !strings.Contains(bodyStr, fragment) {
					t.Fatalf("GET / body does not contain %q\nbody=%s", fragment, bodyStr)
				}
			}
			for _, fragment := range tt.notContains {
				if strings.Contains(bodyStr, fragment) {
					t.Fatalf("GET / body should not contain %q\nbody=%s", fragment, bodyStr)
				}
			}
		})
	}
}

func TestSite_OpenDir_Timezone(t *testing.T) {
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

			s, err := OpenSiteDir(siteDir, newDiscardLogger())
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

func TestSite_OpenDir_ConfigPriority(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yaml"), "title: \"yaml\"\ntimezone: \"Asia/Tokyo\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"yml\"\ntimezone: \"UTC\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "contentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "root\n")

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if got, want := s.location.String(), "Asia/Tokyo"; got != want {
		t.Fatalf("location = %s, want %s", got, want)
	}
}

func TestSite_OpenDir_InvalidSiteTemplatesDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		siteTemplatesDir string
	}{
		{name: "ParentTraversal", siteTemplatesDir: "../bad"},
		{name: "AbsolutePath", siteTemplatesDir: "/bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			siteDir := t.TempDir()
			writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"test\"\nsiteTemplatesDir: \""+tt.siteTemplatesDir+"\"\nrootContentDir: \"content\"\n")
			writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "contentsDir: []\n")
			writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "root\n")

			_, err := OpenSiteDir(siteDir, newDiscardLogger())
			if err == nil {
				t.Fatal("OpenSiteDir() error = nil, want invalid siteTemplatesDir error")
			}
			if !strings.Contains(err.Error(), "invalid siteTemplatesDir") {
				t.Fatalf("OpenSiteDir() error = %v, want invalid siteTemplatesDir", err)
			}
		})
	}
}

func TestSite_OpenDir_InvalidSiteTemplateEntryPoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		siteTemplate string
	}{
		{name: "ParentTraversal", siteTemplate: "../layout.html"},
		{name: "AbsolutePath", siteTemplate: "/layout.html"},
		{name: "CurrentDir", siteTemplate: "."},
		{name: "NestedPath", siteTemplate: "layout/main.html"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			siteDir := t.TempDir()
			writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"test\"\nsiteTemplatesDir: \"site_templates\"\nsiteTemplate: \""+tt.siteTemplate+"\"\nrootContentDir: \"content\"\n")
			writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "contentsDir: []\n")
			writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "root\n")

			_, err := OpenSiteDir(siteDir, newDiscardLogger())
			if err == nil {
				t.Fatal("OpenSiteDir() error = nil, want invalid siteTemplate error")
			}
			if !strings.Contains(err.Error(), "invalid siteTemplate") {
				t.Fatalf("OpenSiteDir() error = %v, want invalid siteTemplate", err)
			}
		})
	}
}

func TestSite_ServeHTTP_SiteTemplatePipeline(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"pipeline\"\ntimezone: \"UTC\"\nsiteTemplatesDir: \"site_templates\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "site_templates", "site-template.md"), "SITE|BODY={{.contents.body}}|META={{.page.meta.title}}\n")

	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "templatesDir: templates\ncontentTemplate: template.md\ncontentsDir:\n  - posts\n  - naked\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "templates", "template.md"), "ROOT|BODY={{.contents.body}}|META={{.page.meta.title}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "---\ntitle: root\n---\nroot\n")

	writeTestFile(t, filepath.Join(siteDir, "content", "posts", ".content.yaml"), "contentType: posts\ntemplatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "templates", "template.md"), "POSTS|BODY={{.contents.body}}|META={{.page.meta.title}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "templates", "tags.md"), "TAGS|COUNT={{len .contents.posts.tags}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "index.md"), "---\ntitle: posts\n---\nposts\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "entry.md"), "---\ntitle: entry\npostedAt: 2026-03-01T00:00:00Z\nvisibility: public\nsummary: entry\ntags: [\"demo\"]\n---\nentry\n")

	writeTestFile(t, filepath.Join(siteDir, "content", "naked", ".content.yaml"), "disableSiteTemplate: true\ntemplatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "naked", "templates", "template.md"), "NAKED|BODY={{.contents.body}}|META={{.page.meta.title}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "naked", "index.md"), "---\ntitle: naked\n---\nnaked\n")

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	server := httptest.NewServer(s)
	t.Cleanup(server.Close)

	tests := []struct {
		name        string
		urlPath     string
		wantStatus  int
		contains    []string
		notContains []string
	}{
		{
			name:       "RootAppliesContentThenSite",
			urlPath:    "/",
			wantStatus: http.StatusOK,
			contains:   []string{"SITE|BODY=ROOT|BODY=<p>root</p>", "META=root"},
		},
		{
			name:       "PostsTagsAlsoApplySiteTemplate",
			urlPath:    "/posts/tags/",
			wantStatus: http.StatusOK,
			contains:   []string{"SITE|BODY=TAGS|COUNT=1", "META="},
		},
		{
			name:        "DisableSiteTemplatePerContent",
			urlPath:     "/naked/",
			wantStatus:  http.StatusOK,
			contains:    []string{"NAKED|BODY=<p>naked</p>", "META=naked"},
			notContains: []string{"SITE|BODY="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, err := server.Client().Get(server.URL + tt.urlPath)
			if err != nil {
				t.Fatalf("GET %s error = %v", tt.urlPath, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if got := resp.StatusCode; got != tt.wantStatus {
				t.Fatalf("GET %s status = %d, want %d", tt.urlPath, got, tt.wantStatus)
			}

			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)
			for _, fragment := range tt.contains {
				if !strings.Contains(bodyStr, fragment) {
					t.Fatalf("GET %s body does not contain %q\nbody=%s", tt.urlPath, fragment, bodyStr)
				}
			}
			for _, fragment := range tt.notContains {
				if strings.Contains(bodyStr, fragment) {
					t.Fatalf("GET %s body should not contain %q\nbody=%s", tt.urlPath, fragment, bodyStr)
				}
			}
		})
	}
}

func TestSite_ServeHTTP_SiteTemplateMissingFallback(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"fallback\"\ntimezone: \"UTC\"\nsiteTemplatesDir: \"site_templates\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "templatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "templates", "template.md"), "CONTENT|BODY={{.contents.body}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "hello\n")

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

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
	if !strings.Contains(bodyStr, "CONTENT|BODY=<p>hello</p>") {
		t.Fatalf("GET / body=%s, want content template result", bodyStr)
	}
	if strings.Contains(bodyStr, "SITE|BODY=") {
		t.Fatalf("GET / body should not contain site template wrapper\nbody=%s", bodyStr)
	}
}

func TestSite_ServeHTTP_SiteTemplateHTMLFallback(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"html-fallback\"\ntimezone: \"UTC\"\nsiteTemplatesDir: \"site_templates\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "site_templates", "site-template.html"), "SITE_HTML|BODY={{.contents.body}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "templatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "templates", "template.md"), "CONTENT|BODY={{.contents.body}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "hello\n")

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

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
	if !strings.Contains(bodyStr, "SITE_HTML|BODY=CONTENT|BODY=<p>hello</p>") {
		t.Fatalf("GET / body=%s, want site-template.html fallback result", bodyStr)
	}
}

func TestSite_ServeHTTP_SiteTemplateEntryPointFromConfig(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"entry-point\"\ntimezone: \"UTC\"\nsiteTemplatesDir: \"site_templates\"\nsiteTemplate: \"site-layout.html\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "site_templates", "site-layout.html"), "SITE_ENTRY|BODY={{.contents.body}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "templatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "templates", "template.md"), "CONTENT|BODY={{.contents.body}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "hello\n")

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

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
	if !strings.Contains(bodyStr, "SITE_ENTRY|BODY=CONTENT|BODY=<p>hello</p>") {
		t.Fatalf("GET / body=%s, want configured site template entry point result", bodyStr)
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
	if err := os.WriteFile(filepath.Join(postsDir, ".content.yaml"), []byte("contentType: posts\nlatest: 1\ntemplatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n"), 0o600); err != nil {
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

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
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

func TestSite_PostsMarkdownTemplateContext(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"posts\"\ntimezone: \"UTC\"\nlatest: 2\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "registerIndexing: true\ncontentsDir:\n  - posts\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "root\n")

	postsDir := filepath.Join(siteDir, "content", "posts")
	writeTestFile(t, filepath.Join(postsDir, ".content.yaml"), "contentType: posts\nlatest: 1\ntemplatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(postsDir, "templates", "template.md"), "BODY={{.contents.body}}\n")
	writeTestFile(t, filepath.Join(postsDir, "index.md"), "---\ntitle: 記事一覧\n---\nSITE_LATEST={{len .site.posts.latest}}|CONTENT_LATEST={{len .contents.posts.latest}}|TITLE={{.page.meta.title}}\n")
	writeTestFile(t, filepath.Join(postsDir, "a.md"), "---\ntitle: A\npostedAt: 2026-03-01T00:00:00Z\nvisibility: public\n---\nA\n")
	writeTestFile(t, filepath.Join(postsDir, "b.md"), "---\ntitle: B\npostedAt: 2026-03-02T00:00:00Z\nvisibility: public\n---\nB\n")

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

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
	for _, fragment := range []string{"SITE_LATEST=2", "CONTENT_LATEST=1", "TITLE=記事一覧"} {
		if !strings.Contains(bodyStr, fragment) {
			t.Fatalf("GET /posts/ body does not contain %q\nbody=%s", fragment, bodyStr)
		}
	}
}

func TestSite_OpenDir_MultiplePostsContents(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"posts\"\ntimezone: \"UTC\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "contentsDir:\n  - posts\n  - diary\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "root\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", ".content.yaml"), "contentType: posts\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "index.md"), "posts\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "diary", ".content.yaml"), "contentType: posts\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "diary", "index.md"), "diary\n")

	_, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err == nil {
		t.Fatal("OpenSiteDir() error = nil, want multiple posts contents error")
	}
	for _, fragment := range []string{"multiple posts contents found", "/posts/", "/diary/"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("OpenSiteDir() error = %v, want contains %q", err, fragment)
		}
	}
}

func TestSite_PostTags(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"posts\"\ntimezone: \"UTC\"\nlatest: 3\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "contentsDir:\n  - posts\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "root\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", ".content.yaml"), "contentType: posts\nlatest: 2\ntemplatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", ".tag.yaml"), "known:\n  defaultLang: ja\n  label:\n    ja: \"既知タグ\"\n    en: \"Known Tag\"\n  about:\n    ja: \"既知タグの説明\"\n    en: \"Known tag description\"\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "index.md"), "---\ntitle: 記事一覧\n---\nposts\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "templates", "template.md"), "{{if .page.meta.title}}TITLE={{.page.meta.title}}|{{range .contents.posts.byTag}}X{{end}}{{range .contents.posts.all}}{{if eq .Title $.page.meta.title}}{{range .Tags}}{{if .URL}}KNOWN_LINK={{.URL}}|KNOWN_LABEL={{index .Label \"default\"}}|{{else}}UNKNOWN_LABEL={{index .Label \"default\"}}|{{end}}{{end}}{{end}}{{end}}{{end}}BODY={{.contents.body}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "templates", "tags.md"), "{{if .contents.posts.currentTag.Key}}DETAIL={{.contents.posts.currentTag.Key}}|COUNT={{.contents.posts.currentTag.Count}}|ABOUT={{index .contents.posts.currentTag.About \"default\"}}|{{range .contents.posts.currentTag.Posts}}{{.Title}};{{end}}{{else}}LIST={{len .contents.posts.tags}}|{{range .contents.posts.tags}}{{.Key}}={{index .Label \"default\"}}@{{.Count}}@{{.URL}};{{end}}{{end}}\n")

	posts := map[string]string{
		"public.md":   "---\ntitle: Public\npostedAt: 2026-03-01T00:00:00Z\nvisibility: public\nsummary: public\ntags: [\"known\", \"unknown\"]\n---\npublic\n",
		"unlisted.md": "---\ntitle: Unlisted\npostedAt: 2026-03-02T00:00:00Z\nvisibility: unlisted\nsummary: unlisted\ntags: [\"known\"]\n---\nunlisted\n",
		"direct.md":   "---\ntitle: Direct\npostedAt: 2026-03-03T00:00:00Z\nvisibility: directOnly\nsummary: direct\ntags: [\"known\"]\n---\ndirect\n",
		"future.md":   "---\ntitle: Future\npostedAt: 2026-03-04T00:00:00Z\npublishAt: 2999-01-01T00:00:00Z\nvisibility: public\nsummary: future\ntags: [\"known\"]\n---\nfuture\n",
	}
	for name, body := range posts {
		writeTestFile(t, filepath.Join(siteDir, "content", "posts", name), body)
	}

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	server := httptest.NewServer(s)
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + "/posts/public")
	if err != nil {
		t.Fatalf("GET /posts/public error = %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	bodyStr := string(body)
	for _, fragment := range []string{"KNOWN_LINK=/posts/tags/known/", "KNOWN_LABEL=既知タグ", "UNKNOWN_LABEL=unknown"} {
		if !strings.Contains(bodyStr, fragment) {
			t.Fatalf("GET /posts/public body does not contain %q\nbody=%s", fragment, bodyStr)
		}
	}

	resp, err = server.Client().Get(server.URL + "/posts/tags/")
	if err != nil {
		t.Fatalf("GET /posts/tags/ error = %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	bodyStr = string(body)
	for _, fragment := range []string{"LIST=2", "known=既知タグ@2@/posts/tags/known/", "unknown=unknown@1@"} {
		if !strings.Contains(bodyStr, fragment) {
			t.Fatalf("GET /posts/tags/ body does not contain %q\nbody=%s", fragment, bodyStr)
		}
	}

	resp, err = server.Client().Get(server.URL + "/posts/tags/known")
	if err != nil {
		t.Fatalf("GET /posts/tags/known error = %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	bodyStr = string(body)
	for _, fragment := range []string{"DETAIL=known", "COUNT=2", "ABOUT=既知タグの説明", "Public;", "Unlisted;"} {
		if !strings.Contains(bodyStr, fragment) {
			t.Fatalf("GET /posts/tags/known body does not contain %q\nbody=%s", fragment, bodyStr)
		}
	}
	for _, fragment := range []string{"Direct;", "Future;"} {
		if strings.Contains(bodyStr, fragment) {
			t.Fatalf("GET /posts/tags/known body should not contain %q\nbody=%s", fragment, bodyStr)
		}
	}

	for _, pathName := range []string{"/posts/tags/unknown", "/posts/tags/none"} {
		resp, err := server.Client().Get(server.URL + pathName)
		if err != nil {
			t.Fatalf("GET %s error = %v", pathName, err)
		}
		_ = resp.Body.Close()
		if got, want := resp.StatusCode, http.StatusNotFound; got != want {
			t.Fatalf("GET %s status = %d, want %d", pathName, got, want)
		}
	}
}

func TestSite_PostTagsWithoutDefinitionFile(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"posts\"\ntimezone: \"UTC\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "contentsDir:\n  - posts\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "root\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", ".content.yaml"), "contentType: posts\ntemplatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "index.md"), "---\ntitle: 記事一覧\n---\nposts\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "templates", "template.md"), "{{range .contents.posts.all}}{{range .Tags}}{{if .URL}}LINK={{.URL}}{{else}}TEXT={{index .Label \"default\"}}{{end}}{{end}}{{end}}")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "templates", "tags.md"), "{{range .contents.posts.tags}}{{.Key}};{{end}}")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "only.md"), "---\ntitle: Only\npostedAt: 2026-03-01T00:00:00Z\nvisibility: public\nsummary: only\ntags: [\"raw-tag\"]\n---\nonly\n")

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	server := httptest.NewServer(s)
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + "/posts/only")
	if err != nil {
		t.Fatalf("GET /posts/only error = %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "TEXT=raw-tag") {
		t.Fatalf("GET /posts/only body=%s, want raw tag text", bodyStr)
	}
	if strings.Contains(bodyStr, "LINK=") {
		t.Fatalf("GET /posts/only body should not contain link\nbody=%s", bodyStr)
	}

	resp, err = server.Client().Get(server.URL + "/posts/tags/raw-tag")
	if err != nil {
		t.Fatalf("GET /posts/tags/raw-tag error = %v", err)
	}
	_ = resp.Body.Close()
	if got, want := resp.StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("GET /posts/tags/raw-tag status = %d, want %d", got, want)
	}
}

func TestSite_PostTagsTemplateMissing(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"posts\"\ntimezone: \"UTC\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "contentsDir:\n  - posts\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "root\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", ".content.yaml"), "contentType: posts\ntemplatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "index.md"), "---\ntitle: 記事一覧\n---\nposts\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "templates", "template.md"), "{{.contents.body}}")
	writeTestFile(t, filepath.Join(siteDir, "content", "posts", "only.md"), "---\ntitle: Only\npostedAt: 2026-03-01T00:00:00Z\nvisibility: public\n---\nonly\n")

	s, err := OpenSiteDir(siteDir, newDiscardLogger())
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	server := httptest.NewServer(s)
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + "/posts/tags/")
	if err != nil {
		t.Fatalf("GET /posts/tags/ error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got, want := resp.StatusCode, http.StatusInternalServerError; got != want {
		t.Fatalf("GET /posts/tags/ status = %d, want %d", got, want)
	}
}

func writeTestFile(t *testing.T, filePath, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
		t.Fatalf("failed to create dir for %s: %v", filePath, err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", filePath, err)
	}
}

func newDiscardLogger() *logging.Logger {
	return logging.NewLogger(slog.NewTextHandler(io.Discard, nil), "test")
}
