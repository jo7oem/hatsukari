package logger

import (
	"context"
	"log/slog"
	"os"
)

const (
	loggerKey = "logger"
)

var Logger *slog.Logge


var Logger *slog.Logger
dlerOptions{L
func init() {
	logger := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})
		Logger = slog.New(logger)
}
// GetLoggerFromContext は、コンテキストからロガーを取得する。
func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Logger
	}

	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}

	return Logger
}

// SetLoggerToContext は、コンテキストにロガーを設定する。
func SetLoggerToContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}
