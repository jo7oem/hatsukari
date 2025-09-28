package render

import (
	"bytes"
	"embed"
	"html/template"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

//go:embed *.html
var emFs embed.FS
var siteTmpl = template.Must(template.New("site").ParseFS(emFs, "site.html"))

type Page struct {
	Title   string
	Content template.HTML
}

func (p *Page) RenderHTML() ([]byte, error) {
	if false {
		// for debug
		html, err := markdownToHTML([]byte(p.Content))
		if err != nil {
			return nil, err
		}
		return []byte(html), nil
	}

	tmp, err := siteTmpl.Clone()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	page := make(map[string]any)
	page["Title"] = p.Title
	page["Content"] = template.HTML("content aaa")

	if err := tmp.Execute(&buf, page); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func markdownToHTML(markdown []byte) (template.HTML, error) {
	// parser.WithHeadingAttribute allows custom ids on headings.
	// html.WithUnsafe allows use of raw HTML, which we need for tables.
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithHeadingAttribute(),
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(util.Prioritized(mdTransformFunc(mdLink), 1)),
		),
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithExtensions(
			extension.NewTypographer(),
			extension.NewLinkify(
				extension.WithLinkifyAllowedProtocols([][]byte{[]byte("http"), []byte("https")}),
				extension.WithLinkifyEmailRegexp(regexp.MustCompile(`[^\x00-\x{10FFFF}]`)), // impossible
			),
			extension.DefinitionList,
			extension.NewTable(),
		),
	)
	var buf bytes.Buffer
	buf.WriteString("<div class='markdown'>\n")
	if err := md.Convert(replaceTabs(markdown), &buf); err != nil {
		return "", err
	}
	buf.WriteString("</div>\n")
	return template.HTML(buf.String()), nil
}

// mdTransformFunc is a func implementing parser.ASTTransformer.
type mdTransformFunc func(*ast.Document, text.Reader, parser.Context)

func (f mdTransformFunc) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	f(node, reader, pc)
}

// mdLink walks doc, adding rel=noreferrer target=_blank to non-relative links.
func mdLink(doc *ast.Document, _ text.Reader, _ parser.Context) {
	mdLinkWalk(doc)
}

func mdLinkWalk(n ast.Node) {
	switch n := n.(type) {
	case *ast.Link:
		dest := string(n.Destination)
		if strings.HasPrefix(dest, "https://") || strings.HasPrefix(dest, "http://") {
			n.SetAttributeString("rel", []byte("noreferrer"))
			n.SetAttributeString("target", []byte("_blank"))
		}
		return
	case *ast.AutoLink:
		// All autolinks are non-relative.
		n.SetAttributeString("rel", []byte("noreferrer"))
		n.SetAttributeString("target", []byte("_blank"))
		return
	}

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		mdLinkWalk(child)
	}
}

// replaceTabs replaces all tabs in text with spaces up to a 4-space tab stop.
//
// In Markdown, tabs used for indentation are required to be interpreted as
// 4-space tab stops. See https://spec.commonmark.org/0.30/#tabs.
// Go also renders nicely and more compactly on the screen with 4-space
// tab stops, while browsers often use 8-space.
// And Goldmark crashes in some inputs that mix spaces and tabs.
// Fix the crashes and make the Go code consistently compact across browsers,
// all while staying Markdown-compatible, by expanding to 4-space tab stops.
//
// This function does not handle multi-codepoint Unicode sequences correctly.
func replaceTabs(text []byte) []byte {
	var buf bytes.Buffer
	col := 0
	for len(text) > 0 {
		r, size := utf8.DecodeRune(text)
		text = text[size:]

		switch r {
		case '\n':
			buf.WriteByte('\n')
			col = 0

		case '\t':
			buf.WriteByte(' ')
			col++
			for col%4 != 0 {
				buf.WriteByte(' ')
				col++
			}

		default:
			buf.WriteRune(r)
			col++
		}
	}
	return buf.Bytes()
}
