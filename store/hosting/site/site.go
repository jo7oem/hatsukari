package site

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

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

func (s *Site) Setup() error {
	mux := http.NewServeMux()
	root, err := contents.OpenContentDir(s.fs, s.Config().RootContentDir)
	if err != nil {
		return err
	}

	mux.Handle("/", root)
	s.mux = mux
	return nil
}
