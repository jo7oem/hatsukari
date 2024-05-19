package resources

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

const BlogTitle = "my blog だよ"

const sniffLen = 512

func StaticRouter(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path[1:]

	FileSys, err := staticFS.Open(path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	defer FileSys.Close()

	fileInfo, err := FileSys.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if fileInfo.IsDir() {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	contentLen := fileInfo.Size()

	w.Header().Set("Content-Length", fmt.Sprintf("%d", contentLen))

	var buf []byte

	var contentType string

	switch getExtension(path) {
	case "css":
		contentType = "text/css"
	case "js":
		contentType = "text/javascript"
	default:
		buf = make([]byte, sniffLen)

		n, err := FileSys.Read(buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		buf = buf[:n]
		contentType = http.DetectContentType(buf)
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)

	if len(buf) > 0 {
		w.Write(buf)
	}

	if contentLen == int64(len(buf)) {
		return
	}

	io.Copy(w, FileSys)
}

func BlogRouter(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/":
		topPage(w, req)
	case "/about", "/about/":
		aboutPage(w, req)
	default:
		http.NotFound(w, req)
	}
}

func getExtension(path string) string {
	if len(path) == 0 {
		return ""
	}

	sl := strings.Split(path, "/")

	ext := strings.Split(sl[len(sl)-1], ".")
	if len(ext) == 0 {
		return ""
	}

	return ext[len(ext)-1]
}

type topPageVal struct {
	BlogTitle string
	HeadVal   HeadVal
	HeaderVal HeaderVal
}

func topPage(w http.ResponseWriter, req *http.Request) {
	tempHead, err0 := template.ParseFS(templateFS, "templates/head.html")
	_ = err0
	temp, err1 := tempHead.ParseFS(templateFS, "templates/index.html")
	_ = err1
	temp = template.Must(temp.ParseFS(templateFS, "templates/header.html"))
	v := topPageVal{
		BlogTitle: BlogTitle,
		HeadVal: HeadVal{
			Title: BlogTitle,
		},
		HeaderVal: HeaderVal{
			BlogTitle: BlogTitle,
		},
	}

	err := temp.ExecuteTemplate(w, "index.html", v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		panic(err)
	}

	w.WriteHeader(http.StatusOK)
}

type HeadVal struct {
	Title string
}

type HeaderVal struct {
	BlogTitle string
}

type aboutPageVal struct {
	BlogTitle string
	HeadVal   HeadVal
	HeaderVal HeaderVal
}

func aboutPage(w http.ResponseWriter, req *http.Request) {
	tempHead, err0 := template.ParseFS(templateFS, "templates/head.html")
	_ = err0
	temp, err1 := tempHead.ParseFS(templateFS, "templates/about.html")
	_ = err1
	temp = template.Must(temp.ParseFS(templateFS, "templates/header.html"))
	v := aboutPageVal{
		BlogTitle: BlogTitle,
		HeadVal: HeadVal{
			Title: BlogTitle,
		},
		HeaderVal: HeaderVal{
			BlogTitle: BlogTitle,
		},
	}

	err := temp.ExecuteTemplate(w, "about.html", v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		panic(err)
	}

	w.WriteHeader(http.StatusOK)
}
