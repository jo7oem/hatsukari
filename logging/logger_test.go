package logging_test

import (
	"encoding/json"
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

func newHandler() *capturingHandler {
	ch := &capturingHandler{}
	ch.handler = slog.NewJSONHandler(&ch.buf, nil)

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
			cph := newHandler()
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
