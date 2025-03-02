//go:build ignore
// +build ignore

package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Version constants
const (
	containerdVersion = "1.6.25"
	runcVersion       = "1.1.10"
	cniVersion        = "1.4.0"
	linuxKitVersion   = "0.8.0"
)

// Binary download URLs for containerd
var containerdURLs = map[string]string{
	"linux_amd64":   fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/containerd-%s-linux-amd64.tar.gz", containerdVersion, containerdVersion),
	"linux_386":     fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/containerd-%s-linux-386.tar.gz", containerdVersion, containerdVersion),
	"linux_arm64":   fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/containerd-%s-linux-arm64.tar.gz", containerdVersion, containerdVersion),
	"darwin_amd64":  fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/containerd-%s-darwin-amd64.tar.gz", containerdVersion, containerdVersion),
	"darwin_arm64":  fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/containerd-%s-darwin-arm64.tar.gz", containerdVersion, containerdVersion),
	"windows_amd64": fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/containerd-%s-windows-amd64.tar.gz", containerdVersion, containerdVersion),
	"windows_386":   fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/containerd-%s-windows-386.tar.gz", containerdVersion, containerdVersion),
	"windows_arm64": fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/containerd-%s-windows-arm64.tar.gz", containerdVersion, containerdVersion),
}

// Binary download URLs for runc
var runcURLs = map[string]string{
	"linux_amd64":   fmt.Sprintf("https://github.com/opencontainers/runc/releases/download/v%s/runc.amd64", runcVersion),
	"linux_386":     fmt.Sprintf("https://github.com/opencontainers/runc/releases/download/v%s/runc.386", runcVersion),
	"linux_arm64":   fmt.Sprintf("https://github.com/opencontainers/runc/releases/download/v%s/runc.arm64", runcVersion),
	"windows_amd64": fmt.Sprintf("https://github.com/opencontainers/runc/releases/download/v%s/runc-windows-amd64.exe", runcVersion),
	// Not available for all platforms
}

// Binary download URLs for CNI plugins
var cniURLs = map[string]string{
	"linux_amd64":   fmt.Sprintf("https://github.com/containernetworking/plugins/releases/download/v%s/cni-plugins-linux-amd64-v%s.tgz", cniVersion, cniVersion),
	"linux_386":     fmt.Sprintf("https://github.com/containernetworking/plugins/releases/download/v%s/cni-plugins-linux-386-v%s.tgz", cniVersion, cniVersion),
	"linux_arm64":   fmt.Sprintf("https://github.com/containernetworking/plugins/releases/download/v%s/cni-plugins-linux-arm64-v%s.tgz", cniVersion, cniVersion),
	"windows_amd64": fmt.Sprintf("https://github.com/containernetworking/plugins/releases/download/v%s/cni-plugins-windows-amd64-v%s.tgz", cniVersion, cniVersion),
	// Not available for all platforms
}

// Binary download URLs for LinuxKit (macOS only)
var linuxKitURLs = map[string]string{
	"linuxkit_kernel": fmt.Sprintf("https://github.com/linuxkit/linuxkit/releases/download/v%s/kernel", linuxKitVersion),
	"linuxkit_initrd": fmt.Sprintf("https://github.com/linuxkit/linuxkit/releases/download/v%s/initrd.img", linuxKitVersion),
	"hyperkit":        "https://github.com/moby/hyperkit/releases/download/v0.20220926/hyperkit-v0.20220926-x86_64.zip",
}

// Binary paths within the archive for containerd
var containerdBinaryPaths = map[string]string{
	"linux":   "bin/containerd",
	"darwin":  "bin/containerd",
	"windows": "bin/containerd.exe",
}

// Binary names for runc (not in archives, direct downloads)
var runcBinaryNames = map[string]string{
	"linux":   "runc",
	"windows": "runc.exe",
}

// Output directories for each component
var outputDirs = map[string]map[string]string{
	"containerd": {
		"linux":   "binaries/linux",
		"windows": "binaries/windows",
		"darwin":  "binaries/darwin",
	},
	"runc": {
		"linux":   "binaries/linux",
		"windows": "binaries/windows",
	},
	"cni": {
		"linux":   "binaries/linux/cni",
		"windows": "binaries/windows/cni",
	},
}

// Output directories for LinuxKit components
var linuxKitOutputDirs = map[string]string{
	"darwin": "binaries/darwin/linuxkit",
}

func main() {
	// Configure logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting download script for containerd %s, runc %s, and CNI plugins %s",
		containerdVersion, runcVersion, cniVersion)

	// Detect if running in GoReleaser environment
	if os.Getenv("GORELEASER_CURRENT_TAG") != "" {
		log.Printf("Running in GoReleaser environment with tag: %s", os.Getenv("GORELEASER_CURRENT_TAG"))
	}

	// Create all output directories
	createOutputDirectories()

	// Download containerd for all platforms
	downloadContainerd()

	// Download runc for supported platforms
	downloadRunc()

	// Download CNI plugins for supported platforms
	downloadCNIPlugins()

	// Download LinuxKit components for macOS
	downloadLinuxKitComponents()

	log.Println("All binaries downloaded successfully")
}

// createOutputDirectories creates all required output directories
func createOutputDirectories() {
	// Create directories for all components
	for _, dirs := range outputDirs {
		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Fatalf("Error creating directory %s: %v", dir, err)
			}
		}
	}

	// Create LinuxKit directories for macOS
	for _, dir := range linuxKitOutputDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Error creating LinuxKit directory %s: %v", dir, err)
		}
	}
}

// downloadContainerd downloads containerd binaries for all platforms
func downloadContainerd() {
	// Download and extract containerd binaries for each platform
	for platform, url := range containerdURLs {
		// Get the OS from the platform string (split by _)
		parts := strings.Split(platform, "_")
		if len(parts) != 2 {
			log.Printf("Skipping invalid platform string: %s", platform)
			continue
		}
		osType := parts[0] // Using osType instead of os to avoid conflict

		// Skip if output directory doesn't exist for this OS
		if _, ok := outputDirs["containerd"][osType]; !ok {
			log.Printf("Skipping download for %s: no output directory defined", platform)
			continue
		}

		outputDir := outputDirs["containerd"][osType]
		binaryPath := containerdBinaryPaths[osType]
		outputPath := filepath.Join(outputDir, filepath.Base(binaryPath))

		// Skip if the binary already exists and is not older than 1 day
		if fileInfo, err := os.Stat(outputPath); err == nil {
			// File exists, check modification time
			modTime := fileInfo.ModTime()
			if time.Since(modTime) < 24*time.Hour {
				log.Printf("Skipping download for containerd %s: binary already exists and is recent", platform)
				continue
			}
		}

		log.Printf("Downloading containerd for %s from %s", platform, url)

		// Download and extract the binary with retry logic
		if err := downloadAndExtractTarGz(url, platform, outputPath, binaryPath, 3); err != nil {
			log.Printf("Error downloading containerd for %s: %v, continuing with other platforms", platform, err)
			continue
		}
	}
}

// downloadRunc downloads runc binaries for supported platforms
func downloadRunc() {
	// Download runc binaries for each platform
	for platform, url := range runcURLs {
		// Get the OS from the platform string (split by _)
		parts := strings.Split(platform, "_")
		if len(parts) != 2 {
			log.Printf("Skipping invalid platform string: %s", platform)
			continue
		}
		osType := parts[0]

		// Skip if output directory doesn't exist for this OS
		if _, ok := outputDirs["runc"][osType]; !ok {
			log.Printf("Skipping download for %s: no output directory defined", platform)
			continue
		}

		outputDir := outputDirs["runc"][osType]
		binaryName := runcBinaryNames[osType]
		outputPath := filepath.Join(outputDir, binaryName)

		// Skip if the binary already exists and is not older than 1 day
		if fileInfo, err := os.Stat(outputPath); err == nil {
			// File exists, check modification time
			modTime := fileInfo.ModTime()
			if time.Since(modTime) < 24*time.Hour {
				log.Printf("Skipping download for runc %s: binary already exists and is recent", platform)
				continue
			}
		}

		log.Printf("Downloading runc for %s from %s", platform, url)

		// Download the binary directly (not in tar.gz)
		if err := downloadBinaryFile(url, platform, outputPath, 3); err != nil {
			log.Printf("Error downloading runc for %s: %v, continuing with other platforms", platform, err)
			continue
		}
	}
}

// downloadCNIPlugins downloads CNI plugins for supported platforms
func downloadCNIPlugins() {
	// Download and extract CNI plugins for each platform
	for platform, url := range cniURLs {
		// Get the OS from the platform string (split by _)
		parts := strings.Split(platform, "_")
		if len(parts) != 2 {
			log.Printf("Skipping invalid platform string: %s", platform)
			continue
		}
		osType := parts[0]

		// Skip if output directory doesn't exist for this OS
		if _, ok := outputDirs["cni"][osType]; !ok {
			log.Printf("Skipping download for %s: no output directory defined", platform)
			continue
		}

		outputDir := outputDirs["cni"][osType]

		// Skip if directory already has files and is not older than 1 day
		if entries, err := os.ReadDir(outputDir); err == nil && len(entries) > 0 {
			// Check the modification time of the first file
			if info, err := entries[0].Info(); err == nil {
				modTime := info.ModTime()
				if time.Since(modTime) < 24*time.Hour {
					log.Printf("Skipping download for CNI plugins %s: files already exist and are recent", platform)
					continue
				}
			}
		}

		log.Printf("Downloading CNI plugins for %s from %s", platform, url)

		// Download and extract all files from the archive to the directory
		if err := downloadAndExtractCNIPlugins(url, platform, outputDir, 3); err != nil {
			log.Printf("Error downloading CNI plugins for %s: %v, continuing with other platforms", platform, err)
			continue
		}
	}
}

// downloadLinuxKitComponents downloads LinuxKit components for macOS
func downloadLinuxKitComponents() {
	// Skip if not building for macOS
	skipDarwin := os.Getenv("SKIP_DARWIN") == "1"
	if skipDarwin {
		log.Printf("Skipping LinuxKit download due to SKIP_DARWIN=1")
		return
	}

	// Get the output directory for macOS
	outputDir, ok := linuxKitOutputDirs["darwin"]
	if !ok {
		log.Printf("Skipping LinuxKit download: no output directory defined for macOS")
		return
	}

	// Download kernel
	kernelURL := linuxKitURLs["linuxkit_kernel"]
	kernelPath := filepath.Join(outputDir, "kernel")

	// Skip if the kernel already exists and is not older than 1 day
	if fileInfo, err := os.Stat(kernelPath); err == nil {
		// File exists, check modification time
		modTime := fileInfo.ModTime()
		if time.Since(modTime) < 24*time.Hour {
			log.Printf("Skipping download for LinuxKit kernel: file already exists and is recent")
		} else {
			log.Printf("Downloading LinuxKit kernel from %s", kernelURL)
			if err := downloadBinaryFile(kernelURL, "darwin", kernelPath, 3); err != nil {
				log.Printf("Error downloading LinuxKit kernel: %v", err)
			} else {
				// Make kernel executable
				if err := os.Chmod(kernelPath, 0755); err != nil {
					log.Printf("Warning: failed to make kernel executable: %v", err)
				}
			}
		}
	} else {
		log.Printf("Downloading LinuxKit kernel from %s", kernelURL)
		if err := downloadBinaryFile(kernelURL, "darwin", kernelPath, 3); err != nil {
			log.Printf("Error downloading LinuxKit kernel: %v", err)
		} else {
			// Make kernel executable
			if err := os.Chmod(kernelPath, 0755); err != nil {
				log.Printf("Warning: failed to make kernel executable: %v", err)
			}
		}
	}

	// Download initrd
	initrdURL := linuxKitURLs["linuxkit_initrd"]
	initrdPath := filepath.Join(outputDir, "initrd.img")

	// Skip if the initrd already exists and is not older than 1 day
	if fileInfo, err := os.Stat(initrdPath); err == nil {
		// File exists, check modification time
		modTime := fileInfo.ModTime()
		if time.Since(modTime) < 24*time.Hour {
			log.Printf("Skipping download for LinuxKit initrd: file already exists and is recent")
		} else {
			log.Printf("Downloading LinuxKit initrd from %s", initrdURL)
			if err := downloadBinaryFile(initrdURL, "darwin", initrdPath, 3); err != nil {
				log.Printf("Error downloading LinuxKit initrd: %v", err)
			}
		}
	} else {
		log.Printf("Downloading LinuxKit initrd from %s", initrdURL)
		if err := downloadBinaryFile(initrdURL, "darwin", initrdPath, 3); err != nil {
			log.Printf("Error downloading LinuxKit initrd: %v", err)
		}
	}

	// Download HyperKit
	hyperkitURL := linuxKitURLs["hyperkit"]
	hyperkitZipPath := filepath.Join(outputDir, "hyperkit.zip")
	hyperkitPath := filepath.Join(outputDir, "hyperkit")

	// Check if HyperKit already exists and is recent
	if fileInfo, err := os.Stat(hyperkitPath); err == nil {
		// File exists, check modification time
		modTime := fileInfo.ModTime()
		if time.Since(modTime) < 24*time.Hour {
			log.Printf("Skipping download for HyperKit: file already exists and is recent")
		} else {
			log.Printf("Downloading HyperKit from %s", hyperkitURL)
			if err := downloadAndExtractHyperKit(hyperkitURL, hyperkitZipPath, hyperkitPath); err != nil {
				log.Printf("Error downloading/extracting HyperKit: %v", err)
			}
		}
	} else {
		log.Printf("Downloading HyperKit from %s", hyperkitURL)
		if err := downloadAndExtractHyperKit(hyperkitURL, hyperkitZipPath, hyperkitPath); err != nil {
			log.Printf("Error downloading/extracting HyperKit: %v", err)
		}
	}
}

// downloadAndExtractHyperKit downloads and extracts the HyperKit zip file
func downloadAndExtractHyperKit(url, zipPath, outputPath string) error {
	// Download the zip file
	if err := downloadFile(url, zipPath); err != nil {
		return fmt.Errorf("failed to download HyperKit: %w", err)
	}

	// Open the zip file for reading
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open HyperKit zip file: %w", err)
	}
	defer zipReader.Close()

	// Find the hyperkit binary in the zip
	var hyperkitFile *zip.File
	for _, file := range zipReader.File {
		if file.Name == "hyperkit" || file.Name == "./hyperkit" {
			hyperkitFile = file
			break
		}
	}

	if hyperkitFile == nil {
		return fmt.Errorf("hyperkit binary not found in zip file")
	}

	// Extract the hyperkit binary
	src, err := hyperkitFile.Open()
	if err != nil {
		return fmt.Errorf("failed to open hyperkit file in zip: %w", err)
	}
	defer src.Close()

	// Create the output file
	dst, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create hyperkit output file: %w", err)
	}
	defer dst.Close()

	// Copy the file
	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("failed to extract hyperkit: %w", err)
	}

	// Remove the zip file
	if err := os.Remove(zipPath); err != nil {
		log.Printf("Warning: failed to remove hyperkit zip file: %v", err)
	}

	return nil
}

// downloadWithRetry is a generic retry function for downloads
func downloadWithRetry(fn func() error, platform string, maxRetries int) error {
	var lastErr error
	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			log.Printf("Retry %d/%d for %s", retry+1, maxRetries, platform)
			time.Sleep(time.Duration(retry) * 2 * time.Second) // Backoff
		}

		if err := fn(); err != nil {
			lastErr = err
			log.Printf("Attempt %d failed: %v", retry+1, err)
			continue
		}

		return nil // Success
	}

	return fmt.Errorf("all retries failed, last error: %v", lastErr)
}

// downloadAndExtractTarGz downloads and extracts a tar.gz file
func downloadAndExtractTarGz(url, platform, outputPath, binaryPath string, maxRetries int) error {
	return downloadWithRetry(func() error {
		return extractBinaryFromTarGz(url, outputPath, binaryPath)
	}, platform, maxRetries)
}

// downloadBinaryFile downloads a binary file directly (not in an archive)
func downloadBinaryFile(url, platform, outputPath string, maxRetries int) error {
	return downloadWithRetry(func() error {
		return downloadFile(url, outputPath)
	}, platform, maxRetries)
}

// downloadAndExtractCNIPlugins downloads and extracts CNI plugins
func downloadAndExtractCNIPlugins(url, platform, outputDir string, maxRetries int) error {
	return downloadWithRetry(func() error {
		return extractAllFromTarGz(url, outputDir)
	}, platform, maxRetries)
}

// extractBinaryFromTarGz extracts a specific binary from a tar.gz file
func extractBinaryFromTarGz(url, outputPath, binaryPath string) error {
	// Create a temporary file to download to
	tmpFile, err := os.CreateTemp("", "archive-*.tar.gz")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download the file
	if err := downloadToFile(url, tmpFile); err != nil {
		return err
	}

	// Rewind the file for reading
	if _, err = tmpFile.Seek(0, 0); err != nil {
		return fmt.Errorf("error seeking in temporary file: %v", err)
	}

	// Open the tar.gz file
	gzReader, err := gzip.NewReader(tmpFile)
	if err != nil {
		return fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Extract the needed binary
	found := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %v", err)
		}

		// Check if this is the file we're looking for
		if strings.HasSuffix(header.Name, binaryPath) {
			// Make sure the directory exists
			if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
				return fmt.Errorf("error creating output directory for %s: %v", outputPath, err)
			}

			// Create the output file
			outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return fmt.Errorf("error creating output file %s: %v", outputPath, err)
			}
			defer outputFile.Close()

			// Copy the file contents
			n, err := io.Copy(outputFile, tarReader)
			if err != nil {
				return fmt.Errorf("error writing to output file: %v", err)
			}

			log.Printf("Successfully extracted %s to %s (%d bytes)", binaryPath, outputPath, n)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("could not find %s in the archive", binaryPath)
	}

	return nil
}

// extractAllFromTarGz extracts all files from a tar.gz to a directory
func extractAllFromTarGz(url, outputDir string) error {
	// Create a temporary file to download to
	tmpFile, err := os.CreateTemp("", "archive-*.tar.gz")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download the file
	if err := downloadToFile(url, tmpFile); err != nil {
		return err
	}

	// Rewind the file for reading
	if _, err = tmpFile.Seek(0, 0); err != nil {
		return fmt.Errorf("error seeking in temporary file: %v", err)
	}

	// Open the tar.gz file
	gzReader, err := gzip.NewReader(tmpFile)
	if err != nil {
		return fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Extract all files
	fileCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %v", err)
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Get the filename from the header
		filename := filepath.Base(header.Name)
		outputPath := filepath.Join(outputDir, filename)

		// Create the output file
		outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return fmt.Errorf("error creating output file %s: %v", outputPath, err)
		}

		// Copy the file contents
		n, err := io.Copy(outputFile, tarReader)
		if err != nil {
			outputFile.Close()
			return fmt.Errorf("error writing to output file: %v", err)
		}
		outputFile.Close()

		fileCount++
		log.Printf("Extracted %s to %s (%d bytes)", filename, outputPath, n)
	}

	if fileCount == 0 {
		return fmt.Errorf("no files extracted from the archive")
	}

	log.Printf("Successfully extracted %d files to %s", fileCount, outputDir)
	return nil
}

// downloadFile downloads a file directly to the outputPath
func downloadFile(url, outputPath string) error {
	// Make sure the directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("error creating output directory for %s: %v", outputPath, err)
	}

	// Create the output file
	outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("error creating output file %s: %v", outputPath, err)
	}
	defer outputFile.Close()

	// Download directly to the file
	return downloadToFile(url, outputFile)
}

// downloadToFile downloads from a URL to an already opened file
func downloadToFile(url string, file *os.File) error {
	// Download the file
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading file, status code: %d", resp.StatusCode)
	}

	// Copy the response body to the file
	n, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	log.Printf("Downloaded %d bytes", n)
	return nil
}
