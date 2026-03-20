package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/hosting/site"
)

// フェーズ5: 全記事を対象に一覧・詳細・タグ別を生成し、assets をコピー
func main() {
	samplePath := "./sample"
	logger := logging.NewLogger(slog.NewTextHandler(os.Stdout, nil), "hatsukari")

	siteMap, err := site.OpenSiteDirWithLogger(samplePath, logger)
	if err != nil {
		logger.Error("failed to open site dir", err)
		return
	}

	if err := siteMap.Setup(); err != nil {
		logger.Error("failed to setup site", err)
		return
	}

	err = http.ListenAndServe(":8080", siteMap)
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("server started at :8080")
}
