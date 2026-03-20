package contents

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/renderer"
	"gopkg.in/yaml.v3"
)

const defaultContentPriority = 0xFFFF
const defaultLatestPosts = 10

const (
	ContentTypePosts = "posts"
)

type Visibility string

const (
	VisibilityPublic     Visibility = "public"
	VisibilityUnlisted   Visibility = "unlisted"
	VisibilityDirectOnly Visibility = "directOnly"
	VisibilityPrivate    Visibility = "private"
)

type Revision struct {
	RevisedAt time.Time
	Summary   string
}

type PostEntry struct {
	URL        string
	Title      string
	PostedAt   time.Time
	PublishAt  *time.Time
	Visibility Visibility
	Tags       []string
	Summary    string
	Revisions  []Revision
}

func (p PostEntry) IsDirectVisible(now time.Time) bool {
	if !p.isPublishedAt(now) {
		return false
	}
	switch p.Visibility {
	case VisibilityPublic, VisibilityUnlisted, VisibilityDirectOnly:
		return true
	default:
		return false
	}
}

func (p PostEntry) IsListVisible(now time.Time) bool {
	return p.isPublishedAt(now) && p.Visibility == VisibilityPublic
}

func (p PostEntry) IsTagVisible(now time.Time) bool {
	if !p.isPublishedAt(now) {
		return false
	}
	return p.Visibility == VisibilityPublic || p.Visibility == VisibilityUnlisted
}

func (p PostEntry) LatestRevision() *Revision {
	if len(p.Revisions) == 0 {
		return nil
	}

	latest := p.Revisions[0]
	for i := 1; i < len(p.Revisions); i++ {
		if p.Revisions[i].RevisedAt.After(latest.RevisedAt) {
			latest = p.Revisions[i]
		}
	}

	return &latest
}

func (p PostEntry) isPublishedAt(now time.Time) bool {
	if p.PublishAt == nil {
		return true
	}
	return !p.PublishAt.After(now)
}

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

	siteVariables  map[string]any
	logger         *logging.Logger
	postsVariables map[string]any
	timezone       *time.Location
	siteLatest     int
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
		if c.isPostsContent() && path.Ext(relPath) == ".md" {
			http.NotFound(w, r)
			return
		}

		f, err := c.root.Open(resolvedPath)
		if err != nil {
			c.logError("failed to open markdown file", err, slog.String("path", resolvedPath))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer func() { _ = f.Close() }()
		b, err := io.ReadAll(f)
		if err != nil {
			c.logError("failed to read markdown file", err, slog.String("path", resolvedPath))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if c.isPostsContent() && !isIndexMarkdownPath(resolvedPath) {
			now := c.now()
			entry, ok, entryErr := c.postEntryByResolvedPath(resolvedPath)
			if entryErr != nil {
				c.logError("failed to load post metadata", entryErr, slog.String("path", resolvedPath))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if !ok || !entry.IsDirectVisible(now) {
				http.NotFound(w, r)
				return
			}
		}

		templateFS, templateName, err := c.resolveTemplate(resolvedPath)
		if err != nil {
			c.logError("failed to resolve template", err, slog.String("path", resolvedPath))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		opts := []renderer.Option{
			renderer.WithContentsVariables(c.ContentConfig.Variables),
			renderer.WithContentsPosts(c.postsVariables),
			renderer.WithSiteVariables(c.requestSiteVariables(r)),
			renderer.WithLogger(c.logger),
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

func (c *Content) SetPostsContext(location *time.Location, siteLatest int, now time.Time) {
	if location == nil {
		location = time.UTC
	}
	c.timezone = location
	c.siteLatest = normalizeLatest(siteLatest)
	c.postsVariables = c.buildPostsVariables(now)
	for _, child := range c.children {
		child.SetPostsContext(location, siteLatest, now)
	}
}

func (c *Content) BuildSitePosts(now time.Time, siteLatest int) map[string]any {
	posts := c.collectSubtreePosts(now)
	return map[string]any{
		"all":    posts,
		"latest": limitPosts(posts, normalizeLatest(siteLatest)),
	}
}

func (c *Content) SetLogger(logger *logging.Logger) {
	c.logger = logger
	for _, child := range c.children {
		child.SetLogger(logger)
	}
}

func (c *Content) logError(msg string, err error, attrs ...slog.Attr) {
	if c.logger == nil {
		return
	}
	c.logger.Error(msg, err, attrs...)
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

func (c *Content) buildPostsVariables(now time.Time) map[string]any {
	posts := c.collectSubtreePosts(now)
	return map[string]any{
		"all":    posts,
		"latest": limitPosts(posts, c.effectiveLatest()),
	}
}

func (c *Content) collectSubtreePosts(now time.Time) []PostEntry {
	posts := make([]PostEntry, 0)
	if c.isPostsContent() {
		posts = append(posts, c.collectOwnPosts(now)...)
	}
	for _, child := range c.children {
		posts = append(posts, child.collectSubtreePosts(now)...)
	}

	sort.SliceStable(posts, func(i, j int) bool {
		if !posts[i].PostedAt.Equal(posts[j].PostedAt) {
			return posts[i].PostedAt.After(posts[j].PostedAt)
		}
		return posts[i].URL < posts[j].URL
	})

	return posts
}

func (c *Content) collectOwnPosts(now time.Time) []PostEntry {
	posts := make([]PostEntry, 0)
	err := fs.WalkDir(c.root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != "." && strings.HasPrefix(path.Base(p), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if path.Ext(p) != ".md" || isIndexMarkdownPath(p) {
			return nil
		}

		entry, ok, parseErr := c.postEntryByResolvedPath(p)
		if parseErr != nil {
			c.logError("failed to parse post metadata", parseErr, slog.String("path", p))
			return nil
		}
		if !ok {
			return nil
		}
		if entry.IsListVisible(now) {
			posts = append(posts, entry)
		}
		return nil
	})
	if err != nil {
		c.logError("failed to walk posts", err)
	}

	return posts
}

func (c *Content) postEntryByResolvedPath(resolvedPath string) (PostEntry, bool, error) {
	f, err := c.root.Open(resolvedPath)
	if err != nil {
		return PostEntry{}, false, err
	}
	defer func() { _ = f.Close() }()

	b, err := io.ReadAll(f)
	if err != nil {
		return PostEntry{}, false, err
	}

	metaData, err := renderer.ExtractMeta(b)
	if err != nil {
		return PostEntry{}, false, err
	}
	if len(metaData) == 0 {
		return PostEntry{}, false, nil
	}

	entry, ok, err := c.postEntryFromMeta(metaData, resolvedPath)
	if err != nil {
		return PostEntry{}, false, err
	}
	if !ok {
		return PostEntry{}, false, nil
	}
	return entry, true, nil
}

func (c *Content) postEntryFromMeta(metaData map[string]any, resolvedPath string) (PostEntry, bool, error) {
	title := strings.TrimSpace(fmt.Sprint(metaData["title"]))
	if title == "" || title == "<nil>" {
		return PostEntry{}, false, nil
	}

	postedAt, ok := parseMetaTime(metaData["postedAt"], c.timezone)
	if !ok {
		return PostEntry{}, false, nil
	}

	entry := PostEntry{
		URL:        buildPostURL(c.Path(), resolvedPath),
		Title:      title,
		PostedAt:   postedAt,
		Visibility: parseVisibility(metaData["visibility"]),
		Summary:    strings.TrimSpace(fmt.Sprint(metaData["summary"])),
		Tags:       parseTags(metaData["tags"]),
		Revisions:  parseRevisions(metaData["revisions"], c.timezone),
	}
	if entry.Summary == "<nil>" {
		entry.Summary = ""
	}

	if publishAt, ok := parseMetaTime(metaData["publishAt"], c.timezone); ok {
		entry.PublishAt = &publishAt
	}

	return entry, true, nil
}

func parseMetaTime(value any, location *time.Location) (time.Time, bool) {
	if location == nil {
		location = time.UTC
	}
	if value == nil {
		return time.Time{}, false
	}

	switch v := value.(type) {
	case time.Time:
		return v.In(location), true
	case string:
		parsed, ok := parseTimeString(v, location)
		return parsed, ok
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return time.Time{}, false
		}
		parsed, ok := parseTimeString(s, location)
		return parsed, ok
	}
}

func parseTimeString(raw string, location *time.Location) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.In(location), true
		}
		if t, err := time.ParseInLocation(layout, value, location); err == nil {
			return t.In(location), true
		}
	}
	return time.Time{}, false
}

func parseVisibility(value any) Visibility {
	v := strings.TrimSpace(fmt.Sprint(value))
	if v == "" || v == "<nil>" {
		return VisibilityPublic
	}
	visibility := Visibility(v)
	switch visibility {
	case VisibilityPublic, VisibilityUnlisted, VisibilityDirectOnly, VisibilityPrivate:
		return visibility
	default:
		return VisibilityPrivate
	}
}

func parseTags(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		tags := make([]string, 0, len(v))
		for _, item := range v {
			t := strings.TrimSpace(fmt.Sprint(item))
			if t == "" || t == "<nil>" {
				continue
			}
			tags = append(tags, t)
		}
		return tags
	default:
		return nil
	}
}

func parseRevisions(value any, location *time.Location) []Revision {
	v, ok := value.([]any)
	if !ok {
		return nil
	}
	revisions := make([]Revision, 0, len(v))
	for _, item := range v {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		revisedAt, ok := parseMetaTime(m["revisedAt"], location)
		if !ok {
			continue
		}
		summary := strings.TrimSpace(fmt.Sprint(m["summary"]))
		if summary == "<nil>" {
			summary = ""
		}
		revisions = append(revisions, Revision{RevisedAt: revisedAt, Summary: summary})
	}
	return revisions
}

func buildPostURL(contentPath, resolvedPath string) string {
	base := strings.TrimSuffix(resolvedPath, path.Ext(resolvedPath))
	return normalizeIndexURL(path.Join(contentPath, base))
}

func normalizeLatest(value int) int {
	if value <= 0 {
		return defaultLatestPosts
	}
	return value
}

func limitPosts(posts []PostEntry, max int) []PostEntry {
	if max <= 0 || len(posts) <= max {
		return posts
	}
	return posts[:max]
}

func isIndexMarkdownPath(p string) bool {
	return path.Base(p) == "index.md"
}

func (c *Content) effectiveLatest() int {
	if c.ContentConfig.Latest == nil {
		return normalizeLatest(c.siteLatest)
	}
	return normalizeLatest(*c.ContentConfig.Latest)
}

func (c *Content) isPostsContent() bool {
	return strings.EqualFold(strings.TrimSpace(c.ContentConfig.ContentType), ContentTypePosts)
}

func (c *Content) now() time.Time {
	location := c.timezone
	if location == nil {
		location = time.UTC
	}
	return time.Now().In(location)
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
	ContentType      string            `yaml:"contentType,omitempty"`
	Priority         *int              `yaml:"priority,omitempty"`
	Latest           *int              `yaml:"latest,omitempty"`
	TemplatesDir     string            `yaml:"templatesDir,omitempty"`
	DefaultLocale    string            `yaml:"defaultLocale,omitempty"`
	IndexTitle       map[string]string `yaml:"indexTitle,omitempty"`
	ContentsDir      []string          `yaml:"contentsDir,omitempty"`
	Variables        map[string]any    `yaml:"variables,omitempty"`
}
