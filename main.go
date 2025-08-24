package main

import (
	"fmt"
	"os"

	"github.com/gomarkdown/markdown"
	"github.com/jo7oem/hatsukari/store/config"
)

func main() {
	configPath := "sample/server_config.yaml"

	serverConfig, err := config.LoadServerConfig(configPath)
	if err != nil {
		panic(err)
	}

	// handlerを構築していく
	siteRoot, err := os.OpenRoot(serverConfig.SiteContentsDir)
	if err != nil {
		panic(err)
	}

	siteConf, err := config.LoadWebsiteConfig(siteRoot.FS(), serverConfig.SiteConfigName)
	if err != nil {
		panic(err)
	}

	root, err := siteRoot.OpenRoot(siteConf.SiteRootDir)
	if err != nil {
		panic(err)
	}

	_ = root
	//

	var r markdown.Renderer
	_ = r

	fmt.Println("End")
}
