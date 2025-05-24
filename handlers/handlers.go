package handlers

import (
	"github.com/jo7oem/hatsukari/config"
	"github.com/jo7oem/hatsukari/logger"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

type StaticSiteHandler struct {
	fs      fs.FS
	pattern string
	http.Handler
}

func NewStaticSiteHandler(conf config.ContentConf, rootFS fs.FS) (*StaticSiteHandler, error) {
	root, err := os.OpenRoot("/home/su-2/GolandProjects/hatsukari/sample")
	if err != nil {
		return nil, err
	}

	rootFS = root.FS()

	conf = config.ContentConf{
		RootDir:  "static",
		ServPath: "/static/",
	}

	// Open the root directory
	subFS, err := fs.Sub(rootFS, conf.RootDir)
	if err != nil {
		return nil, err
	}

	h := http.StripPrefix(conf.ServPath, http.FileServer(http.FS(denyDotFiles(subFS))))

	return &StaticSiteHandler{
		fs:      subFS,
		pattern: "GET " + conf.ServPath,
		Handler: h,
	}, err
}

func (sh *StaticSiteHandler) Pattern() string {
	return sh.pattern
}

type dotFileFS struct {
	fs.FS
}

func (d dotFileFS) Open(name string) (fs.File, error) {
	if haveDotfiles(name) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  os.ErrNotExist,
		}
	}

	f, err := d.FS.Open(name)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func denyDotFiles(f fs.FS) fs.FS {
	return dotFileFS{f}
}

func haveDotfiles(path string) bool {
	// Check if the file system contains any dot files
	for _, part := range strings.Split(path, "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}

	return false
}

func LogHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request
		// fmt.Printf("Request: %s %s\n", r.Method, r.URL.Path)
		ctx := r.Context()

		l := logger.GetLoggerFromContext(ctx).With("URL", r.URL.Path)
		r = r.WithContext(logger.SetLoggerToContext(ctx, l))

		l.DebugContext(ctx, "request invoke")

		lw := newLogWriter(w)

		h.ServeHTTP(lw, r)

		l.DebugContext(ctx, "request invoked", slog.Int("status", lw.statusCode), slog.String("method", r.Method))
	})
}

type logWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLogWriter(w http.ResponseWriter) *logWriter {
	return &logWriter{
		ResponseWriter: w,
	}
}

func (lw *logWriter) WriteHeader(statusCode int) {
	lw.statusCode = statusCode
	lw.ResponseWriter.WriteHeader(statusCode)
}
