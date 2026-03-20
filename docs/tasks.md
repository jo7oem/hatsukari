# 全面再実装 実装タスク

## 関連ファイル
- `docs/requirements.md`
- `README.md`
- `main.go`
- `store/hosting/site/site.go`
- `store/hosting/contents/contents.go`
- `store/hosting/renderer/renderer.go`
- `store/hosting/site/site_test.go`
- `store/hosting/site/site_sample_compat_test.go`
- `store/hosting/contents/contents_test.go`
- `store/hosting/renderer/renderer_test.go`
- `sample/.site.yml`
- `sample/.content.yaml`
- `sample/posts/.content.yaml`
- `sample/posts/.tag.yaml`
- `sample/posts/templates/template.md`
- `sample/posts/templates/tags.md`

## 要件
1. 正本仕様は `docs/requirements.md` とし、`RQ-001` から `RQ-405` を満たすこと
2. 配信形態は現状どおり動的 HTTP 配信を維持すること
3. `sample` 配下の公開エンドポイントを互換維持すること
4. 互換判定は HTTP ステータスと画面ごとの詳細本文断片一致で行うこと
5. 既存 `store/hosting/` 配置を維持しつつ責務を分離すること
6. 依存方向を一方向（Application -> Domain -> Adapter）に制御すること
7. 公開 API は最小化し、不要な公開シンボルを増やさないこと
8. 性能は現状比の相対比較のみ記録し、機能互換と保守性を優先すること

## 実装タスク
- [x] `sample` の全公開エンドポイント互換テストを追加する
- [x] `site` を Application 層として再編し、設定読み込み・起動・配信責務を分離する
- [x] `contents` を Domain/Adapter に分割し、ルーティングと投稿集計の責務を分離する
- [x] `renderer` を描画専任として再編し、テンプレート解決との境界を明確化する
- [x] 公開 API を棚卸しし、外部公開シンボルを最小化する
- [x] RQ 単位の受け入れテストへ命名と構成を統一する
