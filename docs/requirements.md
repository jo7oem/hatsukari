# hatsukari 要件仕様（実装・テスト照合版）

## 1. 本書の位置づけ
- 本書は、リポジトリ内の全 Go コードを対象にした要求仕様である。
- 記載形式は「仕様（期待）」「現実装（現状）」「実装根拠」「テスト根拠」「差分」に統一する。
- 実装とテストが矛盾する場合は、テストで観測できる挙動を優先して差分として明示する。

## 2. スコープ
---

## 3. システム要件（外形）


## 4. ログ要件（`logging`）

### RQ-101 `slog` ハンドラに統一フォーマットで出力できること
- 仕様（期待）
  - namespace を必ず付与し、ログレベル別 API を提供する。

### RQ-102 エラーログにスタックトレースを付与できること
- 仕様（期待）
  - 既定では Error 以上でスタックトレースを追加する。
- 現実装（現状）
  - `saveCodeTrace` 既定は `slog.LevelError`。
  - `SetSaveCodeTraceLevel` で閾値変更可能。
- 実装根拠
  - `logging/logger.go` の `NewLogger`, `SetSaveCodeTraceLevel`, `log`。
- テスト根拠
  - `TestLogger_Error`（`stacktrace` の存在を検証）。
- 差分
  - `SetSaveCodeTraceLevel` の挙動を直接検証するテストは未整備。

### RQ-103 Context 値をログ属性へ注入できること
- 仕様（期待）
  - 指定した context key をログへ転記できる。
- 現実装（現状）
  - `AddContextKeys(logKey, ctxKey)` で定義し、`log()` で `ctx.Value` を属性化する。
- 実装根拠
  - `logging/logger.go` の `AddContextKeys`, `log`。
- テスト根拠
  - 直接テストなし。
- 差分
  - 期待に対して自動検証が不足。

---

---

## 99. 既存ドキュメントの整理方針
- 正本（仕様）: `docs/requirements.md`
- 概要（入口）: `README.md`
- `docs/reports/` は調査・履歴用途とし、仕様の正本としては扱わない。

最終更新時点の運用ルール:
- 仕様変更時は先に `docs/requirements.md` を更新し、その後で実装・テストを更新する。
- `docs/reports/` は判断の経緯や検証メモのみを残す。

