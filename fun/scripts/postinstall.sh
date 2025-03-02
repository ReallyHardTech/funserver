#!/bin/bash
# Post-installation script for Fun Server

# Set permissions
chmod 755 /usr/bin/fun
chmod -R 755 /etc/fun
chmod 644 /etc/systemd/system/fun.service

# Create a symbolic link for easier access
if [ ! -e /usr/local/bin/fun ]; then
    ln -sf /usr/bin/fun /usr/local/bin/fun
fi

# Update icon cache if GTK is installed
if command -v gtk-update-icon-cache &> /dev/null; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor
fi

# Update desktop database if installed
if command -v update-desktop-database &> /dev/null; then
    update-desktop-database -q
fi

# Enable and start systemd service if systemd is available
if command -v systemctl &> /dev/null; then
    echo "Enabling Fun Server systemd service..."
    systemctl daemon-reload
    systemctl enable fun.service
    
    # Start the service (don't fail if it doesn't start)
    systemctl start fun.service || echo "Service failed to start, but installation will continue."
else
    echo "Systemd not found. Fun Server service will not be automatically started."
    echo "You can start it manually by running: fun --daemon"
fi

echo "Fun Server has been successfully installed!"
echo "Run 'fun --help' to get started."

exit 0 