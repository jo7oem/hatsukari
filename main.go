package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/site"
	"github.com/jo7oem/hatsukari/telemetry"
)

func run(args []string, getenv func(string) string, out io.Writer, errOut io.Writer) error {
	conf, err := parseRuntimeConfigWithIO(args, getenv, out, errOut)
	if err != nil {
		if err == errCommandHandled {
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
