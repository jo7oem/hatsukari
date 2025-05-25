package config

import (
	"fmt"
	"github.com/jo7oem/hatsukari/logger"
	"os"

	"github.com/goccy/go-yaml"
)

// ServerConfig はサーバーの動作設定を表します。
type ServerConfig struct {
	// Address はWebサイト提供のためにHTTPサーバーがListenするアドレスを表します。
	// ex.) "localhost:8080"
	Address string `yaml:"address"`

	// SiteContentsDir はWebサイトが配置されたディレクトリを表します。
	// これは、Webサイトのルートディレクトリを指定します。
	// サーバーはこのディレクトリ内のコンテンツを提供します。
	// ex.) "./sample"
	SiteContentsDir string `yaml:"siteContentsDir"`

	// SiteConfigName は`SiteContentsDir`配下に存在するWebサイトの設定ファイル名を表します。
	// これは、Webサイトの設定を定義するYAMLファイルの名前を指定します。
	// SiteContentsDir からの相対パスで指定されます。
	// ex.) "site.yaml"
	SiteConfigName string `yaml:"siteConfigName"`
}

// LoadServerConfig は指定されたパスから設定を読み込みます。
func LoadServerConfig(path string) (*ServerConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.GetLogger().Error("failed to close config file", "error", err)
		}
	}(file)

	var config ServerConfig

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}
