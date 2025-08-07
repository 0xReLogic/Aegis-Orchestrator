# Aegis-Orchestrator Phase 3 Features

## Circuit Breaker Pattern

The Circuit Breaker pattern is a design pattern used to detect failures and prevent cascading failures in distributed systems. It works by monitoring for failures and, once a threshold is reached, "opening" the circuit to prevent further calls to the failing service.

### Circuit Breaker States

The Circuit Breaker has three states:

1. **CLOSED**: The circuit is closed and all requests are allowed to pass through. This is the normal state.
2. **OPEN**: The circuit is open and all requests are blocked. This happens when the failure threshold is reached.
3. **HALF-OPEN**: After a reset timeout, the circuit transitions from OPEN to HALF-OPEN to test if the service has recovered.

### Configuration Example

```yaml
- name: "example-service"
  type: "http"
  endpoint: "http://localhost:8081/health"
  interval: 5s
  timeout: 5s
  retries: 3
  actions:
    onFailure: "restart"
    circuitBreaker:
      enabled: true
      failureThreshold: 3
      resetTimeout: 20s
      successThreshold: 1
```

### Circuit Breaker Parameters

- **enabled**: Enables or disables the circuit breaker.
- **failureThreshold**: Number of consecutive failures before the circuit opens.
- **resetTimeout**: Duration to wait before transitioning from OPEN to HALF-OPEN.
- **successThreshold**: Number of successful health checks required to close the circuit.

## Exponential Backoff with Jitter

Exponential backoff is a technique where retries are performed with increasing delays between attempts. Adding jitter (randomness) helps prevent the "thundering herd" problem where many services restart simultaneously.

### Configuration Example

```yaml
- name: "example-service"
  type: "http"
  endpoint: "http://localhost:8081/health"
  interval: 5s
  timeout: 5s
  retries: 3
  actions:
    onFailure: "restart"
    backoff:
      enabled: true
      initialInterval: 2s
      maxInterval: 30s
      multiplier: 2.0
```

### Backoff Parameters

- **enabled**: Enables or disables the backoff strategy.
- **initialInterval**: Initial delay between restart attempts.
- **maxInterval**: Maximum delay between restart attempts.
- **multiplier**: Factor by which the delay increases with each attempt.

## Automated Service Recovery

Aegis-Orchestrator can automatically recover failing services using different strategies:

### Restart Strategy

When a service fails, Aegis-Orchestrator can automatically restart it. The restart can be immediate or use the backoff strategy.

```yaml
actions:
  onFailure: "restart"
```

### Notify Strategy

Instead of restarting, Aegis-Orchestrator can send notifications about the failure.

```yaml
actions:
  onFailure: "notify"
```

### Ignore Strategy

Aegis-Orchestrator can also be configured to ignore failures for certain services.

```yaml
actions:
  onFailure: "ignore"
```

## Thread Safety

All components in Aegis-Orchestrator are designed to be thread-safe, allowing for concurrent operations without race conditions or data corruption.

- Mutex locks protect shared data structures
- Goroutines are used for concurrent operations
- Channels are used for safe communication between goroutines

## Testing Phase 3 Features

To test the Phase 3 features, you can use the provided test script:

```powershell
.\scripts\test-phase3.ps1
```

This script will:
1. Start the webhook server
2. Start the example HTTP service
3. Start Aegis-Orchestrator
4. Toggle the HTTP service status to DOWN multiple times to trigger the circuit breaker
5. Observe the circuit breaker state transitions
6. Test the backoff strategy by forcing multiple restarts