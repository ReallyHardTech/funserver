#!/bin/bash
# Post-removal script for Fun Server

# Remove symbolic link if it exists
if [ -L /usr/local/bin/fun ]; then
    rm -f /usr/local/bin/fun
fi

# Reload systemd if available
if command -v systemctl &> /dev/null; then
    systemctl daemon-reload || true
fi

# Update icon cache if GTK is installed
if command -v gtk-update-icon-cache &> /dev/null; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor
fi

# Update desktop database if installed
if command -v update-desktop-database &> /dev/null; then
    update-desktop-database -q
fi

# Only remove directories on full removal (not on upgrade)
if [ "$1" = "remove" ] || [ "$1" = "0" ]; then
    # Remove configuration directory if empty
    if [ -d /etc/fun ] && [ -z "$(ls -A /etc/fun)" ]; then
        rmdir /etc/fun
    else
        echo "Configuration directory /etc/fun not removed as it contains user files."
    fi
    
    # Remove data directory if empty
    if [ -d /var/lib/fun ] && [ -z "$(ls -A /var/lib/fun)" ]; then
        rmdir /var/lib/fun
    else
        echo "Data directory /var/lib/fun not removed as it contains user files."
    fi
fi

exit 0 