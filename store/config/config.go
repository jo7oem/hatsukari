package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type config struct {
	Server   ServerConfig  `yaml:"server"`
	DB       DBConfig      `yaml:"database"`
	Contents ContentConfig `yaml:"contents"`
}

func LoadConfig(path string) (*config, error) {
	absConfPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(absConfPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	conf := defaultConfig()
	if err = yaml.Unmarshal(buf, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

func defaultConfig() config {
	return config{
		Server:   defaultServerConfig(),
		DB:       defaultDBConfig(),
		Contents: defaultContentConfig(),
	}
}

type ServerConfig struct{}

func defaultServerConfig() ServerConfig {
	return ServerConfig{}
}

type DBConfig struct {
	DBType   string `yaml:"dbType"`
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbName"`
	Memo     string `yaml:"memo"`
}

func defaultDBConfig() DBConfig {
	return DBConfig{
		DBType:   "sqlite3",
		Host:     "",
		User:     "",
		Password: "",
		DBName:   "",
		Memo:     "AAA",
	}
}

func (c DBConfig) Args() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", c.Host, c.User, c.Password, c.DBName)
}

type ContentConfig struct {
	Path   string `yaml:"path"`
	Remote struct {
		Address        string `yaml:"address"`
		Branch         string `yaml:"branch"`
		AuthType       string `yaml:"authType"`
		AuthSSHKey     string `yaml:"authSSHKey"`
		AuthSSHKeyPath string `yaml:"authSSHKeyPath"`
	}
}

func defaultContentConfig() ContentConfig {
	return ContentConfig{
		Path: "work/content",
	}
}

func (c ContentConfig) IsPathExist() bool {
	if _, err := os.Stat(c.Path); err != nil {
		return false
	}

	return true
}

func (c ContentConfig) CreatePath() error {
	return os.MkdirAll(c.Path, 0o755)
}
