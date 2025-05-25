package logger

import (
	"context"
	"log/slog"
	"os"
)

type key int

const (
	loggerKey key = iota
)

var (
	//nolint: gochecknoglobals
	logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		AddSource:   true,
		ReplaceAttr: nil,
	}))
)

// GetLogger はロガーを返す.
func GetLogger() *slog.Logger {
	return logger
}

// GetLoggerFromContext は、コンテキストからロガーを取得する。
func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return GetLogger()
	}

	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}

	return GetLogger()
}

// SetLoggerToContext は、コンテキストにロガーを設定する。
func SetLoggerToContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}
