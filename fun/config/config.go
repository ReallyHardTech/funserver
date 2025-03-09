package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Config represents the application configuration
type Config struct {
	// Cloud orchestrator settings
	CloudURL     string `json:"cloud_url"`
	APIKey       string `json:"api_key"`
	PollInterval int    `json:"poll_interval"` // In seconds

	// Logging settings
	LogLevel string `json:"log_level"`
	LogFile  string `json:"log_file"`

	// Container settings
	ContainerdSocket    string `json:"containerd_socket"`
	ContainerdNamespace string `json:"containerd_namespace"`
	ContainerRoot       string `json:"container_root"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		CloudURL:            "https://api.thefunserver.com",
		PollInterval:        60,
		LogLevel:            "info",
		LogFile:             getDefaultLogFile(),
		ContainerdSocket:    getDefaultContainerdSocket(),
		ContainerdNamespace: "funserver",
		ContainerRoot:       getDefaultContainerRoot(),
	}
}

// Load loads the configuration from the specified file
func Load(path string) (*Config, error) {
	// Default config
	config := DefaultConfig()

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// If the file doesn't exist, create it with default values
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := config.Save(path); err != nil {
			return nil, fmt.Errorf("failed to create default config file: %w", err)
		}
		return config, nil
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the JSON
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// Save saves the configuration to the specified file
func (c *Config) Save(path string) error {
	// Marshal to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigDir returns the platform-specific configuration directory
func GetConfigDir() string {
	// Get user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home directory can't be determined
		return "."
	}

	// Platform-specific config directory
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "AppData", "Local", "Fun")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Fun")
	default: // Linux and others
		// Check if XDG_CONFIG_HOME is set
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			return filepath.Join(xdgConfig, "fun")
		}
		return filepath.Join(home, ".config", "fun")
	}
}

// GetDefaultConfigPath returns the default path to the configuration file
func GetDefaultConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.json")
}

// getDefaultLogFile returns the default path to the log file
func getDefaultLogFile() string {
	return filepath.Join(GetConfigDir(), "logs", "fun.log")
}

// getDefaultContainerdSocket returns the default path to the containerd socket
func getDefaultContainerdSocket() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\containerd-containerd`
	}
	return "/run/containerd/containerd.sock"
}

// getDefaultContainerRoot returns the default path for container data
func getDefaultContainerRoot() string {
	return filepath.Join(GetConfigDir(), "containers")
}
