{{$locale := index .site "currentLocale"}}{{$first := index .site.indexes 0}}{{$title := index $first "title"}}{{$selected := index $title $locale}}{{if eq $selected ""}}{{$selected = index $title "default"}}{{end}}NAV={{len .site.indexes}}|FIRST={{index $first "url"}}|FIRST_JA={{index $title "ja"}}|CURRENT={{$locale}}|FIRST_SELECTED={{$selected}}|BODY={{.contents.body}}

