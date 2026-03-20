package site

import (
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/contents"
)

type SiteConfig struct {
	Title          string   `yaml:"title"`
	Timezone       string   `yaml:"timezone,omitempty"`
	Latest         *int     `yaml:"latest,omitempty"`
	TemplatesDir   string   `yaml:"templatesDir"`
	RootContentDir string   `yaml:"rootContentDir"`
	ContentsDir    []string `yaml:"contentsDir,omitempty"`
}

func OpenSiteDir(path string) (*Site, error) {
	return OpenSiteDirWithLogger(path, nil)
}

func OpenSiteDirWithLogger(path string, logger *logging.Logger) (*Site, error) {
	path = filepath.Clean(path)
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}

	conf, err := openSiteConfig(root)
	if err != nil {
		return nil, err
	}

	location, err := resolveTimezone(conf.Timezone)
	if err != nil {
		if logger != nil {
			logger.Error("invalid site timezone", err, slog.String("timezone", strings.TrimSpace(conf.Timezone)))
		}
		return nil, err
	}

	site := &Site{
		fs:       root,
		config:   *conf,
		mux:      http.NewServeMux(),
		logger:   logger,
		location: location,
	}

	if err := site.Setup(); err != nil {
		if logger != nil {
			logger.Error("failed to setup site", err)
		}
		return nil, err
	}

	return site, nil

}

type Site struct {
	config   SiteConfig
	fs       *os.Root
	mux      *http.ServeMux
	vars     map[string]any
	logger   *logging.Logger
	location *time.Location
	root     *contents.Content
	latest   int
}

func (s *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Site) Close() error {
	return s.fs.Close()
}

func (s *Site) Config() SiteConfig {
	return s.config
}

func (s *Site) Variables() map[string]any {
	vars := make(map[string]any, len(s.vars)+1)
	maps.Copy(vars, s.vars)
	if s.root != nil {
		vars["posts"] = s.root.BuildSitePosts(time.Now().In(s.location), s.latest)
	}
	return vars
}

func (s *Site) Setup() error {
	mux := http.NewServeMux()
	root, err := contents.OpenContentDir(s.fs, s.Config().RootContentDir)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to open root content", err)
		}
		return err
	}
	root.SetLogger(s.logger)
	siteLatest := resolveLatestLimit(s.config.Latest)
	root.SetPostsContext(s.location, siteLatest)
	postsContents := root.CollectPostsContents()
	if len(postsContents) > 1 {
		paths := make([]string, 0, len(postsContents))
		for _, content := range postsContents {
			paths = append(paths, content.Path())
		}
		return fmt.Errorf("multiple posts contents found: %s", strings.Join(paths, ", "))
	}

	siteIndexes := buildSiteIndexes(root.CollectIndexSeeds())
	s.vars = map[string]any{
		"indexes": siteIndexes,
	}
	s.root = root
	s.latest = siteLatest
	root.SetSiteVariables(s.vars)

	mux.Handle("/", root)
	s.mux = mux
	return nil
}
