# hatsukari 要件仕様（実装・テスト照合版）

## 1. 本書の位置づけ
- 本書は、リポジトリ内の全 Go コードを対象にした要求仕様である。
- 記載形式は「仕様（期待）」「現実装（現状）」「実装根拠」「テスト根拠」「差分」に統一する。
- 実装とテストが矛盾する場合は、テストで観測できる挙動を優先して差分として明示する。

## 2. スコープ
対象コード:
- `main.go`
- `main_runtime_config.go`
- `internal/bootstrap/bootstrap.go`
- `internal/runtimeconfig/config.go`
- `logging/logger.go`
- `telemetry/telemetry.go`
- `telemetry/site_metrics.go`
- `store/hosting/renderer/renderer.go`
- `store/hosting/contents/contents.go`
- `store/hosting/contents/posts/posts.go`
- `store/hosting/contents/tags/definition.go`
- `store/hosting/contents/routing/path.go`
- `store/hosting/contents/rendering/template.go`
- `store/hosting/contents/site_context.go`
- `store/hosting/site/site.go`
- `store/hosting/site/page.go`（現状は空パッケージ）

対象テスト:
- `logging/logger_test.go`
- `store/hosting/renderer/renderer_test.go`
- `store/hosting/contents/contents_test.go`
- `store/hosting/site/site_test.go`

---

## 3. システム要件（外形）

### RQ-001 サイトをローカルディレクトリから HTTP 配信できること
- 仕様（期待）
  - サイト設定と Content ツリーを読み込み、HTTP リクエストに応答する。
- 現実装（現状）
  - `main.go` は `siteDir` と `addr` を実行設定から読み込み、HTTP サーバを起動する。
- 実装根拠
  - `main.go` の `parseRuntimeConfig` と `run`。
  - 起動実体は `internal/bootstrap/bootstrap.go` の `Start`。
  - 実行設定解決の実体は `internal/runtimeconfig/config.go` の `ParseWithIO`。
- テスト根拠
  - `site`/`contents` の `httptest.NewServer(...)` によりハンドラとして配信可能なことを検証。
- 差分
  - なし。

### RQ-003 実行設定を環境変数・設定ファイル・CLI で解決できること
- 仕様（期待）
  - `環境変数 < 設定ファイル < CLI` の優先順位で実行設定を決定する。
  - 設定ファイル形式は YAML とする。
  - 設定ファイル例を標準出力へ出力する機能を提供する。
- 現実装（現状）
  - `-config` / `CONFIG_PATH` で YAML 実行設定を読み込む。
  - `-print-config-example` で実行設定例 YAML を出力して終了する。
  - `siteDir`, `addr`, `telemetry.enabled`, `telemetry.exporterEndpoint`, `telemetry.insecure` を解決する。
- 実装根拠
  - `main_runtime_config.go` の `parseRuntimeConfig`, `runtimeConfigExampleYAML`（互換レイヤ）。
  - `internal/runtimeconfig/config.go` の `ParseWithIO`, `ExampleYAML`, `loadRuntimeConfigFile`（実体）。
- テスト根拠
  - `main_test.go` の `TestRuntimeConfig_Parse`, `TestRuntimeConfig_ExampleYAML`。
- 差分
  - なし。

### RQ-002 サイト設定ファイルを読み込めること
- 仕様（期待）
  - `.site.yaml` と `.site.yml` の両方を受け付ける。
- 現実装（現状）
  - `.site.yaml` を優先し、存在しない場合に `.site.yml` を読む。
- 実装根拠
  - `store/hosting/site/site.go` の `openSiteConfig`。
- テスト根拠
  - `TestSite_OpenDir_Timezone`（`.site.yml` を用いて検証）。
- 差分
  - `.site.yaml` 優先順自体の専用テストは未整備。

---

## 4. ログ要件（`logging`）

### RQ-101 `slog` ハンドラに統一フォーマットで出力できること
- 仕様（期待）
  - namespace を必ず付与し、ログレベル別 API を提供する。
- 現実装（現状）
  - `Debug/Info/Warn/Error` と `Context` 版を提供し、`namespace` 属性を常に付与する。
- 実装根拠
  - `logging/logger.go` の `log`。
- テスト根拠
  - `TestLogger_Info`, `TestLogger_Debug`, `TestLogger_Warn`, `TestLogger_Error`。
- 差分
  - なし。

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

### RQ-104 OpenTelemetry トレースを collector 経由で収集できること
- 仕様（期待）
  - アプリは OTLP 送信先を直接バックエンドへ向けず、`otel-collector` 経由で送信する。
  - collector はトレースを Jaeger と Tempo へ並列転送できる。
  - 開発時の一次確認 UI は Jaeger を優先する。
  - tracer は context 注入で扱い、グローバル tracer へ直接依存しない。
  - exporter 初期化は指数バックオフで再試行し、バックオフ上限は設けない。
  - 再試行回数とバックオフ基準値は `logging` パッケージ内 `var` で管理する。
  - これら設定値の変更は起動時の初期化順序規約で 1 回のみ行う。
  - 開発用途として OTel 送信を明示的に無効化して起動できる。
- 現実装（現状）
  - `telemetry/telemetry.go` で OTLP exporter と tracer provider を初期化し、context 注入 API を提供する。
  - `main.go` は `otelEnabled=false` の場合 `telemetry.Init` を呼ばない。
  - `compose.yaml` は `otel-collector` を経由して Jaeger/Tempo を起動する。
- 実装根拠
  - `telemetry/telemetry.go` の `Init`, `InjectTracer`, `TracerFromContext`, `StartSpan`。
  - `otel/collector-config.yaml` の traces pipeline。
  - `compose.yaml` の `app -> otel-collector` 依存。
- テスト根拠
  - （未整備）
- 差分
  - collector 経由での収集確認は手動確認手順に依存する。

### RQ-105 OpenTelemetry メトリクスを collector 経由で収集できること
- 仕様（期待）
  - `hatsukari_runtime_goroutines` と `hatsukari_runtime_heap_alloc_bytes` を収集できる。
  - `hatsukari_http_requests_total` を `http_method` と `http_status_code` の属性付きで収集できる。
  - collector は Prometheus exporter でメトリクスを公開できる。
  - Grafana は Prometheus datasource 経由で上記メトリクスを可視化できる。
- 現実装（現状）
  - `site.ServeHTTP` でアクセスカウンタを加算し、runtime メトリクスを observable gauge で公開する。
  - `otel/collector-config.yaml` の metrics pipeline が Prometheus exporter（`:9464`）へ出力する。
  - `compose.yaml` で `prometheus` と `grafana` を起動し、Grafana datasource を provision する。
- 実装根拠
  - `store/hosting/site/site.go` の `ServeHTTP` と `telemetry.InitSiteMetrics` 呼び出し。
  - `telemetry/site_metrics.go` の `NewSiteMetrics`, `RecordHTTPRequest`。
  - `telemetry/telemetry.go` の `newMeterProvider`。
  - `otel/prometheus/prometheus.yml`, `otel/grafana/provisioning/datasources/prometheus.yaml`。
- テスト根拠
  - （未整備）
- 差分
  - collector/Prometheus/Grafana を含む E2E 自動検証は未導入。

---

## 5. レンダリング要件（`renderer`）

### RQ-201 Front Matter を抽出できること
- 仕様（期待）
  - Markdown 先頭の Front Matter を抽出し、ページメタとして利用できる。
- 現実装（現状）
  - `goldmark-meta` で抽出し、メタ未定義時は空 map を返す。
- 実装根拠
  - `store/hosting/renderer/renderer.go` の `ExtractMeta`。
- テスト根拠
  - `TestRenderer_Render/ExpandMarkdownMeta`。
- 差分
  - なし。

### RQ-202 Markdown 本文でメタ参照展開できること
- 仕様（期待）
  - 本文を Go テンプレートとして実行し、Front Matter 値を参照できる。
  - 本文テンプレートでは `site` / `contents` / `page.meta` を参照できる。
  - Front Matter とシステム変数のキーが衝突した場合はシステム変数を優先する。
- 現実装（現状）
  - `text/template` (`missingkey=zero`) で事前展開後に Markdown 変換する。
  - 日時は `2006/01/02 15:04:05` に正規化する。
- 実装根拠
  - `renderMarkdownWithMetaTemplate`, `normalizeMetaValue`, `normalizeDateString`。
- テスト根拠
  - `TestRenderer_Render/ExpandMarkdownMeta`。
- 差分
  - なし。

### RQ-203 Content テンプレートを適用できること
- 仕様（期待）
  - `template.<ext>` を適用し、`contents`/`site`/`page` データを参照できる。
- 現実装（現状）
  - `WithTemplateFS` 指定時に `templateName` を実行。
  - 同一ディレクトリ内テンプレートを一括 Parse して部分テンプレートを使用可能。
- 実装根拠
  - `Render`, `renderPageTemplate`。
- テスト根拠
  - `TestRenderer_Render/ApplyTemplateWithPartials`。
- 差分
  - なし。

### RQ-204 指定テンプレート未存在時にフォールバックできること
- 仕様（期待）
  - `template.<ext>` がない場合は変換済み HTML を返す。
- 現実装（現状）
  - `tmpl.Lookup(templateName) == nil` の場合 `fallback` を返す。
- 実装根拠
  - `renderPageTemplate`。
- テスト根拠
  - `TestRenderer_Render/TemplateMissingFallback`。
- 差分
  - なし。

---

## 6. Content 要件（`contents`）

### RQ-301 Content ツリーを再帰構築できること
- 仕様（期待）
  - `.content.yaml` / `.content.yml` と `contentsDir` に基づき子 Content を構築する。
- 現実装（現状）
  - `setupChild` で各 `contentsDir` を `openContentDir` し、mux に mount する。
- 実装根拠
  - `setup`, `setupChild`。
- テスト根拠
  - `TestContent_OpenDir`, `TestContent_ServeHTTP`（nested case）。
- 差分
  - `OpenContentDir` は `logger` に `nil` を許可しない。

### RQ-302 拡張子なし URL の解決優先順位を持つこと
- 仕様（期待）
  - `path -> path.html -> path.md -> path/index.html -> path/index.md` で探索する。
- 現実装（現状）
  - `resolveNoExtRelPath` と `openFirstExistingCandidate` で同順序探索。
- 実装根拠
  - `resolveContentPathByPriority`, `resolveNoExtRelPath`。
  - `store/hosting/contents/routing/path.go` の `RequestURLToRelPath`。
- テスト根拠
  - `TestContent_ServeHTTP`（priority_1..5）。
- 差分
  - なし。

### RQ-303 隠しパス/危険パスを公開しないこと
- 仕様（期待）
  - `.` や `..` を含む相対パスを拒否する。
- 現実装（現状）
  - `isHiddenOrUnsafeRelPath` で 404。
- 実装根拠
  - `customRoutingHandler`。
  - `store/hosting/contents/routing/path.go` の `IsHiddenOrUnsafeRelPath`。
- テスト根拠
  - `TestContent_ServeHTTP` の dotfile/dot dir ケース。
- 差分
  - なし。

### RQ-304 Content 単位でテンプレートを解決できること
- 仕様（期待）
  - `templatesDir` 配下の `contentTemplate` で指定したファイルを使う。
  - `contentTemplate` 未指定時は `template.html` を使う。
  - 描画順序は `render単位展開 -> Content テンプレート -> Site テンプレート` とする。
- 現実装（現状）
  - `resolveTemplate` は `contentTemplate`（未指定時 `template.html`）を解決し、`resolveNamedTemplate` で存在確認する。
  - `renderer.Render` が Content テンプレート適用後、必要に応じて Site テンプレートを適用する。
  - `templatesDir` が絶対パスまたは上位参照ならエラー。
  - `contentTemplate` は `templatesDir` 直下のファイル名のみ許可し、`../`・絶対パス・ネストパスを拒否する。
- 実装根拠
  - `resolveNamedTemplate`, `renderer.Render`。
  - `store/hosting/contents/rendering/template.go` の `ResolveContentTemplateName`。
- テスト根拠
  - `TestContent_ServeHTTP`（template_case）, `TestSite_ServeHTTP_SiteTemplatePipeline`。
- 差分
  - なし。

### RQ-310 Content 単位でサイトテンプレート適用を無効化できること
- 仕様（期待）
  - `disableSiteTemplate: true` の Content はサイトテンプレート適用を行わない。
  - 適用可否は Content ごとに独立し、親子で自動継承しない。
- 現実装（現状）
  - `contentConfig.DisableSiteTemplate` で判定し、true の Content はサイトテンプレート段をスキップする。
  - `tags.md` 経路にも同一判定を適用する。
- 実装根拠
  - `customRoutingHandler`, `renderTagsPage`, `resolveSiteTemplate`。
- テスト根拠
  - `TestSite_ServeHTTP_SiteTemplatePipeline`（`/naked/` と `/posts/tags/`）。
- 差分
  - なし。

### RQ-305 posts Content の記事メタを収集できること
- 仕様（期待）
  - `.md`（`index.md` 除外）から `title/postedAt` を持つ記事を抽出する。
- 現実装（現状）
  - 条件不一致は除外し、`PostEntry` を構築する。
- 実装根拠
  - `collectOwnPosts`, `postEntryByResolvedPath`, `postEntryFromMeta`。
  - `store/hosting/contents/posts/posts.go` の `ParseMetaTime`, `ParseTagKeys`, `ParseRevisions`。
- テスト根拠
  - `TestSite_Posts`, `TestSite_PostTags`。
- 差分
  - なし。

### RQ-306 posts の公開可視性を制御できること
- 仕様（期待）
  - `visibility` と `publishAt` により URL / list / tag の公開可否を制御する。
- 現実装（現状）
  - `IsDirectVisible`, `IsListVisible`, `IsTagVisible` を用途別で適用。
  - 不正 visibility は `private` 扱い。
- 実装根拠
  - `store/hosting/contents/posts/posts.go` の `Entry` 可視判定メソッドと `ParseVisibility`。
  - `customRoutingHandler`。
- テスト根拠
  - `TestSite_Posts`, `TestSite_PostTags`。
- 差分
  - なし。

### RQ-307 tags 定義を読み込み、未知タグをフォールバックできること
- 仕様（期待）
  - `.tag.yaml` を読み込み、既知タグはリンク付き、未知タグは文字列のみとする。
- 現実装（現状）
  - 定義なしは空 map 扱い。
  - 既知タグのみ `URL` を付与、未知タグは `URL` 空。
- 実装根拠
  - `loadTagDefinitions`, `resolvePostTags`, `buildTagMap`。
  - `store/hosting/contents/tags/definition.go` の `NormalizeDefinition`, `BuildLocalizedPublicValue`。
- テスト根拠
  - `TestSite_PostTags`, `TestSite_PostTagsWithoutDefinitionFile`。
- 差分
  - なし。

### RQ-308 tags 一覧/詳細ページを動的生成できること
- 仕様（期待）
  - `/posts/tags/` と `/posts/tags/<key>` を `tags.md` で描画する。
  - `disableSiteTemplate != true` の場合、`tags.md` の描画結果へサイトテンプレート段を適用する。
- 現実装（現状）
  - `tryServeTagsPage` と `renderTagsPage` で処理。
  - 未知タグまたは対象 0 件は 404、テンプレート欠落は 500。
  - `renderTagsPage` でも通常ページと同様にサイトテンプレート判定を行う。
- 実装根拠
  - `tryServeTagsPage`, `renderTagsPage`。
  - `store/hosting/contents/tags/definition.go` の `ParseRoute`。
- テスト根拠
  - `TestSite_PostTags`, `TestSite_PostTagsWithoutDefinitionFile`, `TestSite_ServeHTTP_SiteTemplatePipeline`。
- 差分
  - `tags.md` 欠落時 500 の直接テストは未整備。

### RQ-309 posts の予約パスを保護できること
- 仕様（期待）
  - posts Content 直下の `tags` 系パス衝突を禁止する。
- 現実装（現状）
  - `tags`, `tags.md`, `tags.html` が存在すると setup エラー。
- 実装根拠
  - `validatePostsReservedNames`。
- テスト根拠
  - 直接テストなし。
- 差分
  - 期待に対して自動検証が不足。

---

## 7. Site 要件（`site`）

### RQ-401 timezone をサイト単位で解決できること
- 仕様（期待）
  - 未指定は UTC、空白は除去、不正値は起動エラー。
- 現実装（現状）
  - `resolveTimezone` で実現。
- 実装根拠
  - `OpenSiteDir`, `resolveTimezone`。
- テスト根拠
  - `TestSite_OpenDir_Timezone`。
- 差分
  - `OpenSiteDir` は `logger` に `nil` を許可しない。

### RQ-406 siteTemplatesDir を安全に解決できること
- 仕様（期待）
  - `siteTemplatesDir` はサイトルート基準の相対パスのみ許可する。
  - サイトテンプレートのエントリーポイントは `siteTemplate` で指定できる。
  - `siteTemplate` は `siteTemplatesDir` 直下のファイル名のみ許可し、ネストパスは許可しない。
  - `../` と絶対パスは起動エラーにする。
  - `siteTemplate` 未指定時は `site-template.<ext>`（次点で `site-template.html`）を探索する。
  - `siteTemplate` も `../` と絶対パスを禁止する。
  - 解決したエントリーポイントが未配置の場合はフォールバックとしてサイトテンプレート段をスキップする。
- 現実装（現状）
  - `validateSiteTemplatesDir` で不正パスを拒否する。
  - `validateTemplateEntryPoint` で `siteTemplate` の不正値を拒否する。
  - `resolveSiteTemplateFS` は存在しないディレクトリを非エラー扱いにし、適用段を無効化する。
- 実装根拠
  - `openSiteConfig`, `validateSiteTemplatesDir`, `validateTemplateEntryPoint`, `resolveSiteTemplate`, `Site.resolveSiteTemplateFS`。
- テスト根拠
  - `TestSite_OpenDir_InvalidSiteTemplatesDir`, `TestSite_OpenDir_InvalidSiteTemplateEntryPoint`, `TestSite_ServeHTTP_SiteTemplateMissingFallback`, `TestSite_ServeHTTP_SiteTemplateEntryPointFromConfig`。
- 差分
  - なし。

### RQ-402 `site.indexes` を優先度順に公開できること
- 仕様（期待）
  - `registerIndexing` 対象を `priority` 昇順、同値は読み込み順で公開。
- 現実装（現状）
  - `CollectIndexSeeds` -> `buildSiteIndexes` でソート・整形。
  - 公開 title は `default` と `ja` のみ。
- 実装根拠
  - `Setup`, `buildSiteIndexes`。
- テスト根拠
  - `TestSite_Setup`。
- 差分
  - なし。

### RQ-403 `site.posts` をサイト共通変数として公開できること
- 仕様（期待）
  - `all/latest/tags/byTag` を提供し、`latest` 上限を適用する。
- 現実装（現状）
  - `Variables()` で `root.BuildSitePosts(now, latest)` を毎回計算する。
- 実装根拠
  - `Variables`, `resolveLatestLimit`。
- テスト根拠
  - `TestSite_Posts`, `TestSite_PostTags`。
- 差分
  - なし。

### RQ-404 posts Content をサイト内で単一に制約すること
- 仕様（期待）
  - `contentType: posts` は 1 つのみ許可。
- 現実装（現状）
  - `CollectPostsContents` で検出し、複数なら起動エラー。
- 実装根拠
  - `Setup`。
- テスト根拠
  - `TestSite_OpenDir_MultiplePostsContents`。
- 差分
  - なし。

### RQ-405 リクエスト locale をテンプレートへ公開できること
- 仕様（期待）
  - クエリ `?lang=` を `site.currentLocale` として提供する。
- 現実装（現状）
  - `requestSiteVariables` で空白除去した `lang` を設定。
- 実装根拠
  - `contents.go` の `requestSiteVariables`。
- テスト根拠
  - `TestSite_Setup`（`?lang=en`）。
- 差分
  - なし。

### RQ-407 `site.title` と `site.variables` をテンプレートへ公開できること
- 仕様（期待）
  - `.site.yml` の `title` を `site.title` として公開する。
  - `.site.yml` の `variables`（小文字）を `site.variables` として公開する。
  - `variables` 未指定時は `site.variables` を空 map として公開する。
  - `variables` は `site.variables` 配下に隔離し、`site` 直下キーを上書きしない。
- 現実装（現状）
  - `Setup` で `title` / `variables` / `indexes` を `contents.SiteContext` に設定する。
  - `Variables()` は `contents.SiteContext.VariablesMap` を使って返す。
  - `requestSiteVariables` は `SiteContext` から `site` 変数を生成し、`currentLocale` と `posts` を追加する。
- 実装根拠
  - `site.go` の `SiteConfig`, `Setup`, `Variables`。
  - `contents/site_context.go` の `SiteContext`。
  - `contents.go` の `SetSiteContext`, `requestSiteVariables`。
- テスト根拠
  - `TestSite_Setup`, `TestSite_SiteVariables`, `TestRenderer_Render`。
- 差分
  - なし。

---

## 8. 仕様（期待）と現実装（現状）の差分（横断）

### 8.1 高優先（仕様化または実装追加が必要）
- `main.go` が設定パス/ポートを固定し、CLI 入力や環境変数を受けない。
- `README.md` 以外の旧調査資料に、現実装と異なる将来案（静的生成前提など）が混在しやすい。

### 8.2 中優先（テスト不足）
- `.site.yaml` 優先順位。
- `logging.AddContextKeys` / `SetSaveCodeTraceLevel` の挙動。
- `contents.resolveNamedTemplate` の異常系（不正 `templatesDir`）。
- posts 予約パス（`tags`, `tags.md`, `tags.html`）の起動エラー。
- `tags.md` 欠落時の 500 応答。

### 8.3 低優先（改善余地）
- `store/hosting/site/page.go` は空で、責務未定義。
- 要件 ID 単位の自動トレーサビリティ（要件 -> テスト）の CI 検証は未導入。

### 8.4 エラーメッセージ方針
- 公開 API の前提条件違反は、英小文字・句点なしのメッセージで統一する。
- `nil` 禁止引数は `<argument> must not be nil` 形式に統一する。
- 空文字禁止引数は `<argument> must not be empty` 形式に統一する。
- 方針適用済み:
  - `site.OpenSiteDir(path, logger)` は `logger must not be nil`。
  - `contents.OpenContentDir(fs, path, logger)` は `logger must not be nil`。
  - `main.parseRuntimeConfig(args, getenv)` は `siteDir must not be empty` / `addr must not be empty`。

---

## 9. 既存ドキュメントの整理方針
- 正本（仕様）: `docs/requirements.md`
- 概要（入口）: `README.md`
- `docs/reports/` は調査・履歴用途とし、仕様の正本としては扱わない。

最終更新時点の運用ルール:
- 仕様変更時は先に `docs/requirements.md` を更新し、その後で実装・テストを更新する。
- `docs/reports/` は判断の経緯や検証メモのみを残す。

