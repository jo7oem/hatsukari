package site

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestSite_ServeHTTP_HTTPRequestsTotal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantMethod string
		wantCode   string
	}{
		{name: "OK", path: "/", wantStatus: http.StatusOK, wantMethod: http.MethodGet, wantCode: "200"},
		{name: "NotFound", path: "/missing", wantStatus: http.StatusNotFound, wantMethod: http.MethodGet, wantCode: "404"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			metrics, err := newSiteMetrics(provider.Meter("test/site"))
			if err != nil {
				t.Fatalf("newSiteMetrics() error = %v", err)
			}

			s := newTestSiteForAccessLog(t, newDiscardLogger())
			s.metrics = metrics

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			s.ServeHTTP(rec, req)
			if got := rec.Code; got != tt.wantStatus {
				t.Fatalf("ServeHTTP() status = %d, want %d", got, tt.wantStatus)
			}

			httpRequests := collectMetricData(t, reader, "hatsukari_http_requests_total")
			assertHTTPRequestsMetric(t, httpRequests, tt.wantMethod, tt.wantCode)
		})
	}
}

func TestSite_NewSiteMetrics_RuntimeGauges(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	if _, err := newSiteMetrics(provider.Meter("test/site")); err != nil {
		t.Fatalf("newSiteMetrics() error = %v", err)
	}

	goroutinesMetric := collectMetricData(t, reader, "hatsukari_runtime_goroutines")
	goroutinesGauge, ok := goroutinesMetric.Data.(metricdata.Gauge[int64])
	if !ok {
		t.Fatalf("goroutines metric type = %T, want metricdata.Gauge[int64]", goroutinesMetric.Data)
	}
	if len(goroutinesGauge.DataPoints) != 1 {
		t.Fatalf("goroutines datapoint count = %d, want 1", len(goroutinesGauge.DataPoints))
	}
	if got := goroutinesGauge.DataPoints[0].Value; got <= 0 {
		t.Fatalf("goroutines value = %d, want > 0", got)
	}

	heapMetric := collectMetricData(t, reader, "hatsukari_runtime_heap_alloc_bytes")
	heapGauge, ok := heapMetric.Data.(metricdata.Gauge[int64])
	if !ok {
		t.Fatalf("heap metric type = %T, want metricdata.Gauge[int64]", heapMetric.Data)
	}
	if len(heapGauge.DataPoints) != 1 {
		t.Fatalf("heap datapoint count = %d, want 1", len(heapGauge.DataPoints))
	}
	if got := heapGauge.DataPoints[0].Value; got < 0 {
		t.Fatalf("heap value = %d, want >= 0", got)
	}
}

func collectMetricData(t *testing.T, reader *sdkmetric.ManualReader, name string) metricdata.Metrics {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &resourceMetrics); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		for _, metric := range scopeMetrics.Metrics {
			if metric.Name == name {
				return metric
			}
		}
	}

	t.Fatalf("metric %q not found", name)
	return metricdata.Metrics{}
}

func attributeMap(set attribute.Set) map[string]string {
	attrs := make(map[string]string)
	for _, kv := range set.ToSlice() {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}
	return attrs
}

func assertHTTPRequestsMetric(t *testing.T, metricData metricdata.Metrics, wantMethod, wantCode string) {
	t.Helper()

	sum, ok := metricData.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("metric data type = %T, want metricdata.Sum[int64]", metricData.Data)
	}
	if len(sum.DataPoints) != 1 {
		t.Fatalf("datapoint count = %d, want 1", len(sum.DataPoints))
	}
	dp := sum.DataPoints[0]
	if dp.Value != 1 {
		t.Fatalf("counter value = %d, want 1", dp.Value)
	}
	attrs := attributeMap(dp.Attributes)
	if got := attrs["http.method"]; got != wantMethod {
		t.Fatalf("http.method = %q, want %q", got, wantMethod)
	}
	if got := attrs["http.status_code"]; got != wantCode {
		t.Fatalf("http.status_code = %q, want %q", got, wantCode)
	}
}
