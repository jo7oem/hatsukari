package markdown_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jo7oem/hatsukari/store/markdown"
)

// errReader は常にエラーを返す Reader。
type errReader struct{}

func (e errReader) Read(p []byte) (int, error) { return 0, errors.New("read error") }

func TestParseMarkdown(t *testing.T) {
	type args struct {
		r    io.Reader
		data *any
	}
	tests := []struct {
		name        string
		args        args
		wantHTML    []byte
		wantErr     bool
		wantMeta    any
		wantMetaSet bool // data ポインタにメタデータが設定されることを期待するか
	}{
		{
			name: "with front matter and body",
			args: func() args {
				mdSrc := strings.Join([]string{
					"---",
					"title: Test Title",
					"author: Test Author",
					"---",
					"# Hello",
					"",
					"Paragraph line.",
				}, "\n")
				var v any
				return args{r: strings.NewReader(mdSrc), data: &v}
			}(),
			wantHTML:    []byte("<h1>Hello</h1>\n<p>Paragraph line.</p>\n"),
			wantErr:     false,
			wantMeta:    map[string]any{"title": "Test Title", "author": "Test Author"},
			wantMetaSet: true,
		},
		{
			name: "no front matter heading only",
			args: func() args {
				var v any
				return args{r: strings.NewReader("# Title\n"), data: &v}
			}(),
			wantHTML:    []byte("<h1>Title</h1>\n"),
			wantErr:     false,
			wantMeta:    nil,
			wantMetaSet: false,
		},
		{
			name: "empty front matter",
			args: func() args {
				mdSrc := strings.Join([]string{
					"---",
					"---",
					"# Title",
				}, "\n") + "\n"
				var v any
				return args{r: strings.NewReader(mdSrc), data: &v}
			}(),
			wantHTML:    []byte("<h1>Title</h1>\n"),
			wantErr:     false,
			wantMeta:    map[string]any{},
			wantMetaSet: true,
		},
		{
			name: "front matter but no body (only metadata)",
			args: func() args {
				mdSrc := strings.Join([]string{
					"---",
					"title: Meta Only",
					"description: none",
					"---",
				}, "\n") + "\n"
				var v any
				return args{r: strings.NewReader(mdSrc), data: &v}
			}(),
			// Body が空なので HTML は空文字列 (markdown ライブラリは空入力で空バイト列を返す想定)
			wantHTML:    []byte(""),
			wantErr:     false,
			wantMeta:    map[string]any{"title": "Meta Only", "description": "none"},
			wantMetaSet: true,
		},
		{
			name: "start delimiter only (no closing --- treated as no metadata)",
			args: func() args {
				// メタデータ閉じデリミタが無いのでメタ扱いされず、Markdown として処理される
				mdSrc := strings.Join([]string{
					"---",
					"title: Oops",
					"", // 末尾に空行
				}, "\n")
				var v any
				return args{r: strings.NewReader(mdSrc), data: &v}
			}(),
			// 期待 HTML: '---' は区切り線 <hr />, 続く行は段落。
			wantHTML:    []byte("<hr />\n<p>title: Oops</p>\n"),
			wantErr:     false,
			wantMeta:    nil,
			wantMetaSet: false,
		},
		{
			name:        "nil reader",
			args:        args{r: nil, data: nil},
			wantHTML:    nil,
			wantErr:     true,
			wantMeta:    nil,
			wantMetaSet: false,
		},
		{
			name:        "reader returns error",
			args:        args{r: errReader{}, data: nil},
			wantHTML:    nil,
			wantErr:     true,
			wantMeta:    nil,
			wantMetaSet: false,
		},
		{
			name: "data pointer nil (metadata ignored)",
			args: args{r: strings.NewReader(strings.Join([]string{
				"---", "title: Ignored", "---", "# Title"}, "\n") + "\n"), data: nil},
			wantHTML:    []byte("<h1>Title</h1>\n"),
			wantErr:     false,
			wantMeta:    nil,
			wantMetaSet: false,
		},
	}
	for _, tt := range tests {
		t.Parallel()
		t.Run(tt.name, func(t *testing.T) {
			got, err := markdown.ParseMarkdown(tt.args.r, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseMarkdown() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.wantHTML, got); diff != "" {
				t.Errorf("HTML diff (-want +got)\n%s", diff)
			}
			// メタデータ検証

			if diff := cmp.Diff(tt.wantMeta, tt.args.data); diff != "" {
				t.Errorf("Meta diff (-want +got)\n%s", diff)
			}
		})
	}
}
