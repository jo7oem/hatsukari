package site

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/contents"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	siteMeter = otel.Meter("hatsukari/site")

	siteMetricsInitOnce sync.Once
	siteMetricsInitErr  error

	httpRequestsTotal metric.Int64Counter
	goRoutinesGauge   metric.Int64ObservableGauge
	heapAllocGauge    metric.Int64ObservableGauge
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

	if err := initSiteMetrics(); err != nil {
		logger.Error("failed to initialize site metrics", err)
	}

	return site, nil

}

func initSiteMetrics() error {
	siteMetricsInitOnce.Do(func() {
		var err error

		httpRequestsTotal, err = siteMeter.Int64Counter(
			"hatsukari_http_requests_total",
			metric.WithDescription("HTTP requests total grouped by method and status code"),
		)
		if err != nil {
			siteMetricsInitErr = err
			return
		}

		goRoutinesGauge, err = siteMeter.Int64ObservableGauge(
			"hatsukari_runtime_goroutines",
			metric.WithDescription("Number of goroutines"),
		)
		if err != nil {
			siteMetricsInitErr = err
			return
		}

		heapAllocGauge, err = siteMeter.Int64ObservableGauge(
			"hatsukari_runtime_heap_alloc_bytes",
			metric.WithDescription("Allocated heap bytes"),
		)
		if err != nil {
			siteMetricsInitErr = err
			return
		}

		_, err = siteMeter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			observer.ObserveInt64(goRoutinesGauge, int64(runtime.NumGoroutine()))
			observer.ObserveInt64(heapAllocGauge, int64(ms.HeapAlloc))
			return nil
		}, goRoutinesGauge, heapAllocGauge)
		if err != nil {
			siteMetricsInitErr = err
		}
	})

	return siteMetricsInitErr
}

type Site struct {
	config   SiteConfig
	fs       *os.Root
	mux      *http.ServeMux
	vars     map[string]any
	logger   *logging.Logger
	access   *logging.Logger
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
	responseWriter := &accessLogResponseWriter{ResponseWriter: w}

	reqCtx := logging.InjectTracer(r.Context(), logging.Tracer(s.tracer))
	spanCtx, span := logging.StartSpan(reqCtx, r.Method+" "+r.URL.Path)
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
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
		slog.Int("responseBytes", responseWriter.bytes),
		slog.Int64("durationMs", time.Since(startedAt).Milliseconds()),
		slog.String("userAgent", strings.TrimSpace(r.UserAgent())),
		slog.String("remoteAddr", remoteAddrHost(r.RemoteAddr)),
	)

	if httpRequestsTotal != nil {
		httpRequestsTotal.Add(spanCtx, 1,
			metric.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.status_code", strconv.Itoa(status)),
			),
		)
	}
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
