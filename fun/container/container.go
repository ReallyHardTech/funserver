package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

// Container represents a managed container
type Container struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	ImageRef        string            `json:"image_ref"`
	Command         []string          `json:"command"`
	Args            []string          `json:"args"`
	Env             []string          `json:"env"`
	Labels          map[string]string `json:"labels"`
	Status          string            `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
	RestartPolicy   string            `json:"restart_policy"`
	PrivilegedMode  bool              `json:"privileged_mode"`
	ContainerClient *Client           `json:"-"`
}

// CreateContainerOptions contains options for creating a container
type CreateContainerOptions struct {
	ID             string
	Name           string
	Image          string
	Command        []string
	Args           []string
	Env            []string
	Labels         map[string]string
	Mounts         []specs.Mount
	RestartPolicy  string
	PrivilegedMode bool
}

// CreateContainer creates a new container
func (c *Client) CreateContainer(ctx context.Context, opts CreateContainerOptions) (*Container, error) {
	// Pull the image first
	image, err := c.PullImage(ctx, opts.Image)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pull image")
	}

	// Create a unique container ID if not provided
	if opts.ID == "" {
		opts.ID = opts.Name
	}

	// Prepare container options
	var containerOpts []oci.SpecOpts
	containerOpts = append(containerOpts, oci.WithImageConfig(image))
	containerOpts = append(containerOpts, oci.WithEnv(opts.Env))

	// Set command and args if provided
	if len(opts.Command) > 0 {
		containerOpts = append(containerOpts, oci.WithProcessArgs(append(opts.Command, opts.Args...)...))
	}

	// Add mounts if provided
	if len(opts.Mounts) > 0 {
		containerOpts = append(containerOpts, oci.WithMounts(opts.Mounts))
	}

	// Set privileged mode if requested
	if opts.PrivilegedMode {
		containerOpts = append(containerOpts, oci.WithPrivileged)
	}

	// Create the container
	container, err := c.client.NewContainer(
		ctx,
		opts.ID,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(opts.ID+"-snapshot", image),
		containerd.WithNewSpec(containerOpts...),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create container")
	}

	return &Container{
		ID:              container.ID(),
		Name:            opts.Name,
		ImageRef:        opts.Image,
		Command:         opts.Command,
		Args:            opts.Args,
		Env:             opts.Env,
		Labels:          opts.Labels,
		Status:          "created",
		CreatedAt:       time.Now(),
		RestartPolicy:   opts.RestartPolicy,
		PrivilegedMode:  opts.PrivilegedMode,
		ContainerClient: c,
	}, nil
}

// StartContainer starts a container
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	container, err := c.client.LoadContainer(ctx, containerID)
	if err != nil {
		return errors.Wrap(err, "failed to load container")
	}

	// Create an IO for the container
	logFile, err := os.Create(filepath.Join(os.TempDir(), containerID+".log"))
	if err != nil {
		return errors.Wrap(err, "failed to create log file")
	}

	// Create a task
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		logFile.Close()
		return errors.Wrap(err, "failed to create task")
	}

	// Start the task
	if err := task.Start(ctx); err != nil {
		logFile.Close()
		return errors.Wrap(err, "failed to start task")
	}

	return nil
}

// StopContainer stops a container
func (c *Client) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	container, err := c.client.LoadContainer(ctx, containerID)
	if err != nil {
		return errors.Wrap(err, "failed to load container")
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get task")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try to stop the container gracefully
	exitCh, err := task.Wait(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to wait for task")
	}

	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		return errors.Wrap(err, "failed to send SIGTERM")
	}

	// Wait for container to stop
	select {
	case <-exitCh:
		// Container stopped
		return nil
	case <-ctx.Done():
		// Force stop
		if err := task.Kill(context.Background(), syscall.SIGKILL); err != nil {
			return errors.Wrap(err, "failed to send SIGKILL")
		}
		return nil
	}
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	container, err := c.client.LoadContainer(ctx, containerID)
	if err != nil {
		return errors.Wrap(err, "failed to load container")
	}

	task, err := container.Task(ctx, nil)
	if err == nil {
		// If the container is running and force is true, stop it first
		if force {
			// Force stop the container
			if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
				return errors.Wrap(err, "failed to kill task")
			}

			// Wait for the task to exit
			_, err = task.Wait(ctx)
			if err != nil {
				return errors.Wrap(err, "failed to wait for task")
			}
		} else {
			return fmt.Errorf("container is still running, use force to remove it")
		}

		// Delete the task
		if _, err := task.Delete(ctx); err != nil {
			return errors.Wrap(err, "failed to delete task")
		}
	}

	// Delete the container
	if err := container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
		return errors.Wrap(err, "failed to delete container")
	}

	return nil
}

// GetContainerLogs gets the logs from a container
func (c *Client) GetContainerLogs(ctx context.Context, containerID string, follow bool, writer io.Writer) error {
	// Check if the logfile exists
	logPath := filepath.Join(os.TempDir(), containerID+".log")
	logFile, err := os.Open(logPath)
	if err != nil {
		return errors.Wrap(err, "failed to open log file")
	}
	defer logFile.Close()

	if follow {
		// Implement log following (similar to tail -f)
		// This is a simplified version
		_, err = io.Copy(writer, logFile)
		if err != nil {
			return errors.Wrap(err, "failed to copy logs")
		}

		// In a real implementation, you would watch for new content
		// and stream it to the writer
	} else {
		// Just copy the logs
		_, err = io.Copy(writer, logFile)
		if err != nil {
			return errors.Wrap(err, "failed to copy logs")
		}
	}

	return nil
}

// PullImage pulls an image from a registry
func (c *Client) PullImage(ctx context.Context, ref string) (containerd.Image, error) {
	image, err := c.client.Pull(ctx, ref, containerd.WithPullUnpack)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pull image")
	}
	return image, nil
}

// ListImages lists all images
func (c *Client) ListImages(ctx context.Context) ([]containerd.Image, error) {
	images, err := c.client.ImageService().List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list images")
	}

	var result []containerd.Image
	for _, img := range images {
		image, err := c.client.GetImage(ctx, img.Name)
		if err != nil {
			continue
		}
		result = append(result, image)
	}

	return result, nil
}

// RemoveImage removes an image
func (c *Client) RemoveImage(ctx context.Context, ref string) error {
	err := c.client.ImageService().Delete(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "failed to remove image")
	}
	return nil
}
