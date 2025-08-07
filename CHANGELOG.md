# Changelog

## [0.2.0] - 2025-08-07

### Added - Phase 2: Monitoring and Detection

#### Health Checking
- Added gRPC health checker for monitoring gRPC services
- Added script health checker for custom health checks
- Enhanced timeout and retry policies with exponential backoff and jitter
- Added support for service-specific retry policies

#### Log Analysis
- Added pattern-based log analysis for error detection
- Implemented configurable log file monitoring
- Added support for custom log patterns with severity levels
- Integrated log analysis with alerting system

#### Anomaly Detection
- Added latency monitoring and anomaly detection
- Added throughput monitoring and anomaly detection
- Added error rate monitoring and anomaly detection
- Implemented configurable thresholds for each metric
- Added historical data collection for trend analysis

#### Prometheus Integration
- Enhanced Prometheus metrics for better monitoring
- Added service uptime tracking
- Added health check duration metrics
- Added service restart count metrics
- Added anomaly detection metrics

#### Alerting
- Implemented threshold-based alerting
- Added support for webhook notifications
- Added alert severity levels
- Added alert state management (firing, resolved)
- Added alert labels for better categorization

#### Configuration
- Extended YAML configuration for new features
- Added log analysis configuration
- Added anomaly detection configuration
- Added alerting configuration
- Added webhook notification configuration

#### Examples and Documentation
- Added example HTTP service for testing
- Added webhook server for receiving notifications
- Added documentation for Phase 2 features
- Added testing guide for Phase 2 features
- Updated README with Phase 2 information

## [0.1.0] - 2025-08-01

### Added - Phase 1: Core Functionality

- Implemented event-driven control loop
- Added HTTP health checker
- Added TCP health checker
- Implemented automatic service recovery
- Added configuration via YAML and ENV
- Added basic metrics for monitoring
- Implemented service manager for service lifecycle
- Added plugin registry for health checkers