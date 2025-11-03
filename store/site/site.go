package site

import (
	"os"
	"path/filepath"
)

type Site struct {
	SiteName string
	Contents map[string]Contents
	BlogPost Contents
	root     *os.Root
}

func (s *Site) init() error {
	return nil
}

func NewSite(siteName string, siteRootPath string) (*Site, error) {
	siteRootPath = filepath.Clean(siteRootPath)
	root, err := os.OpenRoot(siteRootPath)
	if err != nil {
		return nil, err
	}

	site := &Site{
		SiteName: siteName,
		Contents: make(map[string]Contents),
		root:     root,
	}

	if err := site.init(); err != nil {
		return nil, err
	}

	return site, nil
}

type Contents interface {
	GetBasePath() string
	IsIndexing() bool
}
