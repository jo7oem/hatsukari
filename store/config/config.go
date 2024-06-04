package config

import (
	"errors"
	"fmt"
	"github.com/go-git/go-billy/v5"
	gogit "github.com/go-git/go-git/v5"
	goGitSSH "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/jo7oem/hatsukari/store/contents"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
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
	fs billy.Filesystem
}

func defaultContentConfig() ContentConfig {
	return ContentConfig{
		Path: "work/content",
	}
}

func (c *ContentConfig) IsPathExist() bool {
	if _, err := os.Stat(c.Path); err != nil {
		return false
	}

	return true
}

func (c *ContentConfig) CreatePath() error {
	return os.MkdirAll(c.Path, 0o755)
}

func (c *ContentConfig) Load() error {
	cloneOpt := &gogit.CloneOptions{
		URL:      c.Remote.Address,
		Progress: os.Stdout,
	}

	switch c.Remote.AuthType {
	case "ssh":
		pubKey, err := goGitSSH.NewPublicKeysFromFile("git", c.Remote.AuthSSHKeyPath, "")
		if err != nil {
			return err
		}

		cloneOpt.Auth = pubKey

	default:
		panic("unsupported auth type")
	}

	// すでにcloneされているか確認する
	r, err := gogit.PlainOpen(c.Path)
	if errors.Is(err, gogit.ErrRepositoryNotExists) {
		r, err = gogit.PlainClone(c.Path, false, cloneOpt)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	if err := w.Pull(&gogit.PullOptions{
		RemoteName: "origin",
	}); err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return err
	}

	c.fs = w.Filesystem

	file, err := w.Filesystem.Open("conf.yml")
	if err != nil {
		return err
	}

	defer file.Close()

	err = contents.Load(file)
	if err != nil {
		return err
	}

	return nil
}
