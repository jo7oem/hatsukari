package config

import (
	"fmt"
	"github.com/jo7oem/hatsukari/logger"
	"os"

	"github.com/goccy/go-yaml"
)

const (
	ContentTypeStaticDir   = "static_dir"
	ContentTypeStaticFiles = "static_files"
	ContentTypePage        = "page"
)

// WebsiteConfig はWebサイトのコンテンツに関する設定。
type WebsiteConfig struct {
	// Title はWebサイトのタイトルを表します。
	Title string `yaml:"title"`

	ContentMappings []ContentMapping `yaml:"contentMappings"`
}

// LoadWebsiteConfig は指定されたパスから設定を読み込みます。
func LoadWebsiteConfig(path string) (*WebsiteConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			l := logger.GetLogger()
			l.Error("failed to close config file", "error", err)
		}
	}(file)

	var config WebsiteConfig

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

// ContentMapping はWebサイトのコンテンツとその設定をマッピングするための型.
type ContentMapping struct {
	// URLPath はコンテンツのURLパスを表します。
	URLPath string `yaml:"urlPath"`
	// Reference はWebサイトルートからの相対パスで、コンテンツのディレクトリを指す
	Reference string `yaml:"reference"`
	// ContentType はコンテンツの種類を表します。
	ContentType string `yaml:"contentType"`
	// ContentConfigName はコンテンツの設定ファイル名を表します。
	// これは、コンテンツの設定を定義するYAMLファイルの名前を指定します。
	// ContentTypeの種別によっては使用されないため無視されます.
	// SiteContentsDir からの相対パスで指定されます。
	// 例: "content.yaml"
	ContentConfigName string `yaml:"contentConfigName,omitempty"`
}
