package telemetry

import (
	"context"
	"runtime"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	defaultSiteMeter = otel.Meter("hatsukari/site")

	siteMetricsInitOnce sync.Once
	siteMetricsInitErr  error
	defaultSiteMetrics  *SiteMetrics
)

type SiteMetrics struct {
	httpRequestsTotal metric.Int64Counter
}

func InitSiteMetrics() (*SiteMetrics, error) {
	siteMetricsInitOnce.Do(func() {
		defaultSiteMetrics, siteMetricsInitErr = NewSiteMetrics(defaultSiteMeter)
	})

	return defaultSiteMetrics, siteMetricsInitErr
}

func NewSiteMetrics(meter metric.Meter) (*SiteMetrics, error) {
	if meter == nil {
		meter = defaultSiteMeter
	}

	httpRequestsTotal, err := meter.Int64Counter(
		"hatsukari_http_requests_total",
		metric.WithDescription("HTTP requests total grouped by method and status code"),
	)
	if err != nil {
		return nil, err
	}

	goRoutinesGauge, err := meter.Int64ObservableGauge(
		"hatsukari_runtime_goroutines",
		metric.WithDescription("Number of goroutines"),
	)
	if err != nil {
		return nil, err
	}

	heapAllocGauge, err := meter.Int64ObservableGauge(
		"hatsukari_runtime_heap_alloc_bytes",
		metric.WithDescription("Allocated heap bytes"),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		observer.ObserveInt64(goRoutinesGauge, int64(runtime.NumGoroutine()))
		observer.ObserveInt64(heapAllocGauge, int64(ms.HeapAlloc))
		return nil
	}, goRoutinesGauge, heapAllocGauge)
	if err != nil {
		return nil, err
	}

	return &SiteMetrics{httpRequestsTotal: httpRequestsTotal}, nil
}

func (m *SiteMetrics) RecordHTTPRequest(ctx context.Context, method string, statusCode int) {
	if m == nil || m.httpRequestsTotal == nil {
		return
	}
	m.httpRequestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.status_code", strconv.Itoa(statusCode)),
		),
	)
}
