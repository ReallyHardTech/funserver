package container

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// ServerConfig holds configuration for the containerd server
type ServerConfig struct {
	// Root directory for containerd data
	Root string
	// State directory for containerd runtime state
	State string
	// Address for containerd socket
	Address string
	// Config file path
	Config string
	// Log level
	LogLevel string
	// Log file path
	LogFile string
}

// Server represents a containerd server instance
type Server struct {
	config         ServerConfig
	cmd            *exec.Cmd
	running        bool
	mutex          sync.Mutex
	stopSignal     chan struct{}
	linuxKitConfig LinuxKitConfig
	wsl2Config     WSL2Config
	vmRunning      bool
	wslRunning     bool
}

// DefaultServerConfig returns a default server configuration
func DefaultServerConfig() ServerConfig {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".fun", "containerd")

	var defaultSocket string
	if runtime.GOOS == "windows" {
		defaultSocket = `\\.\pipe\fun-containerd`
	} else {
		defaultSocket = filepath.Join(dataDir, "containerd.sock")
	}

	return ServerConfig{
		Root:     filepath.Join(dataDir, "root"),
		State:    filepath.Join(dataDir, "state"),
		Address:  defaultSocket,
		Config:   "",
		LogLevel: "info",
		LogFile:  filepath.Join(dataDir, "containerd.log"),
	}
}

// NewServer creates a new containerd server instance
func NewServer(config ServerConfig) *Server {
	// Set defaults for empty values
	if config.Root == "" {
		homeDir, _ := os.UserHomeDir()
		config.Root = filepath.Join(homeDir, ".fun", "containerd", "root")
	}
	if config.State == "" {
		homeDir, _ := os.UserHomeDir()
		config.State = filepath.Join(homeDir, ".fun", "containerd", "state")
	}
	if config.Address == "" {
		config.Address = GetFunSocketPath()
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	if config.LogFile == "" {
		homeDir, _ := os.UserHomeDir()
		config.LogFile = filepath.Join(homeDir, ".fun", "containerd", "containerd.log")
	}

	return &Server{
		config:         config,
		running:        false,
		stopSignal:     make(chan struct{}),
		linuxKitConfig: DefaultLinuxKitConfig(),
		wsl2Config:     DefaultWSL2Config(),
		vmRunning:      false,
		wslRunning:     false,
	}
}

// Start starts the containerd server
func (s *Server) Start(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return nil
	}

	// On macOS, we need to start a LinuxKit VM to run containerd
	if IsRunningOnMacOS() {
		log.Printf("Starting LinuxKit VM for containerd on macOS")
		if err := StartLinuxKitVM(ctx, s.linuxKitConfig); err != nil {
			return errors.Wrap(err, "failed to start LinuxKit VM")
		}
		s.vmRunning = true

		// TODO: Configure containerd to connect to the LinuxKit VM
		// For now, we'll skip the normal containerd startup on macOS
		// and assume the VM has containerd running

		// Wait for VM to fully boot and containerd to start
		time.Sleep(10 * time.Second)

		// Set running to true so we consider the service started
		s.running = true
		return nil
	}

	// On Windows, we'll use WSL2 for Linux containers if available
	if IsRunningOnWindows() {
		// Check if WSL2 is available
		if s.wsl2Config.Enabled && IsWSL2Available() {
			log.Printf("Starting WSL2 environment for containerd on Windows")

			// Start the WSL2 environment
			if err := StartWSL2Environment(ctx, s.wsl2Config); err != nil {
				// WSL2 startup failed - we'll log the error but continue to try
				// native Windows containers as a fallback
				log.Printf("Failed to start WSL2 environment: %v. Falling back to native Windows containers.", err)
			} else {
				// WSL2 started successfully
				s.wslRunning = true

				// Ensure containerd is installed in WSL2
				if err := EnsureContainerdInWSL(ctx, s.wsl2Config); err != nil {
					return errors.Wrap(err, "failed to ensure containerd is installed in WSL2")
				}

				// Update the socket path to use WSL2 socket
				s.config.Address = GetWindowsContainerdSocketPath(s.wsl2Config)

				// Set running to true so we consider the service started
				s.running = true
				return nil
			}
		} else if s.wsl2Config.Enabled {
			// WSL2 is not available but was requested
			log.Printf("WSL2 is not available but was requested for Linux containers. Please install WSL2 from Microsoft Store or run 'wsl --install' in an elevated command prompt. Falling back to native Windows containers which may not support all Linux container features.")
		}

		// Fallback to native Windows containers if WSL2 is not available or failed to start
		log.Printf("Using native Windows container runtime")
	}

	// For non-macOS/non-WSL2 platforms, continue with normal containerd startup
	// Ensure all bundled components are available (containerd, runc, CNI plugins)
	// First try to extract our bundled binaries if needed
	if err := EnsureAllBundledComponentsExtracted(); err != nil {
		// If extraction fails, check if at least containerd and runc are installed on the system
		if !IsContainerdInstalled() {
			return errors.New("containerd is not available and failed to extract bundled binary")
		}

		if !IsRuncInstalled() {
			return errors.New("runc is not available and failed to extract bundled binary")
		}

		// CNI plugins are recommended but not required at this point
		if !HasCNIPlugins() {
			log.Printf("Warning: CNI plugins are not available, networking functionality may be limited")
		}
	}

	// Get the path to containerd
	containerdPath := GetContainerdPath()
	if containerdPath == "" {
		return errors.New("containerd is not available")
	}

	// Get the path to runc
	runcPath := GetRuncPath()
	if runcPath == "" {
		return errors.New("runc is not available")
	}

	// Ensure directories exist
	if err := os.MkdirAll(s.config.Root, 0755); err != nil {
		return errors.Wrap(err, "failed to create root directory")
	}
	if err := os.MkdirAll(s.config.State, 0755); err != nil {
		return errors.Wrap(err, "failed to create state directory")
	}

	// For Unix sockets, ensure the socket directory exists
	if !strings.HasPrefix(s.config.Address, `\\.\pipe\`) {
		if err := os.MkdirAll(filepath.Dir(s.config.Address), 0755); err != nil {
			return errors.Wrap(err, "failed to create socket directory")
		}
	}

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(s.config.LogFile), 0755); err != nil {
		return errors.Wrap(err, "failed to create log directory")
	}

	// Prepare containerd command
	args := []string{
		"--root", s.config.Root,
		"--state", s.config.State,
		"--address", s.config.Address,
		"--log-level", s.config.LogLevel,
	}

	// If runc path is from our bundled binaries, tell containerd about it
	if IsRuncInstalled() && strings.Contains(runcPath, BundledBinaryDir) {
		args = append(args, "--runtime-type", "io.containerd.runc.v2")
		args = append(args, "--runtime-engine", runcPath)
	}

	// If CNI plugins are available from our bundled binaries, configure their path
	cniPath := GetCNIPath()
	if cniPath != "" && strings.Contains(cniPath, BundledBinaryDir) {
		args = append(args, "--cni-bin-dir", cniPath)
		args = append(args, "--cni-conf-dir", filepath.Join(s.config.Root, "cni", "conf"))

		// Ensure CNI config directory exists
		cniConfDir := filepath.Join(s.config.Root, "cni", "conf")
		if err := os.MkdirAll(cniConfDir, 0755); err != nil {
			return errors.Wrap(err, "failed to create CNI configuration directory")
		}
	}

	if s.config.Config != "" {
		args = append(args, "--config", s.config.Config)
	}

	s.cmd = exec.CommandContext(ctx, containerdPath, args...)

	// Open log file
	logFile, err := os.OpenFile(s.config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open log file")
	}

	// Set stdout and stderr to the log file
	s.cmd.Stdout = logFile
	s.cmd.Stderr = logFile

	// Start the command
	if err := s.cmd.Start(); err != nil {
		logFile.Close()
		return errors.Wrap(err, "failed to start containerd")
	}

	// Wait for the socket to become available or timeout
	err = WaitForSocket(s.config.Address, 30*time.Second)
	if err != nil {
		// Try to kill the process
		s.cmd.Process.Kill()
		logFile.Close()
		return errors.Wrap(err, "failed waiting for containerd to start")
	}

	s.running = true
	return nil
}

// Stop stops the containerd server
func (s *Server) Stop(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// If running on macOS and VM is running, stop the VM
	if IsRunningOnMacOS() && s.vmRunning {
		if err := StopLinuxKitVM(s.linuxKitConfig); err != nil {
			return errors.Wrap(err, "failed to stop LinuxKit VM")
		}
		s.vmRunning = false
		s.running = false
		return nil
	}

	// If running on Windows with WSL2, stop the WSL2 environment
	if IsRunningOnWindows() && s.wslRunning {
		if err := StopWSL2Environment(s.wsl2Config); err != nil {
			return errors.Wrap(err, "failed to stop WSL2 environment")
		}
		s.wslRunning = false
		s.running = false
		return nil
	}

	if !s.running || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	// Send termination signal
	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		return errors.Wrap(err, "failed to send interrupt signal")
	}

	// Wait for the process to exit with a timeout
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-time.After(30 * time.Second):
		// Force kill if timeout
		if err := s.cmd.Process.Kill(); err != nil {
			return errors.Wrap(err, "failed to kill containerd process")
		}
		return errors.New("containerd did not exit gracefully, killed")
	case err := <-done:
		s.running = false
		if err != nil {
			return errors.Wrap(err, "containerd exited with error")
		}
		return nil
	case <-ctx.Done():
		// Force kill if context is cancelled
		if err := s.cmd.Process.Kill(); err != nil {
			return errors.Wrap(err, "failed to kill containerd process on context done")
		}
		return errors.New("containerd killed due to context cancellation")
	}
}

// IsRunning returns whether the containerd server is running
func (s *Server) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.running
}

// GetSocketAddress returns the socket address for the containerd server
func (s *Server) GetSocketAddress() string {
	return s.config.Address
}

// GetLogFilePath returns the path to the containerd log file
func (s *Server) GetLogFilePath() string {
	return s.config.LogFile
}
