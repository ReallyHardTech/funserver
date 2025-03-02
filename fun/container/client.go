package container

import (
	"context"
	"fmt"
	"log"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/pkg/errors"
)

// Client wraps the containerd client and provides container management functionality
type Client struct {
	client    *containerd.Client
	namespace string
	ctx       context.Context
}

// NewClient creates a new containerd client
func NewClient(socket, namespace string) (*Client, error) {
	// Special handling for Windows WSL2-based containers
	if IsRunningOnWindows() {
		wsl2Config := DefaultWSL2Config()

		// Check if WSL2 is available and we're using Linux containers
		if wsl2Config.Enabled {
			// Check if Windows has all prerequisites for Linux containers
			hasPrereqs, missingPrereqs := CheckWindowsLinuxContainerPrerequisites()
			if !hasPrereqs {
				// Show instructions for installing prerequisites
				ShowWindowsPrerequisitesInstructions(missingPrereqs)
				return nil, fmt.Errorf("missing prerequisites for Linux containers on Windows")
			}

			// Check if we have a custom socket from WSL
			if socketPath, err := GetContainerdClientConfig(wsl2Config); err == nil {
				// Use the socket from WSL
				socket = socketPath
			}
		}
	}

	// Create the client
	client, err := containerd.New(socket, containerd.WithDefaultNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to containerd at %s: %w", socket, err)
	}

	// Create a namespaced context
	ctx := namespaces.WithNamespace(context.Background(), namespace)

	return &Client{
		client:    client,
		namespace: namespace,
		ctx:       ctx,
	}, nil
}

// Close closes the containerd client
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Ping checks if the containerd daemon is running
func (c *Client) Ping(ctx context.Context) error {
	// Add a timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to ping containerd
	_, err := c.client.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to ping containerd")
	}

	return nil
}

// GetContainers returns a list of all containers
func (c *Client) GetContainers(ctx context.Context) ([]containerd.Container, error) {
	return c.client.Containers(ctx)
}

// GetContainer returns a specific container by ID
func (c *Client) GetContainer(ctx context.Context, id string) (containerd.Container, error) {
	return c.client.LoadContainer(ctx, id)
}

// GetRunningContainers returns a list of running containers
func (c *Client) GetRunningContainers(ctx context.Context) ([]containerd.Container, error) {
	containers, err := c.client.Containers(ctx)
	if err != nil {
		return nil, err
	}

	var running []containerd.Container
	for _, container := range containers {
		task, err := container.Task(ctx, nil)
		if err != nil {
			continue
		}

		status, err := task.Status(ctx)
		if err != nil {
			log.Printf("Warning: Failed to get status for container %s: %v", container.ID(), err)
			continue
		}

		if status.Status == containerd.Running {
			running = append(running, container)
		}
	}

	return running, nil
}

// GetContainerdClient returns the raw containerd client for advanced operations
func (c *Client) GetContainerdClient() *containerd.Client {
	return c.client
}

// GetNamespacedContext returns a context with the client's namespace
func (c *Client) GetNamespacedContext() context.Context {
	return c.ctx
}

// VerifyConnection checks if the connection to containerd is working
func (c *Client) VerifyConnection(ctx context.Context) error {
	// Add a timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to containerd: %w", err)
	}

	return nil
}
