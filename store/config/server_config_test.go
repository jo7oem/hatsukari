package config_test

import (
	"github.com/google/go-cmp/cmp"
	"github.com/jo7oem/hatsukari/store/config"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServerConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tmpConfigPath string
		path          string
		yamlContent   string
		expectErr     bool
		expect        *config.ServerConfig
	}{
		{
			name:          "valid config",
			tmpConfigPath: "server.yaml",
			path:          "server.yaml",
			yamlContent: `
address: "localhost:8080"
siteContentsDir: "./sample"
siteConfigName: "site.yaml"
`,
			expectErr: false,
			expect: &config.ServerConfig{
				Address:        "localhost:8080",
				SiteRoot:       "./sample",
				SiteConfigName: "site.yaml",
			},
		},
		{
			name:          "file not found",
			tmpConfigPath: "not_exist.yml",
			path:          "not_exist.yaml",
			yamlContent:   "./not_exist.yaml",
			expectErr:     true,
			expect:        nil,
		},
		{
			name:          "invalid yaml",
			tmpConfigPath: "invalid.yaml",
			path:          "invalid.yaml",
			yamlContent: `
address: localhost:8080
siteContentsDir: [invalid
siteConfigName: "site.yaml"
`,
			expectErr: true,
			expect:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			path := filepath.Join(tmpDir, "server.yaml")
			if err := os.WriteFile(path, []byte(tt.yamlContent), 0600); err != nil {
				t.Fatalf("failed to write yaml: %v", err)
			}

			cfg, err := config.LoadServerConfig(path)
			if tt.expectErr != (err != nil) {
				t.Errorf("LoadServerConfig() error = %v, expectErr %v", err, tt.expectErr)

				return
			}

			if cmp.Diff(cfg, tt.expect) != "" {
				t.Errorf("LoadServerConfig() mismatch (-got +want):\n%s", cmp.Diff(cfg, tt.expect))
			}
		})
	}
}
