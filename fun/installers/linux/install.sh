#!/bin/bash
# Linux Installer for Fun Server
# This script installs the Fun Server application and service

# Default install locations
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/fun"
LOG_DIR="/var/log/fun"
SERVICE_DIR="/etc/systemd/system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to log messages
log_message() {
    if [ "$2" == "error" ]; then
        echo -e "${RED}ERROR: $1${NC}"
    else
        echo -e "${GREEN}$1${NC}"
    fi
    
    # Also log to a file
    if [ ! -d "$LOG_DIR" ]; then
        mkdir -p "$LOG_DIR"
    fi
    
    TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")
    echo "[$TIMESTAMP] $1" >> "$LOG_DIR/install.log"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log_message "Please run this script as root or with sudo" "error"
    exit 1
fi

# Create installation directories
log_message "Creating installation directories"
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"
mkdir -p "$LOG_DIR"

# Copy binary to installation directory
log_message "Installing Fun Server binary"
cp fun "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/fun"

# Copy documentation
cp LICENSE README.md "$CONFIG_DIR/"

# Create default configuration if it doesn't exist
CONFIG_FILE="$CONFIG_DIR/config.json"
if [ ! -f "$CONFIG_FILE" ]; then
    log_message "Creating default configuration"
    cat > "$CONFIG_FILE" << EOF
{
  "cloud_url": "https://api.thefunserver.com",
  "poll_interval": 60,
  "docker_host": "unix:///var/run/docker.sock",
  "docker_network": "fun_network",
  "log_level": "info",
  "log_file": "/var/log/fun/fun.log"
}
EOF
    chmod 600 "$CONFIG_FILE"
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    log_message "WARNING: Docker does not appear to be installed. Fun Server requires Docker to function properly." "error"
    log_message "Please install Docker using your distribution's package manager or follow instructions at https://docs.docker.com/engine/install/" "error"
else
    log_message "Docker is installed at: $(which docker)"
    
    # Check if user can access Docker socket
    if [ ! -S /var/run/docker.sock ]; then
        log_message "WARNING: Docker socket not found at /var/run/docker.sock" "error"
    elif [ ! -r /var/run/docker.sock ]; then
        log_message "WARNING: Cannot read Docker socket, you may need to add your user to the 'docker' group" "error"
        log_message "Try running: sudo usermod -aG docker $USER" "error"
    fi
fi

# Create systemd service file
SERVICE_FILE="$SERVICE_DIR/fun.service"
log_message "Creating systemd service file"
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=Fun Server
After=network.target docker.service
Requires=docker.service

[Service]
ExecStart=$INSTALL_DIR/fun --daemon
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd configuration
log_message "Reloading systemd configuration"
systemctl daemon-reload

# Enable and start the service
log_message "Enabling and starting Fun Server service"
systemctl enable fun
systemctl start fun

# Check service status
if systemctl is-active --quiet fun; then
    log_message "Fun Server service is running"
else
    log_message "WARNING: Fun Server service failed to start. Check logs for details." "error"
    log_message "You can view logs with: journalctl -u fun -f" "error"
fi

# Installation complete
log_message "Fun Server has been successfully installed"
log_message "Configuration is located at $CONFIG_DIR"
log_message "Logs are located at $LOG_DIR"

echo -e "\n${YELLOW}To use Fun Server, open a new terminal and type 'fun' followed by a command.${NC}"
echo -e "${YELLOW}Example: fun status${NC}"
echo -e "\n${YELLOW}To uninstall, run: /usr/local/bin/uninstall-fun.sh${NC}"

# Create uninstaller script
UNINSTALLER="/usr/local/bin/uninstall-fun.sh"
log_message "Creating uninstaller script at $UNINSTALLER"
cat > "$UNINSTALLER" << 'EOF'
#!/bin/bash
# Linux Uninstaller for Fun Server

# Default install locations
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/fun"
LOG_DIR="/var/log/fun"
SERVICE_DIR="/etc/systemd/system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to log messages
log_message() {
    if [ "$2" == "error" ]; then
        echo -e "${RED}ERROR: $1${NC}"
    else
        echo -e "${GREEN}$1${NC}"
    fi
    
    # Also log to a file
    if [ -d "$LOG_DIR" ]; then
        TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")
        echo "[$TIMESTAMP] $1" >> "$LOG_DIR/uninstall.log"
    fi
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log_message "Please run this script as root or with sudo" "error"
    exit 1
fi

# Confirm uninstall
if [ "$1" != "--force" ]; then
    echo -e "${YELLOW}Are you sure you want to uninstall Fun Server? (y/n)${NC}"
    read -r confirm
    if [ "$confirm" != "y" ]; then
        log_message "Uninstallation canceled"
        exit 0
    fi
fi

# Stop and disable service
log_message "Stopping and disabling Fun Server service"
systemctl stop fun
systemctl disable fun

# Remove service file
log_message "Removing systemd service file"
rm -f "$SERVICE_DIR/fun.service"
systemctl daemon-reload

# Remove binary
log_message "Removing Fun Server binary"
rm -f "$INSTALL_DIR/fun"

# Ask about removing configuration
if [ "$1" != "--force" ]; then
    echo -e "${YELLOW}Do you want to remove configuration and logs? (y/n)${NC}"
    read -r confirm
    if [ "$confirm" = "y" ]; then
        log_message "Removing configuration and logs"
        rm -rf "$CONFIG_DIR"
        rm -rf "$LOG_DIR"
    else
        log_message "Keeping configuration and logs"
    fi
else
    log_message "Removing configuration and logs"
    rm -rf "$CONFIG_DIR"
    rm -rf "$LOG_DIR"
fi

# Check for Docker containers managed by Fun
if command -v docker &> /dev/null; then
    containers=$(docker ps -a --filter "label=managed-by=fun" --format "{{.ID}}" 2>/dev/null)
    if [ -n "$containers" ]; then
        if [ "$1" != "--force" ]; then
            echo -e "${YELLOW}Do you want to remove Docker containers managed by Fun Server? (y/n)${NC}"
            read -r confirm
            if [ "$confirm" = "y" ]; then
                log_message "Removing Docker containers managed by Fun Server"
                docker rm -f $containers
            fi
        else
            log_message "Removing Docker containers managed by Fun Server"
            docker rm -f $containers
        fi
    fi
    
    networks=$(docker network ls --filter "label=managed-by=fun" --format "{{.ID}}" 2>/dev/null)
    if [ -n "$networks" ]; then
        if [ "$1" != "--force" ]; then
            echo -e "${YELLOW}Do you want to remove Docker networks managed by Fun Server? (y/n)${NC}"
            read -r confirm
            if [ "$confirm" = "y" ]; then
                log_message "Removing Docker networks managed by Fun Server"
                docker network rm $networks
            fi
        else
            log_message "Removing Docker networks managed by Fun Server"
            docker network rm $networks
        fi
    fi
fi

# Remove this uninstaller
log_message "Removing uninstaller script"
rm -f "$0"

# Uninstallation complete
log_message "Fun Server has been successfully uninstalled"
echo -e "\n${YELLOW}Thank you for using Fun Server. We hope to see you again soon!${NC}"
EOF

chmod +x "$UNINSTALLER"

exit 0 