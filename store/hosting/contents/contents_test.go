package contents_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
		name       string
		url        string
		wantStatus int
		wantBody   string
		wantError  bool
	}
	tests := []struct {
		name    string
		path    string
		subtest subtest
	}{
		{
			name: "simple content dir",
			path: "simple",
			subtest: subtest{
				name:       "serve index.html",
				url:        "/",
				wantStatus: http.StatusOK,
				wantBody:   "index.md",
				wantError:  false,
			},
		},
	}
	for _, tt := range tests {
		content, err := contents.OpenContentDir(testFS, tt.path)
		if err != nil {
			t.Fatalf("failed to open content dir: %v", err)
		}

		server := httptest.NewServer(content)
		for _, subTest := range []subtest{tt.subtest} {
			t.Run(subTest.name, func(t *testing.T) {
				t.Parallel()
				resp, err := server.Client().Get(server.URL + subTest.url)
				if (err != nil) != subTest.wantError {
					t.Errorf("Get() error = %v, wantError %v", err, subTest.wantError)
					return
				}

				if resp.StatusCode != subTest.wantStatus {
					t.Errorf("Get() status = %v, wantStatus %v", resp.StatusCode, subTest.wantStatus)
				}

				body, _ := io.ReadAll(resp.Body)

				if diff := cmp.Diff(subTest.wantBody, string(body)); diff != "" {
					t.Errorf("Get() body mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}
