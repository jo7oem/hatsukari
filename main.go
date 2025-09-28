package main

import (
	"fmt"

	"github.com/jo7oem/hatsukari/logic/render"
	"github.com/jo7oem/hatsukari/store/cache"
)

func main() {
	cachePath := "./tmp/cache"
	cm, err := cache.NewCacheManager(cachePath)
	if err != nil {
		fmt.Println("Error initializing cache manager:", err)
		return
	}
	defer cm.Close()
	fmt.Println("Cache manager initialized at:", cachePath)

	p := render.Page{}
	html, err := p.RenderHTML()
	if err != nil {
		fmt.Println("Error rendering page:", err)
		return
	}
	fmt.Println("Rendered HTML:", html)
	fmt.Println("End")
}
