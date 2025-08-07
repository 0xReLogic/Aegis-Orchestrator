package core

import (
	"context"
	"fmt"
	"math"
	"math/rand"
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

	// Phase 3 additions
	restartStates        map[string]*restartState
	circuitBreakerStates map[string]*circuitBreakerState
}

// restartState holds the state for a service's restart backoff strategy.
type restartState struct {
	attempts           int
	nextRestartAttempt time.Time
}

// circuitBreakerState holds the state of the circuit breaker for a service.
type circuitBreakerState struct {
	state               string
	consecutiveFailures int
	openTime            time.Time // Time when the circuit was opened
}

const (
	StateClosed   = "CLOSED"
	StateOpen     = "OPEN"
	StateHalfOpen = "HALF_OPEN"
)

// NewEngine creates a new orchestration engine
func NewEngine(cfg *config.Config) *Engine {
	// Initialize random seed for jitter calculations
	rand.Seed(time.Now().UnixNano())

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
		alertManager:       monitor.NewAlertManager(),
		retryPolicies:      make(map[string]*health.RetryPolicy),

		// Phase 3 additions
		restartStates:        make(map[string]*restartState),
		circuitBreakerStates: make(map[string]*circuitBreakerState),
	}
}

// Initialize initializes the engine
func (e *Engine) Initialize() error {
	// Initialize anomaly detector if enabled
	if e.config.AnomalyDetection.Enabled {
		// Create a new anomaly detector with settings from config
		e.anomalyDetector = monitor.NewAnomalyDetector(
			e.config.AnomalyDetection.HistorySize,
			e.config.AnomalyDetection.DetectionInterval,
		)
		logger.Info("Initialized anomaly detector with history size %d and interval %s", e.config.AnomalyDetection.HistorySize, e.config.AnomalyDetection.DetectionInterval)
	}

	// Register health checkers
	for i := range e.config.Services {
		svc := e.config.Services[i] // Create a local copy of the loop variable
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

		// Set up anomaly detection thresholds if the detector is enabled
		if e.anomalyDetector != nil && svc.Metadata != nil {
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

		// Initialize circuit breaker state if enabled
		if svc.Actions.CircuitBreaker.Enabled {
			e.circuitBreakerStates[svc.Name] = &circuitBreakerState{
				state: StateClosed,
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
	}

	// Configure and start anomaly detector
	if e.config.AnomalyDetection.Enabled && e.anomalyDetector != nil {
		// Configure thresholds from global config
		for _, threshold := range e.config.AnomalyDetection.Thresholds.Latency {
			e.anomalyDetector.SetLatencyThreshold(threshold.Service, threshold.Value)
			logger.Info("Set global latency threshold for service %s: %.2f ms", threshold.Service, threshold.Value)
		}
		for _, threshold := range e.config.AnomalyDetection.Thresholds.ErrorRate {
			e.anomalyDetector.SetErrorRateThreshold(threshold.Service, threshold.Value)
			logger.Info("Set global error rate threshold for service %s: %.2f%%", threshold.Service, threshold.Value*100)
		}
		for _, threshold := range e.config.AnomalyDetection.Thresholds.Throughput {
			e.anomalyDetector.SetThroughputThreshold(threshold.Service, threshold.Value)
			logger.Info("Set global throughput threshold for service %s: %.2f req/s", threshold.Service, threshold.Value)
		}

		// Start anomaly detector
		e.anomalyDetector.Start()
		logger.Info("Started anomaly detector with interval %s", e.config.AnomalyDetection.DetectionInterval)
	}

	// Initialize alert manager
	if e.config.Notifications.Webhook.Enabled {
		e.alertManager.SetWebhook(e.config.Notifications.Webhook.Endpoint, e.config.Notifications.Webhook.Headers)
		logger.Info("Set alert webhook URL: %s", e.config.Notifications.Webhook.Endpoint)
		e.alertManager.Start()
		logger.Info("Started alert manager")
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
		svc := svc // Create a local copy of the loop variable
		checker, ok := e.healthCheckers[svc.Name]
		if !ok {
			logger.Warn("No health checker found for service '%s', skipping health check loop.", svc.Name)
			continue
		}
		e.wg.Add(1)
		go e.runHealthCheckLoop(ctx, svc, checker)
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
	if e.anomalyDetector != nil {
		e.anomalyDetector.Stop()
	}
	e.alertManager.Stop()

	logger.Info("Aegis-Orchestrator engine stopped")
}

// runHealthCheckLoop runs a health check loop for a service
func (e *Engine) runHealthCheckLoop(ctx context.Context, svc config.ServiceConfig, checker plugin.HealthChecker) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic recovered in health check loop for service %s: %v", svc.Name, r)
		}
	}()

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
			e.performHealthCheck(svc, checker)
		}
	}
}

// performHealthCheck performs a health check for a service
func (e *Engine) performHealthCheck(svc config.ServiceConfig, checker plugin.HealthChecker) {
	e.mu.Lock()
	cbState, cbEnabled := e.circuitBreakerStates[svc.Name]
	cbConfig := svc.Actions.CircuitBreaker
	e.mu.Unlock()

	// Section 1: Check Circuit Breaker state before proceeding
	if cbEnabled {
		e.mu.Lock()
		if cbState.state == StateOpen {
			if time.Since(cbState.openTime) > cbConfig.ResetTimeout {
				cbState.state = StateHalfOpen
				logger.Info("Circuit breaker for %s is now HALF-OPEN", svc.Name)
			} else {
				e.mu.Unlock()
				logger.Debug("Circuit breaker for %s is OPEN. Skipping health check.", svc.Name)
				return // Skip health check
			}
		}
		e.mu.Unlock()
	}

	// Section 2: Perform health check with manual retry loop
	retryPolicy, exists := e.retryPolicies[svc.Name]
	if !exists {
		retryPolicy = health.DefaultRetryPolicy()
	}

	var finalResult *plugin.HealthCheckResult
	var finalErr error

	for i := 0; i < retryPolicy.MaxRetries; i++ {
		startTime := time.Now()
		result, err := checker.Check()
		duration := time.Since(startTime)

		finalResult = result
		finalErr = err

		// Record metrics for every attempt
		if e.config.Global.MetricsEnabled {
			e.prometheusExporter.RecordHealthCheck(svc.Name, checker.Type(), result, duration)
		}

		// If check is successful, handle success and break the loop
		if err == nil && result.Status == plugin.StatusUp {
			e.mu.Lock()
			if cbEnabled {
				if cbState.state == StateHalfOpen {
					cbState.state = StateClosed
					cbState.consecutiveFailures = 0
					logger.Info("Circuit breaker for %s is now CLOSED.", svc.Name)
				} else {
					cbState.consecutiveFailures = 0 // Reset on success
				}
			}
			e.mu.Unlock()
			break // Exit retry loop on success
		}

		// If check fails, update circuit breaker and continue loop
		if cbEnabled {
			e.mu.Lock()
			cbState.consecutiveFailures++
			logger.Debug("Service %s health check failed (attempt %d/%d). Consecutive failures: %d",
				svc.Name, i+1, retryPolicy.MaxRetries, cbState.consecutiveFailures)

			if cbState.state == StateHalfOpen || (cbState.state == StateClosed && cbState.consecutiveFailures >= cbConfig.FailureThreshold) {
				// If tripping from half-open, reset failure count to 1 for this new failure.
				if cbState.state == StateHalfOpen {
					cbState.consecutiveFailures = 1
				}
				cbState.state = StateOpen
				cbState.openTime = time.Now()
				logger.Warn("Circuit breaker for %s is now OPEN due to %d consecutive failures.", svc.Name, cbState.consecutiveFailures)
				e.mu.Unlock()
				break // Exit retry loop as circuit is now open
			}
			e.mu.Unlock()
		}

		// Wait before next retry
		if i < retryPolicy.MaxRetries-1 {
			time.Sleep(retryPolicy.InitialBackoff)
		}
	}

	// Section 3: Process the final result of the health check
	e.mu.RLock()
	prevStatus := e.serviceStates[svc.Name]
	e.mu.RUnlock()

	newStatus := plugin.StatusDown
	if finalResult != nil {
		newStatus = finalResult.Status
	}

	if prevStatus != newStatus {
		e.mu.Lock()
		e.serviceStates[svc.Name] = newStatus
		e.mu.Unlock()

		logger.Info("Service %s status changed from %s to %s", svc.Name, prevStatus, newStatus)

		var eventType EventType
		var eventData map[string]interface{}

		if newStatus == plugin.StatusUp {
			eventType = EventServiceRecovered
			eventData = map[string]interface{}{"message": finalResult.Message}
		} else {
			eventType = EventServiceUnhealthy
			if finalErr != nil {
				eventData = map[string]interface{}{"error": finalErr.Error()}
			} else if finalResult != nil {
				eventData = map[string]interface{}{"error": finalResult.Message}
			}
		}
		e.eventChan <- Event{Type: eventType, ServiceID: svc.Name, Timestamp: time.Now(), Data: eventData}
	}
} // Added closing brace here

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
			e.scheduleRestart(svcInfo)
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

	case EventServiceRestarted:
		logger.Info("Service restarted: %s", event.ServiceID)

		// After a restart attempt, the service's state is unknown until the next health check.
		e.mu.Lock()
		e.serviceStates[event.ServiceID] = plugin.StatusUnknown
		e.mu.Unlock()

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

// scheduleRestart schedules a service restart using a backoff strategy.
func (e *Engine) scheduleRestart(svcInfo *service.ServiceInfo) {
	serviceID := svcInfo.Config.Name
	policy := svcInfo.Config.Actions.Backoff

	// If backoff is not enabled, restart immediately.
	if !policy.Enabled {
		logger.Info("Attempting to restart service immediately (backoff disabled): %s", serviceID)
		go e.attemptRestart(serviceID) // Run in a goroutine to not block the event loop
		return
	}

	// Use a critical section to safely access and update restart state
	e.mu.Lock()
	defer e.mu.Unlock()

	state, exists := e.restartStates[serviceID]
	if !exists {
		state = &restartState{}
		e.restartStates[serviceID] = state
	}

	// If a restart is already scheduled and is in the future, do nothing.
	if !state.nextRestartAttempt.IsZero() && time.Now().Before(state.nextRestartAttempt) {
		logger.Info("Restart for service %s already scheduled at %v", serviceID, state.nextRestartAttempt)
		return
	}

	state.attempts++

	// Calculate backoff duration with jitter to prevent thundering herd
	baseBackoff := policy.InitialInterval * time.Duration(math.Pow(policy.Multiplier, float64(state.attempts-1)))
	if baseBackoff > policy.MaxInterval {
		baseBackoff = policy.MaxInterval
	}

	// Add jitter (±20%)
	jitterFactor := 0.8 + (rand.Float64() * 0.4) // 0.8 to 1.2
	backoffDuration := time.Duration(float64(baseBackoff) * jitterFactor)

	state.nextRestartAttempt = time.Now().Add(backoffDuration)
	logger.Info("Scheduling restart for service %s in %v (attempt %d)", serviceID, backoffDuration, state.attempts)

	// Schedule the restart in a new goroutine
	// We need to copy serviceID to avoid closure issues
	svcID := serviceID
	go func() {
		time.Sleep(backoffDuration)
		e.attemptRestart(svcID)
	}()
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

// attemptRestart performs the actual service restart and sends corresponding events.
func (e *Engine) attemptRestart(serviceID string) {
	logger.Info("Attempting to restart service: %s", serviceID)

	if err := e.serviceManager.RestartService(serviceID); err != nil {
		logger.Error("Failed to restart service %s: %v", serviceID, err)
		e.eventChan <- Event{
			Type:      EventServiceFailed,
			ServiceID: serviceID,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"error": err.Error()},
		}
	} else {
		logger.Info("Successfully restarted service: %s", serviceID)
		e.eventChan <- Event{
			Type:      EventServiceRestarted,
			ServiceID: serviceID,
			Timestamp: time.Now(),
		}
		// Record restart in Prometheus
		if e.config.Global.MetricsEnabled {
			e.prometheusExporter.RecordServiceRestart(serviceID)
		}
	}
}
