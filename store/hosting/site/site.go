package site

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/contents"
	"github.com/jo7oem/hatsukari/telemetry"
)

type SiteConfig struct {
	Title            string         `yaml:"title"`
	Timezone         string         `yaml:"timezone,omitempty"`
	Latest           *int           `yaml:"latest,omitempty"`
	SiteTemplatesDir string         `yaml:"siteTemplatesDir"`
	SiteTemplate     string         `yaml:"siteTemplate,omitempty"`
	RootContentDir   string         `yaml:"rootContentDir"`
	ContentsDir      []string       `yaml:"contentsDir,omitempty"`
	Variables        map[string]any `yaml:"variables,omitempty"`
}

func OpenSiteDir(path string, logger *logging.Logger) (*Site, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger must not be nil")
	}

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
		logger.Error("invalid site timezone", err, slog.String("timezone", strings.TrimSpace(conf.Timezone)))
		return nil, err
	}

	site := &Site{
		fs:       root,
		config:   *conf,
		mux:      http.NewServeMux(),
		logger:   logger,
		access:   logger.WithGroup("access"),
		tracer:   "hatsukari/site",
		location: location,
	}

	if err := site.Setup(); err != nil {
		logger.Error("failed to setup site", err)
		return nil, err
	}

	metrics, metricsErr := telemetry.InitSiteMetrics()
	if metricsErr != nil {
		logger.Error("failed to initialize site metrics", metricsErr)
	}
	site.metrics = metrics

	return site, nil

}

type Site struct {
	config   SiteConfig
	fs       *os.Root
	mux      *http.ServeMux
	vars     map[string]any
	logger   *logging.Logger
	access   *logging.Logger
	metrics  *telemetry.SiteMetrics
	tracer   string
	location *time.Location
	root     *contents.Content
	latest   int
}

type accessLogResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *accessLogResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *accessLogResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func remoteAddrHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return strings.TrimSpace(remoteAddr)
	}
	return strings.TrimSpace(host)
}

func (s *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	requestPath := r.URL.Path
	responseWriter := &accessLogResponseWriter{ResponseWriter: w}

	reqCtx := telemetry.InjectTracer(r.Context(), telemetry.Tracer(s.tracer))
	spanCtx, span := telemetry.StartSpan(reqCtx, r.Method+" "+requestPath)
	defer span.End()

	s.mux.ServeHTTP(responseWriter, r.WithContext(spanCtx))

	status := responseWriter.status
	if status == 0 {
		status = http.StatusOK
	}

	accessLogger := s.access
	if accessLogger == nil {
		accessLogger = s.logger
	}

	accessLogger.InfoContext(spanCtx, "request",
		slog.String("method", r.Method),
		slog.String("path", requestPath),
		slog.Int("status", status),
		slog.Int("responseBytes", responseWriter.bytes),
		slog.Int64("durationMs", time.Since(startedAt).Milliseconds()),
		slog.String("userAgent", strings.TrimSpace(r.UserAgent())),
		slog.String("remoteAddr", remoteAddrHost(r.RemoteAddr)),
	)

	s.metrics.RecordHTTPRequest(spanCtx, r.Method, status)
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
	if siteVariables, ok := vars["variables"].(map[string]any); ok {
		vars["variables"] = maps.Clone(siteVariables)
	}
	if s.root != nil {
		vars["posts"] = s.root.BuildSitePosts(time.Now().In(s.location), s.latest)
	}
	return vars
}

func (s *Site) Setup() error {
	mux := http.NewServeMux()
	root, err := contents.OpenContentDir(s.fs, s.Config().RootContentDir, s.logger)
	if err != nil {
		s.logger.Error("failed to open root content", err)
		return err
	}
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
	siteConfigVariables := maps.Clone(s.config.Variables)
	if siteConfigVariables == nil {
		siteConfigVariables = map[string]any{}
	}
	s.vars = map[string]any{
		"title":     s.config.Title,
		"variables": siteConfigVariables,
		"indexes":   siteIndexes,
	}
	s.root = root
	s.latest = siteLatest
	root.SetSiteVariables(s.vars)

	siteTemplateFS, err := s.resolveSiteTemplateFS()
	if err != nil {
		return err
	}
	root.SetSiteTemplateFS(siteTemplateFS)
	root.SetSiteTemplateEntryPoint(s.config.SiteTemplate)

	mux.Handle("/", root)
	s.mux = mux
	return nil
}

func (s *Site) resolveSiteTemplateFS() (fs.FS, error) {
	templatesDir := strings.TrimSpace(s.config.SiteTemplatesDir)
	if templatesDir == "" {
		return nil, nil
	}

	baseFS := s.fs.FS()
	if templatesDir == "." {
		return baseFS, nil
	}

	templateFS, err := fs.Sub(baseFS, templatesDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	return templateFS, nil
}
