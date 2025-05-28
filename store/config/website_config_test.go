package config_test

import (
	"github.com/google/go-cmp/cmp"
	"github.com/jo7oem/hatsukari/store/config"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWebsiteConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tmpConfigPath string
		path          string
		yamlContent   string
		expectErr     bool
		expect        *config.WebsiteConfig
	}{
		{
			name:          "valid config",
			tmpConfigPath: "website.yaml",
			path:          "website.yaml",
			yamlContent: `
title: "My Site"
siteRootDir: "/var/www"
siteRoot:
  themeEntrypoint: "theme.html"
  topPageEntryPoint: "index.html"
  distributionFiles:
    - "robots.txt"
    - "favicon.ico"
contentMappings:
  - urlPath: "/static/"
    reference: "static"
    contentType: "static_dir"
  - urlPath: "/about/"
    reference: "about"
    contentType: "page"
    contentConfigName: "about.yaml"
`,
			expectErr: false,
			expect: &config.WebsiteConfig{
				Title:       "My Site",
				SiteRootDir: "/var/www",
				SiteRoot: config.SiteRoot{
					ThemaEntryPoint:   "theme.html",
					TopPageEntryPoint: "index.html",
					DistributionFiles: []string{"robots.txt", "favicon.ico"},
				},
				ContentMappings: []config.ContentMapping{
					{
						URLPath:           "/static/",
						Reference:         "static",
						ContentType:       "static_dir",
						ContentConfigName: "",
					},
					{
						URLPath:           "/about/",
						Reference:         "about",
						ContentType:       "page",
						ContentConfigName: "about.yaml",
					},
				},
			},
		},
		{
			name:          "file not found",
			tmpConfigPath: "website.yml",
			path:          "website.yaml",
			yamlContent:   "",
			expectErr:     true,
			expect:        nil,
		},
		{
			name:          "invalid yaml",
			tmpConfigPath: "invalid.yaml",
			path:          "invalid.yaml",
			yamlContent: `
title: "My Site"
siteRootDir: "/var/www"
siteRoot:
  themeEntrypoint: "theme.html"
  topPageEntryPoint: "index.html"
  distributionFiles:
    - "robots.txt"
    - "favicon.ico"
contentMappings:
  - urlPath: "/static/"
    reference: [invalid
    contentType: "static_dir"
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

			cfg, err := config.LoadWebsiteConfig(path)
			if tt.expectErr != (err != nil) {
				t.Errorf("LoadWebsiteConfig() error = %v, expectErr %v", err, tt.expectErr)

				return
			}

			if cmp.Diff(cfg, tt.expect) != "" {
				t.Errorf("LoadWebsiteConfig() mismatch (-got +want):\n%s", cmp.Diff(cfg, tt.expect))
			}
		})
	}
}
