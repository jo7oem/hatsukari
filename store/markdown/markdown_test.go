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
		{
			name: "メタデータなし: 冒頭が --- ではない",
			args: args{
				r: io.NopCloser(strings.NewReader("# heading\n---\nこの --- は本文内")),
			},
			wantMeta: nil,
			wantMd:   "# heading\n---\nこの --- は本文内",
			wantErr:  false,
		},
		{
			name: "メタデータ終端なし: 冒頭 --- から始まるが閉じない",
			args: args{
				r: io.NopCloser(strings.NewReader("---\ntitle: 未終端")),
			},
			wantMeta: nil,
			wantMd:   "---\ntitle: 未終端",
			wantErr:  false,
		},
		{
			name:     "nil リーダー",
			args:     args{r: nil},
			wantMeta: nil,
			wantMd:   "",
			wantErr:  false,
		},
		{
			name:     "空文字列",
			args:     args{r: io.NopCloser(strings.NewReader(""))},
			wantMeta: nil,
			wantMd:   "",
			wantErr:  false,
		},
		{
			name:     "短いコンテンツ (<8 bytes)",
			args:     args{r: io.NopCloser(strings.NewReader("abc"))},
			wantMeta: nil,
			wantMd:   "abc",
			wantErr:  false,
		},
		{
			name:     "Windows改行: メタ判定されない",
			args:     args{r: io.NopCloser(strings.NewReader("---\r\ntitle: Win改行\r\n---\r\n# 本文"))},
			wantMeta: nil,
			wantMd:   "---\r\ntitle: Win改行\r\n---\r\n# 本文",
			wantErr:  false,
		},
		{
			name:     "本文中に追加の --- がある",
			args:     args{r: io.NopCloser(strings.NewReader("---\ntitle: Second --- test\n---\n# 本文1\n---\n本文2"))},
			wantMeta: []byte("title: Second --- test\n"),
			wantMd:   "# 本文1\n---\n本文2",
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
				t.Errorf("ReadMarkdownWithMeta() diff meta:\n%s\n", diff)
			}

			if diff := cmp.Diff(tt.wantMd, gotMd); diff != "" {
				t.Errorf("ReadMarkdownWithMeta() diff markdown:\n%s\n", diff)
			}
		})
	}
}
