package unified_rate

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// AuthService handles authentication for Prayog Unified API
type AuthService struct {
	httpClient interfaces.HTTPClient
	logger     interfaces.Logger
	config     *AuthConfig
	token      *TokenInfo
	mutex      sync.RWMutex
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	LoginURL      string `json:"login_url"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	SigninType    string `json:"signin_type"`
	TimeoutMs     int    `json:"timeout_ms"`
	RefreshBuffer int    `json:"refresh_buffer_minutes"` // Refresh token before expiry
}

// TokenInfo holds token information
type TokenInfo struct {
	IDToken      string    `json:"id_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	ObtainedAt   time.Time `json:"obtained_at"`
	UserID       string    `json:"user_id"`
	UserEmail    string    `json:"user_email"`
	TenantID     string    `json:"tenant_id"`
}

// LoginRequest represents the login API request
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	SigninType string `json:"signinType"`
}

// LoginResponse represents the login API response
type LoginResponse struct {
	Success         bool                   `json:"success"`
	IDToken         string                 `json:"id_token"`
	RefreshToken    string                 `json:"refresh_token"`
	ExpiresIn       int                    `json:"expires_in"`
	TokenType       string                 `json:"token_type"`
	PlatformRole    string                 `json:"platform_role"`
	UserID          string                 `json:"user_id"`
	UserEmail       string                 `json:"user_email"`
	TenantID        string                 `json:"tenant_id"`
	TenantRole      string                 `json:"tenant_role"`
	CustomRoles     []string               `json:"custom_roles"`
	HasTenantAccess bool                   `json:"has_tenant_access"`
	TenantSource    string                 `json:"tenant_source"`
	TenantHierarchy map[string]interface{} `json:"tenant_hierarchy"`
	RoleDescription string                 `json:"role_description"`
	// Error response fields
	Status  string `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// NewAuthService creates a new authentication service
func NewAuthService(httpClient interfaces.HTTPClient, logger interfaces.Logger) *AuthService {
	return &AuthService{
		httpClient: httpClient,
		logger:     logger,
		config: &AuthConfig{
			LoginURL:      "https://sandbox-apis.prayog.io/auth/login",
			Username:      "avinash.singh@prayog.io",
			Password:      "Prayog@Avinash539",
			SigninType:    "EMAIL",
			TimeoutMs:     15000, // 15 seconds
			RefreshBuffer: 30,    // Refresh 30 minutes before expiry
		},
	}
}

// UpdateConfig updates the authentication configuration
func (a *AuthService) UpdateConfig(config map[string]interface{}) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if loginURL, ok := config["login_url"].(string); ok {
		a.config.LoginURL = loginURL
	}
	if username, ok := config["username"].(string); ok {
		a.config.Username = username
	}
	if password, ok := config["password"].(string); ok {
		a.config.Password = password
	}
	if signinType, ok := config["signin_type"].(string); ok {
		a.config.SigninType = signinType
	}
	if timeoutMs, ok := config["timeout_ms"].(int); ok {
		a.config.TimeoutMs = timeoutMs
	}
	if timeoutMs, ok := config["timeout_ms"].(float64); ok {
		a.config.TimeoutMs = int(timeoutMs)
	}
	if refreshBuffer, ok := config["refresh_buffer_minutes"].(int); ok {
		a.config.RefreshBuffer = refreshBuffer
	}
	if refreshBuffer, ok := config["refresh_buffer_minutes"].(float64); ok {
		a.config.RefreshBuffer = int(refreshBuffer)
	}

	return nil
}

// GetBearerToken returns a valid bearer token, refreshing if necessary
func (a *AuthService) GetBearerToken(ctx context.Context) (string, error) {
	a.mutex.RLock()

	// Check if we have a valid token
	if a.token != nil && a.isTokenValid() {
		token := a.token.IDToken
		a.mutex.RUnlock()
		return fmt.Sprintf("Bearer %s", token), nil
	}

	a.mutex.RUnlock()

	// Need to get a new token
	return a.refreshToken(ctx)
}

// isTokenValid checks if the current token is still valid
func (a *AuthService) isTokenValid() bool {
	if a.token == nil {
		return false
	}

	// Check if token is expired (with buffer)
	expiryTime := a.token.ObtainedAt.Add(time.Duration(a.token.ExpiresIn) * time.Second)
	bufferTime := time.Duration(a.config.RefreshBuffer) * time.Minute

	return time.Now().Add(bufferTime).Before(expiryTime)
}

// refreshToken gets a new token from the auth API
func (a *AuthService) refreshToken(ctx context.Context) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Double-check if another goroutine already refreshed the token
	if a.token != nil && a.isTokenValid() {
		return fmt.Sprintf("Bearer %s", a.token.IDToken), nil
	}

	a.logger.Info("Refreshing authentication token")

	// Prepare login request
	loginRequest := LoginRequest{
		Username:   a.config.Username,
		Password:   a.config.Password,
		SigninType: a.config.SigninType,
	}

	// Prepare headers
	headers := map[string]string{
		"Accept":             "application/json, text/plain, */*",
		"Accept-Language":    "en-GB,en-US;q=0.9,en;q=0.8",
		"Connection":         "keep-alive",
		"Content-Type":       "application/json",
		"Origin":             "https://viasetu.sandbox-app.prayog.io",
		"Referer":            "https://viasetu.sandbox-app.prayog.io/",
		"Sec-Fetch-Dest":     "empty",
		"Sec-Fetch-Mode":     "cors",
		"Sec-Fetch-Site":     "same-site",
		"User-Agent":         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
		"sec-ch-ua":          `"Google Chrome";v="143", "Chromium";v="143", "Not A(Brand";v="24"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": `"macOS"`,
	}
	
	// Note: tenantId header is not sent in login request to match the working API behavior
	// The tenantId can be used for API calls after authentication, but login uses user's default tenant

	// Set timeout for auth request
	authCtx, cancel := context.WithTimeout(ctx, time.Duration(a.config.TimeoutMs)*time.Millisecond)
	defer cancel()

	// Make login request
	httpResponse, err := a.httpClient.Post(authCtx, a.config.LoginURL, loginRequest, headers)
	if err != nil {
		a.logger.Error("Authentication request failed", "error", err)
		return "", fmt.Errorf("authentication request failed: %w", err)
	}

	if httpResponse.StatusCode != 200 {
		a.logger.Error("Authentication API returned error",
			"status_code", httpResponse.StatusCode,
			"response", string(httpResponse.Body))
		return "", fmt.Errorf("authentication failed with status %d: %s",
			httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse login response
	var loginResponse LoginResponse
	if err := json.Unmarshal(httpResponse.Body, &loginResponse); err != nil {
		a.logger.Error("Failed to parse authentication response", "error", err)
		return "", fmt.Errorf("failed to parse authentication response: %w", err)
	}

	// Check if authentication was successful
	// If we have an id_token, treat as success (API may not include success field)
	// Only fail if success is explicitly false AND we don't have a token, or if token is missing
	if loginResponse.IDToken == "" {
		// No token - check if success field indicates failure
		if loginResponse.Success == false {
			errorMsg := loginResponse.Message
			if errorMsg == "" {
				errorMsg = loginResponse.Error
			}
			if errorMsg == "" {
				errorMsg = loginResponse.Status
			}
			if errorMsg == "" {
				errorMsg = "authentication failed"
			}
			a.logger.Error("Authentication failed",
				"success", loginResponse.Success,
				"status", loginResponse.Status,
				"error", loginResponse.Error,
				"message", loginResponse.Message,
				"tenant_id", loginResponse.TenantID)
			return "", fmt.Errorf("authentication failed: %s", errorMsg)
		}
		// No token and no explicit failure - still error
		a.logger.Error("Authentication response missing ID token")
		return "", fmt.Errorf("authentication response missing ID token")
	}

	// Store token info
	a.token = &TokenInfo{
		IDToken:      loginResponse.IDToken,
		RefreshToken: loginResponse.RefreshToken,
		ExpiresIn:    loginResponse.ExpiresIn,
		TokenType:    loginResponse.TokenType,
		ObtainedAt:   time.Now(),
		UserID:       loginResponse.UserID,
		UserEmail:    loginResponse.UserEmail,
		TenantID:     loginResponse.TenantID,
	}

	a.logger.Info("Authentication successful",
		"user_email", a.token.UserEmail,
		"user_id", a.token.UserID,
		"tenant_id", a.token.TenantID,
		"expires_in", a.token.ExpiresIn)

	return fmt.Sprintf("Bearer %s", a.token.IDToken), nil
}

// GetTokenInfo returns current token information (for debugging/monitoring)
func (a *AuthService) GetTokenInfo() *TokenInfo {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if a.token == nil {
		return nil
	}

	// Return a copy to avoid external modifications
	return &TokenInfo{
		IDToken:      "***HIDDEN***", // Don't expose actual token
		RefreshToken: "***HIDDEN***", // Don't expose refresh token
		ExpiresIn:    a.token.ExpiresIn,
		TokenType:    a.token.TokenType,
		ObtainedAt:   a.token.ObtainedAt,
		UserID:       a.token.UserID,
		UserEmail:    a.token.UserEmail,
		TenantID:     a.token.TenantID,
	}
}

// IsAuthenticated checks if we have a valid authentication
func (a *AuthService) IsAuthenticated() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.token != nil && a.isTokenValid()
}

// ClearToken clears the stored token (for logout or reset)
func (a *AuthService) ClearToken() {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.logger.Info("Clearing authentication token")
	a.token = nil
}

// ValidateConfig validates the authentication configuration
func (a *AuthService) ValidateConfig() error {
	if a.config.LoginURL == "" {
		return fmt.Errorf("login_url is required")
	}
	if a.config.Username == "" {
		return fmt.Errorf("username is required")
	}
	if a.config.Password == "" {
		return fmt.Errorf("password is required")
	}
	if a.config.SigninType == "" {
		return fmt.Errorf("signin_type is required")
	}
	if a.config.TimeoutMs <= 0 {
		return fmt.Errorf("timeout_ms must be positive")
	}
	if a.config.RefreshBuffer < 0 {
		return fmt.Errorf("refresh_buffer_minutes cannot be negative")
	}

	return nil
}
