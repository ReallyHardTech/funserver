package container

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// LinuxKitService represents a service configuration for LinuxKit
type LinuxKitService struct {
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	Command  []string          `json:"command,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Binds    []string          `json:"binds,omitempty"`
	BindsRO  []string          `json:"binds_ro,omitempty"`
	Rootfs   bool              `json:"rootfs,omitempty"`
	Readonly bool              `json:"readonly,omitempty"`
	Net      string            `json:"net,omitempty"`
	Pid      string            `json:"pid,omitempty"`
	Ipc      string            `json:"ipc,omitempty"`
	Uts      string            `json:"uts,omitempty"`
	Userns   string            `json:"userns,omitempty"`
	Hostname string            `json:"hostname,omitempty"`
}

// LinuxKitConfig represents a LinuxKit YAML configuration
type LinuxKitYAMLConfig struct {
	Kernel struct {
		Image   string   `json:"image"`
		Cmdline string   `json:"cmdline"`
		Args    []string `json:"args,omitempty"`
	} `json:"kernel"`
	Init     []string          `json:"init,omitempty"`
	Onboot   []LinuxKitService `json:"onboot,omitempty"`
	Services []LinuxKitService `json:"services,omitempty"`
	Files    []struct {
		Path     string `json:"path"`
		Contents string `json:"contents"`
		Mode     string `json:"mode,omitempty"`
	} `json:"files,omitempty"`
}

const linuxKitConfigTemplate = `kernel:
  image: linuxkit/kernel:5.4.129
  cmdline: "console=tty0 console=ttyS0 console=ttyAMA0"

init:
  - linuxkit/init:14df799bb3b9e0eb0491da9fda7f32a108a2e2a5
  - linuxkit/runc:4ca27ce9ac9db402c138f63d7b59f3533bd4d41c
  - linuxkit/containerd:9cd43727ed66a7605f655ff95df83abaf5816f31

onboot:
  - name: sysctl
    image: linuxkit/sysctl:cf67133f5601826f5326d67d697111c880f9a57d
  - name: dhcpcd
    image: linuxkit/dhcpcd:63f26d54f8bf33821403286a40b93593b3f7e788
    command: ["/sbin/dhcpcd", "--nobackground", "-f", "/dhcpcd.conf", "-1"]

services:
  - name: rngd
    image: linuxkit/rngd:f66c0b06f7b543c9a779a8749dc477c7e1694f3a
  - name: containerd
    image: linuxkit/containerd:9cd43727ed66a7605f655ff95df83abaf5816f31
    rootfs: true
    env:
      - CONTAINERD_CONFIG=/etc/containerd/config.toml
    binds:
      - /etc/containerd/config.toml:/etc/containerd/config.toml
      - /containers:/containers
      - /run:/run
      - /var:/var
      - /var/lib/containerd:/var/lib/containerd
      - /var/run:/var/run
      - /cni:/cni
files:
  - path: /etc/containerd/config.toml
    contents: |
      [plugins.cri]
        sandbox_image = "k8s.gcr.io/pause:3.6"
      [plugins.cri.containerd.runtimes.runc]
        runtime_type = "io.containerd.runc.v2"
      [plugins.cri.cni]
        bin_dir = "/cni/bin"
        conf_dir = "/cni/conf"
      [plugins.scheduler]
        pause_threshold = 0.02
        deletion_threshold = 0
        mutation_threshold = 100
        schedule_delay = "0s"
        startup_delay = "100ms"
`

// GenerateLinuxKitConfig generates a LinuxKit YAML configuration for macOS
func GenerateLinuxKitConfig(outputPath string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("LinuxKit configuration is only needed for macOS")
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create a basic config file
	return os.WriteFile(outputPath, []byte(linuxKitConfigTemplate), 0644)
}

// BuildLinuxKitImage builds a LinuxKit image for macOS
func BuildLinuxKitImage(ctx context.Context, configPath, outputDir string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("LinuxKit image building is only needed for macOS")
	}

	// Ensure linuxkit CLI is installed
	linuxkitPath, err := exec.LookPath("linuxkit")
	if err != nil {
		// Try to install linuxkit
		if err := installLinuxKitCLI(); err != nil {
			return fmt.Errorf("LinuxKit CLI is not installed and could not be installed: %w", err)
		}

		// Try again
		linuxkitPath, err = exec.LookPath("linuxkit")
		if err != nil {
			return fmt.Errorf("linuxkit command not found after installation attempt: %w", err)
		}
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build the LinuxKit image
	cmd := exec.CommandContext(ctx, linuxkitPath, "build", "-format", "kernel+initrd", "-output", outputDir, configPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build LinuxKit image: %w. Stderr: %s", err, stderr.String())
	}

	fmt.Printf("LinuxKit image built successfully. Output in: %s\n", outputDir)
	return nil
}

// installLinuxKitCLI attempts to install the LinuxKit CLI
func installLinuxKitCLI() error {
	// Check if Homebrew is available
	brewPath, err := exec.LookPath("brew")
	if err == nil {
		// Install LinuxKit using Homebrew
		cmd := exec.Command(brewPath, "install", "linuxkit")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install LinuxKit via Homebrew: %w. Error: %s", err, stderr.String())
		}
		return nil
	}

	// If Homebrew is not available, try using go get (assuming Go is installed)
	goPath, err := exec.LookPath("go")
	if err == nil {
		cmd := exec.Command(goPath, "install", "github.com/linuxkit/linuxkit/src/cmd/linuxkit@latest")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install LinuxKit via Go: %w. Error: %s", err, stderr.String())
		}
		return nil
	}

	return fmt.Errorf("cannot install LinuxKit CLI: neither Homebrew nor Go are available")
}
