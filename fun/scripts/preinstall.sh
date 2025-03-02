#!/bin/bash
# Pre-installation script for Fun Server

# Create required directories if they don't exist
mkdir -p /etc/fun
mkdir -p /var/lib/fun

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "WARNING: Docker is not installed. Fun Server works best with Docker."
    echo "Please install Docker to get the full functionality of Fun Server."
fi

exit 0 