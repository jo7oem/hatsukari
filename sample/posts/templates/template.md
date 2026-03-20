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
{{.contents.body}}
</main>
</body>
</html>

