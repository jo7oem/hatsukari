# `failed to render` デバッグ手順（`?lang=` クエリ）

## 目的
`http://127.0.0.1:8080/?lang=en` で `failed to render` が返るとき、原因を短時間で特定する。

## 手順
1. ベースライン比較
   - `curl -sv 'http://127.0.0.1:8080/'`
   - `curl -sv 'http://127.0.0.1:8080/?lang=en'`
   - ステータス、本文、レスポンスヘッダの差分を確認する。
2. Docker ログ確認
   - `docker compose -f compose.yaml logs --tail=200 app`
   - テンプレート再読込後のビルド失敗、ポート競合、実行時エラーを確認する。
3. ポート競合確認
   - `ss -ltnp` で `:8080` の占有を確認する。
   - コンテナ側が `listen tcp :8080: bind: address already in use` を出している場合、想定外プロセスにリクエストが到達している可能性が高い。
4. テンプレート安全性確認
   - `header.html` で `title.<locale>` を参照するときは、`title.default` へフォールバックする。
   - locale 未指定時も空文字で処理できるようにする。
   - `site.indexes.title` の公開キーは `default` と `ja` のみ（`title.en` は公開しない）ことを前提に確認する。
5. 実装確認
   - `store/hosting/contents/contents.go` で `lang` を `site.currentLocale` としてテンプレートへ注入しているか確認する。

## 今回の対策要点
- リクエスト単位で `site.currentLocale` を注入する。
- ヘッダーの表示名を `title[site.currentLocale] -> title.default` の順で解決する。
- locale 指定有無にかかわらずテンプレートが 500 を返さない構成にする。
- `site.indexes` は公開契約を最小化し、`title.en` は追加しない。


