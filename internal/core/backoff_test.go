package core

import (
	"testing"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/internal/config"
	"github.com/0xReLogic/Aegis-Orchestrator/internal/service"
	"github.com/stretchr/testify/assert"
)

// MockServiceManager is a mock implementation of the service manager
type MockServiceManager struct {
	restartCount int
	services     map[string]*service.ServiceInfo
}

func NewMockServiceManager() *MockServiceManager {
	return &MockServiceManager{
		services: make(map[string]*service.ServiceInfo),
	}
}

func (m *MockServiceManager) RegisterService(svc *service.ServiceInfo) {
	m.services[svc.Name] = svc
}

func (m *MockServiceManager) GetService(name string) (*service.ServiceInfo, error) {
	svc, ok := m.services[name]
	if !ok {
		return nil, service.ErrServiceNotFound
	}
	return svc, nil
}

func (m *MockServiceManager) RestartService(name string) error {
	m.restartCount++
	return nil
}

func TestBackoffStrategy(t *testing.T) {
	// Create a basic configuration with backoff enabled
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
					OnFailure: "restart",
					Backoff: config.BackoffPolicy{
						Enabled:         true,
						InitialInterval: 100 * time.Millisecond,
						MaxInterval:     1 * time.Second,
						Multiplier:      2.0,
					},
				},
			},
		},
	}

	// Create engine with the configuration
	engine := NewEngine(cfg)

	// Replace service manager with mock
	mockServiceManager := NewMockServiceManager()
	mockServiceManager.RegisterService(&service.ServiceInfo{
		Name:   "test-service",
		Config: cfg.Services[0],
	})
	engine.serviceManager = mockServiceManager

	// Initialize the engine
	err := engine.Initialize()
	assert.NoError(t, err)

	// Test first restart - should use initial interval
	svcInfo, _ := engine.serviceManager.GetService("test-service")
	engine.scheduleRestart(svcInfo)

	// Check that restart state was created
	state, exists := engine.restartStates["test-service"]
	assert.True(t, exists)
	assert.Equal(t, 1, state.attempts)

	// Wait for restart to happen
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 1, mockServiceManager.restartCount)

	// Test second restart - should use backoff
	engine.scheduleRestart(svcInfo)
	assert.Equal(t, 2, state.attempts)

	// Wait for second restart
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, 2, mockServiceManager.restartCount)

	// Test third restart - should use even longer backoff
	engine.scheduleRestart(svcInfo)
	assert.Equal(t, 3, state.attempts)

	// Wait for third restart
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 3, mockServiceManager.restartCount)
}

func TestBackoffWithoutJitter(t *testing.T) {
	// Create a basic configuration with backoff enabled
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
					OnFailure: "restart",
					Backoff: config.BackoffPolicy{
						Enabled:         true,
						InitialInterval: 100 * time.Millisecond,
						MaxInterval:     1 * time.Second,
						Multiplier:      2.0,
					},
				},
			},
		},
	}

	// Create engine with the configuration
	engine := NewEngine(cfg)

	// Replace service manager with mock
	mockServiceManager := NewMockServiceManager()
	mockServiceManager.RegisterService(&service.ServiceInfo{
		Name:   "test-service",
		Config: cfg.Services[0],
	})
	engine.serviceManager = mockServiceManager

	// Initialize the engine
	err := engine.Initialize()
	assert.NoError(t, err)

	// Test multiple restarts to verify backoff calculation
	svcInfo, _ := engine.serviceManager.GetService("test-service")

	// First attempt - should use initial interval (100ms)
	engine.scheduleRestart(svcInfo)
	state := engine.restartStates["test-service"]
	assert.Equal(t, 1, state.attempts)

	// Second attempt - should use 200ms (100ms * 2^1)
	engine.scheduleRestart(svcInfo)
	assert.Equal(t, 2, state.attempts)

	// Third attempt - should use 400ms (100ms * 2^2)
	engine.scheduleRestart(svcInfo)
	assert.Equal(t, 3, state.attempts)

	// Fourth attempt - should use 800ms (100ms * 2^3)
	engine.scheduleRestart(svcInfo)
	assert.Equal(t, 4, state.attempts)

	// Fifth attempt - should use 1000ms (max interval)
	engine.scheduleRestart(svcInfo)
	assert.Equal(t, 5, state.attempts)
}
