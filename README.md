# Aegis-Orchestrator

A zero-downtime, self-healing micro-services orchestrator built entirely in Go.

## Overview

Aegis-Orchestrator continuously monitors health metrics, detects anomalies, and automatically recovers failing services via graceful restart, rollback, or isolation—before users ever notice. Powered by an event-driven control loop, pluggable health-check adapters (HTTP, gRPC, TCP, custom scripts), and a declarative policy engine, Aegis keeps your fleet resilient while integrating seamlessly with Prometheus, Grafana, Alertmanager, and any container runtime (Docker, Kubernetes, Nomad).

Ship as a single static binary with no external dependencies, configure through YAML or ENV, and scale from a single host to a multi-cluster mesh without changing a line of code.

## Features

### Phase 1: Core Functionality
- **Event-driven control loop**: Efficiently monitors and manages services
- **Pluggable health checkers**: HTTP and TCP health checks
- **Automatic service recovery**: Restart services when they fail
- **Configuration via YAML and ENV**: Flexible configuration options
- **Basic metrics**: Monitor service health and status

### Phase 2: Monitoring and Detection
- **Advanced health checking**: gRPC and custom script health checkers
- **Enhanced timeout and retry policies**: Exponential backoff with jitter
- **Log analysis**: Pattern-based log analysis for error detection
- **Anomaly detection**: Latency, throughput, and error rate monitoring
- **Prometheus integration**: Rich metrics for better monitoring
- **Threshold-based alerting**: Generate alerts based on configurable thresholds
- **Webhook notifications**: Send alerts to external systems

## Getting Started

### Prerequisites

- Go 1.18 or higher

### Installation

```bash
# Clone the repository
git clone https://github.com/0xReLogic/Aegis-Orchestrator.git
cd Aegis-Orchestrator

# Build the binary
go build -o aegis ./cmd/aegis
```

### Configuration

Create a configuration file in YAML format:

```yaml
# config.yaml
global:
  logLevel: "info"
  metricsEnabled: true
  metricsPort: 9090

services:
  - name: "example-service"
    type: "http"
    endpoint: "http://localhost:8081/health"
    interval: 5s
    timeout: 5s
    retries: 3
    actions:
      onFailure: "restart"
    metadata:
      latency_threshold: "500"  # milliseconds
      error_rate_threshold: "0.05"  # 5%

  - name: "example-tcp-service"
    type: "tcp"
    endpoint: "localhost:5433"
    interval: 15s
    timeout: 3s
    retries: 2
    actions:
      onFailure: "restart"
      
  - name: "example-grpc-service"
    type: "grpc"
    endpoint: "localhost:50051"
    interval: 20s
    timeout: 5s
    retries: 2
    actions:
      onFailure: "notify"
    metadata:
      service: "health.v1.HealthService"  # Optional gRPC service name
      
  - name: "example-script-service"
    type: "script"
    endpoint: "/path/to/check_service.ps1"
    interval: 30s
    timeout: 10s
    retries: 1
    actions:
      onFailure: "notify"
    metadata:
      shell: "powershell"
      args: "-Service ExampleService -Timeout 5"

notifications:
  webhook:
    enabled: true
    endpoint: "http://localhost:8080/webhook"
    headers:
      Authorization: "Bearer example-token"
      Content-Type: "application/json"
      
logs:
  enabled: true
  scan_interval: "1m"
  files:
    - service: "example-service"
      path: "/path/to/example-service.log"
  patterns:
    - name: "OutOfMemory"
      pattern: "(?i)(out of memory|java\\.lang\\.OutOfMemoryError)"
      severity: "CRITICAL"
      description: "Out of memory error detected"
      
anomaly_detection:
  enabled: true
  detection_interval: "30s"
  history_size: 100
  thresholds:
    latency:
      - service: "example-service"
        value: 500  # milliseconds
    error_rate:
      - service: "example-service"
        value: 0.05  # 5%
    throughput:
      - service: "example-service"
        value: 10  # requests/second
```

### Running

```bash
# Run with default configuration
./aegis

# Run with custom configuration
./aegis --config=/path/to/config.yaml

# Run with custom log level
./aegis --log-level=debug
```

## Architecture

Aegis-Orchestrator is built with a modular architecture:

- **Core Engine**: Implements the event-driven control loop
- **Health Checkers**: Pluggable health check adapters (HTTP, TCP, gRPC, Script)
- **Service Manager**: Manages service lifecycle
- **Configuration**: Flexible configuration system
- **Metrics**: Prometheus integration for monitoring
- **Log Analyzer**: Pattern-based log analysis
- **Anomaly Detector**: Detects abnormal service behavior
- **Alert Manager**: Generates and sends alerts

For more details on Phase 2 features, see [Phase 2 Features Documentation](docs/phase2-features.md).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Author

Allen Elzayn
