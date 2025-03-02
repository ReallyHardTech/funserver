# Windows Container Support

This document explains how containers are supported on Windows using the bundled containerd runtime and WSL2 integration.

## Overview

On Windows, we offer two different approaches to run containers:

1. **Linux containers via WSL2**: This is the recommended approach for most users, especially for Linux containers.
2. **Native Windows containers**: For Windows-specific containers (less common).

The application is designed to automatically set up the appropriate environment based on your system and the type of containers you need to run.

## Linux Containers on Windows

### Prerequisites

To run Linux containers on Windows, you need:

1. **Windows 10 version 2004 or higher / Windows 11**
2. **WSL2 installed and enabled**
3. **Virtualization support enabled in BIOS/UEFI**

The application will automatically detect if these prerequisites are met and provide guidance if any components are missing.

### How It Works

1. When running on Windows, the application checks if WSL2 is available
2. If available, a dedicated WSL2 distribution named "wsl-fun" is created
3. Containerd and required container tools are installed within this distribution
4. The application communicates with the containerd instance running in WSL2
5. All container operations (pull, run, stop, etc.) are forwarded to the WSL2 containerd
6. File sharing is configured for seamless access between Windows and containers

## Installation

The installation process is completely self-contained:

1. Run the Windows installer (.msi)
2. If WSL2 is not detected, you'll be guided to install it
3. After WSL2 is installed, the application will automatically set up the required environment
4. No additional Docker installation is needed

### Installing WSL2 (if required)

If WSL2 is not detected on your system, follow these steps to install it:

1. Open PowerShell as Administrator and run:
   ```powershell
   wsl --install
   ```

2. Restart your computer to complete the installation

3. After restart, open PowerShell and run:
   ```powershell
   wsl --set-default-version 2
   ```

4. Run the application again - it will now detect WSL2 and set up the container environment

## Troubleshooting

### WSL2 not detected

If the application cannot detect WSL2:

1. Verify that Windows 10 version 2004 or later is installed
2. Ensure virtualization is enabled in your BIOS/UEFI settings
3. Check if WSL is installed by running `wsl --version` in PowerShell
4. If installed but not working, try `wsl --shutdown` and then restart WSL

### Container network issues

If containers cannot access the network:

1. Check Windows Firewall settings
2. Verify that the WSL2 distribution has internet access
3. Try restarting the WSL service with `wsl --shutdown` followed by running the app again

### Performance issues

If you experience performance issues with WSL2:

1. You can adjust resource limits in the `.wslconfig` file located at `C:\Users\<YourUsername>\.wslconfig`
2. Provide more memory/CPU by modifying the configuration
3. Keep the containers and their data on the WSL2 filesystem for best I/O performance

## Native Windows Container Support

In addition to Linux containers via WSL2, the application also supports native Windows containers. These are useful when you need to run Windows-specific applications in containers.

### Prerequisites for Native Windows Containers

1. Windows 10/11 Professional or Enterprise edition
2. Hyper-V and Containers Windows features enabled

### Using Native Windows Containers

The application will automatically fall back to native Windows containers if:

1. WSL2 is not available or
2. You explicitly configure it to use Windows containers

## Comparison with Docker Desktop

Our approach is similar to Docker Desktop for Windows but:

1. More lightweight (focused only on containerd)
2. No licensing restrictions or costs
3. Integrated directly with our application
4. No dependency on Docker Engine

## Additional Resources

- [Microsoft WSL2 Documentation](https://docs.microsoft.com/en-us/windows/wsl/install)
- [Windows Container Documentation](https://docs.microsoft.com/en-us/virtualization/windowscontainers/about/)
- [Containerd Documentation](https://containerd.io/docs/) 