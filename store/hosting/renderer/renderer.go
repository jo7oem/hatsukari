package renderer

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
)

var wakeUpTime = time.Now()
var markdown = goldmark.New(
	goldmark.WithExtensions(
		meta.Meta,
	),
)

type Renderer struct {
	b []byte
}

func NewRenderer(b []byte) *Renderer {
	return &Renderer{
		b: b,
	}
}

func (r *Renderer) Render() ([]byte, error) {
	html := bytes.NewBuffer([]byte{})
	context := parser.NewContext()
	err := markdown.Convert(r.b, html, parser.WithContext(context))
	metaData := meta.Get(context)
	fmt.Println(metaData)
	return html.Bytes(), err
}
func (r *Renderer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	b, err := r.Render()
	if err != nil {
		http.Error(w, "failed to render", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, req, "a.html", wakeUpTime, bytes.NewReader(b))
}
