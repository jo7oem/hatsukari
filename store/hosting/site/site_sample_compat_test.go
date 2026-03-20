package site

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSite_ServeHTTP_SampleCompatibility(t *testing.T) {
	t.Parallel()

	samplePath := sampleDirForTest(t)
	s, err := OpenSiteDir(samplePath)
	if err != nil {
		t.Fatalf("OpenSiteDir(%q) error = %v", samplePath, err)
	}
	t.Cleanup(func() { _ = s.Close() })

	server := httptest.NewServer(s)
	t.Cleanup(server.Close)

	tests := []struct {
		rq          string
		name        string
		urlPath     string
		wantStatus  int
		contains    []string
		notContains []string
	}{
		{
			rq:         "RQ-001",
			name:       "TopPage",
			urlPath:    "/",
			wantStatus: http.StatusOK,
			contains: []string{
				"<title>サンプルサイト トップ</title>",
				"最新記事（サイト全体）",
				"<a href='/posts/sample-post/'>サンプル記事</a>",
				"サンプルサイト トップ へ！",
				"<a href='/posts/'>記事一覧</a>",
			},
		},
		{
			rq:         "RQ-405",
			name:       "TopPageWithLocale",
			urlPath:    "/?lang=en",
			wantStatus: http.StatusOK,
			contains: []string{
				"<title>サンプルサイト トップ</title>",
				"<a href='/posts/'>記事一覧</a>",
			},
		},
		{
			rq:         "RQ-001",
			name:       "AboutPage",
			urlPath:    "/about/",
			wantStatus: http.StatusOK,
			contains: []string{
				"<title>このサイトについて</title>",
				"このサイトは静的サイトジェネレーターの挙動確認用サンプルです。",
			},
		},
		{
			rq:         "RQ-306",
			name:       "PostsIndex",
			urlPath:    "/posts/",
			wantStatus: http.StatusOK,
			contains: []string{
				"<title>記事一覧</title>",
				"最新記事（この配下）",
				"最新記事（サイト全体）",
				"<a href='/posts/sample-post/'>サンプル記事</a>",
			},
		},
		{
			rq:         "RQ-305",
			name:       "PostDetail",
			urlPath:    "/posts/sample-post",
			wantStatus: http.StatusOK,
			contains: []string{
				"<title>サンプル記事</title>",
				"<h1>サンプル記事</h1>",
				"sample/</code> 配下での生成を確認するためのサンプル投稿です。",
				"<a href='/posts/tags/intro/'>導入</a>",
				"<a href='/posts/tags/sample/'>サンプル</a>",
				"undefined-tag",
			},
		},
		{
			rq:         "RQ-308",
			name:       "TagsIndex",
			urlPath:    "/posts/tags/",
			wantStatus: http.StatusOK,
			contains: []string{
				"<h1>タグ一覧</h1>",
				"<a href='/posts/tags/intro/'>導入</a> (1)",
				"<a href='/posts/tags/sample/'>サンプル</a> (2)",
				"undefined-tag (2)",
			},
		},
		{
			rq:         "RQ-308",
			name:       "TagDetailIntro",
			urlPath:    "/posts/tags/intro",
			wantStatus: http.StatusOK,
			contains: []string{
				"<h1>導入</h1>",
				"導入向けの記事",
				"<a href='/posts/sample-post/'>サンプル記事</a>",
			},
		},
		{
			rq:         "RQ-308",
			name:       "TagDetailSample",
			urlPath:    "/posts/tags/sample",
			wantStatus: http.StatusOK,
			contains: []string{
				"<h1>サンプル</h1>",
				"サンプル実装の記事",
				"<a href='/posts/sample-post/'>サンプル記事</a>",
			},
		},
		{
			rq:          "RQ-308",
			name:        "TagDetailUnknown",
			urlPath:     "/posts/tags/undefined-tag",
			wantStatus:  http.StatusNotFound,
			contains:    []string{"404 page not found"},
			notContains: []string{"<h1>タグ一覧</h1>"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.rq+"/"+tt.name, func(t *testing.T) {
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

func sampleDirForTest(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	sampleDir := filepath.Clean(filepath.Join(wd, "..", "..", "..", "sample"))
	if _, err := os.Stat(sampleDir); err != nil {
		t.Fatalf("sample dir check error: %v", err)
	}

	return sampleDir
}
