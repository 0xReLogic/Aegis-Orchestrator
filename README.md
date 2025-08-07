# Aegis-Orchestrator

A zero-downtime, self-healing micro-services orchestrator built entirely in Go.

## Overview

Aegis-Orchestrator continuously monitors health metrics, detects anomalies, and automatically recovers failing services via graceful restart, rollback, or isolation—before users ever notice. Powered by an event-driven control loop, pluggable health-check adapters (HTTP, gRPC, TCP, custom scripts), and a declarative policy engine, Aegis keeps your fleet resilient while integrating seamlessly with Prometheus, Grafana, Alertmanager, and any container runtime (Docker, Kubernetes, Nomad).

Ship as a single static binary with no external dependencies, configure through YAML or ENV, and scale from a single host to a multi-cluster mesh without changing a line of code.

## Features

- **Event-driven control loop**: Efficiently monitors and manages services
- **Pluggable health checkers**: HTTP and TCP health checks with more to come
- **Automatic service recovery**: Restart services when they fail
- **Configuration via YAML and ENV**: Flexible configuration options
- **Basic metrics**: Monitor service health and status

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
    endpoint: "http://localhost:8080/health"
    interval: 30s
    timeout: 5s
    retries: 3
    actions:
      onFailure: "restart"
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
- **Health Checkers**: Pluggable health check adapters
- **Service Manager**: Manages service lifecycle
- **Configuration**: Flexible configuration system
- **Metrics**: Basic metrics for monitoring

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Author

Allen Elzayn
