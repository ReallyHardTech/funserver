#!/bin/bash
# macOS Installer for Fun Server
# This script installs the Fun Server application and service

# Default install locations
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/Library/Application Support/Fun"
LOG_DIR="/Library/Logs/Fun"
LAUNCH_DAEMON_DIR="/Library/LaunchDaemons"
LINUXKIT_DIR="/usr/local/opt/fun/linuxkit"

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
mkdir -p "$LINUXKIT_DIR"

# Copy binary to installation directory
log_message "Installing Fun Server binary"
cp fun "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/fun"

# Copy documentation
cp LICENSE README.md "$CONFIG_DIR/"

# Install LinuxKit components for container support
log_message "Installing LinuxKit components for container support on macOS"
if [ -d "binaries/darwin/linuxkit" ]; then
    # Copy LinuxKit kernel and initrd 
    cp binaries/darwin/linuxkit/kernel "$LINUXKIT_DIR/"
    cp binaries/darwin/linuxkit/initrd.img "$LINUXKIT_DIR/"
    chmod +x "$LINUXKIT_DIR/kernel"
    
    # Copy bundled HyperKit (no Homebrew dependency)
    if [ -f "binaries/darwin/linuxkit/hyperkit" ]; then
        log_message "Installing bundled HyperKit"
        cp binaries/darwin/linuxkit/hyperkit "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/hyperkit"
        log_message "HyperKit installed at: $INSTALL_DIR/hyperkit"
    else
        log_message "Bundled HyperKit not found. Container functionality may be limited." "error"
    fi
else
    log_message "LinuxKit components not found in the package. Container functionality may be limited on macOS." "error"
fi

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
  "log_file": "/Library/Logs/Fun/fun.log",
  "linuxkit": {
    "enabled": true,
    "kernel": "$LINUXKIT_DIR/kernel",
    "initrd": "$LINUXKIT_DIR/initrd.img",
    "state_dir": "$LINUXKIT_DIR/state"
  }
}
EOF
    chmod 600 "$CONFIG_FILE"
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    log_message "WARNING: Docker does not appear to be installed. Docker is not required, but can be used as an alternative to the bundled containerd." "error"
    log_message "Please install Docker Desktop for Mac from https://www.docker.com/products/docker-desktop if you want to use Docker instead of containerd" "error"
else
    log_message "Docker is installed at: $(which docker)"
    
    # Check if user can access Docker socket
    if [ ! -S /var/run/docker.sock ]; then
        log_message "WARNING: Docker socket not found at /var/run/docker.sock" "error"
    elif [ ! -r /var/run/docker.sock ]; then
        log_message "WARNING: Cannot read Docker socket, you may need to run Docker Desktop" "error"
    fi
fi

# Create launch daemon
PLIST_FILE="$LAUNCH_DAEMON_DIR/com.funserver.fun.plist"
log_message "Creating launch daemon"
cat > "$PLIST_FILE" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.funserver.fun</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/fun</string>
        <string>--daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$LOG_DIR/fun.log</string>
    <key>StandardErrorPath</key>
    <string>$LOG_DIR/fun.log</string>
</dict>
</plist>
EOF
chmod 644 "$PLIST_FILE"

# Load and start the service
log_message "Loading and starting Fun Server service"
launchctl load -w "$PLIST_FILE"

# Check service status
sleep 2
if launchctl list | grep -q com.funserver.fun; then
    log_message "Fun Server service is running"
else
    log_message "WARNING: Fun Server service failed to start. Check logs for details." "error"
    log_message "You can view logs with: cat $LOG_DIR/fun.log" "error"
fi

# Create uninstaller script
UNINSTALLER="$INSTALL_DIR/uninstall-fun"
log_message "Creating uninstaller script at $UNINSTALLER"
cat > "$UNINSTALLER" << 'EOF'
#!/bin/bash
# macOS Uninstaller for Fun Server

# Default install locations
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/Library/Application Support/Fun"
LOG_DIR="/Library/Logs/Fun"
LAUNCH_DAEMON_DIR="/Library/LaunchDaemons"
PLIST_FILE="$LAUNCH_DAEMON_DIR/com.funserver.fun.plist"

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

# Unload service
log_message "Unloading Fun Server service"
launchctl unload -w "$PLIST_FILE" 2>/dev/null

# Remove launch daemon
log_message "Removing launch daemon"
rm -f "$PLIST_FILE"

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

# Installation complete
log_message "Fun Server has been successfully installed"
log_message "Configuration is located at $CONFIG_DIR"
log_message "Logs are located at $LOG_DIR"

echo -e "\n${YELLOW}To use Fun Server, open a new terminal and type 'fun' followed by a command.${NC}"
echo -e "${YELLOW}Example: fun status${NC}"
echo -e "\n${YELLOW}To uninstall, run: sudo $UNINSTALLER${NC}"

exit 0 