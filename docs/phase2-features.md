# Aegis-Orchestrator Phase 2 Features

## Health Checking Advanced

### gRPC Health Checker

The gRPC health checker allows monitoring of gRPC services using the standard gRPC health checking protocol.

**Configuration Example:**
```yaml
- name: "example-grpc-service"
  type: "grpc"
  endpoint: "localhost:50051"
  interval: 20s
  timeout: 5s
  retries: 2
  actions:
    onFailure: "notify"
  metadata:
    service: "health.v1.HealthService"  # Optional gRPC service name for health check
```

### Script Health Checker

The script health checker allows running custom scripts to check service health. This is useful for complex health checks that can't be done with standard HTTP, TCP, or gRPC checks.

**Configuration Example:**
```yaml
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
```

### Timeout and Retry Policies

Enhanced timeout and retry policies with exponential backoff and jitter for more robust health checking.

**Configuration Example:**
```yaml
- name: "example-service"
  type: "http"
  endpoint: "http://localhost:8081/health"
  interval: 5s
  timeout: 5s
  retries: 3  # Number of retries before marking as failed
  actions:
    onFailure: "restart"
```

## Anomaly Detection

### Latency Monitoring

Monitors service response times and detects abnormal latency spikes.

**Configuration Example:**
```yaml
anomaly_detection:
  enabled: true
  detection_interval: "30s"
  history_size: 100
  thresholds:
    latency:
      - service: "example-service"
        value: 500  # milliseconds
```

### Error Rate Monitoring

Monitors service error rates and detects abnormal increases.

**Configuration Example:**
```yaml
anomaly_detection:
  enabled: true
  detection_interval: "30s"
  history_size: 100
  thresholds:
    error_rate:
      - service: "example-service"
        value: 0.05  # 5%
```

### Throughput Monitoring

Monitors service throughput and detects abnormal decreases.

**Configuration Example:**
```yaml
anomaly_detection:
  enabled: true
  detection_interval: "30s"
  history_size: 100
  thresholds:
    throughput:
      - service: "example-service"
        value: 10  # requests/second
```

## Log Analysis

### Pattern-Based Log Analysis

Analyzes service logs for patterns that indicate problems.

**Configuration Example:**
```yaml
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
```

## Prometheus Integration

Enhanced Prometheus metrics for better monitoring and alerting.

**Metrics Available:**
- `aegis_service_status` - Current status of the service (1=up, 0=down)
- `aegis_service_uptime_seconds` - Uptime of the service in seconds
- `aegis_service_restarts_total` - Total number of service restarts
- `aegis_health_checks_total` - Total number of health checks
- `aegis_health_check_errors_total` - Total number of health check errors
- `aegis_health_check_duration_seconds` - Duration of health checks in seconds
- `aegis_last_health_check_timestamp` - Timestamp of the last health check

## Alerting

### Threshold-Based Alerting

Generates alerts based on configurable thresholds.

**Alert Types:**
- Service Down
- Service Failed to Restart
- High Latency
- High Error Rate
- Low Throughput

### Webhook Notifications

Sends alerts to a webhook endpoint for integration with external systems.

**Configuration Example:**
```yaml
notifications:
  webhook:
    enabled: true
    endpoint: "http://localhost:8080/webhook"
    headers:
      Authorization: "Bearer example-token"
      Content-Type: "application/json"
```

## Testing

To test the Phase 2 features, you can use the provided test script:

```powershell
.\scripts\test-phase2.ps1
```

This script will:
1. Start the webhook server
2. Start the example HTTP service
3. Start Aegis-Orchestrator
4. Toggle the HTTP service status to DOWN
5. Wait for Aegis to detect the service is down and send alerts
6. Toggle the HTTP service status back to UP
7. Wait for Aegis to detect the service is up again