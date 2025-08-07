package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	// AlertSeverityInfo represents an informational alert
	AlertSeverityInfo AlertSeverity = "INFO"
	// AlertSeverityWarning represents a warning alert
	AlertSeverityWarning AlertSeverity = "WARNING"
	// AlertSeverityCritical represents a critical alert
	AlertSeverityCritical AlertSeverity = "CRITICAL"
)

// AlertState represents the state of an alert
type AlertState string

const (
	// AlertStateFiring indicates the alert is currently firing
	AlertStateFiring AlertState = "FIRING"
	// AlertStateResolved indicates the alert has been resolved
	AlertStateResolved AlertState = "RESOLVED"
)

// Alert represents an alert
type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ServiceName string            `json:"service_name"`
	Severity    AlertSeverity     `json:"severity"`
	State       AlertState        `json:"state"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// AlertRule represents a rule for generating alerts
type AlertRule struct {
	ID          string
	Name        string
	Description string
	ServiceName string
	MetricName  string
	Threshold   float64
	Operator    string // "gt", "lt", "eq", "ne", "ge", "le"
	Duration    time.Duration
	Severity    AlertSeverity
	Labels      map[string]string
}

// AlertManager manages alerts
type AlertManager struct {
	rules          []AlertRule
	activeAlerts   map[string]*Alert
	resolvedAlerts []*Alert
	webhookURL     string
	webhookHeaders map[string]string
	mu             sync.RWMutex
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		rules:          make([]AlertRule, 0),
		activeAlerts:   make(map[string]*Alert),
		resolvedAlerts: make([]*Alert, 0),
		webhookHeaders: make(map[string]string),
		stopChan:       make(chan struct{}),
	}
}

// AddRule adds an alert rule
func (a *AlertManager) AddRule(rule AlertRule) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rules = append(a.rules, rule)
	logger.Info("Added alert rule: %s for service %s", rule.Name, rule.ServiceName)
}

// SetWebhook sets the webhook URL and headers for alert notifications
func (a *AlertManager) SetWebhook(url string, headers map[string]string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.webhookURL = url
	a.webhookHeaders = headers
	logger.Info("Set alert webhook URL: %s", url)
}

// Start starts the alert manager
func (a *AlertManager) Start() {
	logger.Info("Started alert manager")
}

// Stop stops the alert manager
func (a *AlertManager) Stop() {
	close(a.stopChan)
	a.wg.Wait()
	logger.Info("Alert manager stopped")
}

// CheckMetric checks a metric against alert rules
func (a *AlertManager) CheckMetric(serviceName, metricName string, value float64, timestamp time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check each rule
	for _, rule := range a.rules {
		if rule.ServiceName == serviceName && rule.MetricName == metricName {
			// Check if the value violates the threshold
			if a.checkThreshold(value, rule.Threshold, rule.Operator) {
				// Generate alert ID
				alertID := fmt.Sprintf("%s-%s-%s", serviceName, metricName, rule.ID)

				// Check if alert already exists
				if alert, exists := a.activeAlerts[alertID]; exists {
					// Alert already exists, update it
					alert.Value = value
				} else {
					// Create new alert
					alert := &Alert{
						ID:          alertID,
						Name:        rule.Name,
						Description: rule.Description,
						ServiceName: serviceName,
						Severity:    rule.Severity,
						State:       AlertStateFiring,
						Value:       value,
						Threshold:   rule.Threshold,
						StartTime:   timestamp,
						Labels:      rule.Labels,
					}
					a.activeAlerts[alertID] = alert

					// Send alert notification
					a.sendAlertNotification(alert)
					logger.Warn("Alert firing: %s for service %s (value: %.2f, threshold: %.2f)",
						rule.Name, serviceName, value, rule.Threshold)
				}
			} else {
				// Check if we need to resolve an alert
				alertID := fmt.Sprintf("%s-%s-%s", serviceName, metricName, rule.ID)
				if alert, exists := a.activeAlerts[alertID]; exists {
					// Resolve the alert
					alert.State = AlertStateResolved
					alert.EndTime = timestamp
					a.resolvedAlerts = append(a.resolvedAlerts, alert)
					delete(a.activeAlerts, alertID)

					// Send resolution notification
					a.sendAlertNotification(alert)
					logger.Info("Alert resolved: %s for service %s", rule.Name, serviceName)
				}
			}
		}
	}
}

// GetActiveAlerts returns the active alerts
func (a *AlertManager) GetActiveAlerts() []*Alert {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make([]*Alert, 0, len(a.activeAlerts))
	for _, alert := range a.activeAlerts {
		alertCopy := *alert
		result = append(result, &alertCopy)
	}

	return result
}

// GetResolvedAlerts returns the resolved alerts
func (a *AlertManager) GetResolvedAlerts() []*Alert {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make([]*Alert, len(a.resolvedAlerts))
	for i, alert := range a.resolvedAlerts {
		alertCopy := *alert
		result[i] = &alertCopy
	}

	return result
}

// checkThreshold checks if a value violates a threshold based on the operator
func (a *AlertManager) checkThreshold(value, threshold float64, operator string) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "eq":
		return value == threshold
	case "ne":
		return value != threshold
	case "ge":
		return value >= threshold
	case "le":
		return value <= threshold
	default:
		logger.Warn("Unknown operator: %s", operator)
		return false
	}
}

// sendAlertNotification sends an alert notification via webhook
func (a *AlertManager) sendAlertNotification(alert *Alert) {
	if a.webhookURL == "" {
		return
	}

	// Create JSON payload
	payload, err := json.Marshal(alert)
	if err != nil {
		logger.Error("Failed to marshal alert: %v", err)
		return
	}

	// Send webhook request in a goroutine to avoid blocking
	go func() {
		req, err := http.NewRequest("POST", a.webhookURL, bytes.NewBuffer(payload))
		if err != nil {
			logger.Error("Failed to create webhook request: %v", err)
			return
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		for key, value := range a.webhookHeaders {
			req.Header.Set(key, value)
		}

		// Send request
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logger.Error("Failed to send webhook request: %v", err)
			return
		}
		defer resp.Body.Close()

		// Check response
		if resp.StatusCode != http.StatusOK {
			logger.Error("Webhook request failed with status code: %d", resp.StatusCode)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logger.Info("Successfully sent alert notification for %s", alert.Name)
		} else {
			logger.Error("Failed to send alert notification, status code: %d", resp.StatusCode)
		}
	}()
}
