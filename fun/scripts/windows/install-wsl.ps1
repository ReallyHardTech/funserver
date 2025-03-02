# WSL2 Installation Helper Script
# This script guides the user through installing WSL2

param (
  [string]$InstallDir = $PSScriptRoot  # Default to script directory if not provided
)

# Function to check if running as administrator
function Test-Admin {
  $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
  return $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Ensure we're running as admin
if (-not (Test-Admin)) {
  Write-Host "WSL2 installation requires administrator privileges." -ForegroundColor Red
  Write-Host "Please run this script as an administrator." -ForegroundColor Red
    
  # Try to restart as admin
  Start-Process powershell.exe -ArgumentList "-ExecutionPolicy Bypass -NoProfile -File `"$PSCommandPath`" -InstallDir `"$InstallDir`"" -Verb RunAs
  exit
}

# Function to check if a restart is pending
function Test-PendingReboot {
  if (Get-ChildItem "HKLM:\Software\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending" -EA Ignore) { 
    return $true 
  }
  if (Get-Item "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired" -EA Ignore) { 
    return $true 
  }
  try { 
    $util = [wmiclass]"\\.\root\ccm\clientsdk:CCM_ClientUtilities"
    $status = $util.DetermineIfRebootPending()
    if (($status -ne $null) -and $status.RebootPending) {
      return $true
    }
  }
  catch {}
    
  return $false
}

# Function to install WSL2
function Install-WSL2 {
  Write-Host "Starting WSL2 installation..." -ForegroundColor Cyan
    
  try {
    # Enable Windows features required for WSL2
    Write-Host "Enabling required Windows features..." -ForegroundColor Cyan
        
    # Enable Virtual Machine Platform
    Write-Host "Enabling Virtual Machine Platform..." -ForegroundColor Cyan
    Enable-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform -NoRestart -ErrorAction Stop
        
    # Enable Windows Subsystem for Linux
    Write-Host "Enabling Windows Subsystem for Linux..." -ForegroundColor Cyan
    Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux -NoRestart -ErrorAction Stop
        
    # If a restart is required, prompt the user
    if (Test-PendingReboot) {
      $result = [System.Windows.Forms.MessageBox]::Show(
        "A system restart is required to complete the installation of WSL2 components.`n`nWould you like to restart your computer now?",
        "Restart Required",
        [System.Windows.Forms.MessageBoxButtons]::YesNo,
        [System.Windows.Forms.MessageBoxIcon]::Question
      )
            
      if ($result -eq [System.Windows.Forms.DialogResult]::Yes) {
        # Create a temporary script to run after restart to continue WSL2 setup
        $setupScriptPath = Join-Path -Path $env:TEMP -ChildPath "WSL2PostRestartSetup.ps1"
                
        # Create a script that will run after restart
        @"
# Post-restart WSL2 setup script
Start-Sleep -Seconds 30  # Give the system time to fully initialize
powershell.exe -ExecutionPolicy Bypass -WindowStyle Normal -File "$PSCommandPath" -InstallDir "$InstallDir" -PostRestart
"@ | Out-File -FilePath $setupScriptPath -Encoding ASCII
                
        # Create a scheduled task to run the script after restart
        $action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -WindowStyle Normal -File `"$setupScriptPath`""
        $trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
        $settings = New-ScheduledTaskSettingsSet -DeleteExpiredTaskAfter 00:00:01
        Register-ScheduledTask -TaskName "WSL2PostRestartSetup" -Action $action -Trigger $trigger -Settings $settings -Force
                
        # Initiate the restart
        Write-Host "Restarting your computer in 10 seconds..." -ForegroundColor Yellow
        Start-Sleep -Seconds 10
        Restart-Computer -Force
        exit
      }
      else {
        Write-Host "You chose not to restart. The installation will not be complete until you restart your computer." -ForegroundColor Yellow
      }
    }
        
    # Set WSL default version to 2
    Write-Host "Setting WSL default version to 2..." -ForegroundColor Cyan
    wsl --set-default-version 2
        
    # Installation complete
    Write-Host "WSL2 installation complete!" -ForegroundColor Green
    Write-Host "You can now run Linux containers with Fun Server." -ForegroundColor Green
        
    # Offer to run Fun Server now
    $result = [System.Windows.Forms.MessageBox]::Show(
      "WSL2 has been successfully installed. You can now run Linux containers with Fun Server.`n`nWould you like to start Fun Server now?",
      "Installation Complete",
      [System.Windows.Forms.MessageBoxButtons]::YesNo,
      [System.Windows.Forms.MessageBoxIcon]::Information
    )
        
    if ($result -eq [System.Windows.Forms.DialogResult]::Yes) {
      $funExePath = Join-Path -Path $InstallDir -ChildPath "fun.exe"
      if (Test-Path $funExePath) {
        Start-Process $funExePath
      }
    }
        
  }
  catch {
    Write-Host "An error occurred during WSL2 installation:" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
        
    # Show error message to user
    [System.Windows.Forms.MessageBox]::Show(
      "An error occurred during WSL2 installation:`n`n$($_.Exception.Message)`n`nPlease check the console for more details.",
      "Installation Error",
      [System.Windows.Forms.MessageBoxButtons]::OK,
      [System.Windows.Forms.MessageBoxIcon]::Error
    )
  }
}

# Main script execution
Write-Host "WSL2 Installation Helper" -ForegroundColor Cyan
Write-Host "=======================" -ForegroundColor Cyan

# Add Windows Forms assembly for message boxes
Add-Type -AssemblyName System.Windows.Forms

# Check for alternative installation methods on newer Windows versions
$osVersion = [System.Environment]::OSVersion.Version
$useWslInstallCommand = ($osVersion.Major -eq 10 -and $osVersion.Build -ge 19041) -or ($osVersion.Major -gt 10)

if ($useWslInstallCommand) {
  # Newer Windows 10/11 versions support the streamlined 'wsl --install' command
  Write-Host "Your Windows version supports the streamlined WSL installation command." -ForegroundColor Green
    
  $result = [System.Windows.Forms.MessageBox]::Show(
    "WSL2 can be easily installed using the 'wsl --install' command.`n`nThis will install WSL2 with Ubuntu as the default Linux distribution. Would you like to proceed with this installation method?`n`nNote: This will require a system restart to complete.",
    "WSL2 Installation",
    [System.Windows.Forms.MessageBoxButtons]::YesNo,
    [System.Windows.Forms.MessageBoxIcon]::Question
  )
    
  if ($result -eq [System.Windows.Forms.DialogResult]::Yes) {
    try {
      Write-Host "Installing WSL2 with Ubuntu..." -ForegroundColor Cyan
      $process = Start-Process wsl.exe -ArgumentList "--install" -NoNewWindow -Wait -PassThru
            
      if ($process.ExitCode -eq 0) {
        # Installation initiated successfully
        $result = [System.Windows.Forms.MessageBox]::Show(
          "WSL2 installation has been initiated. Your system needs to restart to complete the installation.`n`nWould you like to restart your computer now?",
          "Restart Required",
          [System.Windows.Forms.MessageBoxButtons]::YesNo,
          [System.Windows.Forms.MessageBoxIcon]::Question
        )
                
        if ($result -eq [System.Windows.Forms.DialogResult]::Yes) {
          Write-Host "Restarting your computer in 10 seconds..." -ForegroundColor Yellow
          Start-Sleep -Seconds 10
          Restart-Computer -Force
        }
        else {
          Write-Host "You chose not to restart. The installation will not be complete until you restart your computer." -ForegroundColor Yellow
        }
      }
      else {
        # Something went wrong, try the manual method
        Write-Host "The automatic installation method failed. Trying the manual method..." -ForegroundColor Yellow
        Install-WSL2
      }
    }
    catch {
      # Error occurred, try the manual method
      Write-Host "The automatic installation method failed with error: $($_.Exception.Message)" -ForegroundColor Red
      Write-Host "Trying the manual method..." -ForegroundColor Yellow
      Install-WSL2
    }
  }
  else {
    # User chose the manual method
    Write-Host "You chose the manual installation method." -ForegroundColor Cyan
    Install-WSL2
  }
}
else {
  # Older Windows versions need the manual approach
  Write-Host "Your Windows version requires the manual WSL2 installation steps." -ForegroundColor Cyan
  Install-WSL2
}

Write-Host "WSL2 Installation Helper script completed." -ForegroundColor Cyan 