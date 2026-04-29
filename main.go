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
	"strings"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/site"
)

type runtimeConfig struct {
	siteDir string
	addr    string
}

func parseRuntimeConfig(args []string, getenv func(string) string) (runtimeConfig, error) {
	fs := flag.NewFlagSet("hatsukari", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

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

	var conf runtimeConfig
	fs.StringVar(&conf.siteDir, "site", defaultSiteDir, "path to site directory")
	fs.StringVar(&conf.addr, "addr", defaultAddr, "listen address")

	if err := fs.Parse(args); err != nil {
		return runtimeConfig{}, err
	}

	conf.siteDir = strings.TrimSpace(conf.siteDir)
	conf.addr = strings.TrimSpace(conf.addr)
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
	ctx := context.Background()

	shutdownTelemetry, err := logging.InitTelemetry(ctx, logging.TelemetryConfig{
		ServiceName:      "hatsukari",
		ExporterEndpoint: strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		Insecure:         strings.EqualFold(strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE")), "true"),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = shutdownTelemetry(ctx) }()

	logger := logging.NewLogger(slog.NewTextHandler(os.Stdout, nil), "hatsukari")

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
