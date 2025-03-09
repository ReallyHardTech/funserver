package container

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	outputPath := flag.String("output", "", "Path to output the LinuxKit configuration")
	flag.Parse()

	// Only proceed on macOS
	if runtime.GOOS != "darwin" {
		fmt.Println("Skipping LinuxKit generation: not running on macOS")
		os.Exit(0)
	}

	if *outputPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -output flag is required")
		os.Exit(1)
	}

	// Ensure output directory exists
	dir := filepath.Dir(*outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate LinuxKit configuration
	if err := GenerateLinuxKitConfig(*outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating LinuxKit configuration: %v\n", err)
		os.Exit(1)
	}

	// Build LinuxKit image
	cmd := exec.Command("linuxkit", "build", "-format", "kernel+initrd", "-output", dir, *outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error building LinuxKit image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("LinuxKit configuration and image generated at: %s\n", dir)
}
