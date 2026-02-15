package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/site"
)

// フェーズ5: 全記事を対象に一覧・詳細・タグ別を生成し、assets をコピー
func main() {
	samplePath := "./sample"

	s, err := site.OpenSiteDir(samplePath)
	if err != nil {
		log.Fatalf("build: failed to open site dir: %v", err)
	}
	defer s.Close()
	sl := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})
	ll := slog.New(sl)
	ll.Debug("start")

	l := logging.NewLogger(sl)
	l.Debug("start")
	l.Error("sample error message", slog.String("key", "value"))

	fmt.Println(s.Config())
	log.Println("build: done")
	err = http.ListenAndServe(":8080", s)
	if err != nil {
		log.Fatalf("server: failed to start server: %v", err)
	}
}
