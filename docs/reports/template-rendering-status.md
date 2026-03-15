# Contents テンプレート展開 実装メモ

## 目的
Markdown から HTML への変換後に、contents レベルのテンプレートを適用する仕様を実装し、参照ルールを明確化する。

## 実装済みの挙動
- `store/hosting/renderer/renderer.go`
  - Front Matter を抽出後、Markdown 本文を Go template で事前展開する
  - 展開後の Markdown を HTML へ変換する
  - 変換結果を `contents.body`、メタを `page.meta` としてテンプレートへ渡す
- `store/hosting/contents/contents.go`
  - 配信対象の拡張子に応じて `template.<ext>` を選択する
  - テンプレート探索は `TemplatesDir` 直下のみ
  - `template.<ext>` がない場合は素の HTML を返す

## 変数参照ルール
- Markdown 本文: top-level meta を Go template 形式で参照
  - 例: `{{.title}}`, `{{index .Suuji 0}}`
- テンプレート本文:
  - `{{.contents.body}}`
  - `{{.page.meta.title}}`
  - `{{index .contents.variables "label"}}`

## 日付仕様
- Markdown 本文の簡易参照では日付を `2006/01/02 15:04:05` へ正規化する。
- `page.meta` は `goldmark-meta` の生値を保持する。

## 適用スコープ
- 各 `Content` は自身の `TemplatesDir` のみ使用する。
- 子 `Content` が独自 `TemplatesDir` を持つ場合は、その子の配信時のみ有効。

