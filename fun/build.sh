#!/bin/bash
# Build script for Fun Server on Unix platforms using GoReleaser

set -e

# Parse arguments
RELEASE=0
SNAPSHOT=0
CLEAN=0

for arg in "$@"; do
  case $arg in
    --release)
      RELEASE=1
      ;;
    --snapshot)
      SNAPSHOT=1
      ;;
    --clean)
      CLEAN=1
      ;;
    *)
      echo "Unknown option: $arg"
      echo "Usage: ./build.sh [--release] [--snapshot] [--clean]"
      exit 1
      ;;
  esac
done

# Check if GoReleaser is installed
if ! command -v goreleaser &> /dev/null; then
  echo "GoReleaser not found. Would you like to install it? (y/n)"
  read -r choice
  if [ "$choice" = "y" ]; then
    echo "Installing GoReleaser via go install..."
    go install github.com/goreleaser/goreleaser@latest
    if [ $? -ne 0 ]; then
      echo "Failed to install GoReleaser. Please install it manually: https://goreleaser.com/install/"
      exit 1
    fi
  else
    echo "Please install GoReleaser to continue: https://goreleaser.com/install/"
    exit 1
  fi
fi

# Clean build artifacts if requested
if [ $CLEAN -eq 1 ]; then
  echo "Cleaning build artifacts..."
  rm -rf dist
fi

# Determine build mode
if [ $RELEASE -eq 1 ]; then
  # This is a real release
  echo "Building release with GoReleaser..."
  goreleaser release --clean
elif [ $SNAPSHOT -eq 1 ]; then
  # This is a snapshot/testing build
  echo "Building snapshot with GoReleaser..."
  goreleaser release --snapshot --clean --skip-publish
else
  # Default: local build for testing
  echo "Building locally with GoReleaser..."
  goreleaser build --clean --snapshot --single-target
fi

if [ $? -eq 0 ]; then
  echo "Build completed successfully! Check the ./dist directory for outputs."
else
  echo "Build failed."
  exit 1
fi 