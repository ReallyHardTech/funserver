package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"fun/cloud"
	"fun/config"
	"fun/container"
	"fun/service"
)

// Version information
var (
	Version   = "0.1.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Command line flags
var (
	daemonMode  bool
	showVersion bool
	configPath  string
)

func init() {
	flag.BoolVar(&daemonMode, "daemon", false, "Run in daemon mode")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.StringVar(&configPath, "config", config.GetDefaultConfigPath(), "Path to configuration file")
	flag.Parse()
}

func main() {
	// Show version if requested
	if showVersion {
		fmt.Printf("Fun Server %s\n", Version)
		fmt.Printf("Build time: %s\n", BuildTime)
		fmt.Printf("Git commit: %s\n", GitCommit)
		return
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// If not in daemon mode, process CLI commands
	if !daemonMode {
		handleCLICommands(cfg)
		return
	}

	// Configure logging
	setupLogging(cfg.LogFile, cfg.LogLevel)

	// Run daemon mode
	runDaemon(cfg)
}

// setupLogging configures the logging system
func setupLogging(logFile, logLevel string) {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Set log output to the file
	log.SetOutput(file)
	log.SetPrefix("[Fun] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Log startup message
	log.Printf("Starting Fun Server version %s", Version)
}

// handleCLICommands processes command line arguments and executes the appropriate command
func handleCLICommands(cfg *config.Config) {
	args := flag.Args()
	if len(args) == 0 {
		showHelp()
		return
	}

	// Create service instance
	svc := service.New()

	switch args[0] {
	case "start":
		fmt.Println("Starting Fun Server...")
		if err := svc.Start(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Fun Server started successfully")
	case "stop":
		fmt.Println("Stopping Fun Server...")
		if err := svc.Stop(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Fun Server stopped successfully")
	case "status":
		fmt.Println("Checking Fun Server status...")
		status, err := svc.Status()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Fun Server is %s\n", status)
	case "container":
		if len(args) < 2 {
			fmt.Println("Missing container subcommand")
			showContainerHelp()
			os.Exit(1)
		}
		handleContainerCommands(cfg, args[1:])
	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		showHelp()
	}
}

// handleContainerCommands handles container-related commands
func handleContainerCommands(cfg *config.Config, args []string) {
	if len(args) == 0 {
		showContainerHelp()
		return
	}

	// Create container client
	client, err := container.NewClient(cfg.ContainerdSocket, cfg.ContainerdNamespace)
	if err != nil {
		fmt.Printf("Error: Failed to connect to containerd: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Verify connection to containerd
	if err := client.VerifyConnection(context.Background()); err != nil {
		fmt.Printf("Error: Failed to connect to containerd: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	switch args[0] {
	case "list":
		fmt.Println("Listing containers...")
		containers, err := client.GetContainers(ctx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("ID\t\t\tIMAGE\t\t\tSTATUS")
		for _, c := range containers {
			task, err := c.Task(ctx, nil)
			status := "created"
			if err == nil {
				s, _ := task.Status(ctx)
				status = string(s.Status)
			}

			image := "unknown"
			i, err := c.Image(ctx)
			if err == nil {
				image = i.Name()
			}

			fmt.Printf("%s\t%s\t%s\n", c.ID(), image, status)
		}

	case "create":
		if len(args) < 3 {
			fmt.Println("Usage: fun container create <name> <image> [command]")
			os.Exit(1)
		}

		name := args[1]
		image := args[2]
		var command []string
		if len(args) > 3 {
			command = args[3:]
		}

		fmt.Printf("Creating container '%s' from image '%s'...\n", name, image)

		c, err := client.CreateContainer(ctx, container.CreateContainerOptions{
			Name:    name,
			Image:   image,
			Command: command,
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Container created with ID: %s\n", c.ID)

	case "start":
		if len(args) != 2 {
			fmt.Println("Usage: fun container start <id>")
			os.Exit(1)
		}

		id := args[1]
		fmt.Printf("Starting container %s...\n", id)

		if err := client.StartContainer(ctx, id); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Container started successfully")

	case "stop":
		if len(args) != 2 {
			fmt.Println("Usage: fun container stop <id>")
			os.Exit(1)
		}

		id := args[1]
		fmt.Printf("Stopping container %s...\n", id)

		if err := client.StopContainer(ctx, id, 10*time.Second); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Container stopped successfully")

	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: fun container remove <id> [--force]")
			os.Exit(1)
		}

		id := args[1]
		force := len(args) > 2 && args[2] == "--force"

		fmt.Printf("Removing container %s...\n", id)

		if err := client.RemoveContainer(ctx, id, force); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Container removed successfully")

	case "images":
		fmt.Println("Listing images...")
		images, err := client.ListImages(ctx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("REPOSITORY\t\tTAG\t\tDIGEST\t\tSIZE")
		for _, img := range images {
			size, _ := img.Size(ctx)
			fmt.Printf("%s\t%s\t%s\t%.2f MB\n", img.Name(), "latest", img.Target().Digest.String()[:12], float64(size)/(1024*1024))
		}

	default:
		fmt.Printf("Unknown container command: %s\n", args[0])
		showContainerHelp()
	}
}

// showHelp displays usage information
func showHelp() {
	fmt.Println("Usage: fun [options] <command>")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
	fmt.Println("\nCommands:")
	fmt.Println("  start        Start the Fun Server service")
	fmt.Println("  stop         Stop the Fun Server service")
	fmt.Println("  status       Check the status of Fun Server")
	fmt.Println("  container    Manage containers")
	fmt.Println("\nNote: Service installation and removal is handled by platform-specific installers.")
}

// showContainerHelp displays container command usage
func showContainerHelp() {
	fmt.Println("Usage: fun container <command>")
	fmt.Println("\nCommands:")
	fmt.Println("  list                   List all containers")
	fmt.Println("  create <n> <image>  Create a new container")
	fmt.Println("  start <id>             Start a container")
	fmt.Println("  stop <id>              Stop a container")
	fmt.Println("  remove <id> [--force]  Remove a container")
	fmt.Println("  images                 List all images")
}

// runDaemon starts the background service
func runDaemon(cfg *config.Config) {
	log.Println("Starting Fun Server daemon...")

	// Create a context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v\n", sig)
		cancel()
	}()

	// Create cloud client
	cloudClient := cloud.New(cfg.CloudURL, cfg.APIKey)

	// Register host with cloud orchestrator
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Warning: Failed to get hostname: %v", err)
		hostname = "unknown-host"
	}

	err = cloudClient.RegisterHost(ctx, &cloud.RegistrationRequest{
		Hostname:     hostname,
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
		Version:      Version,
		Labels:       []string{"funserver"},
	})
	if err != nil {
		log.Printf("Warning: Failed to register host: %v", err)
	} else {
		log.Printf("Successfully registered host with cloud orchestrator")
	}

	// Initialize containerd client
	containerClient, err := container.NewClient(cfg.ContainerdSocket, cfg.ContainerdNamespace)
	if err != nil {
		log.Printf("Warning: Failed to connect to containerd: %v", err)
	} else {
		log.Printf("Successfully connected to containerd")
		defer containerClient.Close()
	}

	// Start the main service routines
	var wg sync.WaitGroup

	// Start the cloud communication service
	wg.Add(1)
	go func() {
		defer wg.Done()
		runCloudCommunication(ctx, cfg, cloudClient, hostname)
	}()

	// Start the container management service if containerd is available
	if containerClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runContainerManagement(ctx, cfg, containerClient)
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	log.Println("Fun Server daemon shutdown complete")
}

// runCloudCommunication handles communication with the Fun orchestrator in the cloud
func runCloudCommunication(ctx context.Context, cfg *config.Config, cloudClient *cloud.Client, hostname string) {
	log.Println("Starting cloud communication service...")
	ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down cloud communication service...")
			return
		case <-ticker.C:
			// Update status with cloud orchestrator
			err := cloudClient.UpdateStatus(ctx, &cloud.StatusUpdateRequest{
				Hostname: hostname,
				Status:   "running",
				// TODO: Add resource usage metrics
			})
			if err != nil {
				log.Printf("Error updating status: %v", err)
			}
		}
	}
}

// runContainerManagement manages containers based on cloud orchestration
func runContainerManagement(ctx context.Context, cfg *config.Config, containerClient *container.Client) {
	log.Println("Starting container management service...")

	// Simplified container management without compose functionality
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down container management service...")
			return
		case <-ticker.C:
			// Basic container health check
			if err := containerClient.VerifyConnection(ctx); err != nil {
				log.Printf("Connection to containerd lost: %v", err)
				continue
			}

			// Container maintenance operations could be added here
		}
	}
}
