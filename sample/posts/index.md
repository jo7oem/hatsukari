---
title: 記事一覧
---

aa

## 最新記事（この配下）

{{range .contents.posts.latest}}
- [{{.Title}}]({{.URL}})
{{end}}

## 最新記事（サイト全体）

{{range .site.posts.latest}}
- [{{.Title}}]({{.URL}})
{{end}}

{{if .page.meta.title}}
## タグ

{{range .contents.posts.all}}
{{if eq .Title $.page.meta.title}}
{{range .Tags}}
- {{if .URL}}[{{index .Label "default"}}]({{.URL}}){{else}}{{index .Label "default"}}{{end}}
{{end}}
{{end}}
{{end}}
{{end}}
