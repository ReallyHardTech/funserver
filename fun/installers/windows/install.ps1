# Windows Installer for Fun Server
# This script installs the Fun Server application and service

param (
    [string]$InstallDir = "$env:ProgramFiles\Fun",
    [switch]$Silent = $false
)

# Function to log messages
function Write-Message {
    param (
        [string]$Message,
        [switch]$IsError
    )
    
    if ($IsError) {
        Write-Host "ERROR: $Message" -ForegroundColor Red
    }
    else {
        Write-Host $Message -ForegroundColor Green
    }
    
    # Also log to a file
    $logDir = "$env:ProgramData\Fun\logs"
    if (-not (Test-Path $logDir)) {
        New-Item -Path $logDir -ItemType Directory -Force | Out-Null
    }
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] $Message"
    Add-Content -Path "$logDir\install.log" -Value $logMessage
}

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Message "Please run this script as Administrator" -IsError
    exit 1
}

# Create installation directory
Write-Message "Creating installation directory: $InstallDir"
if (-not (Test-Path $InstallDir)) {
    New-Item -Path $InstallDir -ItemType Directory -Force | Out-Null
}

# Create configuration directory
$configDir = "$env:ProgramData\Fun"
Write-Message "Creating configuration directory: $configDir"
if (-not (Test-Path $configDir)) {
    New-Item -Path $configDir -ItemType Directory -Force | Out-Null
}

# Extract files from the current directory to the installation directory
Write-Message "Copying files to installation directory"
Copy-Item -Path ".\fun.exe" -Destination "$InstallDir\fun.exe" -Force
Copy-Item -Path ".\LICENSE" -Destination "$InstallDir\LICENSE" -Force
Copy-Item -Path ".\README.md" -Destination "$InstallDir\README.md" -Force

# Create default configuration if it doesn't exist
$configFile = "$configDir\config.json"
if (-not (Test-Path $configFile)) {
    Write-Message "Creating default configuration"
    $defaultConfig = @{
        cloud_url      = "https://api.thefunserver.com"
        poll_interval  = 60
        docker_host    = "npipe:////./pipe/docker_engine"
        docker_network = "fun_network"
        log_level      = "info"
        log_file       = "$env:ProgramData\Fun\logs\fun.log"
    } | ConvertTo-Json -Depth 10
    
    Set-Content -Path $configFile -Value $defaultConfig -Force
}

# Create logs directory
$logDir = "$env:ProgramData\Fun\logs"
if (-not (Test-Path $logDir)) {
    New-Item -Path $logDir -ItemType Directory -Force | Out-Null
}

# Add to PATH environment variable if not already there
$currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
if (-not $currentPath.Contains($InstallDir)) {
    Write-Message "Adding installation directory to PATH"
    [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$InstallDir", "Machine")
}

# Check if Docker is installed
try {
    $docker = Get-Command docker -ErrorAction Stop
    Write-Message "Docker is installed at: $($docker.Source)"
}
catch {
    Write-Message "WARNING: Docker does not appear to be installed. Fun Server requires Docker to function properly." -IsError
    Write-Message "Please install Docker Desktop for Windows from https://www.docker.com/products/docker-desktop" -IsError
}

# Install the service using the install-service.cmd script
Write-Message "Installing Fun Server service"
Start-Process -FilePath "$InstallDir\install-service.cmd" -ArgumentList "$InstallDir" -Wait -NoNewWindow

# Start the service using SC command directly
Write-Message "Starting Fun Server service"
Start-Process -FilePath "sc.exe" -ArgumentList "start", "fun" -Wait -NoNewWindow

# Installation complete
Write-Message "Fun Server has been successfully installed to $InstallDir"
Write-Message "Configuration is located at $configDir"
Write-Message "Logs are located at $logDir"

if (-not $Silent) {
    Write-Host "`nTo use Fun Server, open a new command prompt and type 'fun' followed by a command." -ForegroundColor Yellow
    Write-Host "Example: fun status" -ForegroundColor Yellow
    Write-Host "`nTo uninstall, run: $InstallDir\uninstall.ps1" -ForegroundColor Yellow
} 