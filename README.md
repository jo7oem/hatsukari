# hatsukari

## 全体概要
hatsukari は、ディレクトリ構成と YAML 設定に基づいて Markdown/HTML を配信する、Go 製の軽量サイトホスティング実装です。

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
make test
```

## ドキュメント
- 要件仕様（正本）: `docs/requirements.md`
- 調査/履歴レポート: `docs/reports/`
