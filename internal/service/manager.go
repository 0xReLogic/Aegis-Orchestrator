package service

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/internal/config"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
)

// ServiceState represents the current state of a service
type ServiceState string

const (
	// ServiceStateRunning indicates the service is running
	ServiceStateRunning ServiceState = "RUNNING"
	// ServiceStateStopped indicates the service is stopped
	ServiceStateStopped ServiceState = "STOPPED"
	// ServiceStateRestarting indicates the service is restarting
	ServiceStateRestarting ServiceState = "RESTARTING"
	// ServiceStateFailed indicates the service failed to start or restart
	ServiceStateFailed ServiceState = "FAILED"
	// ServiceStateUnknown indicates the service state is unknown
	ServiceStateUnknown ServiceState = "UNKNOWN"
)

// ServiceInfo contains information about a service
type ServiceInfo struct {
	Name         string
	State        ServiceState
	LastError    string
	StartTime    time.Time
	RestartCount int
	Config       config.ServiceConfig
}

// Manager manages services and their lifecycle
type Manager struct {
	services map[string]*ServiceInfo
	commands map[string]*exec.Cmd
	mu       sync.RWMutex
}

// NewManager creates a new service manager
func NewManager() *Manager {
	return &Manager{
		services: make(map[string]*ServiceInfo),
		commands: make(map[string]*exec.Cmd),
	}
}

// RegisterService registers a service with the manager
func (m *Manager) RegisterService(serviceConfig config.ServiceConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.services[serviceConfig.Name]; exists {
		return fmt.Errorf("service '%s' already registered", serviceConfig.Name)
	}

	m.services[serviceConfig.Name] = &ServiceInfo{
		Name:         serviceConfig.Name,
		State:        ServiceStateUnknown,
		LastError:    "",
		StartTime:    time.Time{},
		RestartCount: 0,
		Config:       serviceConfig,
	}

	logger.Info("Registered service: %s", serviceConfig.Name)
	return nil
}

// GetService returns information about a service
func (m *Manager) GetService(name string) (*ServiceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	service, exists := m.services[name]
	if !exists {
		return nil, fmt.Errorf("service '%s' not found", name)
	}

	return service, nil
}

// ListServices returns information about all registered services
func (m *Manager) ListServices() []*ServiceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*ServiceInfo
	for _, service := range m.services {
		result = append(result, service)
	}

	return result
}

// RestartService restarts a service
func (m *Manager) RestartService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	service, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service '%s' not found", name)
	}

	// In a real implementation, this would use the appropriate method to restart the service
	// For now, we'll just simulate a restart by updating the state
	logger.Info("Restarting service: %s", name)

	service.State = ServiceStateRestarting
	service.RestartCount++

	// Simulate restart delay
	go func() {
		time.Sleep(2 * time.Second)

		m.mu.Lock()
		defer m.mu.Unlock()

		// In a real implementation, we would check if the restart was successful
		service.State = ServiceStateRunning
		service.StartTime = time.Now()
		logger.Info("Service restarted successfully: %s", name)
	}()

	return nil
}

// StopService stops a service
func (m *Manager) StopService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	service, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service '%s' not found", name)
	}

	// In a real implementation, this would use the appropriate method to stop the service
	// For now, we'll just simulate stopping by updating the state
	logger.Info("Stopping service: %s", name)

	service.State = ServiceStateStopped

	return nil
}

// StartService starts a service
func (m *Manager) StartService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	service, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service '%s' not found", name)
	}

	// In a real implementation, this would use the appropriate method to start the service
	// For now, we'll just simulate starting by updating the state
	logger.Info("Starting service: %s", name)

	service.State = ServiceStateRunning
	service.StartTime = time.Now()

	return nil
}

// UpdateServiceState updates the state of a service
func (m *Manager) UpdateServiceState(name string, state ServiceState, lastError string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	service, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service '%s' not found", name)
	}

	service.State = state
	service.LastError = lastError

	return nil
}
