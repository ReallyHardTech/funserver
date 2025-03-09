//go:build ignore
// +build ignore

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	containerdVersion = "2.0.3"
	runcVersion       = "1.2.5"
	linuxkitVersion   = "v1.5.3" // Latest stable version
	cniVersion        = "1.6.2"  // Latest stable CNI version
)

var (
	// Platform-specific binary names
	binaryExt = map[string]string{
		"darwin": "",
		"linux":  "",
	}
)

func main() {
	platforms := []string{"darwin", "linux"} // Removed windows since we'll use Linux binaries in WSL2
	arches := []string{"amd64", "arm64"}

	// Create platform-specific directories
	for _, platform := range platforms {
		for _, arch := range arches {
			// Skip unsupported combinations
			if platform == "darwin" && arch != runtime.GOARCH {
				log.Printf("Warning: Skipping cross-compilation for %s-%s\n", platform, arch)
				continue
			}
			binDir := filepath.Join("bin", platform+"-"+arch)
			if err := os.MkdirAll(binDir, 0755); err != nil {
				log.Fatalf("Fatal: Failed to create bin directory for %s-%s: %v\n", platform, arch, err)
			}
		}
	}

	// Download dependencies for each platform/arch combination
	for _, platform := range platforms {
		for _, arch := range arches {
			// Skip unsupported combinations
			if platform == "darwin" && arch != runtime.GOARCH {
				log.Printf("Warning: Skipping cross-compilation for %s-%s\n", platform, arch)
				continue
			}

			binDir := filepath.Join("bin", platform+"-"+arch)

			switch platform {
			case "linux":
				// Download both containerd and runc for Linux
				containerdURL := fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v%s/containerd-%s-%s-%s.tar.gz",
					containerdVersion, containerdVersion, platform, arch)
				containerdBin := filepath.Join(binDir, "containerd"+binaryExt[platform])

				err := downloadAndExtractContainerd(containerdURL, containerdBin)
				if err != nil {
					log.Fatalf("Fatal: Failed to download containerd for %s/%s: %v\n", platform, arch, err)
				}

				// Download runc for Linux
				runcURL := fmt.Sprintf("https://github.com/opencontainers/runc/releases/download/v%s/runc.%s",
					runcVersion, arch)
				runcBin := filepath.Join(binDir, "runc"+binaryExt[platform])
				err = downloadFile(runcURL, runcBin)
				if err != nil {
					log.Fatalf("Fatal: Failed to download runc for %s/%s: %v\n", platform, arch, err)
				}
				os.Chmod(runcBin, 0755)
				os.Chmod(containerdBin, 0755)

				// Download CNI plugins for Linux
				cniURL := fmt.Sprintf("https://github.com/containernetworking/plugins/releases/download/v%s/cni-plugins-%s-%s-v%s.tgz",
					cniVersion, platform, arch, cniVersion)
				cniDir := filepath.Join(binDir, "cni")
				err = downloadAndExtractCNI(cniURL, cniDir)
				if err != nil {
					log.Fatalf("Fatal: Failed to download CNI plugins for %s/%s: %v\n", platform, arch, err)
				}

			case "darwin":
				// Download LinuxKit for macOS
				linuxkitURL := fmt.Sprintf("https://github.com/linuxkit/linuxkit/releases/download/%s/linuxkit-%s-%s",
					linuxkitVersion, platform, arch)
				linuxkitBin := filepath.Join(binDir, "linuxkit"+binaryExt[platform])
				err := downloadFile(linuxkitURL, linuxkitBin)
				if err != nil {
					log.Fatalf("Fatal: Failed to download LinuxKit for %s/%s: %v\n", platform, arch, err)
				}
				os.Chmod(linuxkitBin, 0755)
			}
		}
	}
}

func downloadAndExtractContainerd(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.HasSuffix(header.Name, "containerd"+binaryExt[runtime.GOOS]) {
			out, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("containerd binary not found in archive")
}

func downloadFile(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func downloadAndExtractCNI(url, outputDir string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create CNI directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Extract each binary from the archive
		if header.Typeflag == tar.TypeReg {
			outPath := filepath.Join(outputDir, filepath.Base(header.Name))
			out, err := os.Create(outPath)
			if err != nil {
				return err
			}

			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()

			// Make the binary executable
			if err := os.Chmod(outPath, 0755); err != nil {
				return err
			}
		}
	}
	return nil
}
