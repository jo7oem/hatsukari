package main

import (
	"fmt"

	"github.com/gomarkdown/markdown"
	"github.com/jo7oem/hatsukari/store/cache"
)

func main() {
	// はじめに、記事のMarkdownをHTMLに変換します。
	md := []byte("# Hello, Markdown!\nThis is a sample markdown text.")
	html := markdown.ToHTML(md, nil, nil)
	fmt.Println(string(html))

	// 次に、キャッシュマネージャを初期化します。
	cachePath := "./tmp/cache"
	cm, err := cache.NewCacheManager(cachePath)
	if err != nil {
		fmt.Println("Error initializing cache manager:", err)
		return
	}
	defer cm.Close()
	fmt.Println("Cache manager initialized at:", cachePath)

	if err := cm.Store("articles/article1.html", html); err != nil {
		panic(err)
	}

	fmt.Println("End")
}
