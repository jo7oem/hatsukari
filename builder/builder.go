package builder

import (
	"errors"
	"fmt"
	"github.com/jo7oem/hatsukari/store/config"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

func NewHandler(conf *config.ContentConfig) (http.Handler, error) {
	if conf == nil {
		return nil, errors.New("invalid arguments. conf is nil")
	}

	blg, err := loadTopConf(conf.GetFS())
	if err != nil {
		return nil, err
	}

	return blg.NewHandler(), nil
}

type HatsukariConf struct {
	BlogName string             `yaml:"blogName"`
	Contents map[string]content `yaml:"contents"`
	fs       fs.FS
}

func loadTopConf(fSys fs.FS) (*HatsukariConf, error) {
	if fSys == nil {
		return nil, errors.New("nil")
	}

	file, err := fSys.Open("hatsukari.yaml")
	if err != nil {
		return nil, err
	}

	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	res := HatsukariConf{fs: fSys}

	if err := yaml.Unmarshal(buf, &res); err != nil {
		return nil, err
	}

	for k, v := range res.Contents {
		v.name = k
		res.Contents[k] = v
	}

	return &res, nil
}

func (h HatsukariConf) NewHandler() http.Handler {
	handler := http.NewServeMux()
	for _, v := range h.Contents {
		hl, err := v.getHandler(h.fs)
		if err != nil {
			continue
		}

		hp := http.StripPrefix(v.URL, hl)
		p := v.URL
		handler.Handle(p, hp)
	}

	return handler
}

const (
	TypeStaticFiles = "staticFiles"
	TypeStaticDir   = "staticDir"
	TypeTemplate    = "template"
)

type content struct {
	name     string
	FilePath string `yaml:"filePath"`
	URL      string `yaml:"url"`
	Type     string `yaml:"type"`
	confName string `yaml:"confName"`
}

func (c content) getHandler(fSys fs.FS) (http.Handler, error) {
	switch c.Type {
	case TypeStaticDir:
		return NewStaticDirHandler(fSys, c.URL, c.FilePath), nil
	}

	return nil, errors.New("invalid type")
}

func NewStaticDirHandler(fSys fs.FS, httpPath string, rootDir string) http.Handler {
	return staticFileDirHandler{
		fs:       fSys,
		httpPath: httpPath,
		rootDir:  rootDir,
	}
}

type staticFileDirHandler struct {
	fs       fs.FS
	httpPath string
	rootDir  string
}

func newReaderAt(buf []byte) *readerAt {
	return &readerAt{
		buf: buf,
	}
}

type readerAt struct {
	buf []byte
}

func (ra *readerAt) ReadAt(buf []byte, off int64) (int, error) {
	if off >= int64(len(ra.buf)) {
		buf = buf[:0]

		return 0, io.EOF
	}

	n := copy(buf, ra.buf[off:])
	ra.buf = ra.buf[int(off)+n:]

	if n < len(buf) {
		return n, io.EOF
	}

	return n, nil
}

func (h staticFileDirHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.RequestURI
	fName := filepath.FromSlash(urlPath)
	if filepath.Base(fName)[0] == '.' {
		http.Error(w, "Not Found", http.StatusNotFound)

		return
	}

	if !strings.HasPrefix(urlPath, h.httpPath) {
		http.Error(w, "Not Found", http.StatusNotFound)

		return
	}

	tPath := filepath.Join(h.rootDir, fName)
	f, err := h.fs.Open(tPath)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)

		return
	}

	/*w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, f); err != nil {
		fmt.Println(err)
	}
	*/

	buf := make([]byte, 512)

	if _, err := f.Read(buf); err != nil {
		fmt.Println(err)
	}

	ra := newReaderAt(buf)
	sr := io.NewSectionReader(ra, 0, 0)
	fi, _ := f.Stat()

	http.ServeContent(w, r, fName, fi.ModTime(), sr)
}
