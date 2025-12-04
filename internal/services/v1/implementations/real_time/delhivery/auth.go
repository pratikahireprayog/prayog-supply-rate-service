package delhivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// AuthManager manages authentication tokens for Delhivery API
type AuthManager struct {
	config     *Config
	httpClient interfaces.HTTPClient
	logger     interfaces.Logger

	// Token management
	token          string
	tokenExpiresAt time.Time
	mu             sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *Config, httpClient interfaces.HTTPClient, logger interfaces.Logger) *AuthManager {
	return &AuthManager{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
	}
}

// GetToken returns a valid authentication token, refreshing if necessary
func (am *AuthManager) GetToken(ctx context.Context) (string, error) {
	am.mu.RLock()
	// Check if we have a valid token
	if am.token != "" && time.Now().Before(am.tokenExpiresAt) {
		token := am.token
		am.mu.RUnlock()
		return token, nil
	}
	am.mu.RUnlock()

	// Token is expired or not available, acquire lock and refresh
	am.mu.Lock()
	defer am.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have refreshed)
	if am.token != "" && time.Now().Before(am.tokenExpiresAt) {
		return am.token, nil
	}

	// Perform login to get new token
	return am.login(ctx)
}

// login performs the login operation and stores the token
func (am *AuthManager) login(ctx context.Context) (string, error) {
	am.logger.Info("Performing Delhivery API login", "username", am.config.Username)

	// Prepare login request
	loginReq := LoginRequest{
		Username: am.config.Username,
		Password: am.config.Password,
	}

	// Prepare headers
	headers := map[string]string{
		"Content-Type": "Application/json",
	}

	// Make login request
	startTime := time.Now()
	httpResponse, err := am.httpClient.Post(ctx, am.config.GetLoginURL(), loginReq, headers)
	if err != nil {
		am.logger.Error("Delhivery login request failed", "error", err, "duration_ms", time.Since(startTime).Milliseconds())
		return "", fmt.Errorf("login request failed: %w", err)
	}

	am.logger.Debug("Delhivery login response", "status_code", httpResponse.StatusCode, "duration_ms", time.Since(startTime).Milliseconds())

	// Check HTTP status
	if httpResponse.StatusCode != http.StatusOK {
		am.logger.Error("Delhivery login failed with non-200 status", "status_code", httpResponse.StatusCode, "response", string(httpResponse.Body))
		return "", fmt.Errorf("login failed with status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response
	var loginResp LoginResponse
	if err := json.Unmarshal(httpResponse.Body, &loginResp); err != nil {
		am.logger.Error("Failed to parse Delhivery login response", "error", err, "response", string(httpResponse.Body))
		return "", fmt.Errorf("failed to parse login response: %w", err)
	}

	// Validate response
	if !loginResp.Success || loginResp.Data.JWT == "" {
		am.logger.Error("Delhivery login response missing JWT token", "success", loginResp.Success, "response", string(httpResponse.Body))
		return "", fmt.Errorf("login response missing JWT token")
	}

	// Store token with expiry
	am.token = loginResp.Data.JWT
	
	// Set token expiry (use configured expiry or default to 7 days)
	expiryDuration := time.Duration(am.config.TokenExpirySec) * time.Second
	
	// Subtract 5 minutes as buffer to refresh before actual expiry
	am.tokenExpiresAt = time.Now().Add(expiryDuration - 5*time.Minute)

	am.logger.Info("Delhivery login successful", 
		"expires_in_sec", expiryDuration.Seconds(),
		"expires_at", am.tokenExpiresAt.Format(time.RFC3339))

	return am.token, nil
}

// InvalidateToken invalidates the current token, forcing a refresh on next request
func (am *AuthManager) InvalidateToken() {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.logger.Info("Invalidating Delhivery authentication token")
	am.token = ""
	am.tokenExpiresAt = time.Time{}
}

// IsTokenValid checks if the current token is valid
func (am *AuthManager) IsTokenValid() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	return am.token != "" && time.Now().Before(am.tokenExpiresAt)
}

// GetTokenExpiresAt returns when the current token expires
func (am *AuthManager) GetTokenExpiresAt() time.Time {
	am.mu.RLock()
	defer am.mu.RUnlock()

	return am.tokenExpiresAt
}

// RefreshToken forces a token refresh
func (am *AuthManager) RefreshToken(ctx context.Context) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.logger.Info("Forcing Delhivery token refresh")
	_, err := am.login(ctx)
	return err
}

