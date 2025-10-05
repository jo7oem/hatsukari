package main

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	address  = "0.0.0.0:8080"
	siteRoot = "sample/site_root"
	title    = "My Site"
)

func main() {
	rootFS, err := os.OpenRoot(siteRoot)
	if err != nil {
		panic(err)
	}

	defer rootFS.Close()

	sc := &SiteConfig{
		Title:   title,
		rootDir: "root",
	}

	sr, err := NewSiteRouter(rootFS, sc)
	if err != nil {
		panic(err)
	}

	if err := http.ListenAndServe(address, sr); err != nil {
		panic(err)
	}

	fmt.Println("End")
}

type SiteConfig struct {
	Title   string
	rootDir string
}

type SiteRouter struct {
	RootFS   *os.Root
	router   *http.ServeMux
	renderer *SiteRender
}

func (s SiteRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func NewSiteRouter(rootFS *os.Root, conf *SiteConfig) (*SiteRouter, error) {
	res := &SiteRouter{RootFS: rootFS}

	siteTmplFile, err := rootFS.Open("templates/site.tmpl")
	if err != nil {
		return nil, err
	}

	defer siteTmplFile.Close()

	siteTmplRow, err := io.ReadAll(siteTmplFile)
	if err != nil {
		return nil, err
	}

	siteTmpl, err := template.New("entry").Parse(string(siteTmplRow))
	if err != nil {
		return nil, err
	}

	res.renderer = &SiteRender{
		template: siteTmpl,
		title:    conf.Title,
	}

	// Static file server
	rootDir, err := rootFS.OpenRoot(conf.rootDir)
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(FilterDotFiles{FS: rootDir.FS()}))

	r := http.NewServeMux()
	reIndexFile := regexp.MustCompile(`^index\.(html|htm)?$`)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)
		dirPath, filePath := filepath.Split(path)
		if filePath == "" || reIndexFile.MatchString(filePath) {
			filePath = "index"
		}

		path = filepath.Join(dirPath, filePath)

		if len(strings.SplitN(filePath, ".", 2)) > 1 {
			fileServer.ServeHTTP(w, r)

			return
		}

		// Render page
		for _, ext := range []string{".html", ".tmpl", ".md", ""} {
			filename := filepath.Join(".", path+ext)
			fStat, err := rootDir.Stat(filename)
			if os.IsNotExist(err) {
				continue
			}

			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if fStat.IsDir() {
				http.NotFound(w, r)
				return
			}

			if ext == "" {
				path = path + ext
				r.URL.Path = path

				fileServer.ServeHTTP(w, r)

				return
			}

			f, err := rootDir.Open(filename)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			defer f.Close()

			contentRow, err := io.ReadAll(f)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			siteTmpl.Execute(w, map[string]any{
				"Title":   conf.Title,
				"Content": template.HTML(contentRow),
			})

			return

		}

		http.NotFound(w, r)
	})

	res.router = r

	return res, nil
}

type SiteRender struct {
	title    string
	template *template.Template
}

func (s *SiteRender) Execute(w io.Writer, content template.HTML) error {
	return s.template.Execute(w, map[string]any{
		"Title":   s.title,
		"Content": content,
	})
}

type FilterDotFiles struct {
	fs.FS
}

var reDotFile = regexp.MustCompile(`(^|/)\.`)

func (f FilterDotFiles) Open(name string) (fs.File, error) {
	if reDotFile.Match([]byte(name)) {
		return nil, fs.ErrNotExist
	}

	return f.FS.Open(name)
}

type RootFSHandler struct {
	os.Root
	prefix string
	http.FileSystem
}

func (r RootFSHandler) Open(name string) (http.File, error) {
	if r.prefix != "" {
		name = r.prefix + name
	}
	return r.Root.Open(name)
}
