package site

import (
	"fmt"
	"os"
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
	return &config, nil
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
