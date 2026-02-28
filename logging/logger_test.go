package logging_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jo7oem/hatsukari/logging"
)

type capturingHandler struct {
	buf     strings.Builder
	handler slog.Handler
}

func (ch *capturingHandler) GetLogRecord() (*logRecord, error) {
	lr, err := jsonToLogRecord(ch.buf.String())
	if err != nil {
		return nil, err
	}

	return lr, nil
}

type logRecord struct {
	Time  string
	Level string
	Msg   string
	Attrs map[string]any
}

func mapToLogRecord(m map[string]any) (*logRecord, error) {
	lr := &logRecord{}
	if time, ok := m["time"].(string); ok {
		lr.Time = time
		delete(m, "time")
	}
	if level, ok := m["level"].(string); ok {
		lr.Level = level
		delete(m, "level")
	}
	if msg, ok := m["msg"].(string); ok {
		lr.Msg = msg
		delete(m, "msg")
	}

	lr.Attrs = m

	return lr, nil
}

func jsonToLogRecord(r string) (*logRecord, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(r), &m); err != nil {
		return nil, err
	}

	return mapToLogRecord(m)
}

func newHandler(level slog.Level) *capturingHandler {
	ch := &capturingHandler{}
	ch.handler = slog.NewJSONHandler(&ch.buf, &slog.HandlerOptions{Level: level})

	return ch
}

func TestLogger_Info(t *testing.T) {
	tests := []struct {
		name       string
		msg        string
		attrs      []slog.Attr
		wantsLevel string
		wants      map[string]any
	}{
		{
			name: "simple log",
			msg:  "Hello, World!",
			attrs: []slog.Attr{
				slog.String("key1", "value1"),
				slog.Int("key2", 42),
			},
			wantsLevel: "INFO",
			wants: map[string]any{
				"namespace": "test",
				"key1":      "value1",
				"key2":      float64(42),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cph := newHandler(slog.LevelInfo)
			l := logging.NewLogger(cph.handler, "test")
			l.Info(tt.msg, tt.attrs...)
			lr, err := cph.GetLogRecord()
			if err != nil {
				t.Fatalf("failed to get log record: %v", err)
			}

			if lr.Level != tt.wantsLevel {
				t.Errorf("unexpected log level: got %s, want %s", lr.Level, tt.wantsLevel)
			}
			if lr.Msg != tt.msg {
				t.Errorf("unexpected log message: got %s, want %s", lr.Msg, tt.msg)
			}

			if diff := cmp.Diff(tt.wants, lr.Attrs); diff != "" {
				t.Errorf("unexpected log record attrs (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLogger_Debug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		level      slog.Level
		msg        string
		attrs      []slog.Attr
		wantsLevel string
		wantsMsg   string
		wants      map[string]any
	}{
		{
			name:       "debug log",
			level:      slog.LevelDebug,
			msg:        "debug message",
			attrs:      []slog.Attr{slog.String("key", "value")},
			wantsLevel: "DEBUG",
			wantsMsg:   "debug message",
			wants: map[string]any{
				"namespace": "test",
				"key":       "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cph := newHandler(tt.level)
			l := logging.NewLogger(cph.handler, "test")
			l.Debug(tt.msg, tt.attrs...)

			lr, err := cph.GetLogRecord()
			if err != nil {
				t.Fatalf("failed to get log record: %v", err)
			}
			if lr.Level != tt.wantsLevel {
				t.Fatalf("unexpected log level: got %s, want %s", lr.Level, tt.wantsLevel)
			}
			if lr.Msg != tt.wantsMsg {
				t.Fatalf("unexpected log message: got %s, want %s", lr.Msg, tt.wantsMsg)
			}
			if diff := cmp.Diff(tt.wants, lr.Attrs); diff != "" {
				t.Fatalf("unexpected log record attrs (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLogger_Warn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		level      slog.Level
		msg        string
		attrs      []slog.Attr
		wantsLevel string
		wantsMsg   string
		wants      map[string]any
	}{
		{
			name:       "warn log",
			level:      slog.LevelInfo,
			msg:        "warn message",
			attrs:      []slog.Attr{slog.String("key", "value")},
			wantsLevel: "WARN",
			wantsMsg:   "warn message",
			wants: map[string]any{
				"namespace": "test",
				"key":       "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cph := newHandler(tt.level)
			l := logging.NewLogger(cph.handler, "test")
			l.Warn(tt.msg, tt.attrs...)

			lr, err := cph.GetLogRecord()
			if err != nil {
				t.Fatalf("failed to get log record: %v", err)
			}
			if lr.Level != tt.wantsLevel {
				t.Fatalf("unexpected log level: got %s, want %s", lr.Level, tt.wantsLevel)
			}
			if lr.Msg != tt.wantsMsg {
				t.Fatalf("unexpected log message: got %s, want %s", lr.Msg, tt.wantsMsg)
			}
			if diff := cmp.Diff(tt.wants, lr.Attrs); diff != "" {
				t.Fatalf("unexpected log record attrs (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLogger_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		level      slog.Level
		msg        string
		err        error
		wantsLevel string
		wantsMsg   string
		wants      map[string]any
	}{
		{
			name:       "error log",
			level:      slog.LevelInfo,
			msg:        "error message",
			err:        errors.New("boom"),
			wantsLevel: "ERROR",
			wantsMsg:   "error message",
			wants: map[string]any{
				"namespace": "test",
				"error":     "boom",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cph := newHandler(tt.level)
			l := logging.NewLogger(cph.handler, "test")
			l.Error(tt.msg, tt.err)

			lr, err := cph.GetLogRecord()
			if err != nil {
				t.Fatalf("failed to get log record: %v", err)
			}
			if lr.Level != tt.wantsLevel {
				t.Fatalf("unexpected log level: got %s, want %s", lr.Level, tt.wantsLevel)
			}
			if lr.Msg != tt.wantsMsg {
				t.Fatalf("unexpected log message: got %s, want %s", lr.Msg, tt.wantsMsg)
			}
			stack, ok := lr.Attrs["stacktrace"].(string)
			if !ok || stack == "" {
				t.Fatalf("stacktrace missing or invalid: %v", lr.Attrs["stacktrace"])
			}
			if got := strings.Count(stack, "\n"); got < 1 {
				t.Fatalf("stacktrace lines too short: got %d", got)
			}
			delete(lr.Attrs, "stacktrace")
			if diff := cmp.Diff(tt.wants, lr.Attrs); diff != "" {
				t.Fatalf("unexpected log record attrs (-want +got):\n%s", diff)
			}
		})
	}
}
