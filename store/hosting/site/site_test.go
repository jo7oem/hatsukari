package site

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	for _, fragment := range []string{"NAV=4", "FIRST=/c/", "FIRST_JA=シー"} {
		if !strings.Contains(bodyStr, fragment) {
			t.Fatalf("GET / body does not contain %q\nbody=%s", fragment, bodyStr)
		}
	}
}
