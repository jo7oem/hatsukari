{{if .contents.posts.currentTag.Key}}<h1>{{index .contents.posts.currentTag.Label "default"}}</h1>
<p>{{index .contents.posts.currentTag.About "default"}}</p>
<ul>
  {{range .contents.posts.currentTag.Posts}}
  <li><a href='{{.URL}}'>{{.Title}}</a></li>
  {{end}}
</ul>
{{else}}<h1>タグ一覧</h1>
<ul>
  {{range .contents.posts.tags}}
  <li>
    {{if .URL}}<a href='{{.URL}}'>{{index .Label "default"}}</a>{{else}}{{index .Label "default"}}{{end}} ({{.Count}})
  </li>
  {{end}}
</ul>
{{end}}

