package indiapost

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// AuthManager manages authentication tokens for India Post API
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
	am.logger.Info("Performing India Post API login", "username", am.config.Username)

	// Prepare login request
	loginReq := LoginRequest{
		Username: am.config.Username,
		Password: am.config.Password,
	}

	// Prepare headers
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Make login request
	startTime := time.Now()
	httpResponse, err := am.httpClient.Post(ctx, am.config.GetLoginURL(), loginReq, headers)
	if err != nil {
		am.logger.Error("India Post login request failed", "error", err, "duration_ms", time.Since(startTime).Milliseconds())
		return "", fmt.Errorf("login request failed: %w", err)
	}

	am.logger.Debug("India Post login response", "status_code", httpResponse.StatusCode, "duration_ms", time.Since(startTime).Milliseconds())

	// Check HTTP status
	if httpResponse.StatusCode != http.StatusOK {
		am.logger.Error("India Post login failed with non-200 status", "status_code", httpResponse.StatusCode, "response", string(httpResponse.Body))
		return "", fmt.Errorf("login failed with status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response
	var loginResp LoginResponse
	if err := json.Unmarshal(httpResponse.Body, &loginResp); err != nil {
		am.logger.Error("Failed to parse India Post login response", "error", err, "response", string(httpResponse.Body))
		return "", fmt.Errorf("failed to parse login response: %w", err)
	}

	// Validate response
	if loginResp.Data.AccessToken == "" {
		am.logger.Error("India Post login response missing access token", "response", string(httpResponse.Body))
		return "", fmt.Errorf("login response missing access token")
	}

	// Store token with expiry
	am.token = loginResp.Data.AccessToken
	
	// Set token expiry (use configured expiry or default to 15 minutes)
	expiryDuration := time.Duration(am.config.TokenExpirySec) * time.Second
	if loginResp.Data.ExpiresIn > 0 {
		expiryDuration = time.Duration(loginResp.Data.ExpiresIn) * time.Second
	}
	
	// Subtract 60 seconds as buffer to refresh before actual expiry
	am.tokenExpiresAt = time.Now().Add(expiryDuration - 60*time.Second)

	am.logger.Info("India Post login successful", 
		"token_type", loginResp.Data.TokenType,
		"expires_in_sec", expiryDuration.Seconds(),
		"expires_at", am.tokenExpiresAt.Format(time.RFC3339))

	return am.token, nil
}

// InvalidateToken invalidates the current token, forcing a refresh on next request
func (am *AuthManager) InvalidateToken() {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.logger.Info("Invalidating India Post authentication token")
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

	am.logger.Info("Forcing India Post token refresh")
	_, err := am.login(ctx)
	return err
}

