package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/site"
	"github.com/jo7oem/hatsukari/telemetry"
)

type RuntimeConfig struct {
	SiteDir      string
	Addr         string
	OTELEnabled  bool
	OTELEndpoint string
	OTELInsecure bool
}

type SiteHandler interface {
	http.Handler
	Close() error
}

type Dependencies struct {
	InitTelemetry  func(context.Context, telemetry.Config) (func(context.Context) error, error)
	NewLogger      func(io.Writer) *logging.Logger
	OpenSite       func(string, *logging.Logger) (SiteHandler, error)
	ListenAndServe func(string, http.Handler) error
}

func Start(ctx context.Context, out io.Writer, conf RuntimeConfig, deps Dependencies) error {
	resolvedDeps := resolveDependencies(deps)

	shutdownTelemetry := func(context.Context) error { return nil }
	var err error
	if conf.OTELEnabled {
		shutdownTelemetry, err = resolvedDeps.InitTelemetry(ctx, telemetry.Config{
			ServiceName:      "hatsukari",
			ExporterEndpoint: conf.OTELEndpoint,
			Insecure:         conf.OTELInsecure,
		})
		if err != nil {
			return err
		}
	}
	defer func() { _ = shutdownTelemetry(ctx) }()

	logger := resolvedDeps.NewLogger(out)
	siteHandler, err := resolvedDeps.OpenSite(conf.SiteDir, logger)
	if err != nil {
		logger.Error("failed to open site dir", err)
		return err
	}
	defer func() { _ = siteHandler.Close() }()

	logger.Info("server starting", slog.String("siteDir", conf.SiteDir), slog.String("addr", conf.Addr))
	return resolvedDeps.ListenAndServe(conf.Addr, siteHandler)
}

func resolveDependencies(overrides Dependencies) Dependencies {
	resolved := Dependencies{
		InitTelemetry: telemetry.Init,
		NewLogger: func(out io.Writer) *logging.Logger {
			return logging.NewLogger(slog.NewJSONHandler(out, nil), "hatsukari")
		},
		OpenSite: func(path string, logger *logging.Logger) (SiteHandler, error) {
			return site.OpenSiteDir(path, logger)
		},
		ListenAndServe: http.ListenAndServe,
	}
	if overrides.InitTelemetry != nil {
		resolved.InitTelemetry = overrides.InitTelemetry
	}
	if overrides.NewLogger != nil {
		resolved.NewLogger = overrides.NewLogger
	}
	if overrides.OpenSite != nil {
		resolved.OpenSite = overrides.OpenSite
	}
	if overrides.ListenAndServe != nil {
		resolved.ListenAndServe = overrides.ListenAndServe
	}
	return resolved
}
