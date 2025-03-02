package container

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// WSL2Config represents configuration for WSL2 integration
type WSL2Config struct {
	// Enable WSL2 for Linux containers
	Enabled bool
	// Distribution to use (default: "wsl-fun")
	Distribution string
	// WSL2 mount directory for sharing files
	MountDir string
	// Resources allocation
	Memory int    // Memory in MB
	CPUs   int    // Number of CPUs
	Swap   int    // Swap in MB
	DiskGB int    // Disk size in GB
	Kernel string // Optional custom kernel path
}

// DefaultWSL2Config returns default WSL2 configuration
func DefaultWSL2Config() WSL2Config {
	homeDir, _ := os.UserHomeDir()
	return WSL2Config{
		Enabled:      true,
		Distribution: "wsl-fun",
		MountDir:     filepath.Join(homeDir, ".fun", "wsl-mounts"),
		Memory:       4096, // 4GB
		CPUs:         2,
		Swap:         2048, // 2GB
		DiskGB:       10,   // 10GB
	}
}

// IsWSL2Available checks if WSL2 is available on the system
func IsWSL2Available() bool {
	// Check if wsl.exe exists and is executable
	wslPath, err := exec.LookPath("wsl.exe")
	if err != nil {
		return false
	}

	// Check if WSL2 is available by running "wsl --status"
	cmd := exec.Command(wslPath, "--status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If command fails, WSL might not be properly installed
		return false
	}

	// Check if WSL2 is mentioned in the output
	outputStr := strings.ToLower(string(output))
	return strings.Contains(outputStr, "wsl 2") || strings.Contains(outputStr, "wsl2")
}

// IsWSL2DistributionAvailable checks if a specific WSL2 distribution is installed
func IsWSL2DistributionAvailable(distribution string) bool {
	cmd := exec.Command("wsl.exe", "--list", "--quiet")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Check if the distribution is in the list
	outputStr := strings.ToLower(string(output))
	return strings.Contains(outputStr, strings.ToLower(distribution))
}

// downloadWSLRootFS downloads and prepares a rootfs for WSL2
func downloadWSLRootFS(ctx context.Context, targetPath string) error {
	// Create the directory for the rootfs
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return errors.Wrap(err, "failed to create rootfs directory")
	}

	// Download a minimal Ubuntu rootfs specifically for containers
	// We're using Ubuntu 20.04 LTS for compatibility
	ubuntuURL := "https://cloud-images.ubuntu.com/minimal/releases/focal/release/ubuntu-20.04-minimal-cloudimg-amd64-root.tar.xz"

	// Create a temporary file to download to
	tempFile, err := os.CreateTemp("", "ubuntu-rootfs-*.tar.xz")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary file for rootfs download")
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Download the rootfs
	fmt.Printf("Downloading Ubuntu rootfs for WSL2... This may take a while.\n")

	// Create an HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", ubuntuURL, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to download rootfs")
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download rootfs: %s", resp.Status)
	}

	// Copy the response body to the temporary file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to save rootfs download")
	}

	// Close the file
	tempFile.Close()

	// Extract the rootfs using tar
	fmt.Printf("Extracting rootfs...\n")

	// Create the target directory if it doesn't exist
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return errors.Wrap(err, "failed to create target directory")
	}

	// Extract the rootfs
	// On Windows we need to use a special approach since tar might not be available
	// We'll use PowerShell's Expand-Archive cmdlet
	extractCmd := exec.CommandContext(ctx, "powershell.exe", "-Command",
		fmt.Sprintf("Expand-Archive -Path \"%s\" -DestinationPath \"%s\"", tempFile.Name(), targetPath))
	output, err := extractCmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to extract rootfs: %s", string(output))
	}

	fmt.Printf("Rootfs prepared for WSL2.\n")
	return nil
}

// Actually implement the WSL2 distribution installation
func InstallWSL2Components(ctx context.Context, config WSL2Config) error {
	// Check if WSL2 is available
	if !IsWSL2Available() {
		return errors.New("WSL2 is not installed. Please install WSL2 from Microsoft Store or run 'wsl --install' in an elevated command prompt")
	}

	// Check if our distribution already exists
	if IsWSL2DistributionAvailable(config.Distribution) {
		return nil // Already installed
	}

	// Create a new WSL2 distribution
	// First, ensure the directory exists for our distribution
	homeDir, _ := os.UserHomeDir()
	wslDir := filepath.Join(homeDir, ".fun", "wsl")
	if err := os.MkdirAll(wslDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create WSL directory")
	}

	// Path for the rootfs
	rootfsPath := filepath.Join(wslDir, "rootfs")

	// Download the rootfs
	if err := downloadWSLRootFS(ctx, rootfsPath); err != nil {
		return errors.Wrap(err, "failed to download rootfs for WSL")
	}

	// Import the distribution
	fmt.Printf("Creating WSL2 distribution '%s'...\n", config.Distribution)
	cmd := exec.CommandContext(ctx, "wsl.exe", "--import", config.Distribution, wslDir, rootfsPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to import WSL distribution: %s", string(output))
	}

	// Configure the distribution
	if err := configureWSL2Distribution(config); err != nil {
		return errors.Wrap(err, "failed to configure WSL distribution")
	}

	// Now ensure containerd is installed in the new distribution
	fmt.Println("Installing containerd in WSL distribution...")
	if err := EnsureContainerdInWSL(ctx, config); err != nil {
		return errors.Wrap(err, "failed to install containerd in WSL")
	}

	fmt.Printf("WSL2 distribution '%s' successfully created and configured.\n", config.Distribution)
	return nil
}

// configureWSL2Distribution configures the WSL2 distribution with our settings
func configureWSL2Distribution(config WSL2Config) error {
	// Set resource limits using .wslconfig file
	homeDir, _ := os.UserHomeDir()
	wslConfigPath := filepath.Join(homeDir, ".wslconfig")

	// Create or append to .wslconfig
	configContent := fmt.Sprintf(`
[wsl2]
memory=%dMB
processors=%d
swap=%dMB
`, config.Memory, config.CPUs, config.Swap)

	// If kernel is specified, add it
	if config.Kernel != "" {
		configContent += fmt.Sprintf("kernel=%s\n", config.Kernel)
	}

	// Write the config file
	if err := os.WriteFile(wslConfigPath, []byte(configContent), 0644); err != nil {
		return errors.Wrap(err, "failed to write WSL config file")
	}

	// Create the mount directory
	if err := os.MkdirAll(config.MountDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create WSL mount directory")
	}

	return nil
}

// StartWSL2Environment starts the WSL2 environment for containers
func StartWSL2Environment(ctx context.Context, config WSL2Config) error {
	// Check if WSL2 is available
	if !IsWSL2Available() {
		return errors.New("WSL2 is not installed. Please install WSL2 from Microsoft Store or run 'wsl --install' in an elevated command prompt")
	}

	// Check if our distribution exists, install if needed
	if !IsWSL2DistributionAvailable(config.Distribution) {
		if err := InstallWSL2Components(ctx, config); err != nil {
			return errors.Wrap(err, "failed to install WSL components")
		}
	}

	// Start the WSL2 distribution
	cmd := exec.CommandContext(ctx, "wsl.exe", "--distribution", config.Distribution, "--")
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start WSL distribution")
	}

	// Give it some time to start up
	time.Sleep(5 * time.Second)

	// Mount the host directory for sharing
	if err := mountWSL2Directory(config); err != nil {
		return errors.Wrap(err, "failed to mount host directory in WSL")
	}

	// Start containerd in the WSL2 environment
	// This assumes containerd is installed in the WSL2 distribution
	cmd = exec.CommandContext(ctx, "wsl.exe", "--distribution", config.Distribution,
		"--", "containerd", "--address", "/run/containerd/containerd.sock")
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start containerd in WSL")
	}

	// Wait for containerd to be ready
	time.Sleep(5 * time.Second)

	return nil
}

// StopWSL2Environment stops the WSL2 environment
func StopWSL2Environment(config WSL2Config) error {
	// Check if our distribution exists
	if !IsWSL2DistributionAvailable(config.Distribution) {
		return nil // Nothing to stop
	}

	// Terminate the WSL2 distribution
	cmd := exec.Command("wsl.exe", "--terminate", config.Distribution)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to terminate WSL distribution: %s", string(output))
	}

	return nil
}

// mountWSL2Directory mounts a host directory in WSL2
func mountWSL2Directory(config WSL2Config) error {
	// Create the mount directory in WSL if it doesn't exist
	cmd := exec.Command("wsl.exe", "--distribution", config.Distribution,
		"--", "mkdir", "-p", "/mnt/fun-host")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to create mount directory in WSL: %s", string(output))
	}

	// Mount the host directory
	cmd = exec.Command("wsl.exe", "--distribution", config.Distribution,
		"--", "mount", "--bind", config.MountDir, "/mnt/fun-host")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to mount host directory in WSL: %s", string(output))
	}

	return nil
}

// IsRunningOnWindows checks if the current OS is Windows
func IsRunningOnWindows() bool {
	return os.Getenv("OS") == "Windows_NT"
}

// GetWindowsContainerdSocketPath returns the path to the containerd socket inside WSL2
func GetWindowsContainerdSocketPath(config WSL2Config) string {
	// For Windows using WSL2, we need a special socket path
	if config.Enabled && IsWSL2Available() {
		// This is a special path that indicates to containerd client to use WSL
		return fmt.Sprintf("wsl://%s/run/containerd/containerd.sock", config.Distribution)
	}

	// Fallback to standard Windows named pipe
	return `\\.\pipe\fun-containerd`
}

// EnsureContainerdInWSL ensures containerd is installed in the WSL2 distribution
func EnsureContainerdInWSL(ctx context.Context, config WSL2Config) error {
	// Check if containerd is installed in WSL
	cmd := exec.CommandContext(ctx, "wsl.exe", "--distribution", config.Distribution,
		"--", "which", "containerd")
	output, err := cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		return nil // Already installed
	}

	// Install containerd in WSL
	cmd = exec.CommandContext(ctx, "wsl.exe", "--distribution", config.Distribution,
		"--", "apt-get", "update", "&&", "apt-get", "install", "-y", "containerd")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to install containerd in WSL: %s", string(output))
	}

	// Configure containerd in WSL
	// Create a basic config file
	configDir := "/etc/containerd"
	cmd = exec.CommandContext(ctx, "wsl.exe", "--distribution", config.Distribution,
		"--", "mkdir", "-p", configDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to create containerd config directory in WSL: %s", string(output))
	}

	// Write a basic config file
	configContent := `
[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "k8s.gcr.io/pause:3.6"
`
	// We need to create a temporary file locally and then copy it to WSL
	tempFile, err := os.CreateTemp("", "containerd-config")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary config file")
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := tempFile.WriteString(configContent); err != nil {
		return errors.Wrap(err, "failed to write to temporary config file")
	}
	tempFile.Close()

	// Copy the config to WSL
	cmd = exec.CommandContext(ctx, "wsl.exe", "--distribution", config.Distribution,
		"--", "cp", tempFile.Name(), "/etc/containerd/config.toml")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to copy containerd config to WSL: %s", string(output))
	}

	return nil
}

// CreateWSLSocketProxy creates a proxy to forward containerd socket communication
// between Windows and WSL2. This is needed because Windows clients can't directly
// connect to a Unix socket inside WSL2.
func CreateWSLSocketProxy(ctx context.Context, config WSL2Config) (string, error) {
	// Create a temporary Windows named pipe
	pipeName := fmt.Sprintf(`\\.\pipe\fun-containerd-wsl-%d`, os.Getpid())

	// Start the proxy process in WSL
	// The proxy uses socat to forward the Unix socket to a Windows named pipe
	cmd := exec.CommandContext(ctx, "wsl.exe", "--distribution", config.Distribution,
		"--", "socat", "UNIX-CONNECT:/run/containerd/containerd.sock",
		fmt.Sprintf("PIPE:%s", pipeName))

	if err := cmd.Start(); err != nil {
		return "", errors.Wrap(err, "failed to start socket proxy")
	}

	// Wait a moment for the proxy to be established
	time.Sleep(1 * time.Second)

	return pipeName, nil
}

// CheckWindowsLinuxContainerPrerequisites checks if Windows has the prerequisites
// for running Linux containers
func CheckWindowsLinuxContainerPrerequisites() (bool, []string) {
	var missingPrereqs []string

	// Check if WSL2 is available
	if !IsWSL2Available() {
		missingPrereqs = append(missingPrereqs, "WSL2 is not installed. Install from Microsoft Store or run 'wsl --install' as administrator.")
	}

	// Check for virtualization support
	// We need to check if Hyper-V or Windows Hypervisor Platform is enabled
	cmd := exec.Command("powershell.exe", "-Command",
		"(Get-CimInstance -ClassName Win32_ComputerSystem).HypervisorPresent")
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) != "True" {
		missingPrereqs = append(missingPrereqs, "Virtualization is not enabled. Enable Hyper-V or Windows Hypervisor Platform in Windows Features.")
	}

	// Check if Windows version is supported (Windows 10 version 2004 or higher)
	cmd = exec.Command("powershell.exe", "-Command",
		"[Environment]::OSVersion.Version")
	output, err = cmd.Output()
	if err == nil {
		version := strings.TrimSpace(string(output))
		// Parse the version - this is simplified, a real implementation would
		// need to parse the version properly
		if !strings.Contains(version, "10.") {
			missingPrereqs = append(missingPrereqs, "Windows 10 version 2004 or higher is required.")
		}
	}

	return len(missingPrereqs) == 0, missingPrereqs
}

// GetContainerdClientConfig returns the configuration for the containerd client
// when running on Windows with WSL2
func GetContainerdClientConfig(config WSL2Config) (string, error) {
	if !IsWSL2Available() {
		return "", errors.New("WSL2 is not available")
	}

	// Check if our distribution exists
	if !IsWSL2DistributionAvailable(config.Distribution) {
		return "", errors.New("WSL2 distribution is not available")
	}

	// Create a proxy for the socket
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	socketPath, err := CreateWSLSocketProxy(ctx, config)
	if err != nil {
		return "", errors.Wrap(err, "failed to create socket proxy")
	}

	return socketPath, nil
}

// ShowWindowsPrerequisitesInstructions displays instructions for installing
// prerequisites for Linux containers on Windows
func ShowWindowsPrerequisitesInstructions(prerequisites []string) {
	// First, try to run our WSL check script if it exists
	executablePath, err := os.Executable()
	if err == nil {
		executableDir := filepath.Dir(executablePath)
		wslCheckScript := filepath.Join(executableDir, "check-wsl.ps1")

		if _, err := os.Stat(wslCheckScript); err == nil {
			// Script exists, try to run it
			fmt.Println("Running WSL2 installation helper...")
			cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-NoProfile",
				"-File", wslCheckScript, executableDir)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Start(); err == nil {
				// Script started successfully, let it handle the rest
				return
			}
		}
	}

	// Fallback to built-in instructions if script not found or failed to run
	fmt.Println("\nTo run Linux containers on Windows, please install the following prerequisites:")
	fmt.Println()

	for i, prereq := range prerequisites {
		fmt.Printf("%d. %s\n", i+1, prereq)
	}

	fmt.Println()
	fmt.Println("WSL2 Installation Instructions:")
	fmt.Println("------------------------------")
	fmt.Println("1. Open PowerShell as Administrator and run: wsl --install")
	fmt.Println("2. Restart your computer to complete the installation")
	fmt.Println("3. After restart, open PowerShell and run: wsl --set-default-version 2")
	fmt.Println("4. Run this application again to set up the container environment")
	fmt.Println()
	fmt.Println("For more information, see: https://docs.microsoft.com/en-us/windows/wsl/install")
}
