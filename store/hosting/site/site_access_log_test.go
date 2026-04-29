package site

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/jo7oem/hatsukari/logging"
)

type capturedJSONLog struct {
	Time      string                 `json:"time"`
	Level     string                 `json:"level"`
	Msg       string                 `json:"msg"`
	Namespace string                 `json:"namespace"`
	Access    map[string]any         `json:"access"`
	Attrs     map[string]interface{} `json:"-"`
}

func newCapturedLogger(buf *bytes.Buffer) *logging.Logger {
	return logging.NewLogger(slog.NewJSONHandler(buf, nil), "test")
}

func decodeCapturedJSONLog(t *testing.T, buf *bytes.Buffer) capturedJSONLog {
	t.Helper()

	var record capturedJSONLog
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to decode log record: %v\nbody=%s", err, buf.String())
	}

	return record
}

func newTestSiteForAccessLog(t *testing.T, logger *logging.Logger) *Site {
	t.Helper()

	siteDir := t.TempDir()
	writeTestFile(t, filepath.Join(siteDir, ".site.yml"), "title: \"access\"\ntimezone: \"UTC\"\nrootContentDir: \"content\"\n")
	writeTestFile(t, filepath.Join(siteDir, "content", ".content.yaml"), "templatesDir: templates\ncontentTemplate: template.md\ncontentsDir: []\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "templates", "template.md"), "BODY={{.contents.body}}\n")
	writeTestFile(t, filepath.Join(siteDir, "content", "index.md"), "hello\n")

	s, err := OpenSiteDir(siteDir, logger)
	if err != nil {
		t.Fatalf("OpenSiteDir() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	return s
}

func TestSite_ServeHTTP_AccessLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		userAgent     string
		remoteAddr    string
		wantStatus    int
		wantPath      string
		wantUserAgent string
	}{
		{
			name:          "StatusDefault200",
			path:          "/",
			userAgent:     "test-agent/1.0",
			remoteAddr:    "203.0.113.10:12345",
			wantStatus:    http.StatusOK,
			wantPath:      "/",
			wantUserAgent: "test-agent/1.0",
		},
		{
			name:          "NotFound",
			path:          "/missing",
			userAgent:     "crawler/2.0",
			remoteAddr:    "198.51.100.8:54321",
			wantStatus:    http.StatusNotFound,
			wantPath:      "/missing",
			wantUserAgent: "crawler/2.0",
		},
		{
			name:          "UserAgentEmpty",
			path:          "/",
			remoteAddr:    "192.0.2.5:9999",
			wantStatus:    http.StatusOK,
			wantPath:      "/",
			wantUserAgent: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logBuffer bytes.Buffer
			logger := newCapturedLogger(&logBuffer)
			s := newTestSiteForAccessLog(t, logger)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}

			s.ServeHTTP(rec, req)

			if got := rec.Code; got != tt.wantStatus {
				t.Fatalf("ServeHTTP() status = %d, want %d", got, tt.wantStatus)
			}

			record := decodeCapturedJSONLog(t, &logBuffer)
			if record.Level != "INFO" {
				t.Fatalf("log level = %s, want INFO", record.Level)
			}
			if record.Msg != "request" {
				t.Fatalf("log msg = %s, want request", record.Msg)
			}
			if record.Access == nil {
				t.Fatal("access group missing")
			}
			if got := record.Access["namespace"]; got != "test" {
				t.Fatalf("access.namespace = %v, want test", got)
			}

			if got := record.Access["method"]; got != http.MethodGet {
				t.Fatalf("access.method = %v, want %s", got, http.MethodGet)
			}
			if got := record.Access["path"]; got != tt.wantPath {
				t.Fatalf("access.path = %v, want %s", got, tt.wantPath)
			}
			if got := int(record.Access["status"].(float64)); got != tt.wantStatus {
				t.Fatalf("access.status = %d, want %d", got, tt.wantStatus)
			}
			if got := record.Access["userAgent"]; got != tt.wantUserAgent {
				t.Fatalf("access.userAgent = %v, want %q", got, tt.wantUserAgent)
			}
			if got := record.Access["remoteAddr"]; got == nil || got == "" {
				t.Fatalf("access.remoteAddr = %v, want non-empty", got)
			}
			if got := record.Access["durationMs"]; got == nil {
				t.Fatal("access.durationMs missing")
			}
			if got := record.Access["responseBytes"]; got == nil {
				t.Fatal("access.responseBytes missing")
			}
		})
	}
}

func TestSite_remoteAddrHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{name: "WithPort", remoteAddr: "203.0.113.10:8080", want: "203.0.113.10"},
		{name: "WithoutPort", remoteAddr: "203.0.113.10", want: "203.0.113.10"},
		{name: "IPv6WithPort", remoteAddr: "[2001:db8::1]:443", want: "2001:db8::1"},
		{name: "Trimmed", remoteAddr: " 198.51.100.4 ", want: "198.51.100.4"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := remoteAddrHost(tt.remoteAddr); got != tt.want {
				t.Fatalf("remoteAddrHost(%q) = %q, want %q", tt.remoteAddr, got, tt.want)
			}
		})
	}
}

func TestSite_ServeHTTP_AccessLog_ResponseBytesPositive(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := newCapturedLogger(&logBuffer)
	s := newTestSiteForAccessLog(t, logger)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.20:3456"
	req.Header.Set("User-Agent", "bytes-check/1.0")

	s.ServeHTTP(rec, req)

	record := decodeCapturedJSONLog(t, &logBuffer)
	responseBytes, ok := record.Access["responseBytes"].(float64)
	if !ok {
		t.Fatalf("access.responseBytes type = %T, want number", record.Access["responseBytes"])
	}
	if responseBytes <= 0 {
		body, _ := io.ReadAll(rec.Result().Body)
		t.Fatalf("access.responseBytes = %v, want > 0, body=%q", responseBytes, string(body))
	}
}
