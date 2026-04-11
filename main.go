package main

import (
	"log/slog"
	"os"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/store/materials"
	"github.com/jo7oem/hatsukari/store/products"
)

func main() {
	logger := logging.NewLogger(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}), "main")

	cnf, err := ReadConfig()
	if err != nil {
		panic(err)
	}

	logger.Debug("Config", slog.String("Server Addr", cnf.ServerAddr), slog.String("ContentsPath", cnf.ContentsPath), slog.String("OutputPath", cnf.OutputPath))

	logger.Info("Start")

	mtls, err := materials.New(cnf.ContentsPath)
	if err != nil {
		panic(err)
	}

	defer mtls.Close()

	prds, err := products.New(cnf.OutputPath)
	if err != nil {
		panic(err)
	}

	defer prds.Close()

	logger.Info("Fin")

}

type Config struct {
	ServerAddr   string
	ContentsPath string
	OutputPath   string
}

func ReadConfig() (*Config, error) {
	config := &Config{
		ServerAddr:   ":8080",
		ContentsPath: "./contents",
		OutputPath:   "./output",
	}
	if sa := os.Getenv("SERVER_ADDR"); sa != "" {
		config.ServerAddr = sa
	}

	if sa := os.Getenv("CONTENTS_PATH"); sa != "" {
		config.ContentsPath = sa
	}

	if sa := os.Getenv("OUTPUT_PATH"); sa != "" {
		config.OutputPath = sa
	}

	return config, nil
}
