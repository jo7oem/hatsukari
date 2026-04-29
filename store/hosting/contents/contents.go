package contents

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jo7oem/hatsukari/logging"
	contentsposts "github.com/jo7oem/hatsukari/store/hosting/contents/posts"
	contentsrendering "github.com/jo7oem/hatsukari/store/hosting/contents/rendering"
	contentstags "github.com/jo7oem/hatsukari/store/hosting/contents/tags"
	"github.com/jo7oem/hatsukari/store/hosting/renderer"
	"github.com/jo7oem/hatsukari/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

const defaultContentPriority = 0xFFFF
const defaultLatestPosts = 10

const (
	ContentTypePosts = "posts"
)

type Visibility = contentsposts.Visibility

const (
	VisibilityPublic     = contentsposts.VisibilityPublic
	VisibilityUnlisted   = contentsposts.VisibilityUnlisted
	VisibilityDirectOnly = contentsposts.VisibilityDirectOnly
	VisibilityPrivate    = contentsposts.VisibilityPrivate
)

type Revision = contentsposts.Revision
type TagDefinition = contentstags.Definition
type PostTag = contentsposts.Tag
type TagFeed = contentsposts.TagFeed
type PostEntry = contentsposts.Entry

type Content struct {
	// 親コンテンツへの参照。ルートコンテンツの場合は nil になる
	parent *Content
	// 親コンテンツからの相対パス。ルートコンテンツの場合は空文字になる
	relPath  string
	path     func() string
	rootPath *Content
	// このコンテンツのルートディレクトリへの参照
	root   *os.Root
	config contentConfig

	// このコンテンツとその子コンテンツのルーティングを管理する ServeMux
	mux http.ServeMux
	// ルーティング先がない場合にはファイルサーバとしてハンドリングする
	serveFS  http.Handler
	children []*Content

	siteContext    SiteContext
	siteTemplateFS fs.FS
	siteTemplate   string
	logger         *logging.Logger
	timezone       *time.Location
	siteLatest     int
	tagDefinitions map[string]TagDefinition
}

func OpenContentDir(fs *os.Root, path string, logger *logging.Logger) (*Content, error) {
	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}
	return openContentDir(fs, path, nil, logger)
}
func openContentDir(fs *os.Root, path string, parent *Content, logger *logging.Logger) (*Content, error) {
	root, err := fs.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	c := &Content{
		parent:  parent,
		relPath: path,
		root:    root,
		serveFS: http.FileServerFS(root.FS()),
		logger:  logger,
	}
	if parent == nil {
		c.rootPath = c
	} else {
		c.rootPath = parent.rootPath
	}

	c.path = sync.OnceValue(c.calcPath)

	if err := c.setup(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Content) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "contents.ServeHTTP",
		trace.WithAttributes(
			attribute.String("contents.path", c.Path()),
			attribute.String("http.path", r.URL.Path),
		),
	)
	defer span.End()

	c.mux.ServeHTTP(w, r.WithContext(ctx))
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
	ctx, span := telemetry.StartSpan(r.Context(), "contents.customRoutingHandler",
		trace.WithAttributes(
			attribute.String("contents.path", c.Path()),
			attribute.String("http.path", r.URL.Path),
		),
	)
	defer span.End()
	r = r.WithContext(ctx)

	relPath, ok := requestURLToRelPath(c.Path(), r.URL.Path)
	if !ok || isHiddenOrUnsafeRelPath(relPath) {
		http.NotFound(w, r)
		return
	}

	if c.isPostsContent() {
		handled, err := c.tryServeTagsPage(w, r, relPath)
		if err != nil {
			c.logError("failed to serve tags page", err, slog.String("path", relPath))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if handled {
			return
		}
	}

	_, resolvePathSpan := telemetry.StartSpan(r.Context(), "contents.resolveContentPath",
		trace.WithAttributes(attribute.String("contents.rel_path", relPath)),
	)
	resolvedPath, ok := c.resolveContentPathByPriority(relPath)
	resolvePathSpan.End()
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

		_, readMarkdownSpan := telemetry.StartSpan(r.Context(), "contents.readMarkdown",
			trace.WithAttributes(attribute.String("contents.resolved_path", resolvedPath)),
		)
		f, err := c.root.Open(resolvedPath)
		if err != nil {
			readMarkdownSpan.RecordError(err)
			readMarkdownSpan.End()
			span.RecordError(err)
			c.logError("failed to open markdown file", err, slog.String("path", resolvedPath))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer func() { _ = f.Close() }()
		b, err := io.ReadAll(f)
		readMarkdownSpan.End()
		if err != nil {
			span.RecordError(err)
			c.logError("failed to read markdown file", err, slog.String("path", resolvedPath))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if c.isPostsContent() && !isIndexMarkdownPath(resolvedPath) {
			_, postMetaSpan := telemetry.StartSpan(r.Context(), "contents.resolvePostMeta",
				trace.WithAttributes(attribute.String("contents.resolved_path", resolvedPath)),
			)
			now := c.now()
			entry, ok, entryErr := c.postEntryByResolvedPath(resolvedPath)
			postMetaSpan.End()
			if entryErr != nil {
				span.RecordError(entryErr)
				c.logError("failed to load post metadata", entryErr, slog.String("path", resolvedPath))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if !ok || !entry.IsDirectVisible(now) {
				http.NotFound(w, r)
				return
			}
		}

		_, templateResolveSpan := telemetry.StartSpan(r.Context(), "contents.resolveTemplates",
			trace.WithAttributes(attribute.String("contents.resolved_path", resolvedPath)),
		)
		templateFS, templateName, err := c.resolveTemplate(resolvedPath)
		if err != nil {
			templateResolveSpan.RecordError(err)
			templateResolveSpan.End()
			span.RecordError(err)
			c.logError("failed to resolve template", err, slog.String("path", resolvedPath))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		siteTemplateFS, siteTemplateName, err := c.resolveSiteTemplate(resolvedPath)
		templateResolveSpan.End()
		if err != nil {
			span.RecordError(err)
			c.logError("failed to resolve site template", err, slog.String("path", resolvedPath))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		opts := []renderer.Option{
			renderer.WithContentsVariables(c.config.Variables),
			renderer.WithContentsPosts(c.requestContentsPosts()),
			renderer.WithSiteVariables(c.requestSiteVariables(r)),
			renderer.WithLogger(c.logger),
		}
		if templateFS != nil && templateName != "" {
			opts = append(opts, renderer.WithTemplateFS(templateFS, templateName))
		}
		if siteTemplateFS != nil && siteTemplateName != "" {
			opts = append(opts, renderer.WithSiteTemplateFS(siteTemplateFS, siteTemplateName))
		}

		render := renderer.NewRenderer(b, opts...)
		render.ServeHTTP(w, r)

		return
	}

	c.serveFS.ServeHTTP(w, r)
}

func (c *Content) tryServeTagsPage(w http.ResponseWriter, r *http.Request, relPath string) (bool, error) {
	handled, tagKey, invalid := contentstags.ParseRoute(relPath)
	if !handled {
		return false, nil
	}
	if invalid {
		http.NotFound(w, r)
		return true, nil
	}
	return true, c.renderTagsPage(w, r, tagKey)
}

func (c *Content) renderTagsPage(w http.ResponseWriter, r *http.Request, tagKey string) error {
	ctx, span := telemetry.StartSpan(r.Context(), "contents.renderTagsPage",
		trace.WithAttributes(
			attribute.String("contents.path", c.Path()),
			attribute.String("contents.tag_key", tagKey),
		),
	)
	defer span.End()
	r = r.WithContext(ctx)

	postsData := c.requestContentsPosts()
	byTag, _ := postsData["byTag"].(map[string]TagFeed)
	if tagKey != "" {
		feed, ok := byTag[tagKey]
		if !ok || len(feed.Posts) == 0 {
			http.NotFound(w, r)
			return nil
		}
		postsData["currentTag"] = feed
	} else {
		postsData["currentTag"] = TagFeed{}
	}

	templateFS, templateName, err := c.resolveNamedTemplate("tags.md")
	if err != nil {
		span.RecordError(err)
		return err
	}
	if templateFS == nil || templateName == "" {
		err = fmt.Errorf("tags template not found")
		span.RecordError(err)
		return err
	}

	siteTemplateFS, siteTemplateName, err := c.resolveSiteTemplate("tags.md")
	if err != nil {
		span.RecordError(err)
		return err
	}

	opts := []renderer.Option{
		renderer.WithTemplateFS(templateFS, templateName),
		renderer.WithContentsVariables(c.config.Variables),
		renderer.WithContentsPosts(postsData),
		renderer.WithSiteVariables(c.requestSiteVariables(r)),
		renderer.WithLogger(c.logger),
	}
	if siteTemplateFS != nil && siteTemplateName != "" {
		opts = append(opts, renderer.WithSiteTemplateFS(siteTemplateFS, siteTemplateName))
	}

	render := renderer.NewRenderer(
		[]byte(""),
		opts...,
	)
	render.ServeHTTP(w, r)
	return nil
}

func (c *Content) resolveSiteTemplate(resolvedPath string) (fs.FS, string, error) {
	if c.config.DisableSiteTemplate {
		return nil, "", nil
	}
	if c.siteTemplateFS == nil {
		return nil, "", nil
	}

	candidates := contentsrendering.ResolveSiteTemplateCandidates(c.siteTemplate, resolvedPath)
	if len(candidates) == 0 {
		return nil, "", nil
	}

	for _, templateName := range candidates {
		f, err := c.siteTemplateFS.Open(templateName)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, "", err
		}

		stat, err := f.Stat()
		_ = f.Close()
		if err != nil {
			return nil, "", err
		}
		if stat.IsDir() {
			continue
		}

		return c.siteTemplateFS, templateName, nil
	}

	return nil, "", nil
}

func (c *Content) resolveTemplate(resolvedPath string) (fs.FS, string, error) {
	templateName := contentsrendering.ResolveContentTemplateName(c.config.ContentTemplate)
	return c.resolveNamedTemplate(templateName)
}

func (c *Content) resolveNamedTemplate(templateName string) (fs.FS, string, error) {
	if strings.TrimSpace(templateName) == "" {
		return nil, "", nil
	}

	templatesDir := strings.TrimSpace(c.config.TemplatesDir)
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

	if err := yaml.NewDecoder(confFile).Decode(&c.config); err != nil {
		return fmt.Errorf("parse error in content config: %w", err)
	}

	contentTemplate, err := validateTemplateEntryPoint(c.config.ContentTemplate, "contentTemplate")
	if err != nil {
		return err
	}
	c.config.ContentTemplate = contentTemplate

	if c.isPostsContent() {
		definitions, err := c.loadTagDefinitions()
		if err != nil {
			return err
		}
		c.tagDefinitions = definitions
		if err := c.validatePostsReservedNames(); err != nil {
			return err
		}
	}

	c.mux.HandleFunc(c.Path(), c.customRoutingHandler)
	if err := c.setupChild(); err != nil {
		return err
	}

	return nil
}

func (c *Content) setupChild() error {
	for _, contentDir := range c.config.ContentsDir {
		child, err := openContentDir(c.root, contentDir, c, c.logger)
		if err != nil {
			return fmt.Errorf("failed to setup content path:%s  error is :%w", path.Join(c.Path(), contentDir), err)
		}

		c.children = append(c.children, child)
		c.mux.Handle(child.Path(), child)
	}

	return nil
}

func (c *Content) loadTagDefinitions() (map[string]TagDefinition, error) {
	const tagFileYAML = ".tag.yaml"
	const tagFileYML = ".tag.yml"

	f, err := c.root.Open(tagFileYAML)
	if os.IsNotExist(err) {
		f, err = c.root.Open(tagFileYML)
	}
	if os.IsNotExist(err) {
		return map[string]TagDefinition{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open tag definitions: %w", err)
	}
	defer func() { _ = f.Close() }()

	definitions := make(map[string]TagDefinition)
	if err := yaml.NewDecoder(f).Decode(&definitions); err != nil {
		return nil, fmt.Errorf("parse error in tag definitions: %w", err)
	}

	for key, definition := range definitions {
		if key == "" {
			return nil, fmt.Errorf("tag key must not be empty")
		}
		if key == "tags" {
			return nil, fmt.Errorf("reserved tag key: %s", key)
		}
		definitions[key] = contentstags.NormalizeDefinition(definition)
	}

	return definitions, nil
}

func (c *Content) validatePostsReservedNames() error {
	for _, reserved := range []string{"tags", "tags.md", "tags.html"} {
		f, err := c.root.Open(reserved)
		if err != nil {
			continue
		}
		_ = f.Close()
		return fmt.Errorf("reserved posts path exists: %s", reserved)
	}
	return nil
}

func (c *Content) SetSiteContext(siteContext SiteContext) {
	c.siteContext = siteContext
	for _, child := range c.children {
		child.SetSiteContext(siteContext)
	}
}

func (c *Content) SetSiteVariables(siteVariables map[string]any) {
	siteContext := SiteContext{}
	if title, ok := siteVariables["title"].(string); ok {
		siteContext.Title = title
	}
	if variables, ok := siteVariables["variables"].(map[string]any); ok {
		siteContext.Variables = variables
	}
	if indexes, ok := siteVariables["indexes"].([]map[string]any); ok {
		siteContext.Indexes = indexes
	}
	c.SetSiteContext(siteContext)
}

func (c *Content) SetSiteTemplateFS(siteTemplateFS fs.FS) {
	c.siteTemplateFS = siteTemplateFS
	for _, child := range c.children {
		child.SetSiteTemplateFS(siteTemplateFS)
	}
}

func (c *Content) SetSiteTemplateEntryPoint(siteTemplate string) {
	c.siteTemplate = strings.TrimSpace(siteTemplate)
	for _, child := range c.children {
		child.SetSiteTemplateEntryPoint(siteTemplate)
	}
}

func (c *Content) SetPostsContext(location *time.Location, siteLatest int) {
	if location == nil {
		location = time.UTC
	}
	c.timezone = location
	c.siteLatest = contentsposts.NormalizeLatest(siteLatest, defaultLatestPosts)
	for _, child := range c.children {
		child.SetPostsContext(location, siteLatest)
	}
}

func (c *Content) BuildSitePosts(now time.Time, siteLatest int) map[string]any {
	listPosts := c.collectSubtreePosts(now, PostEntry.IsListVisible)
	tagPosts := c.collectSubtreePosts(now, PostEntry.IsTagVisible)
	return map[string]any{
		"all":    listPosts,
		"latest": contentsposts.Limit(listPosts, contentsposts.NormalizeLatest(siteLatest, defaultLatestPosts)),
		"tags":   contentsposts.BuildTagList(tagPosts),
		"byTag":  contentsposts.BuildTagMap(tagPosts),
	}
}

func (c *Content) logError(msg string, err error, attrs ...slog.Attr) {
	if c.logger == nil {
		return
	}
	c.logger.Error(msg, err, attrs...)
}

func (c *Content) requestSiteVariables(r *http.Request) map[string]any {
	if c.siteContext.Title == "" && c.siteContext.Variables == nil && c.siteContext.Indexes == nil {
		return nil
	}

	var posts map[string]any
	if c.rootPath != nil {
		posts = c.rootPath.BuildSitePosts(c.now(), c.siteLatest)
	}
	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	return c.siteContext.VariablesMapWithLocale(posts, lang)
}

func (c *Content) requestContentsPosts() map[string]any {
	return c.buildPostsVariables(c.now())
}

func (c *Content) buildPostsVariables(now time.Time) map[string]any {
	listPosts := c.collectSubtreePosts(now, PostEntry.IsListVisible)
	tagPosts := c.collectSubtreePosts(now, PostEntry.IsTagVisible)
	return map[string]any{
		"all":    listPosts,
		"latest": contentsposts.Limit(listPosts, c.effectiveLatest()),
		"tags":   contentsposts.BuildTagList(tagPosts),
		"byTag":  contentsposts.BuildTagMap(tagPosts),
	}
}

func (c *Content) collectSubtreePosts(now time.Time, allow func(PostEntry, time.Time) bool) []PostEntry {
	posts := make([]PostEntry, 0)
	if c.isPostsContent() {
		posts = append(posts, c.collectOwnPosts(now, allow)...)
	}
	for _, child := range c.children {
		posts = append(posts, child.collectSubtreePosts(now, allow)...)
	}

	sort.SliceStable(posts, func(i, j int) bool {
		if !posts[i].PostedAt.Equal(posts[j].PostedAt) {
			return posts[i].PostedAt.After(posts[j].PostedAt)
		}
		return posts[i].URL < posts[j].URL
	})

	return posts
}

func (c *Content) collectOwnPosts(now time.Time, allow func(PostEntry, time.Time) bool) []PostEntry {
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
		if allow(entry, now) {
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

	postedAt, ok := contentsposts.ParseMetaTime(metaData["postedAt"], c.timezone)
	if !ok {
		return PostEntry{}, false, nil
	}

	entry := PostEntry{
		URL:        buildPostURL(c.Path(), resolvedPath),
		Title:      title,
		PostedAt:   postedAt,
		Visibility: contentsposts.ParseVisibility(metaData["visibility"]),
		Summary:    strings.TrimSpace(fmt.Sprint(metaData["summary"])),
		Tags:       c.resolvePostTags(contentsposts.ParseTagKeys(metaData["tags"])),
		Revisions:  contentsposts.ParseRevisions(metaData["revisions"], c.timezone),
	}
	if entry.Summary == "<nil>" {
		entry.Summary = ""
	}

	if publishAt, ok := contentsposts.ParseMetaTime(metaData["publishAt"], c.timezone); ok {
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

func parseTagKeys(value any) []string {
	switch v := value.(type) {
	case []string:
		return uniqueTags(v)
	case []any:
		tags := make([]string, 0, len(v))
		for _, item := range v {
			t := strings.TrimSpace(fmt.Sprint(item))
			if t == "" || t == "<nil>" {
				continue
			}
			tags = append(tags, t)
		}
		return uniqueTags(tags)
	default:
		return nil
	}
}

func uniqueTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	unique := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}
	return unique
}

func (c *Content) resolvePostTags(tagKeys []string) []PostTag {
	resolved := make([]PostTag, 0, len(tagKeys))
	for _, key := range tagKeys {
		definition, ok := c.tagDefinitions[key]
		if !ok {
			resolved = append(resolved, PostTag{
				Key:   key,
				Label: map[string]string{"default": key, "ja": key},
				About: map[string]string{"default": "", "ja": ""},
			})
			continue
		}
		resolved = append(resolved, PostTag{
			Key:   key,
			URL:   normalizeIndexURL(path.Join(c.Path(), "tags", key)),
			Label: contentstags.BuildLocalizedPublicValue(definition.DefaultLang, definition.Label, key),
			About: contentstags.BuildLocalizedPublicValue(definition.DefaultLang, definition.About, ""),
		})
	}
	return resolved
}

func buildTagList(posts []PostEntry) []TagFeed {
	feeds := buildTagFeeds(posts)
	list := make([]TagFeed, 0, len(feeds))
	for _, feed := range feeds {
		list = append(list, feed)
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return list[i].Key < list[j].Key
	})
	return list
}

func buildTagMap(posts []PostEntry) map[string]TagFeed {
	feeds := buildTagFeeds(posts)
	byTag := make(map[string]TagFeed, len(feeds))
	for key, feed := range feeds {
		if feed.URL == "" {
			continue
		}
		byTag[key] = feed
	}
	return byTag
}

func buildTagFeeds(posts []PostEntry) map[string]TagFeed {
	feeds := make(map[string]TagFeed)
	for _, post := range posts {
		for _, tag := range post.Tags {
			feed, ok := feeds[tag.Key]
			if !ok {
				feed = TagFeed{Key: tag.Key, URL: tag.URL, Label: copyStringMap(tag.Label), About: copyStringMap(tag.About)}
			}
			feed.Count++
			feed.Posts = append(feed.Posts, post)
			feeds[tag.Key] = feed
		}
	}
	return feeds
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
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
	if c.config.Latest == nil {
		return contentsposts.NormalizeLatest(c.siteLatest, defaultLatestPosts)
	}
	return contentsposts.NormalizeLatest(*c.config.Latest, defaultLatestPosts)
}

func (c *Content) isPostsContent() bool {
	return strings.EqualFold(strings.TrimSpace(c.config.ContentType), ContentTypePosts)
}

func (c *Content) now() time.Time {
	location := c.timezone
	if location == nil {
		location = time.UTC
	}
	return time.Now().In(location)
}

type contentConfig struct {
	RegisterIndexing    bool              `yaml:"registerIndexing,omitempty"`
	ContentType         string            `yaml:"contentType,omitempty"`
	ContentTemplate     string            `yaml:"contentTemplate,omitempty"`
	DisableSiteTemplate bool              `yaml:"disableSiteTemplate,omitempty"`
	Priority            *int              `yaml:"priority,omitempty"`
	Latest              *int              `yaml:"latest,omitempty"`
	TemplatesDir        string            `yaml:"templatesDir,omitempty"`
	DefaultLocale       string            `yaml:"defaultLocale,omitempty"`
	IndexTitle          map[string]string `yaml:"indexTitle,omitempty"`
	ContentsDir         []string          `yaml:"contentsDir,omitempty"`
	Variables           map[string]any    `yaml:"variables,omitempty"`
}

func validateTemplateEntryPoint(raw, key string) (string, error) {
	entryPoint := strings.TrimSpace(raw)
	if entryPoint == "" {
		return "", nil
	}

	clean := path.Clean(entryPoint)
	if clean == "." || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid %s: %s", key, raw)
	}
	if strings.Contains(clean, "/") {
		return "", fmt.Errorf("invalid %s: %s", key, raw)
	}

	return clean, nil
}
