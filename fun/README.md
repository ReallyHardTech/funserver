# Fun

Fun is the application that manages your local funserver installation, allowing you to install and manage compatible applications. It supports installation on macOS, Windows, and Linux.

## Features

- Manage your local funserver installation
- Install and manage compatible applications
- Cross-platform support (macOS, Windows, Linux)
- Automatic service management through platform-specific installers
- Built-in container support with bundled containerd runtime

## Container Support

Fun includes out-of-the-box container support for all platforms:

- **[macOS Container Support](README-MACOS-CONTAINERS.md)**: Uses a lightweight LinuxKit VM to run Linux containers seamlessly
- **[Windows Container Support](README-WINDOWS-CONTAINERS.md)**: Supports both WSL2-based Linux containers and native Windows containers
- **Linux**: Native container support with zero external dependencies
- **[Bundled containerd Runtime](README-BUNDLED-CONTAINERD.md)**: All container components are included - no need to install Docker!

## Installation

### macOS

1. Download the latest release from the [releases page](https://thefunserver.com/download).
2. Open the downloaded `.dmg` file.
3. Drag the Fun application to the Applications folder.
4. Double click on the Fun application and follow the instructions.
5. The installer will automatically register and start the Fun service.

### Windows

1. Download the latest release from the [releases page](https://thefunserver.com/download).
2. Run the installer and follow the on-screen instructions.
3. The installer will automatically register and start the Fun service.

### Linux

1. Open the terminal.
2. Run the following command to download and install Fun:

   ```sh
   curl -sSL https://thefunserver.com/install-linux-funserver | sudo bash
   ```
3. The installer will automatically register and start the Fun service.

## Building from Source

### Dependencies

#### Required Dependencies
- **Go**: Version 1.16 or higher
- **Git**: For version control and commit information
- **GoReleaser**: For building and packaging (will be installed automatically by the build scripts if missing)

#### Platform-Specific Dependencies
- **Windows**:
  - PowerShell: For running the build script
  - WiX Toolset: Optional, for creating MSI installers

- **macOS**:
  - Bash or compatible shell
  - hdiutil: Built-in, for creating DMG files

- **Linux**:
  - Bash or compatible shell
  - dpkg-deb: Optional, for creating Debian packages
  - rpmbuild: Optional, for creating RPM packages

#### Optional Dependencies
- **zip**: For creating ZIP archives on non-Windows platforms
- **tar**: For creating tarballs (included in most Unix systems)
- **gzip**: For compressing archives

### Building Using Build Scripts

The project includes platform-specific build scripts that work on Windows, macOS, and Linux.

#### Windows (PowerShell)

```powershell
# Default: Local build for testing
.\fun\build.ps1

# Build a snapshot/testing version
.\fun\build.ps1 -Snapshot

# Build a release version
.\fun\build.ps1 -Release

# Clean build artifacts
.\fun\build.ps1 -Clean
```

#### macOS and Linux (Bash)

```sh
# Default: Local build for testing
./fun/build.sh

# Build a snapshot/testing version
./fun/build.sh --snapshot

# Build a release version
./fun/build.sh --release

# Clean build artifacts
./fun/build.sh --clean
```

### Continuous Integration

This project uses GitHub Actions for continuous integration. The workflow automatically builds the project on Windows, macOS, and Linux for every push to the main branch and pull request.

For releases (when a tag is pushed), the workflow builds release versions and creates a draft GitHub release with all the artifacts.

For more details, see the GitHub Actions workflow files in the `.github/workflows/` directory of the repository.

### Build Output

All build output is stored in the `dist/` directory, organized by platform and architecture.

### Platform Support Matrix

| Feature | Windows | macOS | Linux |
|---------|---------|-------|-------|
| Local Build | ✅ | ✅ | ✅ |
| Snapshot Build | ✅ | ✅ | ✅ |
| Release Build | ✅ | ✅ | ✅ |
| Windows Packages | ✅ (.zip, .msi) | ❌ | ❌ |
| macOS Packages | ❌ | ✅ (.tar.gz, .dmg) | ❌ |
| Linux Packages | ❌ | ❌ | ✅ (.tar.gz, .deb, .rpm) |

## Installation Packages

Fun Server is distributed in various formats depending on the platform:

### Windows
- `.msi` - Windows Installer Package (recommended for most users)
- `.zip` - ZIP archive for manual installation

### macOS
- `.dmg` - macOS Disk Image (recommended for most users)
- `.tar.gz` - Tarball archive for manual installation

### Linux
- `.deb` - Debian/Ubuntu package
- `.rpm` - Red Hat/Fedora package
- `.tar.gz` - Tarball archive for manual installation

## Service Management

Fun Server runs as a system service on all supported platforms. The service is automatically installed, configured, and started by the platform-specific installers:

### Windows
- The MSI installer creates and configures a Windows service named "fun"
- Service management is handled using standard Windows service tools
- You can start/stop the service using the `fun start` and `fun stop` commands, or through Windows Service Manager

### macOS
- The installer creates and registers a LaunchDaemon
- Service is configured to start automatically on system boot
- You can start/stop the service using the `fun start` and `fun stop` commands, or through `launchctl`

### Linux
- For Debian/Ubuntu (.deb) and Red Hat/Fedora (.rpm) packages, a systemd service is configured
- Service is enabled to start automatically on system boot
- You can start/stop the service using the `fun start` and `fun stop` commands, or through `systemctl`

To check the status of the service, use the command:
```
fun status
```

## Contributing

We welcome contributions! Please see our [contributing guidelines](CONTRIBUTING.md) for more details.

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for more information.

## Contact

For any questions or issues, please open an issue on the [GitHub repository](https://github.com/jdconley/funserver/issues).
