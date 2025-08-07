package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Global        GlobalConfig       `yaml:"global"`
	Services      []ServiceConfig    `yaml:"services"`
	Notifications NotificationConfig `yaml:"notifications"`
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
		if serviceType != "http" && serviceType != "tcp" {
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
	}

	// Validate webhook configuration if enabled
	if config.Notifications.Webhook.Enabled {
		if config.Notifications.Webhook.Endpoint == "" {
			return fmt.Errorf("webhook endpoint must be specified if webhook notifications are enabled")
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
}
