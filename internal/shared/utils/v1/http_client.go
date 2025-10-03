package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// HTTPClientImpl implements the HTTPClient interface
type HTTPClientImpl struct {
	client  *http.Client
	timeout time.Duration
	retries int
}

// NewHTTPClient creates a new HTTP client implementation
func NewHTTPClient(timeout time.Duration, retries int) interfaces.HTTPClient {
	return &HTTPClientImpl{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
		retries: retries,
	}
}

// Get performs a GET request
func (h *HTTPClientImpl) Get(ctx context.Context, url string, headers map[string]string) (*interfaces.HTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.doRequestWithRetry(req)
}

// Post performs a POST request
func (h *HTTPClientImpl) Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*interfaces.HTTPResponse, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.doRequestWithRetry(req)
}

// Put performs a PUT request
func (h *HTTPClientImpl) Put(ctx context.Context, url string, body interface{}, headers map[string]string) (*interfaces.HTTPResponse, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create PUT request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.doRequestWithRetry(req)
}

// Delete performs a DELETE request
func (h *HTTPClientImpl) Delete(ctx context.Context, url string, headers map[string]string) (*interfaces.HTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create DELETE request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.doRequestWithRetry(req)
}

// SetTimeout sets the default timeout for requests
func (h *HTTPClientImpl) SetTimeout(timeout time.Duration) {
	h.timeout = timeout
	h.client.Timeout = timeout
}

// SetRetryCount sets the number of retry attempts
func (h *HTTPClientImpl) SetRetryCount(count int) {
	h.retries = count
}

// doRequestWithRetry performs the HTTP request with retry logic
func (h *HTTPClientImpl) doRequestWithRetry(req *http.Request) (*interfaces.HTTPResponse, error) {
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= h.retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, etc.
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		resp, err := h.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Convert headers
		headers := make(map[string]string)
		for key, values := range resp.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		httpResp := &interfaces.HTTPResponse{
			StatusCode: resp.StatusCode,
			Body:       body,
			Headers:    headers,
			Duration:   time.Since(startTime),
		}

		// Return successful responses (2xx) immediately
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return httpResp, nil
		}

		// For 4xx errors, don't retry
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return httpResp, fmt.Errorf("client error %d: %s", resp.StatusCode, string(body))
		}

		// For 5xx errors, retry
		lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", h.retries+1, lastErr)
}
