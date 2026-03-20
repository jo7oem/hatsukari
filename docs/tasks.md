# site.indexes 実装タスク

## 関連ファイル
- `store/hosting/contents/contents.go`
- `store/hosting/renderer/renderer.go`
- `store/hosting/site/site.go`
- `store/hosting/renderer/renderer_test.go`
- `store/hosting/site/site_test.go`
- `store/hosting/site/assets_test/indexing_site/.site.yml`
- `store/hosting/site/assets_test/indexing_site/content/.content.yaml`
- `store/hosting/site/assets_test/indexing_site/content/index.md`
- `store/hosting/site/assets_test/indexing_site/content/templates/template.md`
- `store/hosting/site/assets_test/indexing_site/content/a/.content.yaml`
- `store/hosting/site/assets_test/indexing_site/content/a/index.md`
- `store/hosting/site/assets_test/indexing_site/content/b/.content.yaml`
- `store/hosting/site/assets_test/indexing_site/content/b/index.md`
- `store/hosting/site/assets_test/indexing_site/content/c/.content.yaml`
- `store/hosting/site/assets_test/indexing_site/content/c/index.md`
- `README.md`

## 要件
1. `registerIndexing: true` の Content を `site.indexes` で参照できること
2. `site.indexes` はソート済みを保証し、公開要素は `url`, `title.default`, `title.ja` のみであること
3. `url` は目次用として末尾 `/` を付与すること（ルートは `/`）
4. `priority` は負値を許容し、小さい値ほど高優先とすること
5. `priority` 未指定時の既定値は `0xFFFF` であること
6. `priority` 同値時は読み込み順で安定すること
7. `title.default` は `indexTitle[defaultLocale]` を優先し、未設定時は `indexTitle.ja`、さらに未設定なら `url` を使うこと
8. `title.ja` は `indexTitle.ja` を優先し、未設定時は `url` を使うこと
9. `site` 変数はテンプレートで参照できること
10. README に変数仕様を追記すること

## 実装タスク
- [x] `ContentConfig` に `priority` を追加し、既定値 `0xFFFF` を適用
- [x] Content ツリーから index 用シードを収集する API を追加
- [x] Site 初期化時に index を生成・ソートし、`site.indexes` を保持
- [x] renderer のデータモデルに `site` を追加
- [x] `site.indexes` の仕様テストを追加
- [x] README のテンプレート変数仕様を更新
