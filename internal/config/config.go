package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Global           GlobalConfig       `yaml:"global"`
	Services         []ServiceConfig    `yaml:"services"`
	Notifications    NotificationConfig `yaml:"notifications"`
	Logs             LogConfig          `yaml:"logs"`
	AnomalyDetection AnomalyConfig      `yaml:"anomaly_detection"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	LogLevel       string `yaml:"logLevel"`
	MetricsEnabled bool   `yaml:"metricsEnabled"`
	MetricsPort    int    `yaml:"metricsPort"`
}

// ServiceConfig represents a service to be monitored
type ServiceConfig struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"`
	Endpoint string            `yaml:"endpoint"`
	Interval time.Duration     `yaml:"interval"`
	Timeout  time.Duration     `yaml:"timeout"`
	Retries  int               `yaml:"retries"`
	Actions  ActionConfig      `yaml:"actions"`
	Metadata map[string]string `yaml:"metadata"`
}

// ActionConfig defines actions to take on service state changes
type ActionConfig struct {
	OnFailure string `yaml:"onFailure"`
}

// NotificationConfig contains notification settings
type NotificationConfig struct {
	Webhook WebhookConfig `yaml:"webhook"`
}

// WebhookConfig contains webhook notification settings
type WebhookConfig struct {
	Enabled  bool              `yaml:"enabled"`
	Endpoint string            `yaml:"endpoint"`
	Headers  map[string]string `yaml:"headers"`
}

// LogConfig contains log analysis settings
type LogConfig struct {
	Enabled      bool               `yaml:"enabled"`
	ScanInterval time.Duration      `yaml:"scan_interval"`
	Files        []LogFileConfig    `yaml:"files"`
	Patterns     []LogPatternConfig `yaml:"patterns"`
}

// LogFileConfig contains configuration for a log file
type LogFileConfig struct {
	Service string `yaml:"service"`
	Path    string `yaml:"path"`
}

// LogPatternConfig contains configuration for a log pattern
type LogPatternConfig struct {
	Name        string `yaml:"name"`
	Pattern     string `yaml:"pattern"`
	Severity    string `yaml:"severity"`
	Description string `yaml:"description"`
}

// AnomalyConfig contains anomaly detection settings
type AnomalyConfig struct {
	Enabled           bool                   `yaml:"enabled"`
	DetectionInterval time.Duration          `yaml:"detection_interval"`
	HistorySize       int                    `yaml:"history_size"`
	Thresholds        AnomalyThresholdConfig `yaml:"thresholds"`
}

// AnomalyThresholdConfig contains threshold configurations for anomaly detection
type AnomalyThresholdConfig struct {
	Latency    []ServiceThreshold `yaml:"latency"`
	ErrorRate  []ServiceThreshold `yaml:"error_rate"`
	Throughput []ServiceThreshold `yaml:"throughput"`
}

// ServiceThreshold contains a threshold for a specific service
type ServiceThreshold struct {
	Service string  `yaml:"service"`
	Value   float64 `yaml:"value"`
}

// LoadConfig loads configuration from a YAML file and overrides with environment variables
func LoadConfig(configPath string) (*Config, error) {
	// Default configuration
	config := &Config{
		Global: GlobalConfig{
			LogLevel:       "info",
			MetricsEnabled: true,
			MetricsPort:    9090,
		},
	}

	// Read configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Override with environment variables
	overrideWithEnv(config)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// overrideWithEnv overrides configuration with environment variables
func overrideWithEnv(config *Config) {
	// Override global settings
	if logLevel := os.Getenv("AEGIS_LOG_LEVEL"); logLevel != "" {
		config.Global.LogLevel = logLevel
	}

	if metricsEnabled := os.Getenv("AEGIS_METRICS_ENABLED"); metricsEnabled != "" {
		config.Global.MetricsEnabled = metricsEnabled == "true"
	}

	if metricsPort := os.Getenv("AEGIS_METRICS_PORT"); metricsPort != "" {
		fmt.Sscanf(metricsPort, "%d", &config.Global.MetricsPort)
	}

	// Override service configurations
	// This is a simplified approach; a more robust solution would be needed for production
	for i, service := range config.Services {
		envPrefix := fmt.Sprintf("AEGIS_SERVICE_%s_", strings.ToUpper(service.Name))

		if endpoint := os.Getenv(envPrefix + "ENDPOINT"); endpoint != "" {
			config.Services[i].Endpoint = endpoint
		}

		if interval := os.Getenv(envPrefix + "INTERVAL"); interval != "" {
			if duration, err := time.ParseDuration(interval); err == nil {
				config.Services[i].Interval = duration
			}
		}

		if timeout := os.Getenv(envPrefix + "TIMEOUT"); timeout != "" {
			if duration, err := time.ParseDuration(timeout); err == nil {
				config.Services[i].Timeout = duration
			}
		}

		if retries := os.Getenv(envPrefix + "RETRIES"); retries != "" {
			fmt.Sscanf(retries, "%d", &config.Services[i].Retries)
		}

		if onFailure := os.Getenv(envPrefix + "ON_FAILURE"); onFailure != "" {
			config.Services[i].Actions.OnFailure = onFailure
		}
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate log level
	logLevel := strings.ToLower(config.Global.LogLevel)
	if logLevel != "debug" && logLevel != "info" && logLevel != "warn" && logLevel != "error" && logLevel != "fatal" {
		return fmt.Errorf("invalid log level: %s", config.Global.LogLevel)
	}

	// Validate services
	for _, service := range config.Services {
		// Validate service type
		serviceType := strings.ToLower(service.Type)
		if serviceType != "http" && serviceType != "tcp" && serviceType != "grpc" && serviceType != "script" {
			return fmt.Errorf("invalid service type for %s: %s", service.Name, service.Type)
		}

		// Validate interval and timeout
		if service.Interval < time.Second {
			return fmt.Errorf("interval for service %s must be at least 1 second", service.Name)
		}

		if service.Timeout < time.Millisecond*100 {
			return fmt.Errorf("timeout for service %s must be at least 100 milliseconds", service.Name)
		}

		// Validate retries
		if service.Retries < 0 {
			return fmt.Errorf("retries for service %s must be non-negative", service.Name)
		}

		// Validate actions
		onFailure := strings.ToLower(service.Actions.OnFailure)
		if onFailure != "restart" && onFailure != "notify" && onFailure != "ignore" {
			return fmt.Errorf("invalid onFailure action for %s: %s", service.Name, service.Actions.OnFailure)
		}

		// Validate script type specific configuration
		if serviceType == "script" {
			if _, err := os.Stat(service.Endpoint); os.IsNotExist(err) {
				return fmt.Errorf("script file for service %s does not exist: %s", service.Name, service.Endpoint)
			}
		}
	}

	// Validate webhook configuration if enabled
	if config.Notifications.Webhook.Enabled {
		if config.Notifications.Webhook.Endpoint == "" {
			return fmt.Errorf("webhook endpoint must be specified if webhook notifications are enabled")
		}
	}

	// Validate log analysis configuration if enabled
	if config.Logs.Enabled {
		if config.Logs.ScanInterval < time.Second*10 {
			return fmt.Errorf("log scan interval must be at least 10 seconds")
		}

		for _, logFile := range config.Logs.Files {
			if logFile.Path == "" {
				return fmt.Errorf("log file path must be specified for service %s", logFile.Service)
			}
		}

		for _, pattern := range config.Logs.Patterns {
			if pattern.Pattern == "" {
				return fmt.Errorf("pattern must be specified for log pattern %s", pattern.Name)
			}

			// Try to compile the pattern to validate it
			_, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				return fmt.Errorf("invalid regex pattern for %s: %v", pattern.Name, err)
			}
		}
	}

	// Validate anomaly detection configuration if enabled
	if config.AnomalyDetection.Enabled {
		if config.AnomalyDetection.DetectionInterval < time.Second*5 {
			return fmt.Errorf("anomaly detection interval must be at least 5 seconds")
		}

		if config.AnomalyDetection.HistorySize < 10 {
			return fmt.Errorf("anomaly detection history size must be at least 10")
		}
	}

	return nil
}

// PrintConfig prints the current configuration
func PrintConfig(config *Config) {
	logger.Info("Aegis-Orchestrator Configuration:")
	logger.Info("Global:")
	logger.Info("  Log Level: %s", config.Global.LogLevel)
	logger.Info("  Metrics Enabled: %v", config.Global.MetricsEnabled)
	logger.Info("  Metrics Port: %d", config.Global.MetricsPort)

	logger.Info("Services:")
	for _, service := range config.Services {
		logger.Info("  - Name: %s", service.Name)
		logger.Info("    Type: %s", service.Type)
		logger.Info("    Endpoint: %s", service.Endpoint)
		logger.Info("    Interval: %s", service.Interval)
		logger.Info("    Timeout: %s", service.Timeout)
		logger.Info("    Retries: %d", service.Retries)
		logger.Info("    On Failure: %s", service.Actions.OnFailure)
	}

	logger.Info("Notifications:")
	logger.Info("  Webhook Enabled: %v", config.Notifications.Webhook.Enabled)
	if config.Notifications.Webhook.Enabled {
		logger.Info("  Webhook Endpoint: %s", config.Notifications.Webhook.Endpoint)
	}

	if config.Logs.Enabled {
		logger.Info("Log Analysis:")
		logger.Info("  Enabled: %v", config.Logs.Enabled)
		logger.Info("  Scan Interval: %s", config.Logs.ScanInterval)
		logger.Info("  Log Files: %d", len(config.Logs.Files))
		logger.Info("  Patterns: %d", len(config.Logs.Patterns))
	}

	if config.AnomalyDetection.Enabled {
		logger.Info("Anomaly Detection:")
		logger.Info("  Enabled: %v", config.AnomalyDetection.Enabled)
		logger.Info("  Detection Interval: %s", config.AnomalyDetection.DetectionInterval)
		logger.Info("  History Size: %d", config.AnomalyDetection.HistorySize)
	}
}
