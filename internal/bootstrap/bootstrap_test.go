package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/jo7oem/hatsukari/logging"
	"github.com/jo7oem/hatsukari/telemetry"
)

type fakeSiteHandler struct{}

func (fakeSiteHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}
func (fakeSiteHandler) Close() error                                 { return nil }

func TestBootstrap_Start(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		config            RuntimeConfig
		wantInitTelemetry bool
	}{
		{
			name:              "OTELDisabled",
			config:            RuntimeConfig{SiteDir: "./sample", Addr: ":8080", OTELEnabled: false},
			wantInitTelemetry: false,
		},
		{
			name:              "OTELEnabled",
			config:            RuntimeConfig{SiteDir: "./sample", Addr: ":8080", OTELEnabled: true, OTELEndpoint: "collector:4317", OTELInsecure: true},
			wantInitTelemetry: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			telemetryCalled := false
			listenCalled := false

			err := Start(context.Background(), io.Discard, tt.config, Dependencies{
				InitTelemetry: func(context.Context, telemetry.Config) (func(context.Context) error, error) {
					telemetryCalled = true
					return func(context.Context) error { return nil }, nil
				},
				NewLogger: func(io.Writer) *logging.Logger {
					return logging.NewLogger(slog.NewTextHandler(io.Discard, nil), "test")
				},
				OpenSite: func(string, *logging.Logger) (SiteHandler, error) {
					return fakeSiteHandler{}, nil
				},
				ListenAndServe: func(string, http.Handler) error {
					listenCalled = true
					return nil
				},
			})
			if err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			if telemetryCalled != tt.wantInitTelemetry {
				t.Fatalf("init telemetry called = %v, want %v", telemetryCalled, tt.wantInitTelemetry)
			}
			if !listenCalled {
				t.Fatal("listen was not called")
			}
		})
	}
}
