# Testing Phase 2 Features

This guide explains how to test the Phase 2 features of Aegis-Orchestrator.

## Prerequisites

- Windows with PowerShell
- Go 1.18 or higher

## Components

The Phase 2 demo consists of the following components:

1. **Aegis-Orchestrator**: The main service orchestrator
2. **Example HTTP Service**: A simple HTTP service that can be toggled between UP and DOWN states
3. **Webhook Server**: A server that receives alert notifications from Aegis-Orchestrator

## Running the Demo

1. Build all components:

```powershell
# Build Aegis-Orchestrator
cd "d:\Project Utama\Golang\Aegis-Orchestrator"
go build -o aegis.exe .\cmd\aegis\main.go

# Build webhook server
cd "d:\Project Utama\Golang\Aegis-Orchestrator\examples\webhook-server"
go build -o webhook-server.exe main.go

# Build HTTP service
cd "d:\Project Utama\Golang\Aegis-Orchestrator\examples\http-service"
go build -o http-service.exe main.go
```

2. Run the demo script:

```powershell
cd "d:\Project Utama\Golang\Aegis-Orchestrator"
.\run-phase2-demo.ps1
```

This script will:
- Start the webhook server
- Start the example HTTP service
- Start Aegis-Orchestrator
- Provide instructions for testing

## Testing Features

### 1. Health Checking

The example HTTP service has a health endpoint at `http://localhost:8081/health` that returns:
- `{"status":"UP","message":"Service is healthy"}` when the service is up
- `{"status":"DOWN","message":"Service is unhealthy"}` when the service is down

Aegis-Orchestrator is configured to check this endpoint every 5 seconds.

### 2. Service Status Toggle

You can toggle the status of the example HTTP service by visiting `http://localhost:8081/toggle` in your web browser. This will switch the service between UP and DOWN states.

### 3. Log Analysis

The log analyzer is configured to scan log files in the `logs` directory. You can add log entries to test pattern matching:

```powershell
# Add a critical error to the example-service.log
Add-Content -Path "logs\example-service.log" -Value "[2025-08-07 12:00:00] [ERROR] Out of memory error: Java heap space"
```

### 4. Anomaly Detection

The example HTTP service simulates random latency between 50-200ms. You can modify the code to simulate higher latency to trigger anomaly detection:

```go
// Simulate high latency (e.g., 600ms)
time.Sleep(600 * time.Millisecond)
```

### 5. Alerting

When the service status changes to DOWN, Aegis-Orchestrator will:
1. Detect the status change
2. Generate an event
3. Take the configured action (restart or notify)
4. Send an alert to the webhook server if notification is enabled

You can observe the alerts in the webhook server console and in the `alerts.log` file.

### 6. Prometheus Metrics

Aegis-Orchestrator exposes Prometheus metrics at `http://localhost:9090/metrics`. You can use a tool like Prometheus or simply view the metrics in a web browser.

## Cleanup

When you're done testing, close all windows and press Enter in the demo script window to clean up processes.

## Troubleshooting

If you encounter any issues:

1. Check the logs in each window for error messages
2. Verify that all services are running on the expected ports
3. Ensure the configuration in `configs/config.yaml` is correct
4. Restart the demo script if necessary