package config

// SiteRoot はWebサイトのルートに関する設定を表します.
// `/`直下で提供されるコンテンツの設定を定義します.
// これは、Webサイトの全体の共通テーマの定義、トップページのエントリポイントなどを含みます.
type SiteRoot struct {
	// ThemaEntryPoint はテンプレートファイルを示します.
	// これは、サイト全体の構成を形作るファイルです。
	// `template/html`で解釈できるHTMLテンプレートであることが期待されます。
	ThemaEntryPoint string `yaml:"themeEntrypoint"`

	// TopPageEntryPoint はトップページのHTMLテンプレートファイルを示します.
	// `template/html`で解釈できるHTMLテンプレートであることが期待されます。
	TopPageEntryPoint string `yaml:"topPageEntryPoint"`

	// DistributionFiles は配布ファイルのリストを表します.
	// これはルート直下で提供されるファイルのリストを指定します.
	DistributionFiles []string `yaml:"distributionFiles"`
}
