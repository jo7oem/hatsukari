package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/site"
	"github.com/jo7oem/hatsukari/telemetry"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type runtimeConfig struct {
	siteDir            string
	addr               string
	configPath         string
	printConfigExample bool
	otelEnabled        bool
	otelEndpoint       string
	otelInsecure       bool
}

type runtimeConfigFile struct {
	SiteDir   string                    `yaml:"siteDir,omitempty"`
	Addr      string                    `yaml:"addr,omitempty"`
	Telemetry runtimeTelemetryConfigYML `yaml:"telemetry,omitempty"`
}

type runtimeTelemetryConfigYML struct {
	Enabled          *bool  `yaml:"enabled,omitempty"`
	ExporterEndpoint string `yaml:"exporterEndpoint,omitempty"`
	Insecure         *bool  `yaml:"insecure,omitempty"`
}

var errCommandHandled = errors.New("command already handled")

func loadRuntimeConfigFile(path string) (runtimeConfigFile, error) {
	if strings.TrimSpace(path) == "" {
		return runtimeConfigFile{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return runtimeConfigFile{}, err
	}
	var conf runtimeConfigFile
	if err := yaml.Unmarshal(b, &conf); err != nil {
		return runtimeConfigFile{}, err
	}
	return conf, nil
}

func runtimeConfigExampleYAML() string {
	example := runtimeConfigFile{
		SiteDir: "./sample",
		Addr:    ":8080",
		Telemetry: runtimeTelemetryConfigYML{
			Enabled:          new(true),
			ExporterEndpoint: "otel-collector:4317",
			Insecure:         new(true),
		},
	}
	b, err := yaml.Marshal(example)
	if err != nil {
		return ""
	}
	return string(b)
}

func extractConfigPathFromArgs(args []string, fallback string) (string, error) {
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

func parseRuntimeConfig(args []string, getenv func(string) string) (runtimeConfig, error) {
	return parseRuntimeConfigWithIO(args, getenv, io.Discard, io.Discard)
}

func parseRuntimeConfigWithIO(args []string, getenv func(string) string, out io.Writer, errOut io.Writer) (runtimeConfig, error) {
	defaultSiteDir := strings.TrimSpace(getenv("HATSUKARI_SITE_DIR"))
	if defaultSiteDir == "" {
		defaultSiteDir = "./sample"
	}

	defaultAddr := strings.TrimSpace(getenv("HATSUKARI_ADDR"))
	if defaultAddr == "" {
		port := strings.TrimSpace(getenv("PORT"))
		if port != "" {
			if strings.HasPrefix(port, ":") {
				defaultAddr = port
			} else {
				defaultAddr = ":" + port
			}
		}
	}
	if defaultAddr == "" {
		defaultAddr = ":8080"
	}
	defaultConfigPath := strings.TrimSpace(getenv("CONFIG_PATH"))
	defaultOTelEndpoint := strings.TrimSpace(getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	defaultOTelInsecure := strings.EqualFold(strings.TrimSpace(getenv("OTEL_EXPORTER_OTLP_INSECURE")), "true")
	defaultOTelEnabled := true
	if raw := strings.TrimSpace(getenv("HATSUKARI_OTEL_ENABLED")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return runtimeConfig{}, fmt.Errorf("invalid HATSUKARI_OTEL_ENABLED: %w", err)
		}
		defaultOTelEnabled = parsed
	}

	preConfigPath, err := extractConfigPathFromArgs(args, defaultConfigPath)
	if err != nil {
		return runtimeConfig{}, err
	}

	fileConf, err := loadRuntimeConfigFile(preConfigPath)
	if err != nil {
		return runtimeConfig{}, err
	}

	siteDir := defaultSiteDir
	if v := strings.TrimSpace(fileConf.SiteDir); v != "" {
		siteDir = v
	}
	addr := defaultAddr
	if v := strings.TrimSpace(fileConf.Addr); v != "" {
		addr = v
	}
	otelEnabled := defaultOTelEnabled
	if fileConf.Telemetry.Enabled != nil {
		otelEnabled = *fileConf.Telemetry.Enabled
	}
	otelEndpoint := defaultOTelEndpoint
	if v := strings.TrimSpace(fileConf.Telemetry.ExporterEndpoint); v != "" {
		otelEndpoint = v
	}
	otelInsecure := defaultOTelInsecure
	if fileConf.Telemetry.Insecure != nil {
		otelInsecure = *fileConf.Telemetry.Insecure
	}

	conf := runtimeConfig{
		siteDir:      siteDir,
		addr:         addr,
		configPath:   strings.TrimSpace(preConfigPath),
		otelEnabled:  otelEnabled,
		otelEndpoint: otelEndpoint,
		otelInsecure: otelInsecure,
	}
	actionInvoked := false
	cmd := &cli.Command{
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
			&cli.StringFlag{Name: "site", Value: siteDir, Destination: &conf.siteDir, Usage: "サイトディレクトリのパス"},
			&cli.StringFlag{Name: "addr", Value: addr, Destination: &conf.addr, Usage: "リッスンアドレス"},
			&cli.StringFlag{Name: "config", Value: strings.TrimSpace(preConfigPath), Destination: &conf.configPath, Usage: "実行設定 YAML のパス"},
			&cli.BoolFlag{Name: "print-config-example", Destination: &conf.printConfigExample, Usage: "設定ファイル例を標準出力に出して終了"},
			&cli.BoolFlag{Name: "otel-enabled", Value: otelEnabled, Destination: &conf.otelEnabled, Usage: "OpenTelemetry 送信を有効化する"},
			&cli.StringFlag{Name: "otel-endpoint", Value: otelEndpoint, Destination: &conf.otelEndpoint, Usage: "OTLP Exporter エンドポイント"},
			&cli.BoolFlag{Name: "otel-insecure", Value: otelInsecure, Destination: &conf.otelInsecure, Usage: "OTLP の insecure 通信を有効化する"},
		},
		Action: func(context.Context, *cli.Command) error {
			actionInvoked = true
			conf.siteDir = strings.TrimSpace(conf.siteDir)
			conf.addr = strings.TrimSpace(conf.addr)
			conf.configPath = strings.TrimSpace(conf.configPath)
			conf.otelEndpoint = strings.TrimSpace(conf.otelEndpoint)
			if conf.siteDir == "" {
				return fmt.Errorf("siteDir must not be empty")
			}
			if conf.addr == "" {
				return fmt.Errorf("addr must not be empty")
			}
			return nil
		},
	}

	runArgs := append([]string{"hatsukari"}, args...)
	if err := cmd.Run(context.Background(), runArgs); err != nil {
		return runtimeConfig{}, err
	}
	if !actionInvoked {
		return runtimeConfig{}, errCommandHandled
	}

	return conf, nil
}

func run(args []string, getenv func(string) string, out io.Writer, errOut io.Writer) error {
	conf, err := parseRuntimeConfigWithIO(args, getenv, out, errOut)
	if err != nil {
		if errors.Is(err, errCommandHandled) {
			return nil
		}
		return err
	}
	if conf.printConfigExample {
		_, _ = fmt.Fprint(out, runtimeConfigExampleYAML())
		return nil
	}
	ctx := context.Background()
	shutdownTelemetry := func(context.Context) error { return nil }
	if conf.otelEnabled {
		shutdownTelemetry, err = telemetry.Init(ctx, telemetry.Config{
			ServiceName:      "hatsukari",
			ExporterEndpoint: conf.otelEndpoint,
			Insecure:         conf.otelInsecure,
		})
		if err != nil {
			return err
		}
	}
	defer func() { _ = shutdownTelemetry(ctx) }()

	logger := logging.NewLogger(slog.NewJSONHandler(out, nil), "hatsukari")

	siteMap, err := site.OpenSiteDir(conf.siteDir, logger)
	if err != nil {
		logger.Error("failed to open site dir", err)
		return err
	}
	defer func() { _ = siteMap.Close() }()

	logger.Info("server starting", slog.String("siteDir", conf.siteDir), slog.String("addr", conf.addr))
	if err := http.ListenAndServe(conf.addr, siteMap); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}
