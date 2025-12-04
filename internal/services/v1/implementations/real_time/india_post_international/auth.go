package india_post_international

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// AuthManager manages authentication tokens for India Post International API
type AuthManager struct {
	config     *Config
	httpClient interfaces.HTTPClient
	logger     interfaces.Logger

	// Token management
	accessToken   string
	refreshToken  string
	tokenExpiry   time.Time
	refreshExpiry time.Time
	tokenMutex    sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *Config, httpClient interfaces.HTTPClient, logger interfaces.Logger) *AuthManager {
	return &AuthManager{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
	}
}

// Authenticate authenticates with India Post API and stores the access token
func (am *AuthManager) Authenticate(ctx context.Context) error {
	am.tokenMutex.Lock()
	defer am.tokenMutex.Unlock()

	// Check if token is still valid
	if am.accessToken != "" && time.Now().Before(am.tokenExpiry) {
		am.logger.Info("Using existing valid token", "partner", "IndiaPostInternational")
		return nil
	}

	// Login to get new access token
	loginReq := LoginRequest{
		Username: am.config.Username,
		Password: am.config.Password,
	}

	url := am.config.GetLoginURL()
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	am.logger.Info("Authenticating with India Post International", "url", url)

	httpResponse, err := am.httpClient.Post(ctx, url, loginReq, headers)
	if err != nil {
		am.logger.Error("India Post login request failed", "error", err)
		return fmt.Errorf("login request failed: %w", err)
	}

	if httpResponse.StatusCode != http.StatusOK {
		am.logger.Error("India Post login failed with non-200 status",
			"status_code", httpResponse.StatusCode,
			"response", string(httpResponse.Body))
		return &IndiaPostAPIError{
			StatusCode: httpResponse.StatusCode,
			Message:    fmt.Sprintf("India Post login failed with status %d", httpResponse.StatusCode),
			RawBody:    string(httpResponse.Body),
		}
	}

	// Parse login response - India Post returns nested structure
	var loginResp LoginResponse
	if err := json.Unmarshal(httpResponse.Body, &loginResp); err != nil {
		am.logger.Error("Failed to parse India Post login response", "error", err, "response", string(httpResponse.Body))
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	// Verify login was successful
	if !loginResp.Success {
		return fmt.Errorf("login failed: success=false in response")
	}

	// Store token and expiry
	am.accessToken = loginResp.Data.AccessToken
	am.refreshToken = loginResp.Data.RefreshToken
	am.tokenExpiry = time.Now().Add(time.Duration(loginResp.Data.ExpiresIn) * time.Second)
	if loginResp.Data.RefreshExpiresIn > 0 {
		am.refreshExpiry = time.Now().Add(time.Duration(loginResp.Data.RefreshExpiresIn) * time.Second)
	}

	am.logger.Info("Successfully authenticated with India Post International",
		"token_expiry", am.tokenExpiry,
		"refresh_expiry", am.refreshExpiry)

	return nil
}

// GetAccessToken returns the current access token, authenticating if necessary
func (am *AuthManager) GetAccessToken(ctx context.Context) (string, error) {
	am.tokenMutex.RLock()
	token := am.accessToken
	expiry := am.tokenExpiry
	am.tokenMutex.RUnlock()

	// Check if token needs refresh
	if token == "" || time.Now().After(expiry.Add(-am.config.TokenExpiryBuffer)) {
		// Need to acquire write lock to update token
		am.tokenMutex.Lock()
		// Double-check token is still invalid (another goroutine might have refreshed it)
		if am.accessToken == "" || time.Now().After(am.tokenExpiry.Add(-am.config.TokenExpiryBuffer)) {
			// Try to refresh first if we have a refresh token
			if am.refreshToken != "" && (am.refreshExpiry.IsZero() || time.Now().Before(am.refreshExpiry)) {
				// Call internal refresh method (without locking, since we already have the lock)
				refreshErr := am.refreshTokenUnsafe(ctx)
				if refreshErr == nil {
					token = am.accessToken
					am.tokenMutex.Unlock()
					return token, nil
				}
				// If refresh failed, fall through to full authentication
			}

			// Fall back to full authentication (unlock first since Authenticate will lock)
			am.tokenMutex.Unlock()
			if err := am.Authenticate(ctx); err != nil {
				return "", err
			}
			am.tokenMutex.RLock()
			token = am.accessToken
			am.tokenMutex.RUnlock()
			return token, nil
		} else {
			// Token was refreshed by another goroutine
			token = am.accessToken
			am.tokenMutex.Unlock()
			return token, nil
		}
	}

	return token, nil
}

// RefreshToken refreshes the access token using the refresh token
func (am *AuthManager) RefreshToken(ctx context.Context) error {
	am.tokenMutex.Lock()
	defer am.tokenMutex.Unlock()
	return am.refreshTokenUnsafe(ctx)
}

// refreshTokenUnsafe refreshes the access token without locking
// Caller must hold the write lock (tokenMutex.Lock())
func (am *AuthManager) refreshTokenUnsafe(ctx context.Context) error {
	if am.refreshToken == "" {
		return fmt.Errorf("no refresh token available, need to re-authenticate")
	}

	// Check if refresh token is still valid
	if !am.refreshExpiry.IsZero() && time.Now().After(am.refreshExpiry) {
		am.logger.Warn("Refresh token expired, need to re-authenticate", "partner", "IndiaPostInternational")
		// Clear tokens to force re-authentication
		am.accessToken = ""
		am.refreshToken = ""
		return fmt.Errorf("refresh token expired, need to re-authenticate")
	}

	// Build refresh token URL
	refreshURL := am.config.GetRefreshTokenURL()

	// Create request with refresh token in Authorization header
	// Note: The httpClient interface doesn't support custom request creation,
	// so we'll need to use a workaround. For now, we'll use Authenticate as fallback.
	// In a real implementation, you might need to extend the HTTPClient interface
	// or use a raw HTTP client for this specific case.

	// For now, we'll use a workaround: create a POST request manually
	// Since the HTTPClient interface doesn't support custom headers for refresh,
	// we'll fall back to full authentication
	am.logger.Info("Refreshing India Post access token", "url", refreshURL)

	// Create a simple request body (some APIs require this)
	reqBody := bytes.NewBuffer([]byte{})
	
	// We need to make a request with Bearer token in Authorization header
	// Since our HTTPClient interface doesn't support this directly,
	// we'll need to use the http package directly or extend the interface
	// For now, let's use a workaround by making a raw HTTP request
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create refresh token request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", am.refreshToken))
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Use http.DefaultClient for refresh token request
	// In production, you might want to use a shared HTTP client
	httpClient := &http.Client{
		Timeout: am.config.Timeout,
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		am.logger.Warn("Refresh token request failed, will re-authenticate", "error", err)
		// Clear tokens to force re-authentication
		am.accessToken = ""
		am.refreshToken = ""
		return fmt.Errorf("refresh token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read refresh token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// If refresh fails, clear tokens to force re-authentication
		am.accessToken = ""
		am.refreshToken = ""
		return &IndiaPostAPIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("India Post token refresh failed with status %d", resp.StatusCode),
			RawBody:    string(respBody),
		}
	}

	// Parse refresh response
	var refreshResp RefreshTokenResponse
	if err := json.Unmarshal(respBody, &refreshResp); err != nil {
		return fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	// Update access token
	am.accessToken = refreshResp.AccessToken
	if refreshResp.ExpiresIn > 0 {
		am.tokenExpiry = time.Now().Add(time.Duration(refreshResp.ExpiresIn) * time.Second)
	}

	am.logger.Info("Successfully refreshed India Post access token", "token_expiry", am.tokenExpiry)
	return nil
}

// InvalidateToken invalidates the current token, forcing a refresh on next request
func (am *AuthManager) InvalidateToken() {
	am.tokenMutex.Lock()
	defer am.tokenMutex.Unlock()

	am.logger.Info("Invalidating India Post International authentication token")
	am.accessToken = ""
	am.refreshToken = ""
	am.tokenExpiry = time.Time{}
	am.refreshExpiry = time.Time{}
}

// IsTokenValid checks if the current token is valid
func (am *AuthManager) IsTokenValid() bool {
	am.tokenMutex.RLock()
	defer am.tokenMutex.RUnlock()

	return am.accessToken != "" && time.Now().Before(am.tokenExpiry)
}

