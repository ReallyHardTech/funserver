# Windows Uninstaller for Fun Server
# This script uninstalls the Fun Server application and service

param (
  [string]$InstallDir = "$env:ProgramFiles\Fun",
  [switch]$Silent = $false,
  [switch]$KeepConfig = $false,
  [switch]$Force = $false
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
  if (Test-Path $logDir) {
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] $Message"
    Add-Content -Path "$logDir\uninstall.log" -Value $logMessage
  }
}

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
  Write-Message "Please run this script as Administrator" -IsError
  exit 1
}

# Confirm uninstall if not silent
if (-not $Silent -and -not $Force) {
  $confirmation = Read-Host "Are you sure you want to uninstall Fun Server? (y/n)"
  if ($confirmation -ne "y") {
    Write-Message "Uninstallation canceled"
    exit 0
  }
}

# Stop the service
Write-Message "Stopping Fun Server service"
try {
  Start-Process -FilePath "sc.exe" -ArgumentList "stop", "fun" -Wait -NoNewWindow
  Start-Sleep -Seconds 2
}
catch {
  Write-Message "Unable to stop service: $_" -IsError
}

# Uninstall the service
Write-Message "Uninstalling Fun Server service"
try {
  Start-Process -FilePath "$InstallDir\uninstall-service.cmd" -Wait -NoNewWindow
  Start-Sleep -Seconds 2
}
catch {
  Write-Message "Unable to uninstall service: $_" -IsError
}

# Remove from PATH environment variable
$currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
if ($currentPath.Contains($InstallDir)) {
  Write-Message "Removing installation directory from PATH"
  $newPath = $currentPath -replace [regex]::Escape("$InstallDir;"), "" -replace [regex]::Escape(";$InstallDir"), ""
  [Environment]::SetEnvironmentVariable("PATH", $newPath, "Machine")
}

# Remove configuration if not keeping it
if (-not $KeepConfig) {
  $configDir = "$env:ProgramData\Fun"
  Write-Message "Removing configuration directory: $configDir"
  if (Test-Path $configDir) {
    Remove-Item -Path $configDir -Recurse -Force
  }
}

# Remove installation directory
Write-Message "Removing installation directory: $InstallDir"
if (Test-Path $InstallDir) {
  Remove-Item -Path $InstallDir -Recurse -Force
}

# Check for Docker containers managed by Fun
try {
  $containers = docker ps -a --filter "label=managed-by=fun" --format "{{.ID}}"
  if ($containers) {
    if (-not $Silent -and -not $Force) {
      $confirmation = Read-Host "Do you want to remove Docker containers managed by Fun Server? (y/n)"
      if ($confirmation -eq "y") {
        Write-Message "Removing Docker containers managed by Fun Server"
        docker rm -f $containers
      }
    }
    elseif ($Force) {
      Write-Message "Removing Docker containers managed by Fun Server"
      docker rm -f $containers
    }
  }
}
catch {
  Write-Message "Unable to check for Docker containers: $_" -IsError
}

# Check for Docker networks managed by Fun
try {
  $networks = docker network ls --filter "label=managed-by=fun" --format "{{.ID}}"
  if ($networks) {
    if (-not $Silent -and -not $Force) {
      $confirmation = Read-Host "Do you want to remove Docker networks managed by Fun Server? (y/n)"
      if ($confirmation -eq "y") {
        Write-Message "Removing Docker networks managed by Fun Server"
        docker network rm $networks
      }
    }
    elseif ($Force) {
      Write-Message "Removing Docker networks managed by Fun Server"
      docker network rm $networks
    }
  }
}
catch {
  Write-Message "Unable to check for Docker networks: $_" -IsError
}

# Installation complete
Write-Message "Fun Server has been successfully uninstalled from $InstallDir"

if (-not $Silent) {
  Write-Host "`nThank you for using Fun Server. We hope to see you again soon!" -ForegroundColor Yellow
    
  if ($KeepConfig) {
    Write-Host "Configuration has been preserved at $env:ProgramData\Fun" -ForegroundColor Yellow
  }
} 