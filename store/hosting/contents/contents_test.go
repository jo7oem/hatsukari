package contents_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jo7oem/hatsukari/store/hosting/contents"
)

var testFS = func() *os.Root {
	root, err := os.OpenRoot("./assets_test")
	if err != nil {
		panic(err)
	}

	return root
}()

func TestOpenContentDir(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "simple content dir",
			path:    "simple",
			wantErr: false,
		},
		{
			name:    "content dir with nested content",
			path:    "nests",
			wantErr: false,
		},
		{
			name:    "content dir with templates",
			path:    "template_case",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := contents.OpenContentDir(testFS, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenContentDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestContent_ServeHTTP(t *testing.T) {
	type subtest struct {
		name               string
		url                string
		wantStatus         int
		wantBody           string
		wantBodyContains   []string
		wantBodyNotContain []string
		wantError          bool
	}
	tests := []struct {
		name     string
		path     string
		subtests []subtest
	}{

		{
			name: "simple content dir",
			path: "simple",
			subtests: []subtest{
				{
					name:       "serve index.html",
					url:        "/",
					wantStatus: http.StatusOK,
					wantBody:   "<p>index.md</p>\n",
					wantError:  false,
				},
				{
					name:       "deny access to non-existent file",
					url:        "/notfound",
					wantStatus: http.StatusNotFound,
					wantBody:   "404 page not found\n",
					wantError:  false,
				},
				{
					name:       "deny access to dotfile",
					url:        "/.content.yaml",
					wantStatus: http.StatusNotFound,
					wantBody:   "404 page not found\n",
					wantError:  false,
				},
				{
					name:       "deny access to dot dir",
					url:        "/.hidden/",
					wantStatus: http.StatusNotFound,
					wantBody:   "404 page not found\n",
					wantError:  false,
				},
			},
		},
		{
			name: "content dir with nested content",
			path: "nests",
			subtests: []subtest{
				{
					name:       "root",
					url:        "/",
					wantStatus: http.StatusOK,
					wantBody:   "<p>index.md</p>\n",
					wantError:  false,
				},
				{
					name:       "child content",
					url:        "/child/",
					wantStatus: http.StatusOK,
					wantBody:   "index.html",
					wantError:  false,
				},
				{
					name:       "child content without trailing slash",
					url:        "/child",
					wantStatus: http.StatusOK,
					wantBody:   "index.html",
					wantError:  false,
				},
				{
					name:       "grandchild content",
					url:        "/child/grandchild/",
					wantStatus: http.StatusOK,
					wantBody:   "<p>g</p>\n",
					wantError:  false,
				},
				{
					name:       "exist file",
					url:        "/child/priority_1",
					wantStatus: http.StatusOK,
					wantBody:   "p1",
					wantError:  false,
				},
				{
					name:       "html file has higher priority than md file",
					url:        "/child/priority_2",
					wantStatus: http.StatusOK,
					wantBody:   "p2",
					wantError:  false,
				},
				{
					name:       ".md",
					url:        "/child/priority_3",
					wantStatus: http.StatusOK,
					wantBody:   "<p>p3</p>\n",
					wantError:  false,
				},
				{
					name:       "directory with index.html",
					url:        "/child/priority_4",
					wantStatus: http.StatusOK,
					wantBody:   "p4",
					wantError:  false,
				},
				{
					name:       "directory with index.md",
					url:        "/child/priority_5",
					wantStatus: http.StatusOK,
					wantBody:   "<p>p5</p>\n",
					wantError:  false,
				},
				{
					name:       "deny access to dotfile",
					url:        "/child/.content.yaml",
					wantStatus: http.StatusNotFound,
					wantBody:   "404 page not found\n",
					wantError:  false,
				},
				{
					name:       "deny access to dotfile in grandchild",
					url:        "/child/grandchild/.content.yaml",
					wantStatus: http.StatusNotFound,
					wantBody:   "404 page not found\n",
					wantError:  false,
				},
			},
		},
		{
			name: "content dir with templates",
			path: "template_case",
			subtests: []subtest{
				{
					name:             "root uses root template and markdown meta interpolation",
					url:              "/",
					wantStatus:       http.StatusOK,
					wantBodyContains: []string{"ROOT|", "BODY=<p>root-title/7/2026/03/10 22:10:11</p>\n", "META=root-title", "VAR=root-var"},
					wantError:        false,
				},
				{
					name:               "child uses child template only",
					url:                "/child/",
					wantStatus:         http.StatusOK,
					wantBodyContains:   []string{"CHILD|", "BODY=<p>child child-title</p>\n", "META=child-title"},
					wantBodyNotContain: []string{"ROOT|"},
					wantError:          false,
				},
				{
					name:       "missing template file falls back to plain html",
					url:        "/raw/",
					wantStatus: http.StatusOK,
					wantBody:   "<p>raw raw-title</p>\n",
					wantError:  false,
				},
			},
		},
	}
	for _, tt := range tests {
		content, err := contents.OpenContentDir(testFS, tt.path)
		if err != nil {
			t.Fatalf("failed to open content dir: %v", err)
		}

		server := httptest.NewServer(content)
		for _, subTest := range tt.subtests {
			t.Run(tt.name+"/"+subTest.name, func(t *testing.T) {
				t.Parallel()
				p := server.URL + subTest.url
				resp, err := server.Client().Get(p)
				if (err != nil) != subTest.wantError {
					t.Errorf("Get() error = %v, wantError %v", err, subTest.wantError)
					return
				}

				if resp.StatusCode != subTest.wantStatus {
					t.Errorf("Get() status = %v, wantStatus %v", resp.StatusCode, subTest.wantStatus)
				}

				body, _ := io.ReadAll(resp.Body)
				bodyStr := string(body)

				if subTest.wantBody != "" {
					if diff := cmp.Diff(subTest.wantBody, bodyStr); diff != "" {
						t.Errorf("Get() body mismatch (-want +got):\n%s", diff)
					}
				}

				for _, fragment := range subTest.wantBodyContains {
					if !strings.Contains(bodyStr, fragment) {
						t.Errorf("Get() body does not contain %q\nbody: %s", fragment, bodyStr)
					}
				}

				for _, fragment := range subTest.wantBodyNotContain {
					if strings.Contains(bodyStr, fragment) {
						t.Errorf("Get() body should not contain %q\nbody: %s", fragment, bodyStr)
					}
				}
			})
		}
	}
}
