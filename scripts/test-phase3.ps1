# Test script for Phase 3 features of Aegis-Orchestrator

# Ensure logs directory exists
if (-not (Test-Path -Path "logs")) {
    New-Item -ItemType Directory -Path "logs" -Force | Out-Null
    Write-Host "Created logs directory"
}

# Start the webhook server
Write-Host "Starting Webhook Server..."
$webhookProcess = Start-Process -FilePath ".\webhook-server.exe" -WorkingDirectory "d:\Project Utama\Golang\Aegis-Orchestrator\examples\webhook-server" -PassThru
Write-Host "Started webhook server with PID $($webhookProcess.Id)"
Start-Sleep -Seconds 2

# Start the example HTTP service
Write-Host "Starting HTTP Service..."
$httpProcess = Start-Process -FilePath ".\http-service.exe" -WorkingDirectory "d:\Project Utama\Golang\Aegis-Orchestrator\examples\http-service" -PassThru
Write-Host "Started HTTP service with PID $($httpProcess.Id)"
Start-Sleep -Seconds 2

# Start Aegis-Orchestrator
Write-Host "Starting Aegis-Orchestrator..."
$aegisProcess = Start-Process -FilePath ".\aegis.exe" -ArgumentList "--log-level=debug" -WorkingDirectory "d:\Project Utama\Golang\Aegis-Orchestrator" -PassThru
Write-Host "Started Aegis-Orchestrator with PID $($aegisProcess.Id)"

Write-Host "`nAll services are now running. Follow these steps to test Phase 3 features:"
Write-Host "1. Wait for Aegis to initialize and start monitoring services (about 10 seconds)"
Write-Host "2. Testing Circuit Breaker Pattern..."

# Wait for initialization
Start-Sleep -Seconds 10

# Test Circuit Breaker Pattern
Write-Host "`n=== TESTING CIRCUIT BREAKER PATTERN ==="
Write-Host "Toggling HTTP service to DOWN state multiple times to trigger circuit breaker..."

# Toggle service to DOWN state
Write-Host "First failure: Toggling service to DOWN..."
Invoke-WebRequest -Uri "http://localhost:8082/toggle" -Method GET | Out-Null
Start-Sleep -Seconds 10

# Toggle service to DOWN state again
Write-Host "Second failure: Toggling service to DOWN again..."
Invoke-WebRequest -Uri "http://localhost:8082/toggle" -Method GET | Out-Null
Start-Sleep -Seconds 10

# Toggle service to UP state
Write-Host "Recovery: Toggling service back to UP..."
Invoke-WebRequest -Uri "http://localhost:8082/toggle" -Method GET | Out-Null
Start-Sleep -Seconds 10

# Test Backoff Strategy
Write-Host "`n=== TESTING BACKOFF STRATEGY ==="
Write-Host "Toggling HTTP service to DOWN state to trigger restart with backoff..."

# Toggle service to DOWN state
Write-Host "Toggling service to DOWN..."
Invoke-WebRequest -Uri "http://localhost:8082/toggle" -Method GET | Out-Null
Start-Sleep -Seconds 5

# Toggle service to DOWN state again to trigger another restart
Write-Host "Toggling service to DOWN again..."
Invoke-WebRequest -Uri "http://localhost:8082/toggle" -Method GET | Out-Null
Start-Sleep -Seconds 10

# Toggle service to UP state
Write-Host "Toggling service back to UP..."
Invoke-WebRequest -Uri "http://localhost:8082/toggle" -Method GET | Out-Null
Start-Sleep -Seconds 5

Write-Host "`nTest completed. Check the logs for circuit breaker state transitions and backoff strategy in action."
Write-Host "Press Ctrl+C to stop all services when done."

# Wait for user to press Ctrl+C
try {
    while ($true) {
        Start-Sleep -Seconds 1
    }
} finally {
    # Clean up processes
    Write-Host "`nStopping all services..."
    Stop-Process -Id $webhookProcess.Id -Force -ErrorAction SilentlyContinue
    Stop-Process -Id $httpProcess.Id -Force -ErrorAction SilentlyContinue
    Stop-Process -Id $aegisProcess.Id -Force -ErrorAction SilentlyContinue
    Write-Host "All services stopped."
}