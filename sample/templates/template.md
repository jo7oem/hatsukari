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
  <h2>最新記事（サイト全体）</h2>
  <ul>
    {{range .site.posts.latest}}
    <li><a href='{{.URL}}'>{{.Title}}</a> ({{.PostedAt.Format "2006-01-02"}})</li>
    {{end}}
  </ul>
</section>
{{.contents.body}}
</main>
</body>
</html>

