package handlers

import (
	"fmt"
	"github.com/jo7oem/hatsukari/config"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"
)

func NewBlogHandler() (http.Handler, error) {
	handler := http.NewServeMux()

	sh, err := NewStaticSiteHandler(config.ContentConf{}, nil)
	if err != nil {
		return nil, err
	}

	handler.Handle(sh.Pattern(), sh)

	th, err := NewTopPageHandler()
	if err != nil {
		return nil, err
	}

	handler.Handle(th.Pattern(), th)

	h := LogHandler(http.TimeoutHandler(handler, 3*time.Second, "Timeout"))

	return h, nil
}

type TopPageHandler struct {
	Title   string
	handler http.Handler
}

func NewTopPageHandler() (*TopPageHandler, error) {
	conf := config.TopPageConf{
		RootDir:    "/home/su-2/GolandProjects/hatsukari/sample/topPage",
		Title:      "Top Page",
		HeaderFile: "header.html",
		PageFile:   "index.html",
		FooterFile: "footer.html",
	}

	// Open the root directory
	rootFS, err := os.OpenRoot(conf.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open root directory: %w", err)
	}

	return &TopPageHandler{
		Title:   conf.Title,
		handler: http.FileServer(http.FS(rootFS.FS())),
	}, nil
}

func (h *TopPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

func (h TopPageHandler) Pattern() string {
	return "GET /"
}

type maskFS struct {
	maskDotFiles            bool
	allowFileNameExtensions []string
	denyFileNameExtensions  []string
	FS                      fs.FS
}

func (mf maskFS) Open(name string) (fs.File, error) {
	if mf.maskDotFiles && haveDotfiles(name) {
		return nil, errPathError(name)
	}

	for _, ext := range mf.denyFileNameExtensions {
		if strings.HasSuffix(strings.ToLower(name), ext) {
			return nil, errPathError(name)
		}
	}
	// allowリストが有効な場合は、マッチしないものは見せない
	match := len(mf.allowFileNameExtensions) == 0

	for _, ext := range mf.allowFileNameExtensions {
		if strings.HasSuffix(strings.ToLower(name), ext) {
			match = true

			break
		}
	}

	if !match {
		return nil, errPathError(name)
	}

	return mf.FS.Open(name)
}

func NewMaskFS(f fs.FS, maskDotfile bool, denyFileExt []string, allowFileExt []string) fs.FS {
	mf := maskFS{FS: f,
		maskDotFiles:            maskDotfile,
		denyFileNameExtensions:  make([]string, len(denyFileExt)),
		allowFileNameExtensions: make([]string, len(allowFileExt)),
	}

	for i, s := range denyFileExt {
		mf.denyFileNameExtensions[i] = strings.ToLower(s)
	}

	for i, s := range allowFileExt {
		mf.allowFileNameExtensions[i] = strings.ToLower(s)
	}

	return mf
}

func errPathError(name string) *fs.PathError {
	return &fs.PathError{
		Op:   "open",
		Path: name,
		Err:  os.ErrNotExist,
	}
}
