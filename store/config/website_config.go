package config

import (
	"fmt"
	"io/fs"

	"github.com/goccy/go-yaml"
	"github.com/jo7oem/hatsukari/logger"
)

const (
	ContentTypeRoot      = "root"
	ContentTypeStaticDir = "static_dir"
	ContentTypeContent   = "content"
	ContentTypeBlog      = "blog"
)

// WebsiteConfig はWebサイトのコンテンツに関する設定。
type WebsiteConfig struct {
	// Title はWebサイトのタイトルを表します。
	Title    string         `yaml:"title"`
	SiteRoot ContentMapping `yaml:"siteTemplate"`

	ContentMappings map[string]ContentMapping `yaml:"contentMappings"`

	rootFs fs.FS
}

// LoadWebsiteConfig は指定されたパスから設定を読み込みます。
func LoadWebsiteConfig(root fs.FS, path string) (*WebsiteConfig, error) {
	file, err := root.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	defer func(file fs.File) {
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

	config.rootFs = root

	return &config, nil
}

// ContentMapping はWebサイトのコンテンツとその設定をマッピングするための型.
type ContentMapping struct {
	// Type はコンテンツの種類を表します。
	Type string `yaml:"Type"`

	Directory string `yaml:"Directory"`
	// ContentConfigName はコンテンツの設定ファイル名を表します。
	// これは、コンテンツの設定を定義するYAMLファイルの名前を指定します。
	// ContentTypeの種別によっては使用されないため無視されます.
	// SiteContentsDir からの相対パスで指定されます。
	// 例: "content.yaml"
	ContentConfigName string `yaml:"contentConfigName,omitempty"`
	SubTitle          string `yaml:"subTitle,omitempty"`
}
