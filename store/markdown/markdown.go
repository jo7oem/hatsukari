package markdown

import (
	"bytes"
	"io"
)

// ReadMarkdownWithMeta は,読み込んだMarkdownファイルのメタデータと内容を返す関数です.
// r: io.Reader - 読み込むMarkdownファイルのリーダー
// 戻り値:
// []byte - メタデータ
// string - Markdown内容
// error - エラー情報
// メタデータはMarkdownファイルの先頭にYAML形式で記述されていることを想定.
// 例:
// ---
// title: サンプルタイトル
// date: 2024-01-01
// ---
// # これはMarkdownの内容です
// `---` で囲まれた部分がメタデータとして扱われる.
// メタデータが存在しない場合, 空のバイトスライスとMarkdown内容を返す.
// メタデータの終端が存在しない場合、全てをMarkdown内容として返す.
func ReadMarkdownWithMeta(r io.Reader) ([]byte, string, error) {
	if r == nil {
		return nil, "", nil
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}

	if len(data) < 8 || string(data[0:4]) != "---\n" {
		return nil, string(data), nil
	}

	endMetaIndex := bytes.Index(data, []byte("\n---\n"))
	if endMetaIndex == -1 {
		return nil, string(data), nil
	}

	metaData := data[4 : endMetaIndex+1]
	markdownData := data[endMetaIndex+5:]

	return metaData, string(markdownData), nil
}
