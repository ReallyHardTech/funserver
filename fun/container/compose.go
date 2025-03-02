package container

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// ComposeService represents a service in a compose file
type ComposeService struct {
	Image          string                 `yaml:"image"`
	Command        []string               `yaml:"command,omitempty"`
	Entrypoint     []string               `yaml:"entrypoint,omitempty"`
	Environment    map[string]string      `yaml:"environment,omitempty"`
	Volumes        []string               `yaml:"volumes,omitempty"`
	Ports          []string               `yaml:"ports,omitempty"`
	DependsOn      []string               `yaml:"depends_on,omitempty"`
	Restart        string                 `yaml:"restart,omitempty"`
	Labels         map[string]string      `yaml:"labels,omitempty"`
	PrivilegedMode bool                   `yaml:"privileged,omitempty"`
	ExtraHosts     []string               `yaml:"extra_hosts,omitempty"`
	Networks       []string               `yaml:"networks,omitempty"`
	Deploy         map[string]interface{} `yaml:"deploy,omitempty"`
}

// ComposeConfig represents a docker-compose configuration
type ComposeConfig struct {
	Version  string                    `yaml:"version"`
	Services map[string]ComposeService `yaml:"services"`
	Networks map[string]interface{}    `yaml:"networks,omitempty"`
	Volumes  map[string]interface{}    `yaml:"volumes,omitempty"`
}

// Compose is a docker-compose like orchestrator
type Compose struct {
	client        *Client
	config        *ComposeConfig
	project       string
	file          string
	containerRoot string
}

// NewCompose creates a new compose instance
func NewCompose(client *Client, projectName, file, containerRoot string) (*Compose, error) {
	return &Compose{
		client:        client,
		project:       projectName,
		file:          file,
		containerRoot: containerRoot,
	}, nil
}

// LoadConfig loads a compose configuration from a file
func (c *Compose) LoadConfig() error {
	data, err := os.ReadFile(c.file)
	if err != nil {
		return errors.Wrap(err, "failed to read compose file")
	}

	config := &ComposeConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return errors.Wrap(err, "failed to parse compose file")
	}

	c.config = config
	return nil
}

// Up starts all services defined in the compose file
func (c *Compose) Up(ctx context.Context) error {
	if c.config == nil {
		if err := c.LoadConfig(); err != nil {
			return err
		}
	}

	// Create networks first (not implemented in this simplified version)

	// Start services in dependency order
	// A more robust implementation would use a DAG for dependency resolution
	for serviceName, service := range c.config.Services {
		if err := c.startService(ctx, serviceName, service); err != nil {
			return errors.Wrapf(err, "failed to start service %s", serviceName)
		}
	}

	return nil
}

// Down stops and removes all services defined in the compose file
func (c *Compose) Down(ctx context.Context) error {
	if c.config == nil {
		if err := c.LoadConfig(); err != nil {
			return err
		}
	}

	// Stop and remove services in reverse dependency order
	for serviceName := range c.config.Services {
		containerID := c.getContainerID(serviceName)

		// Try to stop the container first
		if err := c.client.StopContainer(ctx, containerID, 10*time.Second); err != nil {
			// Just log the error but continue to remove
			fmt.Printf("Warning: Failed to stop container %s: %v\n", containerID, err)
		}

		// Remove the container
		if err := c.client.RemoveContainer(ctx, containerID, true); err != nil {
			fmt.Printf("Warning: Failed to remove container %s: %v\n", containerID, err)
		}
	}

	return nil
}

// Restart restarts all services
func (c *Compose) Restart(ctx context.Context) error {
	if c.config == nil {
		if err := c.LoadConfig(); err != nil {
			return err
		}
	}

	for serviceName := range c.config.Services {
		containerID := c.getContainerID(serviceName)

		// Try to stop the container first
		if err := c.client.StopContainer(ctx, containerID, 10*time.Second); err != nil {
			fmt.Printf("Warning: Failed to stop container %s: %v\n", containerID, err)
		}

		// Start the container
		if err := c.client.StartContainer(ctx, containerID); err != nil {
			return errors.Wrapf(err, "failed to restart service %s", serviceName)
		}
	}

	return nil
}

// GetStatus returns the status of all services
func (c *Compose) GetStatus(ctx context.Context) (map[string]string, error) {
	if c.config == nil {
		if err := c.LoadConfig(); err != nil {
			return nil, err
		}
	}

	status := make(map[string]string)

	// Get all containers
	containers, err := c.client.GetContainers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get containers")
	}

	// Create a map of container ID to status
	containerStatus := make(map[string]string)
	for _, container := range containers {
		task, err := container.Task(ctx, nil)
		if err != nil {
			containerStatus[container.ID()] = "not running"
			continue
		}

		taskStatus, err := task.Status(ctx)
		if err != nil {
			containerStatus[container.ID()] = "unknown"
			continue
		}

		containerStatus[container.ID()] = string(taskStatus.Status)
	}

	// Map service names to container statuses
	for serviceName := range c.config.Services {
		containerID := c.getContainerID(serviceName)
		if containerState, ok := containerStatus[containerID]; ok {
			status[serviceName] = containerState
		} else {
			status[serviceName] = "not created"
		}
	}

	return status, nil
}

// startService starts a service and its dependencies
func (c *Compose) startService(ctx context.Context, serviceName string, service ComposeService) error {
	// Check if service has dependencies and start them first
	for _, dep := range service.DependsOn {
		if _, ok := c.config.Services[dep]; !ok {
			return fmt.Errorf("service %s depends on %s, but it is not defined", serviceName, dep)
		}

		if err := c.startService(ctx, dep, c.config.Services[dep]); err != nil {
			return err
		}
	}

	// Check if container already exists
	containerID := c.getContainerID(serviceName)
	_, err := c.client.GetContainer(ctx, containerID)
	if err == nil {
		// Container exists, just start it
		if err := c.client.StartContainer(ctx, containerID); err != nil {
			return errors.Wrapf(err, "failed to start existing container for service %s", serviceName)
		}
		return nil
	}

	// Convert environment map to slice
	var env []string
	for k, v := range service.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Convert volumes to mounts
	var mounts []specs.Mount
	for _, vol := range service.Volumes {
		parts := strings.Split(vol, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid volume format: %s", vol)
		}

		src := parts[0]
		dst := parts[1]

		// If source is not absolute, make it relative to container root
		if !filepath.IsAbs(src) {
			src = filepath.Join(c.containerRoot, "volumes", src)
			// Ensure volume directory exists
			if err := os.MkdirAll(src, 0755); err != nil {
				return errors.Wrapf(err, "failed to create volume directory %s", src)
			}
		}

		mounts = append(mounts, specs.Mount{
			Source:      src,
			Destination: dst,
			Type:        "bind",
			Options:     []string{"rbind", "rw"},
		})
	}

	// Create and start the container
	container, err := c.client.CreateContainer(ctx, CreateContainerOptions{
		ID:             containerID,
		Name:           serviceName,
		Image:          service.Image,
		Command:        service.Command,
		Env:            env,
		Labels:         service.Labels,
		Mounts:         mounts,
		RestartPolicy:  service.Restart,
		PrivilegedMode: service.PrivilegedMode,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create container for service %s", serviceName)
	}

	// Start the container
	if err := c.client.StartContainer(ctx, container.ID); err != nil {
		return errors.Wrapf(err, "failed to start container for service %s", serviceName)
	}

	return nil
}

// getContainerID generates a container ID for a service
func (c *Compose) getContainerID(serviceName string) string {
	return fmt.Sprintf("%s-%s", c.project, serviceName)
}
