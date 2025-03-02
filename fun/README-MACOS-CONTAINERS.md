# macOS Container Support

This document explains how containers are supported on macOS using the bundled containerd runtime.

## Overview

On macOS, containers can't run natively because containers are essentially Linux processes isolated by Linux-specific kernel features like namespaces and cgroups, which aren't available on macOS.

To solve this, we use a LinuxKit-based lightweight virtual machine that runs a minimal Linux kernel with containerd inside. This approach is similar to how Docker Desktop for Mac works.

## How It Works

1. When running on macOS, the application detects the platform and starts a LinuxKit VM
2. The LinuxKit VM runs a minimal Linux distribution with containerd pre-configured
3. The application communicates with the containerd instance running inside the VM
4. All container operations (pull, run, stop, etc.) are forwarded to the VM's containerd
5. Networking and volume mounts are properly configured to work seamlessly between macOS and the VM

## Components

The macOS container support consists of several components, all bundled with the application:

1. **HyperKit**: A lightweight virtualization solution for macOS (bundled - no external dependencies required)
2. **Linux Kernel**: A minimal Linux kernel optimized for container workloads
3. **InitRD**: An initial RAM disk with the basic Linux system
4. **Containerd**: Running inside the VM with appropriate configuration

## Installation

The installation process is completely self-contained:

1. All necessary components are bundled with the application
2. The bundled HyperKit is automatically extracted during installation
3. On first run, the VM is automatically configured and started
4. No external dependencies or manual setup required

Unlike other container solutions for macOS, our implementation doesn't require:
- Homebrew installation
- External driver downloads
- Manual VM configuration
- Docker Desktop installation

## Requirements

- macOS 10.14 (Mojave) or later
- Intel or Apple Silicon processor (both architectures supported)
- At least 4GB of RAM (8GB recommended)
- At least 10GB of free disk space

## Troubleshooting

### VM fails to start

If the LinuxKit VM fails to start:

1. Check if HyperKit is properly installed and executable: `ls -la /usr/local/bin/hyperkit`
2. Ensure you have sufficient permissions: The application may need to be run with elevated privileges
3. Check the logs at `~/Library/Logs/Fun/linuxkit.log`

### Containers cannot be pulled or started

If you experience issues with containers:

1. Make sure the VM is running: `ps aux | grep hyperkit`
2. Check if the VM has network connectivity
3. Verify the containerd socket is accessible within the VM

### Performance issues

If containers run slowly:

1. Increase the VM's memory allocation in the configuration file
2. Reduce the number of concurrent containers
3. Use volume mounts sparingly as they can impact I/O performance

## Technical Details

### LinuxKit Configuration

The LinuxKit VM is configured with:

- A minimal Linux kernel optimized for containers
- Containerd as the container runtime
- CNI plugins for networking
- A shared filesystem for seamless file access between macOS and containers

### Networking

Container networking works by:

1. Creating a virtual network interface on the macOS host
2. Bridging it to the VM's network
3. Setting up NAT for outbound connectivity
4. Port forwarding for inbound connections

### File Sharing

Files are shared between macOS and the VM using:

1. A 9P filesystem for efficient file transfers
2. Automatic mounting of the user's home directory
3. Support for volume mounts in container configurations

## Comparison with Docker Desktop

Our approach is similar to Docker Desktop for Mac but:

1. More lightweight (focused only on containerd)
2. Integrated directly with our application
3. Optimized for our specific use cases
4. No dependency on the Docker daemon
5. No external dependencies required for installation (fully bundled)

## Additional Resources

- [LinuxKit GitHub Repository](https://github.com/linuxkit/linuxkit)
- [HyperKit GitHub Repository](https://github.com/moby/hyperkit)
- [Containerd Documentation](https://containerd.io/docs/) 