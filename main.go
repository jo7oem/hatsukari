package main

import (
	"context"
	"flag"
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

type stringFlag struct {
	value string
	set   bool
}

func (f *stringFlag) String() string { return f.value }

func (f *stringFlag) Set(value string) error {
	f.value = value
	f.set = true
	return nil
}

type boolFlag struct {
	value bool
	set   bool
}

func (f *boolFlag) String() string { return strconv.FormatBool(f.value) }

func (f *boolFlag) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	f.value = parsed
	f.set = true
	return nil
}

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
		if arg == "-config" {
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag needs an argument: -config")
			}
			configPath = strings.TrimSpace(args[i+1])
			i++
			continue
		}
		if after, ok := strings.CutPrefix(arg, "-config="); ok {
			configPath = strings.TrimSpace(after)
		}
	}
	return configPath, nil
}

func parseRuntimeConfig(args []string, getenv func(string) string) (runtimeConfig, error) {
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

	fs := flag.NewFlagSet("hatsukari", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var conf runtimeConfig
	fs.StringVar(&conf.siteDir, "site", siteDir, "path to site directory")
	fs.StringVar(&conf.addr, "addr", addr, "listen address")
	fs.StringVar(&conf.configPath, "config", strings.TrimSpace(preConfigPath), "runtime config yaml path")
	fs.BoolVar(&conf.printConfigExample, "print-config-example", false, "print YAML config example and exit")

	otelEnabledFlag := &boolFlag{value: otelEnabled}
	otelEndpointFlag := &stringFlag{value: otelEndpoint}
	otelInsecureFlag := &boolFlag{value: otelInsecure}
	fs.Var(otelEnabledFlag, "otel-enabled", "enable or disable telemetry export")
	fs.Var(otelEndpointFlag, "otel-endpoint", "OTLP exporter endpoint")
	fs.Var(otelInsecureFlag, "otel-insecure", "enable insecure OTLP transport")

	if err := fs.Parse(args); err != nil {
		return runtimeConfig{}, err
	}

	conf.siteDir = strings.TrimSpace(conf.siteDir)
	conf.addr = strings.TrimSpace(conf.addr)
	conf.configPath = strings.TrimSpace(conf.configPath)
	conf.otelEnabled = otelEnabledFlag.value
	conf.otelEndpoint = strings.TrimSpace(otelEndpointFlag.value)
	conf.otelInsecure = otelInsecureFlag.value
	if conf.siteDir == "" {
		return runtimeConfig{}, fmt.Errorf("siteDir must not be empty")
	}
	if conf.addr == "" {
		return runtimeConfig{}, fmt.Errorf("addr must not be empty")
	}

	return conf, nil
}

func main() {
	conf, err := parseRuntimeConfig(os.Args[1:], os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	if conf.printConfigExample {
		_, _ = fmt.Fprint(os.Stdout, runtimeConfigExampleYAML())
		return
	}
	ctx := context.Background()
	shutdownTelemetry := func(context.Context) error { return nil }
	if conf.otelEnabled {
		shutdownTelemetry, err = logging.InitTelemetry(ctx, logging.TelemetryConfig{
			ServiceName:      "hatsukari",
			ExporterEndpoint: conf.otelEndpoint,
			Insecure:         conf.otelInsecure,
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	defer func() { _ = shutdownTelemetry(ctx) }()

	logger := logging.NewLogger(slog.NewJSONHandler(os.Stdout, nil), "hatsukari")

	siteMap, err := site.OpenSiteDir(conf.siteDir, logger)
	if err != nil {
		logger.Error("failed to open site dir", err)
		return
	}
	defer func() { _ = siteMap.Close() }()

	logger.Info("server starting", slog.String("siteDir", conf.siteDir), slog.String("addr", conf.addr))
	err = http.ListenAndServe(conf.addr, siteMap)
	if err != nil {
		log.Fatal(err)
	}
}
