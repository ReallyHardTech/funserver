#!/bin/bash
# Pre-removal script for Fun Server

# Stop systemd service if systemd is available
if command -v systemctl &> /dev/null; then
    echo "Stopping Fun Server service..."
    systemctl stop fun.service || true
    systemctl disable fun.service || true
fi

# Save user data if needed
if [ -d /var/lib/fun/data ]; then
    echo "Backing up user data to /tmp/fun-backup-$(date +%Y%m%d%H%M%S).tar.gz"
    tar -czf "/tmp/fun-backup-$(date +%Y%m%d%H%M%S).tar.gz" /var/lib/fun/data
fi

exit 0 