# Test script for Aegis-Orchestrator Phase 2 features

# Start the webhook server in a new PowerShell window
Start-Process powershell -ArgumentList "-Command", "cd 'd:\Project Utama\Golang\Aegis-Orchestrator\examples\webhook-server'; go run main.go"

Write-Host "Starting webhook server..."
Start-Sleep -Seconds 2

# Start the example HTTP service
Start-Process powershell -ArgumentList "-Command", "cd 'd:\Project Utama\Golang\Aegis-Orchestrator\examples\http-service'; go run main.go"

Write-Host "Starting example HTTP service..."
Start-Sleep -Seconds 2

# Start Aegis-Orchestrator with debug logging
Write-Host "Starting Aegis-Orchestrator..."
Start-Process powershell -ArgumentList "-Command", "cd 'd:\Project Utama\Golang\Aegis-Orchestrator'; .\aegis.exe --log-level=debug"

Write-Host "All services started. Press Enter to toggle HTTP service status..."
Read-Host

# Toggle HTTP service status to DOWN
Invoke-WebRequest -Uri "http://localhost:8081/toggle" -Method GET | Out-Null
Write-Host "HTTP service toggled to DOWN"

Write-Host "Wait for Aegis to detect the service is down and send alerts..."
Start-Sleep -Seconds 10

Write-Host "Press Enter to toggle HTTP service status back to UP..."
Read-Host

# Toggle HTTP service status back to UP
Invoke-WebRequest -Uri "http://localhost:8081/toggle" -Method GET | Out-Null
Write-Host "HTTP service toggled to UP"

Write-Host "Wait for Aegis to detect the service is up again..."
Start-Sleep -Seconds 10

Write-Host "Test completed. Check the logs and webhook server output for alerts."
Write-Host "Press Enter to exit..."
Read-Host

# Note: You'll need to manually stop the processes that were started