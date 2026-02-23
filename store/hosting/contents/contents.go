package contents

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Content struct {
	parent        *Content
	path          string
	root          *os.Root
	ContentConfig ContentConfig
	mux           http.ServeMux
	serveFS       http.Handler
}

func OpenContentDir(fs *os.Root, path string) (*Content, error) {
	return openContentDir(fs, path, nil)
}
func openContentDir(fs *os.Root, path string, parent *Content) (*Content, error) {
	root, err := fs.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	c := &Content{
		parent: parent,
		path:   path,
		root:   root,
	}

	if err := c.setup(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Content) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path, " content path:", c.Path())
	c.mux.ServeHTTP(w, r)
}

func (c *Content) customRoutingHandler() http.Handler {
	if c.serveFS == nil {
		c.serveFS = http.FileServerFS(c.root.FS())
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		relPath, err := filepath.Rel(c.Path(), r.URL.Path)
		if relPath == "." {
			relPath = "index"
		} else {
			for p := range strings.SplitSeq(relPath, "/") {
				if p == "" {
					continue
				}
				if p[0] == '.' {
					http.NotFound(w, r)
					return
				}
			}
		}

		if len(relPath) > 1 && relPath[len(relPath)-1] == '/' {
			relPath = relPath + "index"
		}

		if ext := path.Ext(relPath); ext == "" {
			if err != nil || relPath == ".." || strings.HasPrefix(relPath, "../") {
				http.NotFound(w, r)
				return
			}
			for _, ext := range []string{".html", ".md"} {
				p := relPath + ext
				f, err := c.root.Open(p)
				if err == nil {
					relPath = relPath + ext
					_ = f.Close()
					break
				}
			}
		}
		r.URL.Path = relPath
		c.serveFS.ServeHTTP(w, r)
	})
}

func (c *Content) Path() string {
	if c.parent == nil {
		return "/"
	}
	return path.Join(c.parent.Path(), c.path) + "/"
}

func (c *Content) setup() error {
	confFile, err := c.root.Open(".content.yaml")

	if os.IsNotExist(err) {
		confFile, err = c.root.Open(".content.yml")
		if os.IsNotExist(err) {
			return fmt.Errorf("content config file not found %w", err)
		}
	}

	if err != nil {
		return err
	}

	defer func() { _ = confFile.Close() }()

	if err := yaml.NewDecoder(confFile).Decode(&c.ContentConfig); err != nil {
		return fmt.Errorf("parse error in content config: %w", err)
	}

	c.mux.Handle(c.Path(), c.customRoutingHandler())
	if err := c.setupChild(); err != nil {
		return err
	}

	return nil
}

func (c *Content) setupChild() error {
	for _, contentDir := range c.ContentConfig.ContentsDir {
		child, err := openContentDir(c.root, contentDir, c)
		if err != nil {
			return fmt.Errorf("failed to setup content path:%s  error is :%w", path.Join(c.Path(), contentDir), err)
		}

		p := child.Path()
		_ = p
		c.mux.Handle(child.Path(), child)
	}

	return nil
}

type ContentConfig struct {
	RegisterIndexing bool              `yaml:"registerIndexing,omitempty"`
	TemplatesDir     string            `yaml:"templatesDir,omitempty"`
	DefaultLocale    string            `yaml:"defaultLocale,omitempty"`
	IndexTitle       map[string]string `yaml:"indexTitle,omitempty"`
	ContentsDir      []string          `yaml:"contentsDir,omitempty"`
	Variables        map[string]any    `yaml:"variables,omitempty"`
}
