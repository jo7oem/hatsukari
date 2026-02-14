package logging

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"
)

type Logger struct {
	handler       slog.Handler
	saveCodeTrace slog.Level
	ctxWiths      []ctxWith
}

func NewLogger(h slog.Handler) *Logger {
	return &Logger{
		handler:       h,
		saveCodeTrace: slog.LevelError,
	}
}

func (l *Logger) SetSaveCodeTraceLevel(level slog.Level) {
	l.saveCodeTrace = level
}

type ctxWith struct {
	logKey string
	ctxKey any
}

func (l *Logger) AddContextKeys(logKey string, ctxKey any) {
	for i, v := range l.ctxWiths {
		if v.logKey == logKey && v.ctxKey == ctxKey {
			return
		}

		l.ctxWiths[i] = ctxWith{logKey: logKey, ctxKey: ctxKey}

		return
	}

	l.ctxWiths = append(l.ctxWiths, ctxWith{logKey: logKey, ctxKey: ctxKey})
}

func (l *Logger) Handler() slog.Handler {
	return l.handler
}

func (l *Logger) With(attrs ...slog.Attr) *Logger {
	if len(attrs) == 0 {
		return l
	}

	c := l.clone()

	c.handler = l.handler.WithAttrs(attrs)

	return c
}

func (l *Logger) WithGroup(name string) *Logger {
	if name == "" {
		return l
	}

	c := l.clone()
	c.handler = l.handler.WithGroup(name)

	return c
}

func (l *Logger) Enabled(ctx context.Context, level slog.Level) bool {
	return l.handler.Enabled(ctx, level)
}

func (l *Logger) Debug(msg string, attrs ...slog.Attr) {
	l.log(context.Background(), slog.LevelDebug, msg, attrs...)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelDebug, msg, attrs...)
}

func (l *Logger) Info(msg string, attrs ...slog.Attr) {
	l.log(context.Background(), slog.LevelInfo, msg, attrs...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelInfo, msg, attrs...)
}

func (l *Logger) Warn(msg string, attrs ...slog.Attr) {
	l.log(context.Background(), slog.LevelWarn, msg, attrs...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelWarn, msg, attrs...)
}

func (l *Logger) Error(msg string, attrs ...slog.Attr) {
	l.log(context.Background(), slog.LevelError, msg, attrs...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelError, msg, attrs...)
}

const pcSize = 20

func (l *Logger) log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.Enabled(ctx, level) {
		return
	}
	r := slog.NewRecord(time.Now(), level, msg, 0)
	r.AddAttrs(attrs...)

	for _, cw := range l.ctxWiths {
		v := ctx.Value(cw.ctxKey)
		if v != nil {
			r.AddAttrs(slog.Any(cw.logKey, v))
		}
	}

	if level >= l.saveCodeTrace {
		var pcs [pcSize]uintptr
		// skip [runtime.Callers, this function, this function's caller]
		n := runtime.Callers(3, pcs[:])
		if n > 2 {
			n = n - 2
		}
		frames := runtime.CallersFrames(pcs[:n])
		s := strings.Builder{}
		for {
			frame, more := frames.Next()
			s.WriteString(fmt.Sprintf("%s:%d\n", frame.File, frame.Line))
			if !more {
				break
			}
		}

		r.AddAttrs(slog.String("code_trace", s.String()))
	}

	_ = l.Handler().Handle(ctx, r)
}

func (l *Logger) clone() *Logger {
	cw := make([]ctxWith, len(l.ctxWiths))
	copy(cw, l.ctxWiths)
	return &Logger{
		handler:       l.handler,
		saveCodeTrace: l.saveCodeTrace,
		ctxWiths:      cw,
	}
}
