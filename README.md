# hatsukari

## 概要
個人用途のブログ記事を公開するためのアプリケーションです。

## 現在の配信仕様
- Markdown は Front Matter を `goldmark-meta` で読み取り、本文を HTML に変換します。
- Markdown 本文では Front Matter の top-level meta を Go template 形式で参照できます。
  - 例: `{{.title}}`, `{{index .Suuji 0}}`
- 日付の簡易参照は `2006/01/02 15:04:05` 形式に正規化されます。

## Content テンプレート
- 各 `Content` は `ContentConfig.TemplatesDir` を参照します。
- 配信対象ファイルの拡張子に対応する `template.<拡張子>` を `TemplatesDir` 直下から読み込みます。
  - 例: `.md` の場合は `template.md`
- `template.<拡張子>` が存在しない場合は、変換後の素の HTML を返します。
- `template.*` は同じ `TemplatesDir` 直下のテンプレートを読み込めます。

## テンプレート変数

### 共通
- `contents.body`: Markdown 変換後の HTML
- `contents.variables`: `.content.yaml` の `variables`
- `page.meta`: `goldmark-meta` が返す Front Matter の生値

### サイト全体
- `site.indexes`: `registerIndexing: true` の Content を優先順位ソートした配列
- `site.posts`: サイト全体の記事情報
  - `site.posts.all`: サイト全体の一覧表示対象記事
  - `site.posts.latest`: サイト全体の最新記事
  - `site.posts.tags`: サイト全体のタグ一覧
  - `site.posts.byTag`: タグキーごとの記事一覧

### posts Content 配下
- `contents.posts`: `contentType: posts` の Content で利用できる記事情報
  - `contents.posts.all`: その Content 配下の一覧表示対象記事
  - `contents.posts.latest`: その Content 配下の最新記事
  - `contents.posts.tags`: その Content 配下のタグ一覧
  - `contents.posts.byTag`: その Content 配下のタグごとの記事一覧
  - `contents.posts.currentTag`: タグ詳細ページで表示中のタグ情報

### `site.indexes` の構造
- `url`: 目次用 URL（末尾 `/` 付き）
- `title.default`: `indexTitle[defaultLocale]` を優先し、未設定時は `indexTitle.ja`、さらに未設定なら `url`
- `title.ja`: `indexTitle.ja` を優先し、未設定時は `url`

### `site.indexes` の公開ポリシー
- `title` は `default` と `ja` のみを公開します。
- `title.en` など `ja` 以外の locale キーは `site.indexes` に含めません。
- `?lang=<locale>` 指定時は `title[locale]` を参照し、未定義なら `title.default` にフォールバックします。

### `site.indexes` の並び順
- `priority` 昇順（小さい値ほど高優先、負値を許容）
- `priority` 同値時は読み込み順（`ContentsDir` の定義順を含む）

### 参照例
- `{{.contents.body}}`
- `{{.page.meta.title}}`
- `{{index .contents.variables "label"}}`
- `{{index (index .site.indexes 0) "url"}}`
- `{{range .site.posts.latest}}{{.Title}}{{end}}`
- `{{range .contents.posts.latest}}{{.Title}}{{end}}`
- `{{range .contents.posts.tags}}{{index .Label "default"}}{{end}}`

## Site 設定

### `timezone`
- `.site.yml` の `timezone` で公開判定の基準タイムゾーンを指定できます。
- `timezone` は前後空白を `TrimSpace` して解釈します。
- 未指定時は `UTC` を使用します。
- 不正な値は起動エラーになります。
- エラーメッセージには設定値を含めます。

### `latest`
- `.site.yml` の `latest` がサイト全体の最新記事件数です。
- 未指定時の既定値は `10` です。

## posts 仕様
- `.content.yaml` に `contentType: posts` を指定した Content を記事収集対象にします。
- サイト全体で `contentType: posts` は 1 つだけ許可します。複数ある場合は起動エラーです。
- `index.md` は記事収集対象外です（一覧ページ用途）。
- Front Matter がない `.md` は記事として扱いません。
- `contentType: posts` 配下では `.md` への直接アクセスは常に `404` を返します。
- 拡張子なし URL（例: `/posts/sample-post`）で記事へアクセスします。
- `site.posts` / `contents.posts` は `publishAt` と `visibility` を考慮してリクエストごとに再生成します。

### 記事メタ
- 必須
  - `title`
  - `postedAt`
- 任意
  - `publishAt`
  - `summary`
  - `tags`
  - `revisions`
- 使用しない項目
  - `slug`（指定しても無視します）

### `revisions`
- `revisions` は `[{ revisedAt, summary }]` 形式です。
- 改稿履歴がない場合、`LatestRevision()` は `nil` を返します。
- 一覧や記事上部では `LatestRevision()` による最新改稿表示を想定しています。
- 記事末尾では `revisions` 全体表示を想定しています。

### `visibility`
- `public`: URL OK / tag OK / list OK
- `unlisted`: URL OK / tag OK / list NG
- `directOnly`: URL OK / tag NG / list NG
- `private`: URL NG / tag NG / list NG
- `publishAt` 未到達、または可視条件未達は `404` 扱いです。
- `publishAt` 判定は記事詳細・記事一覧・タグ一覧のすべてで共通に適用します。

### `latest` の解決順
- `.site.yml` の `latest` がサイト全体 `site.posts.latest` の上限です。
- `contentType: posts` の `.content.yaml` で `latest` を指定すると、その Content 配下表示（`contents.posts.latest`）のみ上書きします。

## tags 仕様
- posts Content 直下の `.tag.yaml` がタグ定義ファイルです。
- `.tag.yaml` が存在しない場合は、タグ定義なしとして扱います。
- 記事メタ `tags` は `.tag.yaml` の参照キーを想定します。
- 未知タグはタグ名そのままで返し、リンクを持たせません。
- 未知タグの詳細 URL は `404` を返します。
- `tags` は posts の予約名です。

### `.tag.yaml`
```yaml
sample:
  defaultLang: ja
  label:
    ja: "サンプル"
    en: "Sample"
  about:
    ja: "サンプル記事のタグ"
    en: "Sample articles"
```

### タグ公開データ
- `label.default`
- `label.ja`
- `about.default`
- `about.ja`

### タグ URL
- タグ一覧: `/posts/tags/`
- タグ詳細: `/posts/tags/<tag-key>/`

### タグテンプレート
- タグ一覧/タグ詳細のテンプレート名は `tags.md` です。
- `templatesDir` 配下に配置します。

## テスト
```bash
go test ./store/hosting/site ./store/hosting/contents ./store/hosting/renderer ./logging -count=1
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
