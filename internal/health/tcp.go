package health

import (
	"fmt"
	"net"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/plugin"
)

// TCPHealthChecker implements a health checker for TCP endpoints
type TCPHealthChecker struct {
	name     string
	endpoint string
	timeout  time.Duration
}

// NewTCPHealthChecker creates a new TCP health checker
func NewTCPHealthChecker(name, endpoint string, timeout time.Duration) *TCPHealthChecker {
	return &TCPHealthChecker{
		name:     name,
		endpoint: endpoint,
		timeout:  timeout,
	}
}

// Name returns the name of the health checker
func (t *TCPHealthChecker) Name() string {
	return t.name
}

// Type returns the type of the health checker
func (t *TCPHealthChecker) Type() string {
	return "tcp"
}

// Configure configures the health checker with the given options
func (t *TCPHealthChecker) Configure(options map[string]interface{}) error {
	if endpoint, ok := options["endpoint"].(string); ok {
		t.endpoint = endpoint
	}

	if timeout, ok := options["timeout"].(time.Duration); ok {
		t.timeout = timeout
	}

	return nil
}

// Check performs a health check and returns the result
func (t *TCPHealthChecker) Check() (*plugin.HealthCheckResult, error) {
	logger.Debug("TCP Check for '%s' on endpoint '%s'", t.name, t.endpoint)
	startTime := time.Now()

	// Attempt to establish a TCP connection
	conn, err := net.DialTimeout("tcp", t.endpoint, t.timeout)
	if err != nil {
		return &plugin.HealthCheckResult{
			Status:    plugin.StatusDown,
			Message:   fmt.Sprintf("TCP connection failed: %v", err),
			Timestamp: time.Now().Unix(),
			Metadata:  map[string]string{"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds())},
		}, nil
	}
	defer conn.Close()

	duration := time.Since(startTime).Milliseconds()
	return &plugin.HealthCheckResult{
		Status:    plugin.StatusUp,
		Message:   fmt.Sprintf("TCP connection successful to %s", t.endpoint),
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"duration_ms": fmt.Sprintf("%d", duration),
			"local_addr":  conn.LocalAddr().String(),
			"remote_addr": conn.RemoteAddr().String(),
		},
	}, nil
}
