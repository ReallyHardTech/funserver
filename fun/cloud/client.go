package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a Fun cloud client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// RegistrationRequest represents a host registration request
type RegistrationRequest struct {
	Hostname     string   `json:"hostname"`
	IPAddress    string   `json:"ip_address"`
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
	Labels       []string `json:"labels"`
	Version      string   `json:"version"`
}

// StatusUpdateRequest represents a status update request
type StatusUpdateRequest struct {
	Hostname    string  `json:"hostname"`
	Status      string  `json:"status"`
	MemoryUsage float64 `json:"memory_usage"`
	CPUUsage    float64 `json:"cpu_usage"`
	DiskUsage   float64 `json:"disk_usage"`
}

// New creates a new cloud client
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterHost registers a host with the cloud orchestrator
func (c *Client) RegisterHost(ctx context.Context, req *RegistrationRequest) error {
	// Marshal request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/hosts/register", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to register host: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// UpdateStatus updates the host status with the cloud orchestrator
func (c *Client) UpdateStatus(ctx context.Context, req *StatusUpdateRequest) error {
	// Marshal request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal status update request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/hosts/%s/status", c.baseURL, req.Hostname)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update status: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}
