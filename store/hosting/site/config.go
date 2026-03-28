package site

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func openSiteConfig(fs *os.Root) (*SiteConfig, error) {
	const confNameYAML = ".site.yaml"
	const confNameYML = ".site.yml"

	f, err := fs.Open(confNameYAML)
	if os.IsNotExist(err) {
		f, err = fs.Open(confNameYML)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var config SiteConfig
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	templatesDir, err := validateSiteTemplatesDir(config.SiteTemplatesDir)
	if err != nil {
		return nil, err
	}
	config.SiteTemplatesDir = templatesDir

	siteTemplate, err := validateTemplateEntryPoint(config.SiteTemplate, "siteTemplate")
	if err != nil {
		return nil, err
	}
	config.SiteTemplate = siteTemplate

	return &config, nil
}

func validateSiteTemplatesDir(raw string) (string, error) {
	templatesDir := strings.TrimSpace(raw)
	if templatesDir == "" {
		return "", nil
	}

	clean := path.Clean(templatesDir)
	if strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid siteTemplatesDir: %s", raw)
	}

	return clean, nil
}

func validateTemplateEntryPoint(raw, key string) (string, error) {
	entryPoint := strings.TrimSpace(raw)
	if entryPoint == "" {
		return "", nil
	}

	clean := path.Clean(entryPoint)
	if clean == "." || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid %s: %s", key, raw)
	}
	if strings.Contains(clean, "/") {
		return "", fmt.Errorf("invalid %s: %s", key, raw)
	}

	return clean, nil
}

func resolveTimezone(raw string) (*time.Location, error) {
	timezone := strings.TrimSpace(raw)
	if timezone == "" {
		return time.UTC, nil
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}

	return location, nil
}

func resolveLatestLimit(v *int) int {
	if v == nil || *v <= 0 {
		return 10
	}
	return *v
}
