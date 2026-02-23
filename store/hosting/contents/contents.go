package contents

import (
	"fmt"
	"net/http"
	"os"
	"path"
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
	relPath, ok := requestURLToRelPath(c.Path(), r.URL.Path)
	if !ok || isHiddenOrUnsafeRelPath(relPath) {
		http.NotFound(w, r)
		return
	}

	resolvedPath, ok := c.resolveContentPathByPriority(relPath)
	if !ok {
		http.NotFound(w, r)
		return
	}

	r.URL.Path = resolvedPath
	c.serveFS.ServeHTTP(w, r)
}

func requestURLToRelPath(mountPath, urlPath string) (string, bool) {
	cleanMount := path.Clean("/" + strings.TrimPrefix(mountPath, "/"))
	if cleanMount != "/" && strings.HasSuffix(mountPath, "/") {
		cleanMount += "/"
	}

	cleanURL := path.Clean("/" + strings.TrimPrefix(urlPath, "/"))

	if cleanMount == "/" {
		if cleanURL == "/" {
			return "index", true
		}
		return strings.TrimPrefix(cleanURL, "/"), true
	}

	if cleanURL == strings.TrimSuffix(cleanMount, "/") {
		return "index", true
	}
	if !strings.HasPrefix(cleanURL, cleanMount) {
		return "", false
	}

	rel := strings.TrimPrefix(cleanURL, cleanMount)
	if rel == "" {
		return "index", true
	}
	return rel, true
}

func isHiddenOrUnsafeRelPath(relPath string) bool {
	if relPath == "" || relPath == "." {
		return true
	}
	if relPath == ".." || strings.HasPrefix(relPath, "../") {
		return true
	}
	for seg := range strings.SplitSeq(relPath, "/") {
		if seg == "" {
			continue
		}
		if seg[0] == '.' {
			return true
		}
	}
	return false
}

func (c *Content) resolveContentPathByPriority(relPath string) (string, bool) {
	if path.Ext(relPath) != "" {
		return relPath, true
	}

	candidate, ok := c.resolveNoExtRelPath(relPath)
	if !ok {
		return "", false
	}

	// http.FileServerFS() のディレクトリハンドリング挙動に合わせるため、index.html を URL 末尾の '/' として扱う。
	if strings.HasSuffix(candidate, "index.html") {
		candidate = strings.TrimSuffix(candidate, "index.html")
	}
	return candidate, true
}

func (c *Content) resolveNoExtRelPath(relPath string) (string, bool) {
	base := relPath
	for {
		resolved, isDir, ok := c.openFirstExistingCandidate(base)
		if !ok {
			return "", false
		}
		if !isDir {
			return resolved, true
		}
		base = path.Join(base, "index")
	}
}

func (c *Content) openFirstExistingCandidate(relPath string) (resolved string, isDir bool, ok bool) {
	for _, ext := range []string{"", ".html", ".md"} {
		p := relPath + ext
		f, err := c.root.Open(p)
		if err != nil {
			continue
		}

		if ext == "" {
			stat, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && stat.IsDir() {
				return relPath, true, true
			}
			return relPath, false, true
		}

		_ = f.Close()
		return p, false, true
	}
	return "", false, false
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
