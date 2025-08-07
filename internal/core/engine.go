package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/internal/config"
	"github.com/0xReLogic/Aegis-Orchestrator/internal/health"
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
		default:
			return fmt.Errorf("unsupported service type for '%s': %s", svc.Name, svc.Type)
		}

		// Register the health checker
		if err := e.pluginRegistry.Register(checker); err != nil {
			return fmt.Errorf("failed to register health checker for '%s': %w", svc.Name, err)
		}

		e.healthCheckers[svc.Name] = checker
		e.serviceStates[svc.Name] = plugin.StatusUnknown
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

	// Perform health check
	result, err := checker.Check()
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
		}
	}

	// Log health check result
	if result.Status == plugin.StatusUp {
		logger.Debug("Health check for service %s: %s - %s", svc.Name, result.Status, result.Message)
	} else {
		logger.Warn("Health check for service %s: %s - %s", svc.Name, result.Status, result.Message)
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
			}
		case "notify":
			logger.Info("Notification would be sent for service: %s", event.ServiceID)
			// In a real implementation, this would send a notification
		case "ignore":
			logger.Info("Ignoring failure for service: %s", event.ServiceID)
		default:
			logger.Warn("Unknown action for service %s: %s", event.ServiceID, svcInfo.Config.Actions.OnFailure)
		}

	case EventServiceRecovered:
		logger.Info("Service recovered: %s", event.ServiceID)
		// In a real implementation, this might trigger additional actions

	case EventServiceRestarted:
		logger.Info("Service restarted: %s", event.ServiceID)
		// In a real implementation, this might trigger additional actions

	case EventServiceFailed:
		logger.Error("Service failed: %s - %v", event.ServiceID, event.Data["error"])
		// In a real implementation, this might trigger escalation procedures

	default:
		logger.Warn("Unknown event type: %s", event.Type)
	}
}
