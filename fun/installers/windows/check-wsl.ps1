# WSL2 Check and Installation Helper Script
# This script checks if WSL2 is available and provides guidance for installation

param (
  [string]$InstallDir = $PSScriptRoot  # Default to script directory if not provided
)

# Import Windows container documentation path
$docsPath = Join-Path -Path $InstallDir -ChildPath "README-WINDOWS-CONTAINERS.md"

# Function to check if running as administrator
function Test-Admin {
  $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
  $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Function to check if Windows version is compatible with WSL2
function Test-WindowsCompatibility {
  $osVersion = [System.Environment]::OSVersion.Version
    
  # Windows 10 version 2004 (build 19041) or higher is required for WSL2
  if ($osVersion.Major -eq 10 -and $osVersion.Build -ge 19041) {
    return $true
  }
  elseif ($osVersion.Major -gt 10) {
    # Windows 11 or higher
    return $true
  }
    
  return $false
}

# Function to check if WSL is installed
function Test-WSLInstalled {
  try {
    $wslCommand = Get-Command wsl.exe -ErrorAction SilentlyContinue
    return ($null -ne $wslCommand)
  }
  catch {
    return $false
  }
}

# Function to check if WSL2 is available
function Test-WSL2Available {
  if (-not (Test-WSLInstalled)) {
    return $false
  }
    
  try {
    $wslOutput = wsl.exe --status 2>&1
    return ($wslOutput -match "Default Version: 2" -or $wslOutput -match "WSL 2")
  }
  catch {
    return $false
  }
}

# Function to launch the WSL2 installation helper
function Start-WSL2InstallationHelper {
  $wslHelperPath = Join-Path -Path $InstallDir -ChildPath "install-wsl.ps1"
    
  if (Test-Admin) {
    # We're already admin, just run it
    & powershell.exe -ExecutionPolicy Bypass -NoProfile -File "$wslHelperPath" -InstallDir "$InstallDir"
  }
  else {
    # Try to elevate
    Start-Process powershell.exe -ArgumentList "-ExecutionPolicy Bypass -NoProfile -File `"$wslHelperPath`" -InstallDir `"$InstallDir`"" -Verb RunAs
  }
}

# Main script execution
Write-Host "Checking WSL2 prerequisites for Fun container host features..." -ForegroundColor Cyan

# First check Windows compatibility
if (-not (Test-WindowsCompatibility)) {
  Write-Host "Your Windows version is not compatible with WSL2." -ForegroundColor Red
  Write-Host "Windows 10 version 2004 (build 19041) or higher is required." -ForegroundColor Red
  Write-Host "You can still use Fun Server with Windows containers, but Linux containers will not be available." -ForegroundColor Yellow
    
  # Show a dialog with the information
  $null = [System.Windows.Forms.MessageBox]::Show(
    "Your Windows version is not compatible with WSL2. Windows 10 version 2004 (build 19041) or higher is required.`n`nYou can still use Fun Server with Windows containers, but Linux containers will not be available.",
    "WSL2 Compatibility Check",
    [System.Windows.Forms.MessageBoxButtons]::OK,
    [System.Windows.Forms.MessageBoxIcon]::Warning
  )
    
  exit 0
}

# Check if WSL2 is available
if (Test-WSL2Available) {
  Write-Host "WSL2 is installed and available. Linux containers are supported." -ForegroundColor Green
  exit 0
}
else {
  Write-Host "WSL2 is not available on your system." -ForegroundColor Yellow
    
  # Ask if the user wants to install WSL2
  Add-Type -AssemblyName System.Windows.Forms
  $result = [System.Windows.Forms.MessageBox]::Show(
    "WSL2 (Windows Subsystem for Linux 2) is required for Linux container support, but is not available on your system.`n`nWould you like to view installation instructions and run the WSL2 installation helper?`n`nNote: This may require a system restart to complete.",
    "WSL2 Installation Required",
    [System.Windows.Forms.MessageBoxButtons]::YesNo,
    [System.Windows.Forms.MessageBoxIcon]::Question
  )
    
  if ($result -eq [System.Windows.Forms.DialogResult]::Yes) {
    # Launch the WSL2 installation helper
    Start-WSL2InstallationHelper
  }
  else {
    Write-Host "WSL2 installation skipped. You can still use Fun Server with Windows containers." -ForegroundColor Yellow
    Write-Host "To install WSL2 later, refer to the README-WINDOWS-CONTAINERS.md file in the installation directory." -ForegroundColor Cyan
        
    # Try to open the instructions file
    if (Test-Path $docsPath) {
      Start-Process $docsPath
    }
  }
}

exit 0 