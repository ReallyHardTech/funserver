# Build script for Fun Server on Windows using GoReleaser
param (
  [switch]$Release,
  [switch]$Snapshot,
  [switch]$Clean
)

$ErrorActionPreference = "Stop"

# Ensure GoReleaser is installed
if (-not (Get-Command goreleaser -ErrorAction SilentlyContinue)) {
  Write-Host "GoReleaser not found. Would you like to install it? (y/n)" -ForegroundColor Yellow
  $choice = Read-Host
  if ($choice -eq "y") {
    Write-Host "Installing GoReleaser via go install..."
    go install github.com/goreleaser/goreleaser@latest
    if (-not $?) {
      Write-Host "Failed to install GoReleaser. Please install it manually: https://goreleaser.com/install/" -ForegroundColor Red
      exit 1
    }
  }
  else {
    Write-Host "Please install GoReleaser to continue: https://goreleaser.com/install/" -ForegroundColor Yellow
    exit 1
  }
}

# Clean build artifacts if requested
if ($Clean) {
  Write-Host "Cleaning build artifacts..."
  if (Test-Path "dist") {
    Remove-Item -Path "dist" -Recurse -Force
  }
}

# Determine build mode
if ($Release) {
  # This is a real release
  Write-Host "Building release with GoReleaser..." -ForegroundColor Green
  goreleaser release --clean
}
elseif ($Snapshot) {
  # This is a snapshot/testing build
  Write-Host "Building snapshot with GoReleaser..." -ForegroundColor Cyan
  goreleaser release --snapshot --clean --skip-publish
}
else {
  # Default: local build for testing
  Write-Host "Building locally with GoReleaser..." -ForegroundColor Blue
  goreleaser build --clean --snapshot --single-target
}

if ($?) {
  Write-Host "Build completed successfully! Check the ./dist directory for outputs." -ForegroundColor Green
}
else {
  Write-Host "Build failed." -ForegroundColor Red
  exit 1
} 