# サイト全体テンプレート適用 実装タスク

## 関連ファイル
- `docs/requirements.md`
- `store/hosting/site/config.go`
- `store/hosting/site/site.go`
- `store/hosting/contents/contents.go`
- `store/hosting/renderer/renderer.go`
- `store/hosting/renderer/template_render.go`
- `store/hosting/site/site_test.go`
- `store/hosting/renderer/renderer_test.go`

## 要件
1. 適用順序は `render単位展開 -> contents単位テンプレート適用 -> サイト全体テンプレート適用` とする
2. サイト全体テンプレート名は `site-template.<ext>` とする
3. `disableSiteTemplate: true` を content 単位で独立適用する
4. `siteTemplatesDir` はサイトルート基準で解決し、`../` と絶対パスは起動エラーとする
5. サイトテンプレート未配置時は自動フォールバックとしてサイト段をスキップする
6. `tags.md` にもサイト全体テンプレートを適用する
7. サイトテンプレートのエントリーポイントを `.site.yml` の `siteTemplate` で定義できる
8. Content テンプレートのエントリーポイントを `.content.yaml` の `contentTemplate` で定義できる
9. `contentTemplate` 未指定時は `template.html` を既定値として使用する

## 実装タスク
- [x] `siteTemplatesDir` バリデーションを `site` 設定読み込みへ追加
- [x] `contentConfig` へ `disableSiteTemplate` を追加
- [x] `renderer` へ content/site の2段テンプレート適用を追加
- [x] 通常ページでサイトテンプレート段を適用
- [x] `tags.md` 経路でもサイトテンプレート段を適用
- [x] サイトテンプレート未配置フォールバックを実装
- [x] 主要テスト（順序、オプトアウト、不正パス、未配置、tags）を追加
- [x] `siteTemplate` 設定でエントリーポイントを切り替え可能にする
- [x] `contentTemplate` 設定で Content テンプレートのエントリーポイントを切り替え可能にする
- [x] `contentTemplate` 未指定時の既定値を `template.html` にする
