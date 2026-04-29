package runtimeconfig

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type RuntimeConfig struct {
	SiteDir            string
	Addr               string
	ConfigPath         string
	PrintConfigExample bool
	OTELEnabled        bool
	OTELEndpoint       string
	OTELInsecure       bool
}

type RuntimeConfigFile struct {
	SiteDir   string                  `yaml:"siteDir,omitempty"`
	Addr      string                  `yaml:"addr,omitempty"`
	Telemetry RuntimeTelemetryConfigY `yaml:"telemetry,omitempty"`
}

type RuntimeTelemetryConfigY struct {
	Enabled          *bool  `yaml:"enabled,omitempty"`
	ExporterEndpoint string `yaml:"exporterEndpoint,omitempty"`
	Insecure         *bool  `yaml:"insecure,omitempty"`
}

var ErrCommandHandled = errors.New("command already handled")

func Parse(args []string, getenv func(string) string) (RuntimeConfig, error) {
	return ParseWithIO(args, getenv, io.Discard, io.Discard)
}

func ParseWithIO(args []string, getenv func(string) string, out io.Writer, errOut io.Writer) (RuntimeConfig, error) {
	defaults, err := resolveDefaultConfig(getenv)
	if err != nil {
		return RuntimeConfig{}, err
	}

	configPath, err := extractConfigPath(args, defaults.ConfigPath)
	if err != nil {
		return RuntimeConfig{}, err
	}

	fileConfig, err := loadRuntimeConfigFile(configPath)
	if err != nil {
		return RuntimeConfig{}, err
	}

	resolved := mergeFileConfig(defaults, fileConfig, configPath)
	actionInvoked := false
	cmd := buildCommand(&resolved, out, errOut, &actionInvoked)

	runArgs := append([]string{"hatsukari"}, args...)
	if err := cmd.Run(context.Background(), runArgs); err != nil {
		return RuntimeConfig{}, err
	}
	if !actionInvoked {
		return RuntimeConfig{}, ErrCommandHandled
	}

	return resolved, nil
}

func ExampleYAML() string {
	example := RuntimeConfigFile{
		SiteDir: "./sample",
		Addr:    ":8080",
		Telemetry: RuntimeTelemetryConfigY{
			Enabled:          boolPtr(true),
			ExporterEndpoint: "otel-collector:4317",
			Insecure:         boolPtr(true),
		},
	}
	b, err := yaml.Marshal(example)
	if err != nil {
		return ""
	}
	return string(b)
}

func resolveDefaultConfig(getenv func(string) string) (RuntimeConfig, error) {
	siteDir := strings.TrimSpace(getenv("HATSUKARI_SITE_DIR"))
	if siteDir == "" {
		siteDir = "./sample"
	}

	addr := strings.TrimSpace(getenv("HATSUKARI_ADDR"))
	if addr == "" {
		port := strings.TrimSpace(getenv("PORT"))
		if port != "" {
			if strings.HasPrefix(port, ":") {
				addr = port
			} else {
				addr = ":" + port
			}
		}
	}
	if addr == "" {
		addr = ":8080"
	}

	otelEnabled := true
	if raw := strings.TrimSpace(getenv("HATSUKARI_OTEL_ENABLED")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid HATSUKARI_OTEL_ENABLED: %w", err)
		}
		otelEnabled = parsed
	}

	return RuntimeConfig{
		SiteDir:      siteDir,
		Addr:         addr,
		ConfigPath:   strings.TrimSpace(getenv("CONFIG_PATH")),
		OTELEnabled:  otelEnabled,
		OTELEndpoint: strings.TrimSpace(getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		OTELInsecure: strings.EqualFold(strings.TrimSpace(getenv("OTEL_EXPORTER_OTLP_INSECURE")), "true"),
	}, nil
}

func extractConfigPath(args []string, fallback string) (string, error) {
	configPath := strings.TrimSpace(fallback)
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if arg == "--config" {
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag needs an argument: --config")
			}
			configPath = strings.TrimSpace(args[i+1])
			i++
			continue
		}
		if after, ok := strings.CutPrefix(arg, "--config="); ok {
			configPath = strings.TrimSpace(after)
		}
	}
	return configPath, nil
}

func loadRuntimeConfigFile(path string) (RuntimeConfigFile, error) {
	if strings.TrimSpace(path) == "" {
		return RuntimeConfigFile{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return RuntimeConfigFile{}, err
	}
	var conf RuntimeConfigFile
	if err := yaml.Unmarshal(b, &conf); err != nil {
		return RuntimeConfigFile{}, err
	}
	return conf, nil
}

func mergeFileConfig(base RuntimeConfig, file RuntimeConfigFile, configPath string) RuntimeConfig {
	merged := base
	merged.ConfigPath = strings.TrimSpace(configPath)
	if v := strings.TrimSpace(file.SiteDir); v != "" {
		merged.SiteDir = v
	}
	if v := strings.TrimSpace(file.Addr); v != "" {
		merged.Addr = v
	}
	if file.Telemetry.Enabled != nil {
		merged.OTELEnabled = *file.Telemetry.Enabled
	}
	if v := strings.TrimSpace(file.Telemetry.ExporterEndpoint); v != "" {
		merged.OTELEndpoint = v
	}
	if file.Telemetry.Insecure != nil {
		merged.OTELInsecure = *file.Telemetry.Insecure
	}
	return merged
}

func buildCommand(conf *RuntimeConfig, out io.Writer, errOut io.Writer, actionInvoked *bool) *cli.Command {
	return &cli.Command{
		Name:      "hatsukari",
		Usage:     "Markdown/HTML サイトを配信するローカルサーバ",
		UsageText: "hatsukari [オプション]",
		Description: strings.TrimSpace(`
				指定したサイトディレクトリを読み込み、HTTP で配信します。
				設定値の優先順位は「環境変数 < 設定ファイル < CLI」です。

				例:
				  hatsukari --site ./sample --addr :8080
				  hatsukari --config ./runtime.yaml
				  hatsukari --print-config-example
				`),
		Writer:    out,
		ErrWriter: errOut,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "site", Value: conf.SiteDir, Destination: &conf.SiteDir, Usage: "サイトディレクトリのパス"},
			&cli.StringFlag{Name: "addr", Value: conf.Addr, Destination: &conf.Addr, Usage: "リッスンアドレス"},
			&cli.StringFlag{Name: "config", Value: conf.ConfigPath, Destination: &conf.ConfigPath, Usage: "実行設定 YAML のパス"},
			&cli.BoolFlag{Name: "print-config-example", Destination: &conf.PrintConfigExample, Usage: "設定ファイル例を標準出力に出して終了"},
			&cli.BoolFlag{Name: "otel-enabled", Value: conf.OTELEnabled, Destination: &conf.OTELEnabled, Usage: "OpenTelemetry 送信を有効化する"},
			&cli.StringFlag{Name: "otel-endpoint", Value: conf.OTELEndpoint, Destination: &conf.OTELEndpoint, Usage: "OTLP Exporter エンドポイント"},
			&cli.BoolFlag{Name: "otel-insecure", Value: conf.OTELInsecure, Destination: &conf.OTELInsecure, Usage: "OTLP の insecure 通信を有効化する"},
		},
		Action: func(context.Context, *cli.Command) error {
			*actionInvoked = true
			conf.SiteDir = strings.TrimSpace(conf.SiteDir)
			conf.Addr = strings.TrimSpace(conf.Addr)
			conf.ConfigPath = strings.TrimSpace(conf.ConfigPath)
			conf.OTELEndpoint = strings.TrimSpace(conf.OTELEndpoint)
			if conf.SiteDir == "" {
				return fmt.Errorf("siteDir must not be empty")
			}
			if conf.Addr == "" {
				return fmt.Errorf("addr must not be empty")
			}
			return nil
		},
	}
}

func boolPtr(v bool) *bool {
	return &v
}
