# funserver
The runtime for self-hosting funserver compatible apps. Self host server software as easily as installing apps on your phone. This is a monorepo, where all core applications live.

WORK IN PROGRESSS -- Doesn't actually work yet, but #buildinpublic

## Installation

Funserver is a cross-platform application that can be installed on macOS, Windows, and Linux.

### macOS
1. Download the latest release from the [releases page](https://thefunserver.com/download).
2. Open the downloaded `.dmg` file and drag the Fun application to the Applications folder.
3. Launch the application and follow the on-screen instructions.

### Windows
1. Download the latest release from the [releases page](https://thefunserver.com/download).
2. Run the installer and follow the on-screen instructions.
3. The installer will automatically start the Fun service.

### Linux
```sh
curl -sSL https://thefunserver.com/install-linux-funserver | sudo bash
```

## Container Support

Funserver includes built-in container support on all platforms:

- **macOS**: Uses a lightweight LinuxKit VM for running containers. [Learn more](fun/README-MACOS-CONTAINERS.md)
- **Windows**: Supports both WSL2-based Linux containers and native Windows containers. [Learn more](fun/README-WINDOWS-CONTAINERS.md)
- **Linux**: Native container support with bundled containerd runtime.

All container components are bundled with the application - no need to install Docker separately! [Learn about our bundled containerd approach](fun/README-BUNDLED-CONTAINERD.md)

For detailed installation instructions, build options, and more information, please see the [detailed documentation](fun/README.md).
