package container

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// BundledBinaryDir is the directory where bundled binaries are stored/extracted
var BundledBinaryDir string

func init() {
	// Initialize the directory for bundled binaries
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to temporary directory if user config dir is not available
		userConfigDir = os.TempDir()
	}
	BundledBinaryDir = filepath.Join(userConfigDir, "funserver", "bin")
}

// GetContainerdPath returns the path to the containerd binary
// It first checks if there's a bundled version, then falls back to PATH lookup
func GetContainerdPath() string {
	// First check if we have a bundled binary
	bundledPath := GetBundledContainerdPath()
	if _, err := os.Stat(bundledPath); err == nil {
		return bundledPath
	}

	// Fall back to PATH lookup
	path, err := exec.LookPath("containerd")
	if err == nil {
		return path
	}

	return ""
}

// GetRuncPath returns the path to the runc binary
// It first checks if there's a bundled version, then falls back to PATH lookup
func GetRuncPath() string {
	// First check if we have a bundled version
	bundledPath := GetBundledRuncPath()
	if _, err := os.Stat(bundledPath); err == nil {
		return bundledPath
	}

	// Fall back to PATH lookup
	path, err := exec.LookPath("runc")
	if err == nil {
		return path
	}

	return ""
}

// GetCNIPath returns the path to the CNI plugins directory
// It first checks if there's a bundled version, then falls back to standard locations
func GetCNIPath() string {
	// First check if we have bundled binaries
	bundledPath := GetBundledCNIPath()
	if _, err := os.Stat(bundledPath); err == nil {
		// Check if the directory contains at least one plugin
		entries, err := os.ReadDir(bundledPath)
		if err == nil && len(entries) > 0 {
			return bundledPath
		}
	}

	// Fall back to standard locations
	standardPaths := []string{
		"/opt/cni/bin",           // Linux standard
		"/usr/local/opt/cni/bin", // macOS Homebrew
		"C:\\Program Files\\cni", // Windows
	}

	for _, path := range standardPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// GetBundledContainerdPath returns the path where the bundled containerd binary should be
func GetBundledContainerdPath() string {
	exeName := "containerd"
	if runtime.GOOS == "windows" {
		exeName = "containerd.exe"
	}
	return filepath.Join(BundledBinaryDir, exeName)
}

// GetBundledRuncPath returns the path where the bundled runc binary should be
func GetBundledRuncPath() string {
	exeName := "runc"
	if runtime.GOOS == "windows" {
		exeName = "runc.exe"
	}
	return filepath.Join(BundledBinaryDir, exeName)
}

// GetBundledCNIPath returns the path where the bundled CNI plugins should be
func GetBundledCNIPath() string {
	return filepath.Join(BundledBinaryDir, "cni")
}

// IsContainerdInstalled checks if containerd is available (either bundled or installed)
func IsContainerdInstalled() bool {
	return GetContainerdPath() != ""
}

// IsRuncInstalled checks if runc is available (either bundled or installed)
func IsRuncInstalled() bool {
	return GetRuncPath() != ""
}

// HasCNIPlugins checks if CNI plugins are available (either bundled or installed)
func HasCNIPlugins() bool {
	return GetCNIPath() != ""
}

// EnsureBundledContainerdExtracted extracts the bundled containerd binary if needed
// This function would be called during application startup
func EnsureBundledContainerdExtracted() error {
	// Call the implementation function
	return extractBundledContainerd()
}

// EnsureBundledRuncExtracted extracts the bundled runc binary if needed
func EnsureBundledRuncExtracted() error {
	// Create the directory for bundled binaries if it doesn't exist
	if err := os.MkdirAll(BundledBinaryDir, 0755); err != nil {
		return fmt.Errorf("failed to create bundled binary directory: %w", err)
	}

	bundledPath := GetBundledRuncPath()

	// Check if the binary already exists and is executable
	if info, err := os.Stat(bundledPath); err == nil && info.Mode()&0111 != 0 {
		// Binary already exists and is executable, no need to extract
		return nil
	}

	// Get the appropriate binary path for the current platform
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	executableDir := filepath.Dir(executablePath)
	var sourcePath string

	if runtime.GOOS == "windows" {
		sourcePath = filepath.Join(executableDir, "binaries", "windows", "runc.exe")
	} else {
		sourcePath = filepath.Join(executableDir, "binaries", runtime.GOOS, "runc")
	}

	// Check if the source binary exists
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("bundled runc binary not found at %s: %w", sourcePath, err)
	}

	// Copy the source binary to the destination
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.OpenFile(bundledPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination binary file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy runc binary: %w", err)
	}

	return nil
}

// EnsureBundledCNIPluginsExtracted extracts the bundled CNI plugins if needed
func EnsureBundledCNIPluginsExtracted() error {
	// Create the directory for bundled binaries if it doesn't exist
	cniDir := GetBundledCNIPath()
	if err := os.MkdirAll(cniDir, 0755); err != nil {
		return fmt.Errorf("failed to create bundled CNI directory: %w", err)
	}

	// Check if the directory already has plugins
	entries, err := os.ReadDir(cniDir)
	if err == nil && len(entries) > 0 {
		// Already has plugins, no need to extract
		return nil
	}

	// Get the appropriate plugins directory for the current platform
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	executableDir := filepath.Dir(executablePath)
	var sourceDir string

	if runtime.GOOS == "windows" {
		sourceDir = filepath.Join(executableDir, "binaries", "windows", "cni")
	} else {
		sourceDir = filepath.Join(executableDir, "binaries", runtime.GOOS, "cni")
	}

	// Check if the source directory exists
	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("bundled CNI plugins directory not found at %s: %w", sourceDir, err)
	}

	// Copy all files from the source directory to the destination
	entries, err = os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read CNI plugins directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sourcePath := filepath.Join(sourceDir, entry.Name())
		destPath := filepath.Join(cniDir, entry.Name())

		sourceFile, err := os.Open(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
		}

		destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			sourceFile.Close()
			return fmt.Errorf("failed to create destination file %s: %w", destPath, err)
		}

		_, err = io.Copy(destFile, sourceFile)
		sourceFile.Close()
		destFile.Close()
		if err != nil {
			return fmt.Errorf("failed to copy plugin %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// GetContainerdVersion returns the installed containerd version
func GetContainerdVersion() (string, error) {
	containerdPath := GetContainerdPath()
	if containerdPath == "" {
		return "", fmt.Errorf("containerd not found")
	}

	cmd := exec.Command(containerdPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get containerd version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetDefaultSocketPath returns the default containerd socket path based on OS
func GetDefaultSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\containerd-containerd`
	}
	return "/run/containerd/containerd.sock"
}

// GetFunSocketPath returns the socket path used by the fun application
func GetFunSocketPath() string {
	homeDir, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		return `\\.\pipe\fun-containerd`
	}
	return filepath.Join(homeDir, ".fun", "containerd", "containerd.sock")
}

// CheckContainerdRunning checks if a containerd instance is running at the given socket
func CheckContainerdRunning(socketPath string) bool {
	if socketPath == "" {
		socketPath = GetDefaultSocketPath()
	}

	// For Unix sockets, we can check if the file exists
	if !strings.HasPrefix(socketPath, `\\.\pipe\`) {
		_, err := os.Stat(socketPath)
		return err == nil
	}

	// For Windows named pipes, we need a more complex check
	// We could try to connect to the socket, but that's beyond the scope here
	// Just return true and let the actual client connection attempt handle errors
	return true
}

// WaitForSocket waits for a socket file to become available or until timeout
func WaitForSocket(socketPath string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// For Windows named pipes, we handle differently
	if strings.HasPrefix(socketPath, `\\.\pipe\`) {
		// On Windows, we can only try to connect to see if it's available
		// For simplicity, we'll just wait for the timeout and let the client handle connection
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timeout waiting for containerd socket at %s", socketPath)
		}
		return nil
	}

	// For Unix sockets, we can check if the file exists
	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(socketPath); err == nil {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for containerd socket at %s", socketPath)
		}
	}
}

// LinuxKitConfig holds configuration for the LinuxKit VM on macOS
type LinuxKitConfig struct {
	// Memory allocated to the VM in MB
	Memory int
	// CPUs allocated to the VM
	CPUs int
	// Disk size in GB
	DiskSize int
	// VM name
	Name string
	// Path to the Linux kernel
	KernelPath string
	// Path to the init RAM disk
	InitrdPath string
	// LinuxKit state directory
	StateDir string
}

// DefaultLinuxKitConfig returns a default LinuxKit VM configuration for macOS
func DefaultLinuxKitConfig() LinuxKitConfig {
	homeDir, _ := os.UserHomeDir()
	linuxKitDir := filepath.Join(homeDir, ".fun", "linuxkit")

	return LinuxKitConfig{
		Memory:     1024,
		CPUs:       2,
		DiskSize:   10,
		Name:       "fun-containerd-vm",
		KernelPath: filepath.Join(linuxKitDir, "kernel"),
		InitrdPath: filepath.Join(linuxKitDir, "initrd.img"),
		StateDir:   filepath.Join(linuxKitDir, "state"),
	}
}

// IsRunningOnMacOS returns true if the code is running on macOS
func IsRunningOnMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinuxKitVMRunning checks if the LinuxKit VM is already running
func IsLinuxKitVMRunning(config LinuxKitConfig) bool {
	if !IsRunningOnMacOS() {
		return false
	}

	// Get HyperKit path
	hyperkitPath := GetHyperKitPath()
	if hyperkitPath == "" {
		return false
	}

	// Check for a PID file
	pidFile := filepath.Join(config.StateDir, "hyperkit.pid")
	if !fileExists(pidFile) {
		return false
	}

	// Read the PID
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false
	}

	// Check if the process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On unix systems, FindProcess always succeeds, so we need to send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// downloadFile downloads a file from a URL to a local path
func downloadFile(url, outputPath string) error {
	// Create the destination file
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer out.Close()

	// Get the file from the URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to GET from %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Copy the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// StartLinuxKitVM starts the LinuxKit VM on macOS
func StartLinuxKitVM(ctx context.Context, config LinuxKitConfig) error {
	if !IsRunningOnMacOS() {
		return nil
	}

	// Check if VM is already running
	if IsLinuxKitVMRunning(config) {
		return nil
	}

	// Ensure LinuxKit components are available
	if err := EnsureLinuxKitComponents(); err != nil {
		return fmt.Errorf("failed to ensure LinuxKit components: %w", err)
	}

	// Get HyperKit path
	hyperkitPath := GetHyperKitPath()
	if hyperkitPath == "" {
		return fmt.Errorf("hyperkit binary not found")
	}

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(config.StateDir, 0755); err != nil {
		return fmt.Errorf("failed to create LinuxKit state directory: %w", err)
	}

	// Prepare hyperkit command
	args := []string{
		"-m", fmt.Sprintf("%d", config.Memory),
		"-c", fmt.Sprintf("%d", config.CPUs),
		"-s", fmt.Sprintf("virtio-blk,file://%s,format=raw", filepath.Join(config.StateDir, "disk.img")),
		"-l", "com1,stdio",
		"-F", filepath.Join(config.StateDir, "hyperkit.pid"),
		"-u", // UEFI boot
		"-f", fmt.Sprintf("kexec,%s,%s,", config.KernelPath, config.InitrdPath),
		"-A", // Create disk if it doesn't exist
		config.Name,
	}

	cmd := exec.CommandContext(ctx, hyperkitPath, args...)

	// Start the VM
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start LinuxKit VM: %w", err)
	}

	// Wait for VM to start fully
	time.Sleep(5 * time.Second)

	return nil
}

// StopLinuxKitVM stops the LinuxKit VM on macOS
func StopLinuxKitVM(config LinuxKitConfig) error {
	if !IsRunningOnMacOS() {
		return nil
	}

	// Check if VM is running
	if !IsLinuxKitVMRunning(config) {
		return nil
	}

	// Get PID file path
	pidFile := filepath.Join(config.StateDir, "hyperkit.pid")
	if !fileExists(pidFile) {
		return fmt.Errorf("PID file not found for VM")
	}

	// Read the PID
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("failed to read hyperkit PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID in hyperkit PID file: %w", err)
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find hyperkit process: %w", err)
	}

	// Send interrupt signal
	if err := process.Signal(os.Interrupt); err != nil {
		// If interrupt fails, try terminate
		if err := process.Signal(syscall.SIGTERM); err != nil {
			// If terminate fails, try kill
			if err := process.Kill(); err != nil {
				return fmt.Errorf("failed to kill hyperkit process: %w", err)
			}
		}
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-time.After(10 * time.Second):
		// Process didn't exit in time, force kill
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill hyperkit process after timeout: %w", err)
		}
		return nil
	case err := <-done:
		if err != nil {
			return fmt.Errorf("error waiting for hyperkit process to exit: %w", err)
		}
		return nil
	}
}

// EnsureLinuxKitComponents ensures all required LinuxKit components are available
func EnsureLinuxKitComponents() error {
	if !IsRunningOnMacOS() {
		return nil
	}

	// Check if hyperkit is installed
	hyperKitPath := GetHyperKitPath()
	if hyperKitPath == "" {
		// Try to extract bundled hyperkit
		if err := EnsureBundledHyperKitExtracted(); err != nil {
			return fmt.Errorf("HyperKit is not available and failed to extract bundled binary: %w", err)
		}

		// Check again after extraction
		hyperKitPath = GetHyperKitPath()
		if hyperKitPath == "" {
			return fmt.Errorf("HyperKit is not available. Please ensure the bundled binary is included with the application")
		}
	}

	// Get paths for LinuxKit components
	homeDir, _ := os.UserHomeDir()
	linuxKitDir := filepath.Join(homeDir, ".fun", "linuxkit")
	kernelPath := filepath.Join(linuxKitDir, "kernel")
	initrdPath := filepath.Join(linuxKitDir, "initrd.img")

	// Create LinuxKit directory if it doesn't exist
	if err := os.MkdirAll(linuxKitDir, 0755); err != nil {
		return fmt.Errorf("failed to create LinuxKit directory: %w", err)
	}

	// Check if kernel and initrd exist
	kernelExists := fileExists(kernelPath)
	initrdExists := fileExists(initrdPath)

	// If components exist, we're done
	if kernelExists && initrdExists {
		return nil
	}

	// Extract bundled LinuxKit components
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	executableDir := filepath.Dir(executablePath)
	sourceDirPath := filepath.Join(executableDir, "binaries", "darwin", "linuxkit")

	// Copy kernel if needed
	if !kernelExists {
		sourceKernelPath := filepath.Join(sourceDirPath, "kernel")
		if fileExists(sourceKernelPath) {
			if err := copyFile(sourceKernelPath, kernelPath); err != nil {
				return fmt.Errorf("failed to copy LinuxKit kernel: %w", err)
			}
			// Make kernel executable
			if err := os.Chmod(kernelPath, 0755); err != nil {
				return fmt.Errorf("failed to make kernel executable: %w", err)
			}
		} else {
			return fmt.Errorf("bundled LinuxKit kernel not found at %s", sourceKernelPath)
		}
	}

	// Copy initrd if needed
	if !initrdExists {
		sourceInitrdPath := filepath.Join(sourceDirPath, "initrd.img")
		if fileExists(sourceInitrdPath) {
			if err := copyFile(sourceInitrdPath, initrdPath); err != nil {
				return fmt.Errorf("failed to copy LinuxKit initrd: %w", err)
			}
		} else {
			return fmt.Errorf("bundled LinuxKit initrd not found at %s", sourceInitrdPath)
		}
	}

	return nil
}

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// GetBundledHyperKitPath returns the path where the bundled HyperKit binary should be
func GetBundledHyperKitPath() string {
	return filepath.Join(BundledBinaryDir, "hyperkit")
}

// GetHyperKitPath returns the path to the HyperKit binary
// It first checks if there's a bundled version, then falls back to PATH lookup
// We prioritize the bundled version to avoid version mismatches and dependencies on external installations
func GetHyperKitPath() string {
	// First check if we have a bundled binary
	bundledPath := GetBundledHyperKitPath()
	if _, err := os.Stat(bundledPath); err == nil {
		return bundledPath
	}

	// Fall back to PATH lookup
	path, err := exec.LookPath("hyperkit")
	if err == nil {
		return path
	}

	return ""
}

// IsHyperKitInstalled checks if HyperKit is available (either bundled or on PATH)
func IsHyperKitInstalled() bool {
	return GetHyperKitPath() != ""
}

// EnsureBundledHyperKitExtracted ensures the bundled HyperKit binary is extracted
func EnsureBundledHyperKitExtracted() error {
	// Create the directory for bundled binaries if it doesn't exist
	if err := os.MkdirAll(BundledBinaryDir, 0755); err != nil {
		return fmt.Errorf("failed to create bundled binary directory: %w", err)
	}

	bundledPath := GetBundledHyperKitPath()

	// Check if the binary already exists and is executable
	if info, err := os.Stat(bundledPath); err == nil && info.Mode()&0111 != 0 {
		// Binary already exists and is executable, no need to extract
		return nil
	}

	// Get the executable path for finding bundled binaries
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	executableDir := filepath.Dir(executablePath)
	sourcePath := filepath.Join(executableDir, "binaries", "darwin", "linuxkit", "hyperkit")

	// Check if the source binary exists
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("bundled HyperKit binary not found at %s: %w", sourcePath, err)
	}

	// Open the source binary file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer sourceFile.Close()

	// Create the destination file
	destFile, err := os.OpenFile(bundledPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination binary file: %w", err)
	}
	defer destFile.Close()

	// Copy the source binary to the destination
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}

	return nil
}

// EnsureAllBundledComponentsExtracted ensures all bundled components are extracted
func EnsureAllBundledComponentsExtracted() error {
	// Extract containerd
	if err := EnsureBundledContainerdExtracted(); err != nil {
		return fmt.Errorf("failed to extract bundled containerd: %w", err)
	}

	// Extract runc
	if err := EnsureBundledRuncExtracted(); err != nil {
		return fmt.Errorf("failed to extract bundled runc: %w", err)
	}

	// Extract CNI plugins
	if err := EnsureBundledCNIPluginsExtracted(); err != nil {
		return fmt.Errorf("failed to extract bundled CNI plugins: %w", err)
	}

	// Extract HyperKit (macOS only)
	if runtime.GOOS == "darwin" {
		if err := EnsureBundledHyperKitExtracted(); err != nil {
			return fmt.Errorf("failed to extract bundled HyperKit: %w", err)
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
