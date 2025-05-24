package main

import (
	"fmt"
	"github.com/jo7oem/hatsukari/config"
	"github.com/jo7oem/hatsukari/handlers"
	"net/http"
)

func main() {
	newConfig := config.NewConfig()
	if err := newConfig.SetFromEnv(); err != nil {
		panic(err)
	}

	dh, err := handlers.NewBlogHandler()
	if err != nil {
		panic(err)
	}

	fmt.Println("Server started on", newConfig.Server.Addr)

	if err := http.ListenAndServe(":8080", dh); err != nil {
		panic(err)
	}
}
