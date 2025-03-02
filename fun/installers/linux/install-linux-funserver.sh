#!/bin/bash
set -e

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default install locations - aligning with existing install.sh and goreleaser
INSTALL_DIR="/opt/funserver"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="/etc/funserver"
LOG_DIR="/var/log/funserver"
DATA_DIR="/var/lib/funserver"
SERVICE_DIR="/etc/systemd/system"

# ASCII art logo
print_logo() {
  echo -e "${BLUE}"
  echo "  ______                                           "
  echo " |  ____|                                          "
  echo " | |__ _   _ _ __  ___  ___ _ ____   _____ _ __    "
  echo " |  __| | | | '_ \/ __|/ _ \ '__\ \ / / _ \ '__|   "
  echo " | |  | |_| | | | \__ \  __/ |   \ V /  __/ |      "
  echo " |_|   \__,_|_| |_|___/\___|_|    \_/ \___|_|      "
  echo "                                                    "
  echo -e "${NC}"
  echo -e "${GREEN}The self-hosting platform for the rest of us${NC}"
  echo ""
}

# Function to log messages, similar to existing install.sh
log_message() {
    if [ "$2" == "error" ]; then
        echo -e "${RED}ERROR: $1${NC}"
    elif [ "$2" == "warning" ]; then
        echo -e "${YELLOW}WARNING: $1${NC}"
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

print_logo

# Check if script is run with sudo/root
if [ "$(id -u)" -ne 0 ]; then
    log_message "This script must be run as root or with sudo privileges" "error"
    exit 1
fi

log_message "Starting funserver installation..."

# Detect Linux distribution
if [ -f /etc/os-release ]; then
    . /etc/os-release
    DISTRO=$ID
    VERSION=$VERSION_ID
else
    log_message "Unable to determine Linux distribution. This script supports Ubuntu, Debian, CentOS, Fedora, and RHEL." "error"
    exit 1
fi

log_message "Detected Linux distribution: $DISTRO $VERSION"

# Check system requirements
log_message "Checking system requirements..."
TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
CPU_CORES=$(nproc)

if [ "$TOTAL_MEM" -lt 2048 ]; then
    log_message "Your system has less than 2GB of RAM. Funserver recommends at least 2GB for optimal performance." "warning"
fi

if [ "$CPU_CORES" -lt 2 ]; then
    log_message "Your system has less than 2 CPU cores. Funserver recommends at least 2 cores for optimal performance." "warning"
fi

# Install dependencies based on distribution
log_message "Installing dependencies..."
case $DISTRO in
    ubuntu|debian)
        apt-get update
        apt-get install -y curl wget gnupg lsb-release ca-certificates apt-transport-https software-properties-common
        ;;
    centos|rhel|fedora)
        if [ "$DISTRO" = "centos" ] || [ "$DISTRO" = "rhel" ]; then
            yum install -y curl wget gnupg ca-certificates
        else
            dnf install -y curl wget gnupg ca-certificates
        fi
        ;;
    *)
        log_message "Unsupported distribution: $DISTRO. This script supports Ubuntu, Debian, CentOS, Fedora, and RHEL." "error"
        exit 1
        ;;
esac

# Create necessary directories
log_message "Creating funserver directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"
mkdir -p "$LOG_DIR"
mkdir -p "$DATA_DIR"

# Download the latest funserver release
log_message "Downloading funserver..."
RELEASE_URL="https://thefunserver.com/releases/latest"
DOWNLOAD_DIR=$(mktemp -d)

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        ARCH_PACKAGE="amd64"
        ;;
    i386|i686)
        ARCH_PACKAGE="386"
        ;;
    aarch64|arm64)
        ARCH_PACKAGE="arm64"
        ;;
    *)
        log_message "Unsupported architecture: $ARCH" "error"
        exit 1
        ;;
esac

# Download the appropriate package based on distribution and architecture
if [ "$DISTRO" = "ubuntu" ] || [ "$DISTRO" = "debian" ]; then
    # Check if we can use .deb package
    if command -v dpkg > /dev/null; then
        log_message "Installing via .deb package..."
        PACKAGE_URL="$RELEASE_URL/fun-server_linux_${ARCH_PACKAGE}.deb"
        curl -sSL "$PACKAGE_URL" -o "$DOWNLOAD_DIR/funserver.deb"
        dpkg -i "$DOWNLOAD_DIR/funserver.deb" || {
            apt-get install -f -y  # Fix dependencies if needed
            dpkg -i "$DOWNLOAD_DIR/funserver.deb"
        }
        # Since we used package manager, we can skip systemd setup
        USED_PACKAGE_MANAGER=true
    else
        # Fall back to tarball if dpkg not available
        log_message "Package manager not available, using tarball installation..."
        PACKAGE_URL="$RELEASE_URL/fun-linux-${ARCH_PACKAGE}.tar.gz"
        curl -sSL "$PACKAGE_URL" -o "$DOWNLOAD_DIR/funserver.tar.gz"
        tar -xzf "$DOWNLOAD_DIR/funserver.tar.gz" -C "$INSTALL_DIR"
    fi
elif [ "$DISTRO" = "centos" ] || [ "$DISTRO" = "rhel" ] || [ "$DISTRO" = "fedora" ]; then
    # Check if we can use .rpm package
    if command -v rpm > /dev/null; then
        log_message "Installing via .rpm package..."
        PACKAGE_URL="$RELEASE_URL/fun-server_linux_${ARCH_PACKAGE}.rpm"
        curl -sSL "$PACKAGE_URL" -o "$DOWNLOAD_DIR/funserver.rpm"
        rpm -i "$DOWNLOAD_DIR/funserver.rpm" || {
            if [ "$DISTRO" = "fedora" ] || [ "$DISTRO" = "centos" -a "$VERSION_ID" -ge 8 ] || [ "$DISTRO" = "rhel" -a "$VERSION_ID" -ge 8 ]; then
                dnf install -y "$DOWNLOAD_DIR/funserver.rpm"
            else
                yum install -y "$DOWNLOAD_DIR/funserver.rpm"
            fi
        }
        # Since we used package manager, we can skip systemd setup
        USED_PACKAGE_MANAGER=true
    else
        # Fall back to tarball if rpm not available
        log_message "Package manager not available, using tarball installation..."
        PACKAGE_URL="$RELEASE_URL/fun-linux-${ARCH_PACKAGE}.tar.gz"
        curl -sSL "$PACKAGE_URL" -o "$DOWNLOAD_DIR/funserver.tar.gz"
        tar -xzf "$DOWNLOAD_DIR/funserver.tar.gz" -C "$INSTALL_DIR"
    fi
else
    # For other distributions, use tarball
    log_message "Using tarball installation for $DISTRO..."
    PACKAGE_URL="$RELEASE_URL/fun-linux-${ARCH_PACKAGE}.tar.gz"
    curl -sSL "$PACKAGE_URL" -o "$DOWNLOAD_DIR/funserver.tar.gz"
    tar -xzf "$DOWNLOAD_DIR/funserver.tar.gz" -C "$INSTALL_DIR"
fi

# Clean up download directory
rm -rf "$DOWNLOAD_DIR"

# If we didn't use a package manager, we need to set up symlinks and service
if [ "$USED_PACKAGE_MANAGER" != "true" ]; then
    # Set correct permissions
    log_message "Setting correct permissions..."
    chown -R root:root "$INSTALL_DIR"
    chmod +x "$INSTALL_DIR/bin"/*
    
    # Create symlinks
    log_message "Creating symlinks..."
    ln -sf "$INSTALL_DIR/bin/funserver" "$BIN_DIR/funserver"
    ln -sf "$INSTALL_DIR/bin/fun" "$BIN_DIR/fun"
    
    # Create systemd service file
    log_message "Setting up funserver service..."
    cat > "$SERVICE_DIR/funserver.service" << EOF
[Unit]
Description=Funserver - Self-hosting Platform
After=network.target

[Service]
ExecStart=$INSTALL_DIR/bin/funserver start
Restart=always
User=root
Group=root
Environment=PATH=/usr/bin:/usr/local/bin:$INSTALL_DIR/bin
WorkingDirectory=$INSTALL_DIR

[Install]
WantedBy=multi-user.target
EOF

    # Generate a default configuration if one doesn't exist
    if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
        log_message "Creating default configuration..."
        cat > "$CONFIG_DIR/config.yaml" << EOF
server:
  port: 8080
  address: "0.0.0.0"
storage:
  path: "$DATA_DIR"
logging:
  level: "info"
  file: "$LOG_DIR/funserver.log"
container:
  runtime: "containerd"
  socket: "/var/run/funserver/containerd.sock"
EOF
    fi
    
    # Start and enable service
    log_message "Starting funserver service..."
    systemctl daemon-reload
    systemctl enable funserver.service
    systemctl start funserver.service
fi

# Check for container runtime (supporting both Docker and bundled containerd)
log_message "Checking container runtime..."
if command -v docker &> /dev/null; then
    log_message "Docker is installed at: $(which docker)"
    
    # Check if user can access Docker socket
    if [ ! -S /var/run/docker.sock ]; then
        log_message "Docker socket not found at /var/run/docker.sock" "warning"
    elif [ ! -r /var/run/docker.sock ]; then
        log_message "Cannot read Docker socket, you may need to add your user to the 'docker' group" "warning"
        log_message "Try running: sudo usermod -aG docker $USER" "warning"
    fi
else
    log_message "Docker not found, will use bundled containerd runtime"
    
    # Make sure the containerd directory exists
    mkdir -p /var/run/funserver
    
    # If we used a package manager, the service should already be configured for containerd
    # If not, we might need to adjust the config to use containerd
    if [ "$USED_PACKAGE_MANAGER" != "true" ] && [ -f "$CONFIG_DIR/config.yaml" ]; then
        # Update config to use containerd if not already
        sed -i 's/docker_host:.*/runtime: "containerd"/' "$CONFIG_DIR/config.yaml"
        sed -i 's/docker_network:.*/socket: "\/var\/run\/funserver\/containerd.sock"/' "$CONFIG_DIR/config.yaml"
    fi
fi

# Configure firewall if available
log_message "Configuring firewall..."
if command -v ufw > /dev/null; then
    ufw allow 8080/tcp
    log_message "Opened port 8080 in UFW firewall"
elif command -v firewall-cmd > /dev/null; then
    firewall-cmd --permanent --add-port=8080/tcp
    firewall-cmd --reload
    log_message "Opened port 8080 in firewalld"
fi

# Create uninstaller script
UNINSTALLER="$BIN_DIR/uninstall-funserver.sh"
log_message "Creating uninstaller script at $UNINSTALLER"
cat > "$UNINSTALLER" << 'EOF'
#!/bin/bash
# Linux Uninstaller for Fun Server

# Default install locations
INSTALL_DIR="/opt/funserver"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="/etc/funserver"
LOG_DIR="/var/log/funserver"
DATA_DIR="/var/lib/funserver"
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
    elif [ "$2" == "warning" ]; then
        echo -e "${YELLOW}WARNING: $1${NC}"
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

# Detect package manager
PKG_MANAGER=""
if command -v dpkg > /dev/null && dpkg -l fun-server >/dev/null 2>&1; then
    PKG_MANAGER="deb"
elif command -v rpm > /dev/null && rpm -q fun-server >/dev/null 2>&1; then
    PKG_MANAGER="rpm"
fi

# Confirm uninstall
if [ "$1" != "--force" ]; then
    echo -e "${YELLOW}Are you sure you want to uninstall funserver? (y/n)${NC}"
    read -r confirm
    if [ "$confirm" != "y" ]; then
        log_message "Uninstallation canceled"
        exit 0
    fi
fi

# Stop and disable service
log_message "Stopping and disabling funserver service..."
systemctl stop funserver 2>/dev/null || true
systemctl disable funserver 2>/dev/null || true

# Uninstall based on installation method
if [ "$PKG_MANAGER" = "deb" ]; then
    log_message "Uninstalling funserver package..."
    apt-get remove -y fun-server
elif [ "$PKG_MANAGER" = "rpm" ]; then
    if command -v dnf > /dev/null; then
        dnf remove -y fun-server
    else
        yum remove -y fun-server
    fi
else
    # Manual uninstall
    log_message "Removing funserver files..."
    rm -f "$SERVICE_DIR/funserver.service"
    rm -f "$BIN_DIR/funserver"
    rm -f "$BIN_DIR/fun"
    rm -rf "$INSTALL_DIR"
    systemctl daemon-reload
fi

# Ask about removing configuration
if [ "$1" != "--force" ]; then
    echo -e "${YELLOW}Do you want to remove configuration and data? (y/n)${NC}"
    read -r confirm
    if [ "$confirm" = "y" ]; then
        log_message "Removing configuration and data..."
        rm -rf "$CONFIG_DIR"
        rm -rf "$LOG_DIR"
        rm -rf "$DATA_DIR"
    else
        log_message "Keeping configuration and data"
    fi
else
    log_message "Removing configuration and data..."
    rm -rf "$CONFIG_DIR"
    rm -rf "$LOG_DIR"
    rm -rf "$DATA_DIR"
fi

# Check for containers
if command -v docker &> /dev/null; then
    containers=$(docker ps -a --filter "label=managed-by=funserver" --format "{{.ID}}" 2>/dev/null)
    if [ -n "$containers" ]; then
        if [ "$1" != "--force" ]; then
            echo -e "${YELLOW}Do you want to remove Docker containers managed by funserver? (y/n)${NC}"
            read -r confirm
            if [ "$confirm" = "y" ]; then
                log_message "Removing Docker containers managed by funserver..."
                docker rm -f $containers
            fi
        else
            log_message "Removing Docker containers managed by funserver..."
            docker rm -f $containers
        fi
    fi
    
    networks=$(docker network ls --filter "label=managed-by=funserver" --format "{{.ID}}" 2>/dev/null)
    if [ -n "$networks" ]; then
        if [ "$1" != "--force" ]; then
            echo -e "${YELLOW}Do you want to remove Docker networks managed by funserver? (y/n)${NC}"
            read -r confirm
            if [ "$confirm" = "y" ]; then
                log_message "Removing Docker networks managed by funserver..."
                docker network rm $networks
            fi
        else
            log_message "Removing Docker networks managed by funserver..."
            docker network rm $networks
        fi
    fi
fi

# Remove this uninstaller
log_message "Removing uninstaller script..."
rm -f "$0"

# Uninstallation complete
log_message "Funserver has been successfully uninstalled"
echo -e "\n${YELLOW}Thank you for using funserver. We hope to see you again soon!${NC}"
EOF

chmod +x "$UNINSTALLER"

# Final steps and instructions
log_message "Funserver has been successfully installed!"
echo ""
echo -e "${BLUE}You can access the funserver dashboard at:${NC}"
echo -e "${GREEN}http://localhost:8080${NC}"
echo ""
echo -e "${BLUE}To manage funserver, use the 'fun' command:${NC}"
echo -e "${GREEN}fun apps list${NC} - List installed applications"
echo -e "${GREEN}fun apps install <app-name>${NC} - Install an application"
echo -e "${GREEN}fun status${NC} - Check funserver status"
echo ""
echo -e "${BLUE}For more information, visit:${NC}"
echo -e "${GREEN}https://thefunserver.com/docs${NC}"
echo ""
echo -e "${BLUE}To get started with your first app:${NC}"
echo -e "${GREEN}fun apps install hello-world${NC}"
echo ""
echo -e "${YELLOW}Note: You may need to configure your router/firewall to access funserver from other devices on your network.${NC}"
echo -e "${YELLOW}To uninstall funserver, run: sudo $UNINSTALLER${NC}"

# Check if the service is running
if systemctl is-active --quiet funserver; then
    log_message "Funserver service is running."
else
    log_message "Funserver service failed to start. Please check the logs with: journalctl -u funserver" "error"
fi

echo ""
log_message "Installation complete! Thank you for installing funserver." 