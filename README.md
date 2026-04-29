# hatsukari

## 全体概要
hatsukari は、ディレクトリ構成と YAML 設定に基づいて Markdown/HTML を配信する、Go 製の軽量サイトホスティング実装です。

- `site` がサイト設定とルーティング全体を管理
- `contents` が Content ツリー（子 Content を含む）を解決
- `renderer` が Front Matter 付き Markdown を HTML 化しテンプレート適用
- `logging` が `slog` ベースの薄いラッパーを提供

## API 利用時の注意
- `site.OpenSiteDir(path, logger)` は `logger` に `nil` を許可しません。
- `contents.OpenContentDir(fs, path, logger)` も `logger` に `nil` を許可しません。
- 前提条件エラーは `<argument> must not be nil` 形式のメッセージで統一しています。
- 空文字禁止の前提条件エラーは `<argument> must not be empty` 形式で統一しています。
- テストや静音実行では `slog.NewTextHandler(io.Discard, nil)` を使った logger を渡してください。

## 外形的にできること
- Content ツリー配信: `.content.yaml` の `contentsDir` で階層化した配信
- Markdown 配信: Front Matter 抽出、本文のメタ展開、HTML 変換
- テンプレート適用: `templatesDir` 内の `contentTemplate`（未指定時 `template.html`）と部分テンプレート
- ナビ生成: `registerIndexing` から `site.indexes` を生成
- posts 機能: `site.posts` / `contents.posts` の一覧、latest、tags、byTag
- サイト変数公開: `.site.yml` の `title` を `site.title`、`variables` を `site.variables` として公開
- tags ページ: `/posts/tags/` と `/posts/tags/<key>` を `tags.md` で描画
- 可視性制御: `visibility` と `publishAt` による公開判定
- timezone 制御: `.site.yml` の `timezone` を公開判定時刻に適用

## 実行方法
既定では `./sample` を読み込み、`:8080` で HTTP サーバを起動します。

```bash
go run ./main.go
```

CLI と環境変数で起動先を切り替えられます。

```bash
go run ./main.go -site ./sample -addr :8080
```

- 環境変数: `HATSUKARI_SITE_DIR`, `HATSUKARI_ADDR`
- `PORT` も `HATSUKARI_ADDR` 未指定時のフォールバックとして利用可能

## 設定ファイル
- サイト設定: `.site.yaml`（優先）または `.site.yml`
- Content 設定: `.content.yaml`（優先）または `.content.yml`
- tags 設定（posts 専用）: `.tag.yaml`（優先）または `.tag.yml`

サイト全体テンプレートを使う場合は、`.site.yml` で `siteTemplatesDir` と `siteTemplate` を指定します。

```yaml
title: "サンプル"
siteTemplatesDir: "site_templates"
siteTemplate: "site-template.html"
rootContentDir: "./"
variables:
  title: "サンプルタイトル"
```

- `siteTemplate` は `siteTemplatesDir` 直下のファイル名を指定します。
- 未指定時は `site-template.<ext>`（次点で `site-template.html`）を自動探索します。
- `variables` はテンプレートで `site.variables` から参照します（例: `{{index .site.variables "title"}}`）。
- `site.title` は `.site.yml` の `title` を公開します。

Content 側は `.content.yaml` の `contentTemplate` でエントリーポイントを指定できます。

```yaml
templatesDir: templates
contentTemplate: template.md
```

- `contentTemplate` 未指定時は `template.html` を既定で使用します。
- `contentTemplate` は `templatesDir` 直下のファイル名のみ許可します（`../` やネストパスは不可）。

主要キーの詳細仕様は `docs/requirements.md` を参照してください。

## テスト
```bash
go test ./... -count=1
```

## OpenTelemetry 開発観測
- `app` は OTLP を `OTEL_EXPORTER_OTLP_ENDPOINT` に送信します。
- 開発用 `compose` では `otel-collector` 経由で Jaeger と Tempo の両方へ転送します。
- 一次確認は Jaeger UI（`http://127.0.0.1:16686`）を想定しています。

```bash
docker compose -f compose.yaml up -d
docker compose -f compose.yaml logs --tail=200 otel-collector
```

- Jaeger UI: `http://127.0.0.1:16686`
- Grafana UI: `http://127.0.0.1:3000`

Grafana は anonymous login 有効のため、起動直後から Tempo datasource でトレース確認できます。

## ドキュメント
- 要件仕様（正本）: `docs/requirements.md`
- 調査/履歴レポート: `docs/reports/`
