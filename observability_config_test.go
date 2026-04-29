package main

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

type collectorConfig struct {
	Exporters map[string]map[string]any `yaml:"exporters"`
	Service   struct {
		Pipelines map[string]struct {
			Receivers  []string `yaml:"receivers"`
			Processors []string `yaml:"processors"`
			Exporters  []string `yaml:"exporters"`
		} `yaml:"pipelines"`
	} `yaml:"service"`
}
type prometheusConfig struct {
	Global struct {
		ScrapeInterval     string `yaml:"scrape_interval"`
		EvaluationInterval string `yaml:"evaluation_interval"`
	} `yaml:"global"`
	ScrapeConfigs []struct {
		JobName       string `yaml:"job_name"`
		StaticConfigs []struct {
			Targets []string `yaml:"targets"`
		} `yaml:"static_configs"`
	} `yaml:"scrape_configs"`
}
type grafanaDatasourceConfig struct {
	Datasources []struct {
		Name      string `yaml:"name"`
		UID       string `yaml:"uid"`
		Type      string `yaml:"type"`
		URL       string `yaml:"url"`
		IsDefault bool   `yaml:"isDefault"`
	} `yaml:"datasources"`
}
type grafanaDashboard struct {
	Panels []struct {
		Title   string `json:"title"`
		Targets []struct {
			Expr string `json:"expr"`
		} `json:"targets"`
	} `json:"panels"`
	Title string `json:"title"`
	UID   string `json:"uid"`
}

func TestObservability_CollectorConfig(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("otel/collector-config.yaml")
	if err != nil {
		t.Fatalf("failed to read collector config: %v", err)
	}
	var cfg collectorConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("failed to parse collector config: %v", err)
	}
	traces, ok := cfg.Service.Pipelines["traces"]
	if !ok {
		t.Fatal("traces pipeline missing")
	}
	if len(traces.Exporters) == 0 {
		t.Fatal("traces pipeline exporters missing")
	}
	metrics, ok := cfg.Service.Pipelines["metrics"]
	if !ok {
		t.Fatal("metrics pipeline missing")
	}
	if len(metrics.Exporters) != 1 || metrics.Exporters[0] != "prometheus" {
		t.Fatalf("metrics exporters = %v, want [prometheus]", metrics.Exporters)
	}
	if _, ok := cfg.Exporters["prometheus"]; !ok {
		t.Fatal("prometheus exporter missing")
	}
}
func TestObservability_PrometheusConfig(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("otel/prometheus/prometheus.yml")
	if err != nil {
		t.Fatalf("failed to read prometheus config: %v", err)
	}
	var cfg prometheusConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("failed to parse prometheus config: %v", err)
	}
	if cfg.Global.ScrapeInterval != "5s" {
		t.Fatalf("scrape_interval = %q, want 5s", cfg.Global.ScrapeInterval)
	}
	if len(cfg.ScrapeConfigs) != 1 {
		t.Fatalf("scrape config count = %d, want 1", len(cfg.ScrapeConfigs))
	}
	if got := cfg.ScrapeConfigs[0].JobName; got != "otel-collector" {
		t.Fatalf("job_name = %q, want otel-collector", got)
	}
	if len(cfg.ScrapeConfigs[0].StaticConfigs) != 1 {
		t.Fatalf("static_configs count = %d, want 1", len(cfg.ScrapeConfigs[0].StaticConfigs))
	}
	targets := cfg.ScrapeConfigs[0].StaticConfigs[0].Targets
	if len(targets) != 1 || targets[0] != "otel-collector:9464" {
		t.Fatalf("targets = %v, want [otel-collector:9464]", targets)
	}
}
func TestObservability_GrafanaDatasource(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("otel/grafana/provisioning/datasources/prometheus.yaml")
	if err != nil {
		t.Fatalf("failed to read grafana datasource config: %v", err)
	}
	var cfg grafanaDatasourceConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("failed to parse grafana datasource config: %v", err)
	}
	if len(cfg.Datasources) != 1 {
		t.Fatalf("datasource count = %d, want 1", len(cfg.Datasources))
	}
	ds := cfg.Datasources[0]
	if ds.Name != "Prometheus" {
		t.Fatalf("datasource name = %q, want Prometheus", ds.Name)
	}
	if ds.UID != "prometheus" {
		t.Fatalf("datasource uid = %q, want prometheus", ds.UID)
	}
	if ds.Type != "prometheus" {
		t.Fatalf("datasource type = %q, want prometheus", ds.Type)
	}
	if ds.URL != "http://prometheus:9090" {
		t.Fatalf("datasource url = %q, want http://prometheus:9090", ds.URL)
	}
	if !ds.IsDefault {
		t.Fatal("prometheus datasource should be default")
	}
}
func TestObservability_GrafanaDashboard(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("otel/grafana/provisioning/dashboards/json/hatsukari-metrics.json")
	if err != nil {
		t.Fatalf("failed to read grafana dashboard: %v", err)
	}
	var dashboard grafanaDashboard
	if err := json.Unmarshal(b, &dashboard); err != nil {
		t.Fatalf("failed to parse grafana dashboard: %v", err)
	}
	if dashboard.Title != "hatsukari Metrics" {
		t.Fatalf("dashboard title = %q, want hatsukari Metrics", dashboard.Title)
	}
	if dashboard.UID != "hatsukari-metrics" {
		t.Fatalf("dashboard uid = %q, want hatsukari-metrics", dashboard.UID)
	}
	if len(dashboard.Panels) != 3 {
		t.Fatalf("panel count = %d, want 3", len(dashboard.Panels))
	}
	wantExprs := map[string]string{
		"HTTP Requests by Status": "sum by (http_status_code) (rate(hatsukari_http_requests_total[5m]))",
		"Goroutines":              "hatsukari_runtime_goroutines",
		"Heap Alloc Bytes":        "hatsukari_runtime_heap_alloc_bytes",
	}
	for _, panel := range dashboard.Panels {
		wantExpr, ok := wantExprs[panel.Title]
		if !ok {
			t.Fatalf("unexpected panel title: %s", panel.Title)
		}
		if len(panel.Targets) != 1 {
			t.Fatalf("panel %s target count = %d, want 1", panel.Title, len(panel.Targets))
		}
		if got := panel.Targets[0].Expr; got != wantExpr {
			t.Fatalf("panel %s expr = %q, want %q", panel.Title, got, wantExpr)
		}
	}
}
