package site

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jo7oem/hatsukari/store/hosting/contents"
	"gopkg.in/yaml.v3"
)

type SiteConfig struct {
	Title          string   `yaml:"title"`
	TemplatesDir   string   `yaml:"templatesDir"`
	RootContentDir string   `yaml:"rootContentDir"`
	ContentsDir    []string `yaml:"contentsDir,omitempty"`
}

func openSiteConfig(fs *os.Root) (*SiteConfig, error) {
	const confNameYaml = ".site.yaml"
	const confNameYml = ".site.yml"

	f, err := fs.Open(confNameYaml)
	if os.IsNotExist(err) {
		f, err = fs.Open(confNameYml)
	}

	if err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

	var config SiteConfig
	err = yaml.NewDecoder(f).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func OpenSiteDir(path string) (*Site, error) {
	path = filepath.Clean(path)
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}

	conf, err := openSiteConfig(root)
	if err != nil {
		return nil, err
	}

	site := &Site{
		Title:  "My Site",
		fs:     root,
		config: *conf,
		mux:    http.NewServeMux(),
	}

	if err := site.Setup(); err != nil {
		return nil, err
	}

	return site, nil

}

type Site struct {
	Title  string
	config SiteConfig
	fs     *os.Root
	mux    *http.ServeMux
	vars   map[string]any
}

func (s *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path)
	s.mux.ServeHTTP(w, r)
}

func (s *Site) Close() error {
	return s.fs.Close()
}

func (s *Site) Config() SiteConfig {
	return s.config
}

func (s *Site) Variables() map[string]any {
	return s.vars
}

func (s *Site) Setup() error {
	mux := http.NewServeMux()
	root, err := contents.OpenContentDir(s.fs, s.Config().RootContentDir)
	if err != nil {
		return err
	}

	siteIndexes := buildSiteIndexes(root.CollectIndexSeeds())
	s.vars = map[string]any{
		"indexes": siteIndexes,
	}
	root.SetSiteVariables(s.vars)

	mux.Handle("/", root)
	s.mux = mux
	return nil
}

func buildSiteIndexes(seeds []contents.IndexSeed) []map[string]any {
	sort.SliceStable(seeds, func(i, j int) bool {
		if seeds[i].Priority != seeds[j].Priority {
			return seeds[i].Priority < seeds[j].Priority
		}
		return seeds[i].LoadOrder < seeds[j].LoadOrder
	})

	indexes := make([]map[string]any, 0, len(seeds))
	for _, seed := range seeds {
		jaTitle := resolveTitle(seed, "ja")
		if jaTitle == "" {
			jaTitle = seed.URL
		}

		defaultLocale := strings.TrimSpace(seed.DefaultLocale)
		if defaultLocale == "" {
			defaultLocale = "ja"
		}

		defaultTitle := resolveTitle(seed, defaultLocale)
		if defaultTitle == "" {
			defaultTitle = jaTitle
		}
		if defaultTitle == "" {
			defaultTitle = seed.URL
		}

		indexes = append(indexes, map[string]any{
			"url": seed.URL,
			"title": map[string]string{
				"default": defaultTitle,
				"ja":      jaTitle,
			},
		})
	}

	return indexes
}

func resolveTitle(seed contents.IndexSeed, locale string) string {
	if len(seed.IndexTitle) == 0 {
		return ""
	}
	return strings.TrimSpace(seed.IndexTitle[locale])
}
