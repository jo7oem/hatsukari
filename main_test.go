package main

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMain_ParseRuntimeConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		args            []string
		env             map[string]string
		want            runtimeConfig
		wantErr         bool
		wantErrContains string
		prepare         func(*testing.T) map[string]string
	}{
		{
			name: "DefaultValues",
			args: nil,
			env:  map[string]string{},
			want: runtimeConfig{siteDir: "./sample", addr: ":8080", otelEnabled: true},
		},
		{
			name: "EnvValues",
			args: nil,
			env: map[string]string{
				"HATSUKARI_SITE_DIR": "/srv/site",
				"HATSUKARI_ADDR":     ":9090",
			},
			want: runtimeConfig{siteDir: "/srv/site", addr: ":9090", otelEnabled: true},
		},
		{
			name: "PortFallback",
			args: nil,
			env: map[string]string{
				"PORT": "18080",
			},
			want: runtimeConfig{siteDir: "./sample", addr: ":18080", otelEnabled: true},
		},
		{
			name: "TrimWhitespaceEnv",
			args: nil,
			env: map[string]string{
				"HATSUKARI_SITE_DIR": "  ./trimmed  ",
				"HATSUKARI_ADDR":     "  :8181  ",
			},
			want: runtimeConfig{siteDir: "./trimmed", addr: ":8181", otelEnabled: true},
		},
		{
			name: "AddrEnvOverridesPort",
			args: nil,
			env: map[string]string{
				"HATSUKARI_ADDR": ":9090",
				"PORT":           "18080",
			},
			want: runtimeConfig{siteDir: "./sample", addr: ":9090", otelEnabled: true},
		},
		{
			name: "PortWithColon",
			args: nil,
			env: map[string]string{
				"PORT": ":28080",
			},
			want: runtimeConfig{siteDir: "./sample", addr: ":28080", otelEnabled: true},
		},
		{
			name: "CLIOverridesEnv",
			args: []string{"-site", "./other", "-addr", ":3000"},
			env: map[string]string{
				"HATSUKARI_SITE_DIR": "/srv/site",
				"HATSUKARI_ADDR":     ":9090",
			},
			want: runtimeConfig{siteDir: "./other", addr: ":3000", otelEnabled: true},
		},
		{
			name: "ConfigFileOverridesEnv",
			args: nil,
			env: map[string]string{
				"HATSUKARI_SITE_DIR": "/env/site",
				"HATSUKARI_ADDR":     ":8181",
				"CONFIG_PATH":        "config.yaml",
			},
			prepare: func(t *testing.T) map[string]string {
				t.Helper()
				dir := t.TempDir()
				configPath := filepath.Join(dir, "config.yaml")
				content := "siteDir: /file/site\naddr: :7070\ntelemetry:\n  enabled: false\n"
				if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
					t.Fatalf("write config file: %v", err)
				}
				return map[string]string{"CONFIG_PATH": configPath}
			},
			want: runtimeConfig{siteDir: "/file/site", addr: ":7070", otelEnabled: false},
		},
		{
			name: "CLIOverridesConfig",
			args: []string{"-config", "dummy", "-site", "/cli/site", "-otel-enabled=false", "-otel-endpoint=cli:4317", "-otel-insecure=true"},
			env:  map[string]string{},
			prepare: func(t *testing.T) map[string]string {
				t.Helper()
				dir := t.TempDir()
				configPath := filepath.Join(dir, "config.yaml")
				content := "siteDir: /file/site\naddr: :7070\ntelemetry:\n  enabled: true\n  exporterEndpoint: file:4317\n  insecure: false\n"
				if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
					t.Fatalf("write config file: %v", err)
				}
				return map[string]string{"CONFIG_PATH": configPath}
			},
			want: runtimeConfig{siteDir: "/cli/site", addr: ":7070", otelEnabled: false, otelEndpoint: "cli:4317", otelInsecure: true},
		},
		{
			name: "OTelConfigFromEnv",
			args: nil,
			env: map[string]string{
				"HATSUKARI_OTEL_ENABLED":      "false",
				"OTEL_EXPORTER_OTLP_ENDPOINT": "collector:4317",
				"OTEL_EXPORTER_OTLP_INSECURE": "true",
				"HATSUKARI_SITE_DIR":          "./sample",
				"HATSUKARI_ADDR":              ":8080",
			},
			want: runtimeConfig{siteDir: "./sample", addr: ":8080", otelEnabled: false, otelEndpoint: "collector:4317", otelInsecure: true},
		},
		{
			name: "PrintConfigExampleFlag",
			args: []string{"-print-config-example"},
			env:  map[string]string{},
			want: runtimeConfig{siteDir: "./sample", addr: ":8080", printConfigExample: true, otelEnabled: true},
		},
		{
			name:            "InvalidOTelEnabledEnv",
			args:            nil,
			env:             map[string]string{"HATSUKARI_OTEL_ENABLED": "oops"},
			wantErr:         true,
			wantErrContains: "invalid HATSUKARI_OTEL_ENABLED",
		},
		{
			name:            "ConfigFileNotFound",
			args:            []string{"-config", "./does-not-exist.yaml"},
			env:             map[string]string{},
			wantErr:         true,
			wantErrContains: "no such file",
		},
		{
			name:            "EmptySite",
			args:            []string{"-site", " "},
			env:             map[string]string{},
			wantErr:         true,
			wantErrContains: "siteDir must not be empty",
		},
		{
			name:            "EmptyAddr",
			args:            []string{"-addr", " "},
			env:             map[string]string{},
			wantErr:         true,
			wantErrContains: "addr must not be empty",
		},
		{
			name:            "UnknownFlag",
			args:            []string{"-unknown"},
			env:             map[string]string{},
			wantErr:         true,
			wantErrContains: "flag provided but not defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := append([]string(nil), tt.args...)
			env := make(map[string]string, len(tt.env))
			maps.Copy(env, tt.env)
			if tt.prepare != nil {
				maps.Copy(env, tt.prepare(t))
				if len(args) >= 2 && args[0] == "-config" && args[1] == "dummy" {
					args[1] = env["CONFIG_PATH"]
				}
			}

			getenv := func(key string) string {
				return env[key]
			}
			got, err := parseRuntimeConfig(args, getenv)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRuntimeConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("parseRuntimeConfig() error = %v, want contains %q", err, tt.wantErrContains)
				}
				return
			}
			got.configPath = ""
			want := tt.want
			want.configPath = ""
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("parseRuntimeConfig() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMain_RuntimeConfigExampleYAML(t *testing.T) {
	t.Parallel()

	raw := runtimeConfigExampleYAML()
	if strings.TrimSpace(raw) == "" {
		t.Fatal("runtimeConfigExampleYAML() returned empty")
	}

	var conf runtimeConfigFile
	if err := yaml.Unmarshal([]byte(raw), &conf); err != nil {
		t.Fatalf("failed to parse example yaml: %v", err)
	}
	if conf.SiteDir == "" {
		t.Fatal("siteDir is empty in example")
	}
	if conf.Addr == "" {
		t.Fatal("addr is empty in example")
	}
	if conf.Telemetry.Enabled == nil || !*conf.Telemetry.Enabled {
		t.Fatal("telemetry.enabled should be true in example")
	}
}
