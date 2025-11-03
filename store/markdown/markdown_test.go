package markdown_test

import (
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jo7oem/hatsukari/store/markdown"
)

func TestReadMarkdownWithMeta(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name     string
		args     args
		wantMeta []byte
		wantMd   string
		wantErr  bool
	}{
		{
			name: "正常系: メタデータあり",
			args: args{
				r: io.NopCloser(strings.NewReader("---\ntitle: サンプルタイトル\ndate: 2024-01-01\n---\n# これはMarkdownの内容です")),
			},
			wantMeta: []byte("title: サンプルタイトル\ndate: 2024-01-01\n"),
			wantMd:   "# これはMarkdownの内容です",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMeta, gotMd, err := markdown.ReadMarkdownWithMeta(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMarkdownWithMeta() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if diff := cmp.Diff(tt.wantMeta, gotMeta); diff != "" {
				t.Errorf("ReadMarkdownWithMeta() diff: \n%s\n", diff)
			}

			if diff := cmp.Diff(tt.wantMd, gotMd); diff != "" {
				t.Errorf("ReadMarkdownWithMeta() diff: \n%s\n", diff)
			}
		})
	}
}
