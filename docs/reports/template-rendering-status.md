# Contents テンプレート展開 実装メモ（履歴）

## 位置づけ
この文書はテンプレート実装導入時のメモです。
現行仕様は `docs/requirements.md` の RQ-201, RQ-202, RQ-203, RQ-204, RQ-304 を参照してください。

## 履歴として残す要点
- Markdown 変換前に Front Matter メタで本文テンプレート展開する方針を採用した。
- `template.<ext>` が見つからない場合に素の HTML へフォールバックする方針を採用した。
- `contents.body` / `page.meta` / `contents.variables` をテンプレート入力の基礎データとした。

