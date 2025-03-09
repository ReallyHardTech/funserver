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
  image: linuxkit/kernel:6.6.71
  cmdline: "console=tty0 console=ttyS0 console=ttyAMA0"

init:
  - linuxkit/init:8eea386739975a43af558eec757a7dcb3a3d2e7b
  - linuxkit/runc:667e7ea2c426a2460ca21e3da065a57dbb3369c9
  - linuxkit/containerd:a988a1a8bcbacc2c0390ca0c08f949e2b4b5915d
  - linuxkit/ca-certificates:7b32a26ca9c275d3ef32b11fe2a83dbd2aee2fdb

onboot:
  - name: sysctl
    image: linuxkit/sysctl:5f56434b81004b50b47ed629b222619168c2bcdf
  - name: dhcpcd
    image: linuxkit/dhcpcd:157df9ef45a035f1542ec2270e374f18efef98a5
    command: ["/sbin/dhcpcd", "--nobackground", "-f", "/dhcpcd.conf", "-1"]

onshutdown:
  - name: shutdown
    image: busybox:latest
    command: ["/bin/echo", "peace out"]

services:
  - name: getty
    image: linuxkit/getty:05eca453695984a69617f1f1f0bcdae7f7032967
    env:
      - INSECURE=true
  - name: rngd
    image: linuxkit/rngd:1a18f2149e42a0a1cb9e7d37608a494342c26032
  - name: nginx
    image: nginx:1.19.5-alpine
    capabilities:
      - CAP_NET_BIND_SERVICE
      - CAP_CHOWN
      - CAP_SETUID
      - CAP_SETGID
      - CAP_DAC_OVERRIDE
    binds:
      - /etc/resolv.conf:/etc/resolv.conf

files:
  - path: etc/linuxkit-config
    metadata: yaml`

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
