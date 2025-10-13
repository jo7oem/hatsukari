package markdown

import (
	"io"
)

// ParseMarkdown は、Markdown 形式のデータをパースし、HTML に変換します。
// また、Markdown 内にメタデータが含まれている場合は、それを MdMeta 型で返します。
// r が nil の場合、または読み取りエラーが発生した場合は、エラーを返します。
// data メタデータを転記するデータ型。
// data が nil の場合、メタデータの転記は行われません。
// メタデータのフォーマットは、YAML フロントマター形式を想定しています。
// 先頭3byteが "---" で始まり、次の "---" までがメタデータとみなされます。
// 先頭が "---" で始まらない場合、あるいは終端する"---"が存在しない場合、メタデータは存在しないとみなされます。
func ParseMarkdown(r io.Reader, data *any) ([]byte, error) {
	return nil, nil
}
