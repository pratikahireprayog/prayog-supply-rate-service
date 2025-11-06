package shipcube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	BaseURL   string
	TenantID  string
	HTTP      *http.Client
}

func NewClientFromEnv() *Client {
	return &Client{
		BaseURL:  getEnv("UNIFIED_CALCULATE_RATE_URL", ""),
		TenantID: getEnv("SHIPCUBE_TENANT_ID", ""),
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) CallShipCubeAPI(ctx context.Context, request *ShipCubeRateRequest) (*ShipCubeRateResponse, error) {
	endpoint := fmt.Sprintf("%s/calculate", c.BaseURL)

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Tenant-Id", c.TenantID)
	
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ShipCube API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response ShipCubeRateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}