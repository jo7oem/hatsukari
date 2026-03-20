# hatsukari

## 概要
個人用途のブログ記事を公開するためのアプリケーションです。

## 現在の配信仕様
- Markdown は Front Matter を `goldmark-meta` で読み取り、本文を HTML に変換します。
- Markdown 本文では Front Matter の top-level meta を Go template 形式で参照できます。
  - 例: `{{.title}}`, `{{index .Suuji 0}}`
- 日付の簡易参照は `2006/01/02 15:04:05` 形式に正規化されます。

## Contents テンプレート展開
- 各 `Content` は `ContentConfig.TemplatesDir` を参照します。
- 配信対象ファイルの拡張子に対応する `template.<拡張子>` を `TemplatesDir` 直下から読み込みます。
  - 例: `.md` の場合は `template.md`
- `template.<拡張子>` が存在しない場合は、変換後の素の HTML を返します。
- `template.*` は同じ `TemplatesDir` 直下のテンプレートを読み込めます。

### テンプレートで使える変数
- `contents.body`: Markdown 変換後の HTML
- `contents.variables`: `.content.yaml` の `variables`
- `site.indexes`: `registerIndexing: true` の Content を優先順位ソートした配列
- `page.meta`: `goldmark-meta` が返す Front Matter の生値

`site.indexes` の要素構造:
- `url`: 目次用 URL（末尾 `/` 付き）
- `title.default`: `indexTitle[defaultLocale]` を優先し、未設定時は `indexTitle.ja`、さらに未設定なら `url`
- `title.ja`: `indexTitle.ja` を優先し、未設定時は `url`

ソート順:
- `priority` 昇順（小さい値ほど高優先、負値を許容）
- `priority` 同値時は読み込み順（`ContentsDir` の定義順を含む）

参照例:
- `{{.contents.body}}`
- `{{.page.meta.title}}`
- `{{index .contents.variables "label"}}`
- `{{index (index .site.indexes 0) "url"}}`

## テスト
```bash
go test ./store/hosting/renderer ./store/hosting/contents -count=1
```

## 機能
- 記事管理
  - gitリポジトリによる表示管理
  - markdownからhtmlへの変換
- 配信機能
  - http(s)での配信
  - 表示数カウント

## アーキテクチャ
- Go
- DB
  - postgresql
- git
