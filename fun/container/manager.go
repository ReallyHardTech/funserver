package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pkg/errors"
)

// ManagerConfig contains configuration for the container manager
type ManagerConfig struct {
	// RunAs determines whether to run as a server, client, or both
	// "server" - run as a server only
	// "client" - run as a client only
	// "both" - run as both server and client
	RunAs string

	// Server configuration
	ServerConfig ServerConfig

	// Client configuration
	ClientSocket string
	Namespace    string
}

// Manager manages containerd server and client operations
type Manager struct {
	config      ManagerConfig
	server      *Server
	client      *Client
	useEmbedded bool
}

// DefaultManagerConfig returns default manager configuration
func DefaultManagerConfig() ManagerConfig {
	homeDir, _ := os.UserHomeDir()

	// Default to client mode
	var defaultRunAs string = "client"

	// If containerd is installed, we can optionally run as server
	if IsContainerdInstalled() {
		defaultRunAs = "client"
	} else {
		// If containerd is not available, we can't run at all
		// This is just a default, the caller should check and handle this case
		defaultRunAs = "none"
	}

	var defaultSocket string
	if runtime.GOOS == "windows" {
		defaultSocket = `\\.\pipe\fun-containerd`
	} else {
		defaultSocket = filepath.Join(homeDir, ".fun", "containerd", "containerd.sock")
	}

	return ManagerConfig{
		RunAs:        defaultRunAs,
		ServerConfig: DefaultServerConfig(),
		ClientSocket: defaultSocket,
		Namespace:    "fun",
	}
}

// NewManager creates a new container manager
func NewManager(config ManagerConfig) *Manager {
	if config.RunAs == "" {
		config.RunAs = "client"
	}
	if config.Namespace == "" {
		config.Namespace = "fun"
	}

	// Set the ServerConfig address to ClientSocket if running both
	if config.RunAs == "both" && config.ServerConfig.Address == "" {
		config.ServerConfig.Address = config.ClientSocket
	}

	return &Manager{
		config:      config,
		useEmbedded: config.RunAs == "server" || config.RunAs == "both",
	}
}

// Start starts the container manager
func (m *Manager) Start(ctx context.Context) error {
	// Start the server if configured to do so
	if m.config.RunAs == "server" || m.config.RunAs == "both" {
		// Make sure containerd is installed
		if !IsContainerdInstalled() {
			return errors.New("containerd is not installed, cannot run as server")
		}

		server := NewServer(m.config.ServerConfig)
		if err := server.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start containerd server")
		}
		m.server = server

		// When running both, use the server's socket for the client
		if m.config.RunAs == "both" {
			m.config.ClientSocket = server.GetSocketAddress()
		}
	}

	// Start the client if configured to do so
	if m.config.RunAs == "client" || m.config.RunAs == "both" {
		// When using just client mode, check if containerd is available
		if m.config.RunAs == "client" && !CheckContainerdRunning(m.config.ClientSocket) {
			return fmt.Errorf("no containerd instance available at %s", m.config.ClientSocket)
		}

		client, err := NewClient(m.config.ClientSocket, m.config.Namespace)
		if err != nil {
			// If we started the server, stop it on client error
			if m.server != nil {
				m.server.Stop(ctx)
				m.server = nil
			}
			return errors.Wrap(err, "failed to create containerd client")
		}
		m.client = client
	}

	return nil
}

// Stop stops the container manager
func (m *Manager) Stop(ctx context.Context) error {
	var clientErr, serverErr error

	// Stop the client if it exists
	if m.client != nil {
		clientErr = m.client.Close()
		m.client = nil
	}

	// Stop the server if it exists
	if m.server != nil {
		serverErr = m.server.Stop(ctx)
		m.server = nil
	}

	// Return first error encountered
	if clientErr != nil {
		return errors.Wrap(clientErr, "failed to close containerd client")
	}
	if serverErr != nil {
		return errors.Wrap(serverErr, "failed to stop containerd server")
	}

	return nil
}

// GetClient returns the containerd client
func (m *Manager) GetClient() *Client {
	return m.client
}

// GetServer returns the containerd server
func (m *Manager) GetServer() *Server {
	return m.server
}

// IsUsingEmbeddedServer returns whether the manager is using an embedded server
func (m *Manager) IsUsingEmbeddedServer() bool {
	return m.useEmbedded
}

// IsServerRunning returns whether the containerd server is running
func (m *Manager) IsServerRunning() bool {
	if m.server == nil {
		return false
	}
	return m.server.IsRunning()
}

// GetServiceStatus returns the status of the containerd service
func (m *Manager) GetServiceStatus() string {
	if m.server != nil && m.server.IsRunning() {
		return "Embedded server running"
	}

	if m.client != nil {
		// Try to ping the client
		err := m.client.Ping(context.Background())
		if err == nil {
			return "Connected to external containerd"
		}
		return "Client initialized but not connected"
	}

	return "Not running"
}

// GetContainerdVersion returns the version of containerd being used
func (m *Manager) GetContainerdVersion() (string, error) {
	if m.client == nil {
		return "", errors.New("client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use the client's raw API to get version
	version, err := m.client.GetContainerdClient().Version(ctx)
	if err != nil {
		return "", err
	}

	return version.Version, nil
}

// GetContainerdLogs gets the logs from the embedded containerd server
func (m *Manager) GetContainerdLogs(writer io.Writer) error {
	if m.server == nil {
		return errors.New("embedded server not running")
	}

	logPath := m.server.GetLogFilePath()
	logFile, err := os.Open(logPath)
	if err != nil {
		return errors.Wrap(err, "failed to open containerd log file")
	}
	defer logFile.Close()

	_, err = io.Copy(writer, logFile)
	return err
}

// CreateContainer creates a new container using the client
func (m *Manager) CreateContainer(ctx context.Context, opts CreateContainerOptions) (*Container, error) {
	if m.client == nil {
		return nil, errors.New("containerd client not initialized")
	}
	return m.client.CreateContainer(ctx, opts)
}

// StartContainer starts a container using the client
func (m *Manager) StartContainer(ctx context.Context, containerID string) error {
	if m.client == nil {
		return errors.New("containerd client not initialized")
	}
	return m.client.StartContainer(ctx, containerID)
}

// StopContainer stops a container using the client
func (m *Manager) StopContainer(ctx context.Context, containerID string, timeout int) error {
	if m.client == nil {
		return errors.New("containerd client not initialized")
	}
	return m.client.StopContainer(ctx, containerID, time.Duration(timeout)*time.Second)
}

// RemoveContainer removes a container using the client
func (m *Manager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	if m.client == nil {
		return errors.New("containerd client not initialized")
	}
	return m.client.RemoveContainer(ctx, containerID, force)
}
