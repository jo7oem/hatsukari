package contents

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/jo7oem/hatsukari/store/hosting/renderer"
	"gopkg.in/yaml.v3"
)

const defaultContentPriority = 0xFFFF

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
	serveFS  http.Handler
	children []*Content

	siteVariables map[string]any
}

type IndexSeed struct {
	URL           string
	IndexTitle    map[string]string
	DefaultLocale string
	Priority      int
	LoadOrder     int
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

	if strings.HasSuffix(resolvedPath, ".md") {
		f, err := c.root.Open(resolvedPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer func() { _ = f.Close() }()
		b, err := io.ReadAll(f)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		templateFS, templateName, err := c.resolveTemplate(resolvedPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		opts := []renderer.Option{
			renderer.WithContentsVariables(c.ContentConfig.Variables),
			renderer.WithSiteVariables(c.requestSiteVariables(r)),
		}
		if templateFS != nil && templateName != "" {
			opts = append(opts, renderer.WithTemplateFS(templateFS, templateName))
		}

		render := renderer.NewRenderer(b, opts...)
		render.ServeHTTP(w, r)

		return
	}

	c.serveFS.ServeHTTP(w, r)
}

func (c *Content) resolveTemplate(resolvedPath string) (fs.FS, string, error) {
	ext := path.Ext(resolvedPath)
	if ext == "" {
		return nil, "", nil
	}

	templatesDir := strings.TrimSpace(c.ContentConfig.TemplatesDir)
	if templatesDir == "" {
		return nil, "", nil
	}

	cleanTemplatesDir := path.Clean(templatesDir)
	if strings.HasPrefix(cleanTemplatesDir, "/") || cleanTemplatesDir == ".." || strings.HasPrefix(cleanTemplatesDir, "../") {
		return nil, "", fmt.Errorf("invalid templatesDir: %s", templatesDir)
	}

	templateFS := c.root.FS()
	if cleanTemplatesDir != "." {
		subFS, err := fs.Sub(templateFS, cleanTemplatesDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, "", nil
			}
			return nil, "", err
		}
		templateFS = subFS
	}

	templateName := "template" + ext
	templateFile, err := templateFS.Open(templateName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, "", nil
		}
		return nil, "", err
	}
	defer func() { _ = templateFile.Close() }()

	stat, err := templateFile.Stat()
	if err != nil {
		return nil, "", err
	}
	if stat.IsDir() {
		return nil, "", nil
	}

	return templateFS, templateName, nil
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
	candidate, _ = strings.CutSuffix(candidate, "index.html")
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

		c.children = append(c.children, child)
		c.mux.Handle(child.Path(), child)
	}

	return nil
}

func (c *Content) SetSiteVariables(siteVariables map[string]any) {
	c.siteVariables = siteVariables
	for _, child := range c.children {
		child.SetSiteVariables(siteVariables)
	}
}

func (c *Content) requestSiteVariables(r *http.Request) map[string]any {
	if c.siteVariables == nil {
		return nil
	}

	localized := make(map[string]any, len(c.siteVariables)+1)
	for k, v := range c.siteVariables {
		localized[k] = v
	}

	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	localized["currentLocale"] = lang

	return localized
}

func (c *Content) CollectIndexSeeds() []IndexSeed {
	seeds := make([]IndexSeed, 0)
	loadOrder := 0
	c.collectIndexSeeds(&seeds, &loadOrder)
	return seeds
}

func (c *Content) collectIndexSeeds(seeds *[]IndexSeed, loadOrder *int) {
	if c.ContentConfig.RegisterIndexing {
		seed := IndexSeed{
			URL:           normalizeIndexURL(c.Path()),
			IndexTitle:    copyIndexTitleMap(c.ContentConfig.IndexTitle),
			DefaultLocale: strings.TrimSpace(c.ContentConfig.DefaultLocale),
			Priority:      c.priority(),
			LoadOrder:     *loadOrder,
		}
		*loadOrder = *loadOrder + 1
		*seeds = append(*seeds, seed)
	}

	for _, child := range c.children {
		child.collectIndexSeeds(seeds, loadOrder)
	}
}

func copyIndexTitleMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func normalizeIndexURL(rawURL string) string {
	if rawURL == "/" {
		return rawURL
	}
	return strings.TrimSuffix(rawURL, "/") + "/"
}

func (c *Content) priority() int {
	if c.ContentConfig.Priority == nil {
		return defaultContentPriority
	}
	return *c.ContentConfig.Priority
}

type ContentConfig struct {
	RegisterIndexing bool              `yaml:"registerIndexing,omitempty"`
	Priority         *int              `yaml:"priority,omitempty"`
	TemplatesDir     string            `yaml:"templatesDir,omitempty"`
	DefaultLocale    string            `yaml:"defaultLocale,omitempty"`
	IndexTitle       map[string]string `yaml:"indexTitle,omitempty"`
	ContentsDir      []string          `yaml:"contentsDir,omitempty"`
	Variables        map[string]any    `yaml:"variables,omitempty"`
}
