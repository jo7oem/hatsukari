package main

import (
	"reflect"
	"strings"
	"testing"
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
	}{
		{
			name: "DefaultValues",
			args: nil,
			env:  map[string]string{},
			want: runtimeConfig{siteDir: "./sample", addr: ":8080"},
		},
		{
			name: "EnvValues",
			args: nil,
			env: map[string]string{
				"HATSUKARI_SITE_DIR": "/srv/site",
				"HATSUKARI_ADDR":     ":9090",
			},
			want: runtimeConfig{siteDir: "/srv/site", addr: ":9090"},
		},
		{
			name: "PortFallback",
			args: nil,
			env: map[string]string{
				"PORT": "18080",
			},
			want: runtimeConfig{siteDir: "./sample", addr: ":18080"},
		},
		{
			name: "TrimWhitespaceEnv",
			args: nil,
			env: map[string]string{
				"HATSUKARI_SITE_DIR": "  ./trimmed  ",
				"HATSUKARI_ADDR":     "  :8181  ",
			},
			want: runtimeConfig{siteDir: "./trimmed", addr: ":8181"},
		},
		{
			name: "AddrEnvOverridesPort",
			args: nil,
			env: map[string]string{
				"HATSUKARI_ADDR": ":9090",
				"PORT":           "18080",
			},
			want: runtimeConfig{siteDir: "./sample", addr: ":9090"},
		},
		{
			name: "PortWithColon",
			args: nil,
			env: map[string]string{
				"PORT": ":28080",
			},
			want: runtimeConfig{siteDir: "./sample", addr: ":28080"},
		},
		{
			name: "CLIOverridesEnv",
			args: []string{"-site", "./other", "-addr", ":3000"},
			env: map[string]string{
				"HATSUKARI_SITE_DIR": "/srv/site",
				"HATSUKARI_ADDR":     ":9090",
			},
			want: runtimeConfig{siteDir: "./other", addr: ":3000"},
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

			getenv := func(key string) string {
				return tt.env[key]
			}
			got, err := parseRuntimeConfig(tt.args, getenv)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRuntimeConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("parseRuntimeConfig() error = %v, want contains %q", err, tt.wantErrContains)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseRuntimeConfig() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
