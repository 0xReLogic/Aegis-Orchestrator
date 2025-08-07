package monitor

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
)

// MetricsCollector collects and exposes metrics
type MetricsCollector struct {
	serviceStatus     map[string]string
	serviceUptime     map[string]time.Time
	serviceRestarts   map[string]int
	healthCheckCounts map[string]int
	healthCheckErrors map[string]int
	mu                sync.RWMutex
	server            *http.Server
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		serviceStatus:     make(map[string]string),
		serviceUptime:     make(map[string]time.Time),
		serviceRestarts:   make(map[string]int),
		healthCheckCounts: make(map[string]int),
		healthCheckErrors: make(map[string]int),
	}
}

// Start starts the metrics server
func (m *MetricsCollector) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", m.handleMetrics)
	mux.HandleFunc("/health", m.handleHealth)

	addr := fmt.Sprintf(":%d", port)
	m.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	logger.Info("Starting metrics server on %s", addr)
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Metrics server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the metrics server
func (m *MetricsCollector) Stop() error {
	if m.server != nil {
		logger.Info("Stopping metrics server")
		return m.server.Close()
	}
	return nil
}

// UpdateServiceStatus updates the status of a service
func (m *MetricsCollector) UpdateServiceStatus(serviceName, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serviceStatus[serviceName] = status
}

// UpdateServiceUptime updates the uptime of a service
func (m *MetricsCollector) UpdateServiceUptime(serviceName string, startTime time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serviceUptime[serviceName] = startTime
}

// IncrementServiceRestart increments the restart count for a service
func (m *MetricsCollector) IncrementServiceRestart(serviceName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serviceRestarts[serviceName]++
}

// IncrementHealthCheckCount increments the health check count for a service
func (m *MetricsCollector) IncrementHealthCheckCount(serviceName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthCheckCounts[serviceName]++
}

// IncrementHealthCheckError increments the health check error count for a service
func (m *MetricsCollector) IncrementHealthCheckError(serviceName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthCheckErrors[serviceName]++
}

// handleMetrics handles the /metrics endpoint
func (m *MetricsCollector) handleMetrics(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain")

	// Write service status metrics
	fmt.Fprintln(w, "# HELP aegis_service_status Current status of the service (1=up, 0=down)")
	fmt.Fprintln(w, "# TYPE aegis_service_status gauge")
	for service, status := range m.serviceStatus {
		value := 0
		if status == "UP" {
			value = 1
		}
		fmt.Fprintf(w, "aegis_service_status{service=\"%s\"} %d\n", service, value)
	}

	// Write service uptime metrics
	fmt.Fprintln(w, "# HELP aegis_service_uptime_seconds Uptime of the service in seconds")
	fmt.Fprintln(w, "# TYPE aegis_service_uptime_seconds gauge")
	now := time.Now()
	for service, startTime := range m.serviceUptime {
		if !startTime.IsZero() {
			uptime := now.Sub(startTime).Seconds()
			fmt.Fprintf(w, "aegis_service_uptime_seconds{service=\"%s\"} %f\n", service, uptime)
		}
	}

	// Write service restart metrics
	fmt.Fprintln(w, "# HELP aegis_service_restarts_total Total number of service restarts")
	fmt.Fprintln(w, "# TYPE aegis_service_restarts_total counter")
	for service, count := range m.serviceRestarts {
		fmt.Fprintf(w, "aegis_service_restarts_total{service=\"%s\"} %d\n", service, count)
	}

	// Write health check metrics
	fmt.Fprintln(w, "# HELP aegis_health_checks_total Total number of health checks")
	fmt.Fprintln(w, "# TYPE aegis_health_checks_total counter")
	for service, count := range m.healthCheckCounts {
		fmt.Fprintf(w, "aegis_health_checks_total{service=\"%s\"} %d\n", service, count)
	}

	// Write health check error metrics
	fmt.Fprintln(w, "# HELP aegis_health_check_errors_total Total number of health check errors")
	fmt.Fprintln(w, "# TYPE aegis_health_check_errors_total counter")
	for service, count := range m.healthCheckErrors {
		fmt.Fprintf(w, "aegis_health_check_errors_total{service=\"%s\"} %d\n", service, count)
	}
}

// handleHealth handles the /health endpoint
func (m *MetricsCollector) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"UP"}`))
}
