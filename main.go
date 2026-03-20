package main

import (
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
		return runtimeConfig{}, fmt.Errorf("site directory is required")
	}
	if conf.addr == "" {
		return runtimeConfig{}, fmt.Errorf("listen address is required")
	}

	return conf, nil
}

func main() {
	conf, err := parseRuntimeConfig(os.Args[1:], os.Getenv)
	if err != nil {
		log.Fatal(err)
	}

	logger := logging.NewLogger(slog.NewTextHandler(os.Stdout, nil), "hatsukari")

	siteMap, err := site.OpenSiteDirWithLogger(conf.siteDir, logger)
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
