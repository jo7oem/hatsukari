# 可読性・保守性向上リファクタ 実装タスク

## 関連ファイル
- `docs/requirements.md`
- `main.go`
- `main_runtime_config.go`
- `main_test.go`
- `internal/bootstrap/bootstrap.go`
- `internal/bootstrap/bootstrap_test.go`
- `internal/runtimeconfig/config.go`
- `store/hosting/contents/contents.go`
- `store/hosting/contents/routing.go`
- `store/hosting/contents/indexing.go`
- `store/hosting/site/site.go`
- `store/hosting/site/site_test.go`

## 方針
1. 仕様基準は `docs/requirements.md` と現行テスト挙動を同格とする
2. 命名は実装語彙に準拠し、機能単位で揃える
3. 責務集中を避けるためサブパッケージを導入する
4. 機能完了ごとに統合し、統合ごとにテストを通す

## 実装タスク
- [x] `main.go` の実行設定解決を `internal/runtimeconfig` サブパッケージへ分離
- [x] 既存 CLI オプション挙動を維持する互換レイヤ (`main_runtime_config.go`) を追加
- [x] `main` 起動処理を `bootstrap` 責務へ分離し、依存注入可能な構造に整理
- [x] `store/hosting/contents` を責務別サブパッケージへ再配置（routing/rendering/posts/tags）
- [x] `site` と `contents` 間の `map[string]any` 境界を縮小する型導入
- [x] 命名統一後のテスト名・ヘルパ名を実装語彙へ揃える
- [x] 要件仕様へ再配置後の実装根拠を追記
