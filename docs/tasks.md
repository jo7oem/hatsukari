# 最小構成CMS 計画（初期スコープ）

このドキュメントは、静的サイトCMS（Markdown/HTML入力、フロント最小）の初期実装計画を示す。コードはまだ書かない。

## フェーズ
1. プロジェクト構造と依存セットアップ
2. ドメイン/データモデルと入出力I/O
3. Markdown変換とFront Matter
4. テンプレート設計とページ生成
5. 一覧/タグ生成とアセットコピー
6. CLIとビルドコマンド
7. 品質（lint/テスト）とドキュメント

## タスク詳細

### 1. プロジェクト構造と依存セットアップ
- 作業項目
  - `cmd/hatsugen/`, `internal/site/`, `content/`, `templates/`, `assets/`, `public/`, `docs/` を作成
  - 依存追加: `github.com/yuin/goldmark`, `github.com/yuin/goldmark-meta`
- 対象
  - ディレクトリ群、`go.mod`
- 完了条件
  - ディレクトリ作成済、`go list`が成功
- 想定コマンド
  ```bash
  mkdir -p cmd/hatsugen internal/site content/posts templates assets public docs
  go get github.com/yuin/goldmark
  go get github.com/yuin/goldmark-meta
  ```

### 2. ドメイン/データモデルと入出力I/O
- 作業項目
  - `Post`（Title, Date, Tags, Slug, Summary, BodyHTML, SourcePath, Draft）
  - `Site`（BaseURL, Title, Author）
  - `content/`から`.md`/`.html`を列挙するファイル歩査
  - Slug生成（正規化、衝突時は連番）
- 対象
  - `internal/site/model.go`, `internal/site/fs.go`, `internal/site/slug.go`
- 完了条件
  - サンプル1件を構造体へ読み込める
- 想定コマンド
  ```bash
  touch content/posts/hello-world.md
  go run ./cmd/hatsugen
  ```

### 3. Markdown変換とFront Matter
- 作業項目
  - goldmark初期化（meta拡張）
  - `.md`はFront Matter抽出＋本文HTML化
  - `.html`はFront Matter抽出のみ（本文はそのまま）
  - Summaryは本文先頭から抽出（タグ除去）
  - 日付フォーマットは `YYYY-MM-DD`（ISO拡張のうち日付のみ）を入力として受け付ける
- 対象
  - `internal/site/markdown.go`, `internal/site/parser.go`
- 完了条件
  - サンプルMarkdownからTitle/Date/Tags/BodyHTMLが得られる
- 想定コマンド
  ```bash
  go run ./cmd/hatsugen
  ```

### 4. テンプレート設計とページ生成（更新）
- 作業項目
  - サイト全体テンプレート（`templates/layout.html.tmpl`）とカテゴリ別テンプレート（`templates/post.html.tmpl`, `templates/index.html.tmpl`, `templates/tag.html.tmpl`）を分離
  - カテゴリ別テンプレートは部分テンプレート（`define "post"`, `define "index"`, `define "tag"`）として実装
  - レンダリングは「カテゴリ別テンプレートを部分レンダリング→HTML（`template.HTML`）→レイアウトへ差し込み（`ContentHTML`）」のフローにする
- 対象
  - `internal/site/templates.go`（RenderPartial, WriteLayout を実装）
- 完了条件
  - 記事・一覧・タグページが、共通レイアウト＋カテゴリ別コンテンツで生成される
- 想定コマンド
  ```bash
  rm -rf public && go run ./cmd/hatsugen
  ```

### 5. 一覧/タグ生成とアセットコピー
- 作業項目
  - Dateで降順ソート
  - タグ別一覧（`/tags/<tag>/index.html`）生成
  - `assets/`コピー（CSSなど）
- 対象
  - `internal/site/build.go`, `internal/site/assets.go`
- 完了条件
  - 複数記事で一覧・タグページが生成され、CSSが反映
- 想定コマンド
  ```bash
  go run ./cmd/hatsugen
  ```

### 6. CLIとビルドコマンド
- 作業項目
  - `cmd/hatsugen/main.go`にフラグ: `-input`, `-output`, `-templates`, `-assets`, `-baseURL`
  - ログ（生成件数/エラー）
- 対象
  - `cmd/hatsugen/main.go`
- 完了条件
  - `go run ./cmd/hatsugen -input content -output public` が動作
- 想定コマンド
  ```bash
  go run ./cmd/hatsugen -input content -output public -templates templates -assets assets
  ```

### 7. 品質（lint/テスト）とドキュメント
- 作業項目
  - 単体: `slug_test.go`、Markdown→HTMLの最小テスト
  - 統合: `testdata/`入力→期待出力比較
  - `golangci-lint` 実行
  - `docs/README.md`に使い方記載
- 対象
  - `internal/site/*_test.go`, `testdata/`, `.golangci.yml`, `docs/README.md`
- 完了条件
  - `go test ./...`がPASS、lint重大違反なし
- 想定コマンド
  ```bash
  go test ./...
  dev_tools/bin/golangci-lint run
  ```
