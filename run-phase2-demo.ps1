# Run Phase 2 Demo Script for Aegis-Orchestrator

# Create logs directory if it doesn't exist
if (-not (Test-Path "logs")) {
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

Write-Host "`nAll services are now running. Follow these steps to test Phase 2 features:"
Write-Host "1. Wait for Aegis to initialize and start monitoring services (about 10 seconds)"
Write-Host "2. Open a web browser and go to http://localhost:8081/toggle to toggle the HTTP service status to DOWN"
Write-Host "3. Watch the Aegis-Orchestrator window for alerts and events"
Write-Host "4. Watch the webhook server window for alert notifications"
Write-Host "5. Toggle the service back to UP using the same URL"
Write-Host "6. Observe the recovery events in Aegis-Orchestrator"
Write-Host "`nWhen you're done testing, close all windows and press Enter to clean up processes..."
Read-Host

# Clean up processes
try {
    if ($webhookProcess -ne $null -and -not $webhookProcess.HasExited) {
        $webhookProcess.Kill()
    }
} catch { }

try {
    if ($httpProcess -ne $null -and -not $httpProcess.HasExited) {
        $httpProcess.Kill()
    }
} catch { }

try {
    if ($aegisProcess -ne $null -and -not $aegisProcess.HasExited) {
        $aegisProcess.Kill()
    }
} catch { }

Write-Host "Demo completed and processes cleaned up."