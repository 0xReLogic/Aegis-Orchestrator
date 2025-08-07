package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/internal/config"
	"github.com/0xReLogic/Aegis-Orchestrator/internal/health"
	"github.com/0xReLogic/Aegis-Orchestrator/internal/monitor"
	"github.com/0xReLogic/Aegis-Orchestrator/internal/service"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/plugin"
)

// EventType represents the type of event in the system
type EventType string

const (
	// EventServiceHealthy indicates a service is healthy
	EventServiceHealthy EventType = "SERVICE_HEALTHY"
	// EventServiceUnhealthy indicates a service is unhealthy
	EventServiceUnhealthy EventType = "SERVICE_UNHEALTHY"
	// EventServiceRecovered indicates a service has recovered from an unhealthy state
	EventServiceRecovered EventType = "SERVICE_RECOVERED"
	// EventServiceRestarted indicates a service has been restarted
	EventServiceRestarted EventType = "SERVICE_RESTARTED"
	// EventServiceFailed indicates a service has failed to restart
	EventServiceFailed EventType = "SERVICE_FAILED"
)

// Event represents an event in the system
type Event struct {
	Type      EventType
	ServiceID string
	Timestamp time.Time
	Data      map[string]interface{}
}

// Engine is the core orchestration engine
type Engine struct {
	config         *config.Config
	serviceManager *service.Manager
	pluginRegistry *plugin.Registry
	eventChan      chan Event
	healthCheckers map[string]plugin.HealthChecker
	serviceStates  map[string]plugin.Status
	stopChan       chan struct{}
	wg             sync.WaitGroup
	mu             sync.RWMutex

	// Phase 2 additions
	prometheusExporter *monitor.PrometheusExporter
	logAnalyzer        *monitor.LogAnalyzer
	anomalyDetector    *monitor.AnomalyDetector
	alertManager       *monitor.AlertManager
	retryPolicies      map[string]*health.RetryPolicy
}

// NewEngine creates a new orchestration engine
func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		config:         cfg,
		serviceManager: service.NewManager(),
		pluginRegistry: plugin.NewRegistry(),
		eventChan:      make(chan Event, 100),
		healthCheckers: make(map[string]plugin.HealthChecker),
		serviceStates:  make(map[string]plugin.Status),
		stopChan:       make(chan struct{}),

		// Phase 2 additions
		prometheusExporter: monitor.NewPrometheusExporter(),
		logAnalyzer:        monitor.NewLogAnalyzer(),
		anomalyDetector:    monitor.NewAnomalyDetector(100, 30*time.Second),
		alertManager:       monitor.NewAlertManager(),
		retryPolicies:      make(map[string]*health.RetryPolicy),
	}
}

// Initialize initializes the engine
func (e *Engine) Initialize() error {
	// Register health checkers
	for _, svc := range e.config.Services {
		// Register service with the service manager
		if err := e.serviceManager.RegisterService(svc); err != nil {
			return fmt.Errorf("failed to register service '%s': %w", svc.Name, err)
		}

		// Create and register health checker based on service type
		var checker plugin.HealthChecker
		switch svc.Type {
		case "http":
			checker = health.NewHTTPHealthChecker(svc.Name, svc.Endpoint, svc.Timeout)
		case "tcp":
			checker = health.NewTCPHealthChecker(svc.Name, svc.Endpoint, svc.Timeout)
		case "grpc":
			checker = health.NewGRPCHealthChecker(svc.Name, svc.Endpoint, svc.Timeout)
		case "script":
			checker = health.NewScriptHealthChecker(svc.Name, svc.Endpoint, svc.Timeout)
		default:
			return fmt.Errorf("unsupported service type for '%s': %s", svc.Name, svc.Type)
		}

		// Register the health checker
		if err := e.pluginRegistry.Register(checker); err != nil {
			return fmt.Errorf("failed to register health checker for '%s': %w", svc.Name, err)
		}

		e.healthCheckers[svc.Name] = checker
		e.serviceStates[svc.Name] = plugin.StatusUnknown

		// Create retry policy for the service
		retryPolicy := health.DefaultRetryPolicy()
		if svc.Retries > 0 {
			retryPolicy.MaxRetries = svc.Retries
		}
		e.retryPolicies[svc.Name] = retryPolicy

		// Set up anomaly detection thresholds
		if svc.Metadata != nil {
			if latencyThreshold, ok := svc.Metadata["latency_threshold"]; ok {
				var threshold float64
				if _, err := fmt.Sscanf(latencyThreshold, "%f", &threshold); err == nil {
					e.anomalyDetector.SetLatencyThreshold(svc.Name, threshold)
					logger.Info("Set latency threshold for service %s: %.2f ms", svc.Name, threshold)
				}
			}

			if errorRateThreshold, ok := svc.Metadata["error_rate_threshold"]; ok {
				var threshold float64
				if _, err := fmt.Sscanf(errorRateThreshold, "%f", &threshold); err == nil {
					e.anomalyDetector.SetErrorRateThreshold(svc.Name, threshold)
					logger.Info("Set error rate threshold for service %s: %.2f%%", svc.Name, threshold*100)
				}
			}
		}
	}

	// Initialize Prometheus exporter
	if e.config.Global.MetricsEnabled {
		if err := e.prometheusExporter.Start(e.config.Global.MetricsPort); err != nil {
			return fmt.Errorf("failed to start Prometheus exporter: %w", err)
		}
	}

	// Initialize log analyzer if enabled
	if e.config.Logs.Enabled {
		// Configure log files
		for _, logFile := range e.config.Logs.Files {
			if err := e.logAnalyzer.AddLogFile(logFile.Service, logFile.Path); err != nil {
				logger.Warn("Failed to add log file for service %s: %v", logFile.Service, err)
			} else {
				logger.Info("Added log file for service %s: %s", logFile.Service, logFile.Path)
			}
		}

		// Configure log patterns
		for _, pattern := range e.config.Logs.Patterns {
			if err := e.logAnalyzer.AddPattern(pattern.Name, pattern.Pattern, pattern.Severity, pattern.Description); err != nil {
				logger.Warn("Failed to add log pattern %s: %v", pattern.Name, err)
			} else {
				logger.Info("Added log pattern: %s (%s)", pattern.Name, pattern.Pattern)
			}
		}

		// Start log analyzer
		e.logAnalyzer.Start(e.config.Logs.ScanInterval)
		logger.Info("Started log analyzer with interval %s", e.config.Logs.ScanInterval)
	} else {
		// Add default patterns for backward compatibility
		e.logAnalyzer.AddPattern("OutOfMemory", "(?i)(out of memory|java\\.lang\\.OutOfMemoryError)", "CRITICAL", "Out of memory error detected")
		e.logAnalyzer.AddPattern("HighCPU", "(?i)(high cpu usage|cpu at \\d{2,3}%)", "WARNING", "High CPU usage detected")
		e.logAnalyzer.AddPattern("DatabaseError", "(?i)(database error|sql error|connection refused|timeout)", "WARNING", "Database error detected")
		e.logAnalyzer.AddPattern("Exception", "(?i)(exception|error|fatal|panic)", "WARNING", "Exception detected")

		// Start with default interval
		e.logAnalyzer.Start(1 * time.Minute)
	}

	// Initialize anomaly detector if enabled
	if e.config.AnomalyDetection.Enabled {
		// Configure thresholds from config
		// Latency thresholds
		for _, threshold := range e.config.AnomalyDetection.Thresholds.Latency {
			e.anomalyDetector.SetLatencyThreshold(threshold.Service, threshold.Value)
			logger.Info("Set latency threshold for service %s: %.2f ms", threshold.Service, threshold.Value)
		}

		// Error rate thresholds
		for _, threshold := range e.config.AnomalyDetection.Thresholds.ErrorRate {
			e.anomalyDetector.SetErrorRateThreshold(threshold.Service, threshold.Value)
			logger.Info("Set error rate threshold for service %s: %.2f%%", threshold.Service, threshold.Value*100)
		}

		// Throughput thresholds
		for _, threshold := range e.config.AnomalyDetection.Thresholds.Throughput {
			e.anomalyDetector.SetThroughputThreshold(threshold.Service, threshold.Value)
			logger.Info("Set throughput threshold for service %s: %.2f req/s", threshold.Service, threshold.Value)
		}

		// Start anomaly detector with configured interval
		e.anomalyDetector = monitor.NewAnomalyDetector(
			e.config.AnomalyDetection.HistorySize,
			e.config.AnomalyDetection.DetectionInterval,
		)
		e.anomalyDetector.Start()
		logger.Info("Started anomaly detector with interval %s", e.config.AnomalyDetection.DetectionInterval)
	} else {
		// Start with default settings
		e.anomalyDetector.Start()
	}

	// Initialize alert manager
	e.alertManager.Start()

	// Set up webhook if configured
	if e.config.Notifications.Webhook.Enabled {
		e.alertManager.SetWebhook(e.config.Notifications.Webhook.Endpoint, e.config.Notifications.Webhook.Headers)
	}

	return nil
}

// Start starts the engine
func (e *Engine) Start(ctx context.Context) error {
	logger.Info("Starting Aegis-Orchestrator engine")

	// Start the event processor
	e.wg.Add(1)
	go e.processEvents(ctx)

	// Start health check loops for each service
	for _, svc := range e.config.Services {
		e.wg.Add(1)
		go e.runHealthCheckLoop(ctx, svc)
	}

	logger.Info("Aegis-Orchestrator engine started")
	return nil
}

// Stop stops the engine
func (e *Engine) Stop() {
	logger.Info("Stopping Aegis-Orchestrator engine")
	close(e.stopChan)
	e.wg.Wait()

	// Stop Phase 2 components
	if e.config.Global.MetricsEnabled {
		e.prometheusExporter.Stop()
	}
	e.logAnalyzer.Stop()
	e.anomalyDetector.Stop()
	e.alertManager.Stop()

	logger.Info("Aegis-Orchestrator engine stopped")
}

// runHealthCheckLoop runs a health check loop for a service
func (e *Engine) runHealthCheckLoop(ctx context.Context, svc config.ServiceConfig) {
	defer e.wg.Done()

	ticker := time.NewTicker(svc.Interval)
	defer ticker.Stop()

	logger.Info("Starting health check loop for service: %s", svc.Name)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping health check loop for service: %s", svc.Name)
			return
		case <-e.stopChan:
			logger.Info("Stopping health check loop for service: %s", svc.Name)
			return
		case <-ticker.C:
			e.performHealthCheck(svc)
		}
	}
}

// performHealthCheck performs a health check for a service
func (e *Engine) performHealthCheck(svc config.ServiceConfig) {
	checker, exists := e.healthCheckers[svc.Name]
	if !exists {
		logger.Error("Health checker not found for service: %s", svc.Name)
		return
	}

	startTime := time.Now()

	// Get retry policy for the service
	retryPolicy, exists := e.retryPolicies[svc.Name]
	if !exists {
		retryPolicy = health.DefaultRetryPolicy()
	}

	// Perform health check with retry policy
	result, err := health.HealthCheckWithRetry(checker, retryPolicy)

	// Calculate duration
	duration := time.Since(startTime)

	// Record metrics
	if e.config.Global.MetricsEnabled {
		e.prometheusExporter.RecordHealthCheck(svc.Name, checker.Type(), result, duration)

		if err != nil {
			e.prometheusExporter.RecordHealthCheckError(svc.Name, checker.Type(), "error")
		}

		// Get service info for uptime tracking
		if svcInfo, err := e.serviceManager.GetService(svc.Name); err == nil {
			e.prometheusExporter.UpdateServiceUptime(svc.Name, svcInfo.StartTime)
		}
	}

	// Record latency for anomaly detection
	if result != nil {
		if durationStr, ok := result.Metadata["duration_ms"]; ok {
			var durationMs float64
			if _, err := fmt.Sscanf(durationStr, "%f", &durationMs); err == nil {
				e.anomalyDetector.AddLatencySample(svc.Name, durationMs)
			}
		}
	}

	if err != nil {
		logger.Error("Health check failed for service %s: %v", svc.Name, err)
		return
	}

	// Get previous state
	e.mu.RLock()
	prevStatus := e.serviceStates[svc.Name]
	e.mu.RUnlock()

	// Update service state
	e.mu.Lock()
	e.serviceStates[svc.Name] = result.Status
	e.mu.Unlock()

	// Generate events based on state changes
	if prevStatus != result.Status {
		logger.Info("Service %s state changed from %s to %s", svc.Name, prevStatus, result.Status)

		if result.Status == plugin.StatusUp && (prevStatus == plugin.StatusDown || prevStatus == plugin.StatusUnknown) {
			// Service recovered or is healthy for the first time
			e.eventChan <- Event{
				Type:      EventServiceRecovered,
				ServiceID: svc.Name,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"previous_status": string(prevStatus),
					"current_status":  string(result.Status),
					"message":         result.Message,
				},
			}

			// Check for alerts to resolve
			e.alertManager.CheckMetric(svc.Name, "status", 1.0, time.Now())

		} else if result.Status == plugin.StatusDown && (prevStatus == plugin.StatusUp || prevStatus == plugin.StatusUnknown) {
			// Service became unhealthy
			e.eventChan <- Event{
				Type:      EventServiceUnhealthy,
				ServiceID: svc.Name,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"previous_status": string(prevStatus),
					"current_status":  string(result.Status),
					"message":         result.Message,
				},
			}

			// Generate alert for down service
			e.alertManager.CheckMetric(svc.Name, "status", 0.0, time.Now())
		}
	}

	// Log health check result
	if result.Status == plugin.StatusUp {
		logger.Debug("Health check for service %s: %s - %s", svc.Name, result.Status, result.Message)
	} else {
		logger.Warn("Health check for service %s: %s - %s", svc.Name, result.Status, result.Message)
	}

	// Check for anomalies
	anomalies := e.anomalyDetector.GetAnomalies()
	for _, anomaly := range anomalies {
		if anomaly.ServiceName == svc.Name {
			logger.Warn("Anomaly detected for service %s: %s", svc.Name, anomaly.Description)

			// Generate alert for anomaly
			switch anomaly.Type {
			case monitor.AnomalyTypeLatency:
				e.alertManager.CheckMetric(svc.Name, "latency", anomaly.Value, time.Now())
			case monitor.AnomalyTypeThroughput:
				e.alertManager.CheckMetric(svc.Name, "throughput", anomaly.Value, time.Now())
			case monitor.AnomalyTypeErrorRate:
				e.alertManager.CheckMetric(svc.Name, "error_rate", anomaly.Value, time.Now())
			}
		}
	}
}

// processEvents processes events from the event channel
func (e *Engine) processEvents(ctx context.Context) {
	defer e.wg.Done()

	logger.Info("Starting event processor")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping event processor")
			return
		case <-e.stopChan:
			logger.Info("Stopping event processor")
			return
		case event := <-e.eventChan:
			e.handleEvent(event)
		}
	}
}

// handleEvent handles an event
func (e *Engine) handleEvent(event Event) {
	logger.Info("Handling event: %s for service %s", event.Type, event.ServiceID)

	switch event.Type {
	case EventServiceUnhealthy:
		// Get service configuration
		svcInfo, err := e.serviceManager.GetService(event.ServiceID)
		if err != nil {
			logger.Error("Failed to get service info for %s: %v", event.ServiceID, err)
			return
		}

		// Check action to take
		switch svcInfo.Config.Actions.OnFailure {
		case "restart":
			logger.Info("Attempting to restart service: %s", event.ServiceID)

			// Attempt to restart the service
			if err := e.serviceManager.RestartService(event.ServiceID); err != nil {
				logger.Error("Failed to restart service %s: %v", event.ServiceID, err)

				// Generate service failed event
				e.eventChan <- Event{
					Type:      EventServiceFailed,
					ServiceID: event.ServiceID,
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				}
			} else {
				// Generate service restarted event
				e.eventChan <- Event{
					Type:      EventServiceRestarted,
					ServiceID: event.ServiceID,
					Timestamp: time.Now(),
					Data:      nil,
				}

				// Record restart in Prometheus
				if e.config.Global.MetricsEnabled {
					e.prometheusExporter.RecordServiceRestart(event.ServiceID)
				}
			}
		case "notify":
			logger.Info("Sending notification for service: %s", event.ServiceID)

			// Create alert rule for the service if it doesn't exist
			alertRule := monitor.AlertRule{
				ID:          "service-down-" + event.ServiceID,
				Name:        "Service Down",
				Description: fmt.Sprintf("Service %s is down", event.ServiceID),
				ServiceName: event.ServiceID,
				MetricName:  "status",
				Threshold:   0.0,
				Operator:    "eq",
				Duration:    1 * time.Second,
				Severity:    monitor.AlertSeverityCritical,
				Labels: map[string]string{
					"service": event.ServiceID,
				},
			}
			e.alertManager.AddRule(alertRule)

			// Check the metric to trigger the alert
			e.alertManager.CheckMetric(event.ServiceID, "status", 0.0, time.Now())

		case "ignore":
			logger.Info("Ignoring failure for service: %s", event.ServiceID)
		default:
			logger.Warn("Unknown action for service %s: %s", event.ServiceID, svcInfo.Config.Actions.OnFailure)
		}

	case EventServiceRecovered:
		logger.Info("Service recovered: %s", event.ServiceID)

		// Update service status in Prometheus
		if e.config.Global.MetricsEnabled {
			e.prometheusExporter.RecordHealthCheck(event.ServiceID, "status", &plugin.HealthCheckResult{
				Status:    plugin.StatusUp,
				Message:   "Service recovered",
				Timestamp: time.Now().Unix(),
			}, 0)
		}

		// Check metrics to resolve any alerts
		e.alertManager.CheckMetric(event.ServiceID, "status", 1.0, time.Now())

	case EventServiceRestarted:
		logger.Info("Service restarted: %s", event.ServiceID)

		// Update service info in Prometheus
		if e.config.Global.MetricsEnabled {
			if svcInfo, err := e.serviceManager.GetService(event.ServiceID); err == nil {
				e.prometheusExporter.UpdateServiceUptime(event.ServiceID, svcInfo.StartTime)
			}
		}

	case EventServiceFailed:
		logger.Error("Service failed: %s - %v", event.ServiceID, event.Data["error"])

		// Create critical alert for service failure
		alertRule := monitor.AlertRule{
			ID:          "service-failed-" + event.ServiceID,
			Name:        "Service Failed",
			Description: fmt.Sprintf("Service %s failed to restart", event.ServiceID),
			ServiceName: event.ServiceID,
			MetricName:  "failed",
			Threshold:   1.0,
			Operator:    "eq",
			Duration:    1 * time.Hour,
			Severity:    monitor.AlertSeverityCritical,
			Labels: map[string]string{
				"service": event.ServiceID,
				"error":   fmt.Sprintf("%v", event.Data["error"]),
			},
		}
		e.alertManager.AddRule(alertRule)

		// Check the metric to trigger the alert
		e.alertManager.CheckMetric(event.ServiceID, "failed", 1.0, time.Now())

	default:
		logger.Warn("Unknown event type: %s", event.Type)
	}
}
