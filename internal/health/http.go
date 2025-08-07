package health

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/plugin"
)

// HTTPHealthChecker implements a health checker for HTTP endpoints
type HTTPHealthChecker struct {
	name           string
	endpoint       string
	timeout        time.Duration
	client         *http.Client
	headers        map[string]string
	expectedStatus int
}

// NewHTTPHealthChecker creates a new HTTP health checker
func NewHTTPHealthChecker(name, endpoint string, timeout time.Duration) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		name:           name,
		endpoint:       endpoint,
		timeout:        timeout,
		client:         &http.Client{Timeout: timeout},
		headers:        make(map[string]string),
		expectedStatus: http.StatusOK,
	}
}

// Name returns the name of the health checker
func (h *HTTPHealthChecker) Name() string {
	return h.name
}

// Type returns the type of the health checker
func (h *HTTPHealthChecker) Type() string {
	return "http"
}

// Configure configures the health checker with the given options
func (h *HTTPHealthChecker) Configure(options map[string]interface{}) error {
	if endpoint, ok := options["endpoint"].(string); ok {
		h.endpoint = endpoint
	}

	if timeout, ok := options["timeout"].(time.Duration); ok {
		h.timeout = timeout
		h.client.Timeout = timeout
	}

	if headers, ok := options["headers"].(map[string]string); ok {
		h.headers = headers
	}

	if expectedStatus, ok := options["expectedStatus"].(int); ok {
		h.expectedStatus = expectedStatus
	}

	return nil
}

// Check performs a health check and returns the result
func (h *HTTPHealthChecker) Check() (*plugin.HealthCheckResult, error) {
	startTime := time.Now()

	// Create request
	req, err := http.NewRequest("GET", h.endpoint, nil)
	if err != nil {
		return &plugin.HealthCheckResult{
			Status:    plugin.StatusUnknown,
			Message:   fmt.Sprintf("Failed to create request: %v", err),
			Timestamp: time.Now().Unix(),
			Metadata:  map[string]string{"duration_ms": "0"},
		}, err
	}

	// Add headers
	for key, value := range h.headers {
		req.Header.Add(key, value)
	}

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		return &plugin.HealthCheckResult{
			Status:    plugin.StatusDown,
			Message:   fmt.Sprintf("HTTP request failed: %v", err),
			Timestamp: time.Now().Unix(),
			Metadata:  map[string]string{"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds())},
		}, nil
	}
	defer resp.Body.Close()

	// Read response body (limited to prevent memory issues)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		logger.Warn("Failed to read response body: %v", err)
	}

	// Check status code
	duration := time.Since(startTime).Milliseconds()
	if resp.StatusCode != h.expectedStatus {
		return &plugin.HealthCheckResult{
			Status:    plugin.StatusDown,
			Message:   fmt.Sprintf("Unexpected status code: %d, expected: %d, body: %s", resp.StatusCode, h.expectedStatus, string(body)),
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"duration_ms":  fmt.Sprintf("%d", duration),
				"status_code":  fmt.Sprintf("%d", resp.StatusCode),
				"content_type": resp.Header.Get("Content-Type"),
			},
		}, nil
	}

	return &plugin.HealthCheckResult{
		Status:    plugin.StatusUp,
		Message:   fmt.Sprintf("HTTP health check successful, status code: %d", resp.StatusCode),
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"duration_ms":  fmt.Sprintf("%d", duration),
			"status_code":  fmt.Sprintf("%d", resp.StatusCode),
			"content_type": resp.Header.Get("Content-Type"),
		},
	}, nil
}
