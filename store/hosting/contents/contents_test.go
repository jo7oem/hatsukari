package contents_test

import (
	"os"
	"testing"

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
