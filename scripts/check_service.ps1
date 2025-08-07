param (
    [Parameter(Mandatory=$true)]
    [string]$Service,
    
    [Parameter(Mandatory=$false)]
    [int]$Timeout = 5
)

# Check if service exists
$serviceExists = Get-Service -Name $Service -ErrorAction SilentlyContinue

if (-not $serviceExists) {
    Write-Host "Service '$Service' does not exist"
    exit 1
}

# Check service status
$status = $serviceExists.Status

if ($status -eq "Running") {
    Write-Host "Service '$Service' is running"
    exit 0
} else {
    Write-Host "Service '$Service' is not running (status: $status)"
    exit 1
}