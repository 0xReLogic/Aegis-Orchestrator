package monitor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusExporter exports metrics to Prometheus
type PrometheusExporter struct {
	registry *prometheus.Registry
	server   *http.Server

	// Metrics
	serviceStatus      *prometheus.GaugeVec
	serviceUptime      *prometheus.GaugeVec
	serviceRestarts    *prometheus.CounterVec
	healthCheckCount   *prometheus.CounterVec
	healthCheckErrors  *prometheus.CounterVec
	healthCheckLatency *prometheus.HistogramVec
	lastCheckTimestamp *prometheus.GaugeVec
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter() *PrometheusExporter {
	registry := prometheus.NewRegistry()

	exporter := &PrometheusExporter{
		registry: registry,
		serviceStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "aegis_service_status",
				Help: "Current status of the service (1=up, 0=down)",
			},
			[]string{"service", "type"},
		),
		serviceUptime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "aegis_service_uptime_seconds",
				Help: "Uptime of the service in seconds",
			},
			[]string{"service"},
		),
		serviceRestarts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aegis_service_restarts_total",
				Help: "Total number of service restarts",
			},
			[]string{"service"},
		),
		healthCheckCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aegis_health_checks_total",
				Help: "Total number of health checks",
			},
			[]string{"service", "type", "result"},
		),
		healthCheckErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aegis_health_check_errors_total",
				Help: "Total number of health check errors",
			},
			[]string{"service", "type", "error_type"},
		),
		healthCheckLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "aegis_health_check_duration_seconds",
				Help:    "Duration of health checks in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "type"},
		),
		lastCheckTimestamp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "aegis_last_health_check_timestamp",
				Help: "Timestamp of the last health check",
			},
			[]string{"service", "type"},
		),
	}

	// Register metrics with the registry
	registry.MustRegister(
		exporter.serviceStatus,
		exporter.serviceUptime,
		exporter.serviceRestarts,
		exporter.healthCheckCount,
		exporter.healthCheckErrors,
		exporter.healthCheckLatency,
		exporter.lastCheckTimestamp,
	)

	return exporter
}

// Start starts the Prometheus HTTP server
func (p *PrometheusExporter) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()

	// Add Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{}))

	// Add a simple health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"UP"}`))
	})

	p.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	logger.Info("Starting Prometheus metrics server on %s", addr)
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Prometheus metrics server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the Prometheus HTTP server
func (p *PrometheusExporter) Stop() error {
	if p.server != nil {
		logger.Info("Stopping Prometheus metrics server")
		return p.server.Close()
	}
	return nil
}

// RecordHealthCheck records a health check result
func (p *PrometheusExporter) RecordHealthCheck(serviceName, checkType string, result *plugin.HealthCheckResult, duration time.Duration) {
	// Record health check count
	status := "unknown"
	if result != nil {
		status = string(result.Status)
	}
	p.healthCheckCount.WithLabelValues(serviceName, checkType, status).Inc()

	// Record health check latency
	p.healthCheckLatency.WithLabelValues(serviceName, checkType).Observe(duration.Seconds())

	// Record last check timestamp
	p.lastCheckTimestamp.WithLabelValues(serviceName, checkType).Set(float64(time.Now().Unix()))

	// Update service status if we have a result
	if result != nil {
		if result.Status == plugin.StatusUp {
			p.serviceStatus.WithLabelValues(serviceName, checkType).Set(1)
		} else {
			p.serviceStatus.WithLabelValues(serviceName, checkType).Set(0)
		}
	}
}

// RecordHealthCheckError records a health check error
func (p *PrometheusExporter) RecordHealthCheckError(serviceName, checkType, errorType string) {
	p.healthCheckErrors.WithLabelValues(serviceName, checkType, errorType).Inc()
}

// RecordServiceRestart records a service restart
func (p *PrometheusExporter) RecordServiceRestart(serviceName string) {
	p.serviceRestarts.WithLabelValues(serviceName).Inc()
}

// UpdateServiceUptime updates the service uptime
func (p *PrometheusExporter) UpdateServiceUptime(serviceName string, startTime time.Time) {
	if !startTime.IsZero() {
		uptime := time.Since(startTime).Seconds()
		p.serviceUptime.WithLabelValues(serviceName).Set(uptime)
	}
}
