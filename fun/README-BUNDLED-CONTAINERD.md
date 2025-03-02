# Bundling containerd with your Application

This README explains how to bundle containerd with your application for a zero-dependency installation experience.

## Overview

The project has been modified to bundle containerd and its dependencies with your application, eliminating the need for users to install these components separately. This approach:

1. Downloads containerd, runc, and CNI plugins for multiple platforms (Linux, Windows, macOS) during the build process
2. Packages these binaries with your application
3. Extracts and uses the appropriate binaries at runtime

## Components Included

- **containerd** (v1.6.25): The container runtime
- **runc** (v1.1.10): The OCI container runtime used by containerd
- **CNI plugins** (v1.4.0): Container Network Interface plugins for networking

## Build Process

### Option 1: Manual Build

#### 1. Download required binaries

Run the download script to fetch all required binaries for all supported platforms:

```shell
go run scripts/download_containerd.go
```

This will:
- Create the necessary directory structure in `binaries/`
- Download containerd, runc, and CNI plugins for all supported platforms
- Extract the binaries to their respective platform directories

#### 2. Build your application

Build your application normally:

```shell
go build -o myapp ./cmd/myapp
```

#### 3. Package your application

Package your application with the bundled binaries:

#### For Windows (PowerShell):

```powershell
# Create a release directory
mkdir -p release
# Copy your application
cp myapp.exe release/
# Copy bundled binaries
cp -r binaries release/
```

#### For Linux/macOS:

```bash
# Create a release directory
mkdir -p release
# Copy your application
cp myapp release/
# Copy bundled binaries
cp -r binaries release/
```

### Option 2: Using GoReleaser

The project is configured to use GoReleaser for automated builds and packaging with all components bundled. To build with GoReleaser:

```shell
# Build a snapshot release for testing
goreleaser build --snapshot --clean

# Create a full release (requires a Git tag)
goreleaser release --clean
```

The GoReleaser configuration:
- Automatically downloads all required binaries for all supported platforms
- Includes the appropriate binaries in each platform-specific package:
  - Windows MSI: bundled in `/binaries/windows/`
  - macOS PKG: installed to `/usr/local/bin/fun-containerd` and `/usr/local/opt/fun/cni/`
  - Linux DEB/RPM: installed to `/usr/lib/fun-server/containerd`, `/usr/lib/fun-server/runc`, and `/usr/lib/fun-server/cni/`
  - ZIP/TAR.GZ archives: included in appropriate platform directories

## How It Works

1. When your application starts, it checks if containerd, runc, and CNI plugins are available on the system
2. If not found, it looks for the bundled binaries appropriate for the current platform
3. It extracts the binaries to user-specific locations if needed (only once)
4. It configures containerd to use the bundled runc and CNI plugins
5. This provides a zero-dependency installation experience

## Customization

### Changing component versions

To change the versions of bundled components, edit the version numbers in `scripts/download_containerd.go`:

```go
// Version constants
const (
    containerdVersion = "1.6.25" // Change to your desired containerd version
    runcVersion       = "1.1.10" // Change to your desired runc version
    cniVersion        = "1.4.0"  // Change to your desired CNI plugins version
)
```

### Supporting additional platforms

To support additional platforms or architectures, modify the download URLs and paths in `scripts/download_containerd.go`.

### Modifying GoReleaser configuration

If you need to change how components are bundled with your releases:
1. Edit the `.goreleaser.yml` file
2. Modify the file mappings in the appropriate archive/installer sections
3. Run `goreleaser build --snapshot --clean` to test your changes

## Troubleshooting

### Missing binaries

If the application reports that binaries are not available:

1. Ensure the bundled binaries were correctly packaged with your application
2. Check the logs for specific errors related to binary extraction
3. Verify that the application has permission to write to the extraction directory

### Linux Prerequisites

While we bundle containerd, runc, and CNI plugins with the application, there are some Linux-specific dependencies that are not bundled but required for proper functionality:

1. **Linux Kernel Features**: 
   - Your Linux kernel must support namespaces, cgroups, and seccomp
   - Recommended: Linux kernel version 4.4 or newer

2. **System Utilities**:
   - `mount`/`umount` commands for filesystem operations
   - `iptables` or `nftables` for networking functionality
   - `fuse-overlayfs` for rootless mode (if used)

3. **Security Modules** (distribution-dependent):
   - AppArmor profiles (for Ubuntu/Debian-based distributions)
   - SELinux policies (for RHEL/CentOS/Fedora-based distributions)

4. **Additional Libraries**:
   - `libseccomp` version 2.3.0 or newer
   - Standard C libraries and dependencies

Most modern Linux distributions include these components by default. If you experience issues, you may need to install missing packages using your distribution's package manager:

**For Debian/Ubuntu**:
```bash
sudo apt-get update && sudo apt-get install -y \
  iptables \
  mount \
  seccomp \
  libseccomp-dev
```

**For RHEL/CentOS/Fedora**:
```bash
sudo dnf install -y \
  iptables \
  util-linux \
  libseccomp \
  libseccomp-devel
```

### Platform-specific issues

- **Windows**: Ensure the Windows binaries are properly signed if required by your users' security policies
- **macOS**: Be aware of Gatekeeper restrictions that might require additional handling for binaries
- **Linux**: Consider distribution-specific requirements for container runtime functionality

### GoReleaser issues

- Check the GoReleaser logs for any errors during the build process
- Verify that the download script is correctly executed as part of the before hooks
- Ensure that the file mappings in `.goreleaser.yml` point to the correct locations

## Using with Custom containerd Configurations

If you need to use a custom containerd configuration:

1. Create a custom `config.toml` file
2. Use the Server.Start() method with the custom configuration path:

```go
config := container.ServerConfig{
    Config: "/path/to/your/custom/config.toml",
    // Other settings...
}
server := container.NewServer(config)
if err := server.Start(context.Background()); err != nil {
    log.Fatalf("Failed to start containerd: %v", err)
}
```

## Legal Considerations

Ensure you comply with the licenses of all bundled components:
- containerd: Apache 2.0
- runc: Apache 2.0
- CNI plugins: Apache 2.0 