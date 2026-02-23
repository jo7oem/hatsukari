package contents

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Content struct {
	// 親コンテンツへの参照。ルートコンテンツの場合は nil になる
	parent *Content
	// 親コンテンツからの相対パス。ルートコンテンツの場合は空文字になる
	relPath string
	path    func() string
	// このコンテンツのルートディレクトリへの参照
	root          *os.Root
	ContentConfig ContentConfig

	// このコンテンツとその子コンテンツのルーティングを管理する ServeMux
	mux http.ServeMux
	// ルーティング先がない場合にはファイルサーバとしてハンドリングする
	serveFS http.Handler
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
		parent:  parent,
		relPath: path,
		root:    root,
		serveFS: http.FileServerFS(root.FS()),
	}

	c.path = sync.OnceValue(c.calcPath)

	if err := c.setup(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Content) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

// customRoutingHandler は、URL パスをコンテンツのルートからの相対パスに変換し、適切なファイルを提供するためのハンドラー。
// '/' 終わりであれば index.{html,md} を探し、拡張子がない場合は .html と .md を順に探索する。
// "/posts" へのアクセスは以下の順序で探索する
// 1. /posts
// 2. /posts.html
// 3. /posts.md
// 4. /posts/index.html
// 5. /posts/index.md
// また、セキュリティ上の理由から、相対パスが '.' で始まっている場合は 404 Not Found を返す。
func (c *Content) customRoutingHandler(w http.ResponseWriter, r *http.Request) {
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

	if ext := path.Ext(relPath); ext == "" {
		if err != nil || relPath == ".." || strings.HasPrefix(relPath, "../") {
			http.NotFound(w, r)
			return
		}
	search:
		for _, ext := range []string{"", ".html", ".md"} {
			p := relPath + ext
			f, err := c.root.Open(p)
			switch {
			case err == nil && ext == "":
				if stat, err := f.Stat(); err == nil && stat.IsDir() {
					relPath = relPath + "/index"
					_ = f.Close()
					goto search
				}
				_ = f.Close()
				break

			case err == nil:
				relPath = relPath + ext
				if strings.HasSuffix(relPath, "index.html") {
					relPath = strings.TrimSuffix(relPath, "index.html")
				}
				_ = f.Close()
				break
			default:
				continue
			}
		}
	}
	r.URL.Path = relPath
	c.serveFS.ServeHTTP(w, r)
}

func (c *Content) Path() string {
	return c.path()
}

func (c *Content) calcPath() string {
	if c.parent == nil {
		return "/"
	}
	return path.Join(c.parent.Path(), c.relPath) + "/"
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

	c.mux.HandleFunc(c.Path(), c.customRoutingHandler)
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
