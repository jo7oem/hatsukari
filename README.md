# hatsukari

## 全体概要
hatsukari は、ディレクトリ構成と YAML 設定に基づいて Markdown/HTML を配信する、Go 製の軽量サイトホスティング実装です。

- `site` がサイト設定とルーティング全体を管理
- `contents` が Content ツリー（子 Content を含む）を解決
- `renderer` が Front Matter 付き Markdown を HTML 化しテンプレート適用
- `logging` が `slog` ベースの薄いラッパーを提供

## 外形的にできること
- Content ツリー配信: `.content.yaml` の `contentsDir` で階層化した配信
- Markdown 配信: Front Matter 抽出、本文のメタ展開、HTML 変換
- テンプレート適用: `templatesDir` 内 `template.<ext>` と部分テンプレート
- ナビ生成: `registerIndexing` から `site.indexes` を生成
- posts 機能: `site.posts` / `contents.posts` の一覧、latest、tags、byTag
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

主要キーの詳細仕様は `docs/requirements.md` を参照してください。

## テスト
```bash
go test ./... -count=1
```

## ドキュメント
- 要件仕様（正本）: `docs/requirements.md`
- 調査/履歴レポート: `docs/reports/`
