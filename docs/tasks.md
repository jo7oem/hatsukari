# タグ一覧生成 実装タスク

## 関連ファイル
- `store/hosting/contents/contents.go`
- `store/hosting/site/site.go`
- `store/hosting/renderer/renderer.go`
- `store/hosting/contents/contents_test.go`
- `store/hosting/site/site_test.go`
- `README.md`
- `docs/reports/posts-and-indexes-spec.md`
- `sample/posts/.content.yaml`
- `sample/posts/sample-post.md`
- `sample/posts/templates/template.md`
- `sample/posts/templates/tags.md`
- `sample/posts/.tag.yaml`

## 要件
1. `contentType: posts` はサイト全体で 1 つだけ許可し、複数あれば起動エラーにすること
2. posts Content 直下の `.tag.yaml` をタグ定義として読み込むこと
3. `.tag.yaml` が存在しない場合は「タグ定義なし」として扱うこと
4. 記事 `tags` は `.tag.yaml` の参照キーを使うこと
5. 未知タグはタグ名をそのまま返し、リンクを持たせないこと
6. 未知タグの詳細 URL へは `404` を返すこと
7. `tags` は posts の予約名として扱うこと
8. タグ一覧/タグ詳細は都度生成し、`publishAt` と `visibility` を毎回評価すること
9. タグ一覧テンプレート名は `tags.md` に固定すること
10. タグ公開データは `label.default`, `label.ja`, `about.default`, `about.ja` を持つこと
11. `site.posts.tags` / `site.posts.byTag` と `contents.posts.tags` / `contents.posts.byTag` を公開すること
12. タグ詳細で表示対象記事が 0 件、またはタグが未知のときは `404` を返すこと

## 実装タスク
- [x] posts Content の一意制約を site 起動時に検証する
- [x] `.tag.yaml` ローダーとタグ定義モデルを追加する
- [x] 記事タグを解決して未知タグを非リンクで返すモデルへ拡張する
- [x] `site.posts` / `contents.posts` に `tags` と `byTag` を追加する
- [x] タグ一覧/タグ詳細ルーティングを追加し `tags.md` で描画する
- [x] `publishAt` 問題を解消するため posts/tag データを都度生成する
- [x] サンプルの posts 設定・タグ定義・テンプレートを更新する
- [x] site / contents テストを追加する
- [x] README / reports を更新する
