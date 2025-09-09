package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jo7oem/hatsukari/logger"

	"github.com/goccy/go-yaml"
)

// ServerConfig はサーバーの動作設定を表します。
type ServerConfig struct {
	// Address はWebサイト提供のためにHTTPサーバーがListenするアドレスを表します。
	// ex.) "localhost:8080"
	Address string `yaml:"address"`

	// SiteRoot はWebサイトが配置されたディレクトリを表します。
	// これは、Webサイトのルートディレクトリを指定します。
	// サーバーはこのディレクトリ内のコンテンツを提供します。
	// ex.) "./sample"
	SiteRoot string `yaml:"siteRoot"`

	// SiteConfigName は`SiteContentsDir`配下に存在するWebサイトの設定ファイル名を表します。
	// これは、Webサイトの設定を定義するYAMLファイルの名前を指定します。
	// SiteRoot からの相対パスで指定されます。
	// ex.) "site.yaml"
	SiteConfigName string `yaml:"siteConfigName"`

	readConfigPath string
}

func (sc ServerConfig) ConfigDir() string {
	return filepath.Dir(sc.readConfigPath)
}
func (sc ServerConfig) SiteRootFS() (*os.Root, error) {
	scDir := sc.ConfigDir()

	if filepath.IsAbs(sc.SiteRoot) {
		return os.OpenRoot(sc.SiteRoot)
	}

	absPath := filepath.Join(scDir, sc.SiteRoot)
	return os.OpenRoot(absPath)
}

// LoadServerConfig は指定されたパスから設定を読み込みます。
func LoadServerConfig(path string) (*ServerConfig, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	file, err := os.Open(absPath)
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

	config.readConfigPath = absPath

	return &config, nil
}
