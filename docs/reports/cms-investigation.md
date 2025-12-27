# 私的ブログ向け最小構成CMS 調査レポート

## 目次
1. 要約と要件
2. 技術選定
3. コンテンツモデル（データ構造）
4. Front Matter仕様（必須/任意）
5. ビルドフロー（入力→生成→出力）
6. テンプレート設計（サイト全体 vs カテゴリ別）
7. アセット方針（グローバル/記事別）
8. ディレクトリ構成
9. CLIと運用
10. セキュリティ/品質
11. 拡張余地

## 1. 要約と要件
- 目的: 私的ブログを静的生成で公開する最小構成CMSをGoで構築。
- 入力: Markdown/HTML。記事メタはFront Matter。
- 出力: 静的HTML（`public/`）。フロントは最小（HTML＋軽量CSS）。
- 要件:
  - 記事はMarkdownまたはHTMLで記述可能
  - 静的サイトとして配信（ビルド時生成）
  - フロントエンドは最小限（テンプレート中心）

## 2. 技術選定
- 言語: Go 1.25+（`go.mod`準拠）
- Markdown: `github.com/yuin/goldmark`（meta拡張）
- テンプレート: 標準 `html/template`
- I/O: `os`, `filepath`, `io/fs`
- 日付: `time`（入力は `YYYY-MM-DD`、内部UTC）
- Lint/テスト: `golangci-lint`, `go test`

## 3. コンテンツモデル（データ構造）
- Post
  - Title (string)
  - Date (time.Time)
  - Tags ([]string)
  - Slug (string)
  - Summary (string)
  - Cover (string)
  - BodyHTML (string)
  - SourcePath (string)
  - Draft (bool)
  - TOCHTML（目次HTML）
  - TOCMaxLevel（目次最大レベル、2〜6、既定3）
- Site（必要最小限）
  - BaseURL, Title, Author

## 4. Front Matter仕様（必須/任意）
- 必須
  - `title`: 記事タイトル
  - `date`: 公開日（`YYYY-MM-DD`）
- 任意
  - `tags`: タグ配列
  - `draft`: 下書き（trueで除外）
  - `cover`: カバー画像（相対/絶対/外部URL）
  - `summary`: 記事サマリ（単一言語）。未指定時は本文から自動抽出（先頭200文字、タグ除去）
  - `toc_max_level`: 目次に含める最大見出しレベル（整数 2〜6、既定3）。例: `toc_max_level: 3`

## 5. ビルドフロー（入力→生成→出力）
1. 読み込み
   - `content/**` の `.md`/`.html` を列挙
   - `.md`: Front Matter抽出＋本文をHTML化
   - `.html`: Front Matterのみ抽出（本文はそのまま）
2. 集約
   - タイトル/日付/タグ/スラッグ/サマリ/カバーを構築
3. テンプレート合成（更新）
   - カテゴリ別テンプレートを部分レンダリングして `ContentHTML` を生成
   - サイト全体テンプレートに `ContentHTML` を挿入し最終HTMLへ
   - 記事ページでは本文の見出し（h2〜指定レベル）から目次を自動生成（`TOCHTML`）
4. 出力
   - 記事詳細: `/posts/<slug>/index.html`
   - 一覧: `/index.html`
   - タグ別: `/tags/<tag>/index.html`
   - アセットコピー（後述）

## 6. テンプレート設計（サイト全体 vs カテゴリ別）
- サイト全体テンプレート（レイアウト）: `templates/layout.html.tmpl`
  - 共通のHTML骨格（`<!doctype html>`、`<head>`、ヘッダ/フッタ、CSP/共通CSS）
  - `ContentHTML`（template.HTML）を受け取り挿入のみ行う
- カテゴリ別テンプレート（部分）
  - `templates/index.html.tmpl`（`define "index"`）
  - `templates/post.html.tmpl`（`define "post"`）
  - `templates/tag.html.tmpl`（`define "tag"`）
- 記事テンプレート（`post`）の表示順
  - 「サマリ → 目次 → 本文」の順で出力（サマリ/目次は存在時のみ表示）
- テンプレート関数
  - `dateISO`, `dateJST`, `slug`, `safeHTML`, `coverURL`

## 7. アセット方針（グローバル/記事別）
- グローバルアセット
  - 配置: `assets/` → 出力: `public/assets/`
  - 例: 共通CSS, 画像, フォント
- 記事別アセット（本文と画像の同一ディレクトリ管理）
  - 配置: 記事ファイルと同一ディレクトリ（例: `content/posts/hello-world/hello-world.md`, `content/posts/hello-world/cover.jpg`）
  - 出力: 記事詳細の出力先へ同階層コピー（例: `public/posts/hello-world/cover.jpg`）
  - 記事内参照: 相対パス `./cover.jpg` 推奨（絶対パス `/posts/<slug>/cover.jpg` も可）

## 8. ディレクトリ構成
- `cmd/hatsugen/` … ジェネレータCLI
- `internal/site/` … モデル/ビルド/テンプレート
- `content/` … 記事ソース
- `templates/` … レイアウト/カテゴリ別テンプレート
- `assets/` … 静的アセット
- `public/` … 出力
- `docs/` … ドキュメント

## 9. CLIと運用
- 既定のビルド
  - 入力: `content/`、テンプレート: `templates/`、アセット: `assets/`、出力: `public/`
-（今後）フラグ: `-input`, `-output`, `-templates`, `-assets`, `-baseURL`

## 10. セキュリティ/品質
- セキュリティ
  - `html/template` による自動エスケープ
  - 外部CDN画像利用時はCSPで許可範囲管理
- 品質
  - Lint（`golangci-lint`）、テスト（`go test`）
  - タイムゾーンは内部UTC、表示はローカルフォーマット

## 11. 拡張余地
- ページネーション、検索インデックス、RSS/Atom、sitemap、ドラフト管理、OGP、コードハイライト、画像最適化、ビルドキャッシュ、差分ビルド、ローカルプレビュー
- 目次拡張
  - アンカーリンク付与（見出しへ id 付加、TOC からリンク）
  - 階層的な入れ子リスト（h2/h3/h4 のネスト）
