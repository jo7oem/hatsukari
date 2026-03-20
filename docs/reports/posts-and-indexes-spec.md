# posts / indexes 仕様整理

## 目的
`site.indexes` と `posts` 特別扱いの現行実装を、運用・テンプレート利用・今後の拡張判断のために整理する。

## 対象
- `site.indexes`
- `site.posts`
- `contents.posts`
- `.tag.yaml`
- `timezone`
- `visibility`
- `latest`

## site.indexes
### 概要
`registerIndexing: true` の Content を収集し、ヘッダーや目次のために `site.indexes` として公開する。

### 公開構造
- `url`
- `title.default`
- `title.ja`

### ロケール方針
- 公開する locale は `default` と `ja` のみ
- `title.en` などは公開しない
- `?lang=<locale>` 指定時は `title[locale] -> title.default` の順で表示を解決する

### 並び順
1. `priority` 昇順
2. 同値時は読み込み順

## posts
### 有効化
`.content.yaml` に `contentType: posts` を指定した Content 配下だけを記事収集対象とする。

### 収集対象
- `.md` のみ
- `index.md` は除外
- Front Matter がないファイルは除外

### URL
- `.md` への直接アクセスは禁止
- 記事URLは拡張子なしで配信する
  - 例: `sample-post.md` -> `/posts/sample-post`

### 記事メタ
必須:
- `title`
- `postedAt`

任意:
- `publishAt`
- `summary`
- `tags`
- `revisions`

無視するメタ:
- `slug`

### revisions
構造:
- `revisedAt`
- `summary`

用途:
- 記事上部や一覧で最新改稿を表示
- 記事末尾で全改稿履歴を表示

### visibility
- `public`: URL / tag / list すべて可
- `unlisted`: URL / tag 可、list 不可
- `directOnly`: URL のみ可
- `private`: すべて不可

### publishAt
- 未到達なら 404
- 記事詳細・一覧・タグ一覧すべて同じ判定を使う
- 判定ずれを避けるため、posts/tag データはリクエストごとに再生成する

## tags
### 定義ファイル
- posts Content 直下の `.tag.yaml`
- `.tag.yaml` がない場合はタグ定義なしとして扱う
- サイト全体で posts Content は 1 つのみ許可する

### 記事メタの tags
- `.tag.yaml` の参照キーを使う想定
- 未知タグは文字列のまま表示し、リンクは付与しない
- 未知タグの詳細 URL は 404

### 予約語
- posts では `tags` を予約名として扱う

### 公開データ
- `site.posts.tags`
- `site.posts.byTag`
- `contents.posts.tags`
- `contents.posts.byTag`
- `contents.posts.currentTag`

### タグ公開構造
- `label.default`
- `label.ja`
- `about.default`
- `about.ja`

### URL
- タグ一覧: `/posts/tags/`
- タグ詳細: `/posts/tags/<key>/`

### テンプレート
- タグ一覧/詳細ともに `tags.md`

## latest
### site
`.site.yml` の `latest` が `site.posts.latest` の件数上限。
未指定時は `10`。

### content override
`contentType: posts` の `.content.yaml` に `latest` を書くと、`contents.posts.latest` の件数だけを上書きする。

## timezone
- `.site.yml` の `timezone` を使用する
- 前後空白は無視する
- 未指定は `UTC`
- 不正値は起動エラー
- エラーメッセージには設定値を含める

## テンプレート利用例
### サイト全体の最新記事
```gotemplate
{{range .site.posts.latest}}
<li><a href='{{.URL}}'>{{.Title}}</a></li>
{{end}}
```

### その Content 配下の最新記事
```gotemplate
{{range .contents.posts.latest}}
<li><a href='{{.URL}}'>{{.Title}}</a></li>
{{end}}
```

## 現時点の注意点
- `contents.posts` は `contentType: posts` の Content 以外でも空データとして参照できる前提で扱うと安全
- `title.en` は公開しないため、英語表示が必要になった時点で公開契約の見直しが必要
- `visibility` と `publishAt` は一覧・タグ・直接URLで必ず同一ロジックを使うこと

