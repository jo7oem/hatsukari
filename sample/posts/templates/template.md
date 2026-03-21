<!doctype html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <title>{{.page.meta.title}}</title>
</head>
<body>
{{template "header.html" .}}
<main>
<section>
  <h2>最新記事（この配下）</h2>
  <ul>
    {{range .contents.posts.latest}}
    <li><a href='{{.URL}}'>{{.Title}}</a></li>
    {{end}}
  </ul>
</section>
<section>
  <h2>最新記事（サイト全体）</h2>
  <ul>
    {{range .site.posts.latest}}
    <li><a href='{{.URL}}'>{{.Title}}</a></li>
    {{end}}
  </ul>
</section>
{{if .page.meta.title}}
<section>
  <h2>タグ</h2>
  <ul>
    {{range .contents.posts.all}}
    {{if eq .Title $.page.meta.title}}
    {{range .Tags}}
    <li>
      {{if .URL}}<a href='{{.URL}}'>{{index .Label "default"}}</a>{{else}}{{index .Label "default"}}{{end}}
    </li>
    {{end}}
    {{end}}
    {{end}}
  </ul>
</section>
{{end}}
{{.contents.body}}
</main>
</body>
</html>

