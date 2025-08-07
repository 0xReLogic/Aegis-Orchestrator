package plugin

import (
	"fmt"
	"sync"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
)

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status    Status
	Message   string
	Timestamp int64
	Metadata  map[string]string
}

// Status represents the status of a service
type Status string

const (
	// StatusUp indicates the service is healthy
	StatusUp Status = "UP"
	// StatusDown indicates the service is unhealthy
	StatusDown Status = "DOWN"
	// StatusDegraded indicates the service is functioning but with issues
	StatusDegraded Status = "DEGRADED"
	// StatusUnknown indicates the service status could not be determined
	StatusUnknown Status = "UNKNOWN"
)

// HealthChecker is the interface that all health checker plugins must implement
type HealthChecker interface {
	// Check performs a health check and returns the result
	Check() (*HealthCheckResult, error)
	// Name returns the name of the health checker
	Name() string
	// Type returns the type of the health checker (http, tcp, etc.)
	Type() string
	// Configure configures the health checker with the given options
	Configure(options map[string]interface{}) error
}

// Registry is a registry of health checker plugins
type Registry struct {
	checkers map[string]HealthChecker
	mu       sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		checkers: make(map[string]HealthChecker),
	}
}

// Register registers a health checker plugin
func (r *Registry) Register(checker HealthChecker) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := checker.Name()
	if _, exists := r.checkers[name]; exists {
		return fmt.Errorf("health checker with name '%s' already registered", name)
	}

	r.checkers[name] = checker
	logger.Info("Registered health checker plugin: %s (%s)", name, checker.Type())
	return nil
}

// Get returns a health checker by name
func (r *Registry) Get(name string) (HealthChecker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	checker, exists := r.checkers[name]
	if !exists {
		return nil, fmt.Errorf("health checker with name '%s' not found", name)
	}

	return checker, nil
}

// GetByType returns all health checkers of a specific type
func (r *Registry) GetByType(typeName string) []HealthChecker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []HealthChecker
	for _, checker := range r.checkers {
		if checker.Type() == typeName {
			result = append(result, checker)
		}
	}

	return result
}

// List returns all registered health checkers
func (r *Registry) List() []HealthChecker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []HealthChecker
	for _, checker := range r.checkers {
		result = append(result, checker)
	}

	return result
}

// Unregister removes a health checker from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.checkers[name]; !exists {
		return fmt.Errorf("health checker with name '%s' not found", name)
	}

	delete(r.checkers, name)
	logger.Info("Unregistered health checker plugin: %s", name)
	return nil
}
