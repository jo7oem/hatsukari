package config

import (
	"os"
	"path"
)

const (
	EnvServerListenAddr = "SERVER_ADDRESS"
	EnvConfigPath       = "CONFIG_PATH"
	EvnWorkDir          = "WORK_DIR"
)

type Config struct {
	Server ServerConf
}

func NewConfig() *Config {
	return &Config{
		Server: NewServerConf(),
	}
}

func (c *Config) SetFromEnv() error {
	c.Server.SetFromEnv()

	return nil
}

type ServerConf struct {
	Addr     string
	WorkDir  string
	ConfPath string
}

func NewServerConf() ServerConf {
	return ServerConf{
		Addr:     ":8080",
		WorkDir:  "./sample",
		ConfPath: "config.yaml",
	}
}

func (sc *ServerConf) SetFromEnv() {
	if s := os.Getenv(EnvServerListenAddr); s != "" {
		sc.Addr = s
	}

	if s := os.Getenv(EnvConfigPath); s != "" {
		sc.ConfPath = s
	}

	if s := os.Getenv(EvnWorkDir); s != "" {
		sc.WorkDir = path.Clean(s)
	}
}
