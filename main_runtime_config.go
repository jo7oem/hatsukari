package main

import (
	"io"

	"github.com/jo7oem/hatsukari/internal/runtimeconfig"
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

var errCommandHandled = runtimeconfig.ErrCommandHandled

func runtimeConfigExampleYAML() string {
	return runtimeconfig.ExampleYAML()
}

func parseRuntimeConfig(args []string, getenv func(string) string) (runtimeConfig, error) {
	return parseRuntimeConfigWithIO(args, getenv, io.Discard, io.Discard)
}

func parseRuntimeConfigWithIO(args []string, getenv func(string) string, out io.Writer, errOut io.Writer) (runtimeConfig, error) {
	parsed, err := runtimeconfig.ParseWithIO(args, getenv, out, errOut)
	if err != nil {
		return runtimeConfig{}, err
	}
	return runtimeConfig{
		siteDir:            parsed.SiteDir,
		addr:               parsed.Addr,
		configPath:         parsed.ConfigPath,
		printConfigExample: parsed.PrintConfigExample,
		otelEnabled:        parsed.OTELEnabled,
		otelEndpoint:       parsed.OTELEndpoint,
		otelInsecure:       parsed.OTELInsecure,
	}, nil
}
