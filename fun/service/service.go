package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Service represents a platform-specific service
type Service struct {
	Name        string
	DisplayName string
	Description string
	Executable  string
}

// New creates a new Service instance
func New() *Service {
	return &Service{
		Name:        "fun",
		DisplayName: "Fun Server",
		Description: "Fun Server communicates with the Fun orchestrator",
		Executable:  getExecutablePath(),
	}
}

// Start starts the service
func (s *Service) Start() error {
	switch runtime.GOOS {
	case "windows":
		return s.startWindows()
	case "darwin":
		return s.startMacOS()
	default: // Linux and others
		return s.startLinux()
	}
}

// Stop stops the service
func (s *Service) Stop() error {
	switch runtime.GOOS {
	case "windows":
		return s.stopWindows()
	case "darwin":
		return s.stopMacOS()
	default: // Linux and others
		return s.stopLinux()
	}
}

// Status returns the service status
func (s *Service) Status() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return s.statusWindows()
	case "darwin":
		return s.statusMacOS()
	default: // Linux and others
		return s.statusLinux()
	}
}

// getExecutablePath returns the absolute path to the current executable
func getExecutablePath() string {
	exe, err := os.Executable()
	if err != nil {
		// Return a sensible default if we can't determine the executable path
		if runtime.GOOS == "windows" {
			return filepath.Join(os.Getenv("PROGRAMFILES"), "Fun", "fun.exe")
		} else {
			return "/usr/local/bin/fun"
		}
	}
	// Return the absolute path to the executable
	abs, err := filepath.Abs(exe)
	if err != nil {
		return exe
	}
	return abs
}

// GetServiceFilePath returns the platform-specific service file path
func (s *Service) GetServiceFilePath() string {
	switch runtime.GOOS {
	case "windows":
		// For Windows, this is not a file path but returned for consistency
		return s.Name
	case "darwin":
		return filepath.Join("/Library", "LaunchDaemons", "com.funserver.fun.plist")
	default: // Linux and others
		return "/etc/systemd/system/fun.service"
	}
}

// Windows service implementation
func (s *Service) startWindows() error {
	cmd := exec.Command("sc", "start", s.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start Windows service: %w, output: %s", err, string(output))
	}
	return nil
}

func (s *Service) stopWindows() error {
	cmd := exec.Command("sc", "stop", s.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop Windows service: %w, output: %s", err, string(output))
	}
	return nil
}

func (s *Service) statusWindows() (string, error) {
	cmd := exec.Command("sc", "query", s.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get Windows service status: %w, output: %s", err, string(output))
	}

	// Parse the output to determine the service status
	outputStr := string(output)
	if strings.Contains(outputStr, "RUNNING") {
		return "running", nil
	} else if strings.Contains(outputStr, "STOPPED") {
		return "stopped", nil
	} else {
		return "unknown", nil
	}
}

// macOS service implementation
func (s *Service) startMacOS() error {
	cmd := exec.Command("launchctl", "load", "-w", s.GetServiceFilePath())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start macOS service: %w, output: %s", err, string(output))
	}
	return nil
}

func (s *Service) stopMacOS() error {
	cmd := exec.Command("launchctl", "unload", "-w", s.GetServiceFilePath())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop macOS service: %w, output: %s", err, string(output))
	}
	return nil
}

func (s *Service) statusMacOS() (string, error) {
	cmd := exec.Command("launchctl", "list", "com.funserver.fun")
	output, err := cmd.CombinedOutput()
	if err == nil {
		return "running", nil
	}

	// Check if the error is because the service is not running
	if strings.Contains(string(output), "Could not find service") {
		return "stopped", nil
	}

	return "", fmt.Errorf("failed to get macOS service status: %w, output: %s", err, string(output))
}

// Linux service implementation
func (s *Service) startLinux() error {
	cmd := exec.Command("systemctl", "start", s.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start systemd service: %w, output: %s", err, string(output))
	}
	return nil
}

func (s *Service) stopLinux() error {
	cmd := exec.Command("systemctl", "stop", s.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop systemd service: %w, output: %s", err, string(output))
	}
	return nil
}

func (s *Service) statusLinux() (string, error) {
	cmd := exec.Command("systemctl", "is-active", s.Name)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		if outputStr == "inactive" || outputStr == "failed" {
			return "stopped", nil
		}
		return "", fmt.Errorf("failed to get systemd service status: %w, output: %s", err, outputStr)
	}

	if outputStr == "active" {
		return "running", nil
	}

	return "unknown", nil
}
