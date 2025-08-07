package core

import (
	"testing"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/internal/config"
	"github.com/0xReLogic/Aegis-Orchestrator/internal/service"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/plugin"
	"github.com/stretchr/testify/assert"
)

// MockHealthChecker is a mock implementation of the HealthChecker interface
type MockHealthChecker struct {
	status plugin.Status
	err    error
}

func (m *MockHealthChecker) Check() (plugin.Status, error) {
	return m.status, m.err
}

func (m *MockHealthChecker) Name() string {
	return "mock"
}

func (m *MockHealthChecker) SetStatus(status plugin.Status) {
	m.status = status
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
	// Create a basic configuration with circuit breaker enabled
	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{
				Name:     "test-service",
				Type:     "http",
				Endpoint: "http://localhost:8080/health",
				Interval: 1 * time.Second,
				Timeout:  1 * time.Second,
				Retries:  1,
				Actions: config.ActionConfig{
					OnFailure: "notify",
					CircuitBreaker: config.CircuitBreakerPolicy{
						Enabled:          true,
						FailureThreshold: 2,
						ResetTimeout:     100 * time.Millisecond,
						SuccessThreshold: 1,
					},
				},
			},
		},
	}

	// Create engine with the configuration
	engine := NewEngine(cfg)
	err := engine.Initialize()
	assert.NoError(t, err)

	// Create a mock health checker
	mockChecker := &MockHealthChecker{status: plugin.StatusUp}
	engine.healthCheckers["test-service"] = mockChecker

	// Register the service
	engine.serviceManager.RegisterService(&service.ServiceInfo{
		Name:   "test-service",
		Config: cfg.Services[0],
	})

	// Test initial state - should be CLOSED
	assert.Equal(t, StateClosed, engine.circuitBreakerStates["test-service"].state)

	// Simulate consecutive failures to trigger OPEN state
	mockChecker.SetStatus(plugin.StatusDown)

	// First failure
	engine.performHealthCheck(cfg.Services[0], mockChecker)
	assert.Equal(t, StateClosed, engine.circuitBreakerStates["test-service"].state)
	assert.Equal(t, 1, engine.circuitBreakerStates["test-service"].consecutiveFailures)

	// Second failure - should open the circuit
	engine.performHealthCheck(cfg.Services[0], mockChecker)
	assert.Equal(t, StateOpen, engine.circuitBreakerStates["test-service"].state)

	// Wait for reset timeout to transition to HALF-OPEN
	time.Sleep(150 * time.Millisecond)

	// Next check should set to HALF-OPEN
	engine.performHealthCheck(cfg.Services[0], mockChecker)
	assert.Equal(t, StateHalfOpen, engine.circuitBreakerStates["test-service"].state)

	// Failure in HALF-OPEN should go back to OPEN
	engine.performHealthCheck(cfg.Services[0], mockChecker)
	assert.Equal(t, StateOpen, engine.circuitBreakerStates["test-service"].state)

	// Wait for reset timeout again
	time.Sleep(150 * time.Millisecond)

	// Set service back to healthy
	mockChecker.SetStatus(plugin.StatusUp)

	// Next check should set to HALF-OPEN
	engine.performHealthCheck(cfg.Services[0], mockChecker)
	assert.Equal(t, StateHalfOpen, engine.circuitBreakerStates["test-service"].state)

	// Success in HALF-OPEN should close the circuit
	engine.performHealthCheck(cfg.Services[0], mockChecker)
	assert.Equal(t, StateClosed, engine.circuitBreakerStates["test-service"].state)
}

func TestCircuitBreakerSkipsHealthCheck(t *testing.T) {
	// Create a basic configuration with circuit breaker enabled
	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{
				Name:     "test-service",
				Type:     "http",
				Endpoint: "http://localhost:8080/health",
				Interval: 1 * time.Second,
				Timeout:  1 * time.Second,
				Retries:  1,
				Actions: config.ActionConfig{
					OnFailure: "notify",
					CircuitBreaker: config.CircuitBreakerPolicy{
						Enabled:          true,
						FailureThreshold: 1,
						ResetTimeout:     500 * time.Millisecond,
						SuccessThreshold: 1,
					},
				},
			},
		},
	}

	// Create engine with the configuration
	engine := NewEngine(cfg)
	err := engine.Initialize()
	assert.NoError(t, err)

	// Create a mock health checker that counts the number of checks
	checkCount := 0
	mockChecker := &MockHealthChecker{
		status: plugin.StatusDown,
	}
	engine.healthCheckers["test-service"] = mockChecker

	// Register the service
	engine.serviceManager.RegisterService(&service.ServiceInfo{
		Name:   "test-service",
		Config: cfg.Services[0],
	})

	// First check - should fail and open the circuit
	engine.performHealthCheck(cfg.Services[0], mockChecker)
	assert.Equal(t, StateOpen, engine.circuitBreakerStates["test-service"].state)

	// Second check - should be skipped because circuit is open
	beforeState := engine.serviceStates["test-service"]
	engine.performHealthCheck(cfg.Services[0], mockChecker)
	afterState := engine.serviceStates["test-service"]

	// Service state should not change because health check was skipped
	assert.Equal(t, beforeState, afterState)
}
