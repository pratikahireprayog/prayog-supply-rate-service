package dhl

import (
	"fmt"
	"os"
)

// Config holds DHL-specific configuration
type Config struct {
	BaseURL       string `json:"base_url"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	BasicAuth     string `json:"basic_auth"` // Base64 encoded username:password
	AccountNumber string `json:"account_number,omitempty"`
	TimeoutMs     int    `json:"timeout_ms"`
	RetryCount    int    `json:"retry_count"`
	Enabled       bool   `json:"enabled"`
}

// NewDefaultConfig creates a new default configuration for DHL
// SECURITY: Credentials should be loaded from environment variables
func NewDefaultConfig() *Config {
	// Load from environment variables with fallback to hardcoded test values
	baseURL := getEnv("DHL_BASE_URL", "")
	username := getEnv("DHL_USERNAME", "")
	password := getEnv("DHL_PASSWORD", "")
	basicAuth := getEnv("DHL_BASIC_AUTH", "")
	accountNumber := getEnv("DHL_ACCOUNT_NUMBER", "533748932") // Default test account number
	enabled := getEnv("DHL_ENABLED", "true") == "true"

	return &Config{
		BaseURL:       baseURL,
		Username:      username,
		Password:      password,
		BasicAuth:     basicAuth,
		AccountNumber: accountNumber,
		TimeoutMs:     30000, // 30 seconds
		RetryCount:    2,
		Enabled:       enabled,
	}
}

// getEnv gets an environment variable with a fallback default
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// NewProductionConfig creates a production configuration for DHL (deprecated)
// Use NewDefaultConfig() instead - it loads from environment variables
func NewProductionConfig() *Config {
	return NewDefaultConfig()
}

// LoadFromMap loads configuration from a map
func (c *Config) LoadFromMap(config map[string]interface{}) error {
	if baseURL, ok := config["base_url"].(string); ok && baseURL != "" {
		c.BaseURL = baseURL
	}

	if username, ok := config["username"].(string); ok && username != "" {
		c.Username = username
	}

	if password, ok := config["password"].(string); ok && password != "" {
		c.Password = password
	}

	if basicAuth, ok := config["basic_auth"].(string); ok && basicAuth != "" {
		c.BasicAuth = basicAuth
	}

	if accountNumber, ok := config["account_number"].(string); ok && accountNumber != "" {
		c.AccountNumber = accountNumber
	}

	if timeoutMs, ok := config["timeout_ms"].(int); ok && timeoutMs > 0 {
		c.TimeoutMs = timeoutMs
	}

	if retryCount, ok := config["retry_count"].(int); ok && retryCount >= 0 {
		c.RetryCount = retryCount
	}

	if enabled, ok := config["enabled"].(bool); ok {
		c.Enabled = enabled
	}

	return c.Validate()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required (set DHL_BASE_URL)")
	}

	if c.BasicAuth == "" {
		return fmt.Errorf("basic_auth is required (set DHL_BASIC_AUTH)")
	}

	// Account number is optional - some DHL endpoints don't require it

	if c.TimeoutMs <= 0 {
		return fmt.Errorf("timeout_ms must be positive")
	}

	if c.RetryCount < 0 {
		return fmt.Errorf("retry_count cannot be negative")
	}

	return nil
}

// ToMap converts configuration to a map
func (c *Config) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       c.BaseURL,
		"username":       c.Username,
		"account_number": c.AccountNumber,
		"timeout_ms":     c.TimeoutMs,
		"retry_count":    c.RetryCount,
		"enabled":        c.Enabled,
		// Note: We don't include password/basic_auth in the map for security
	}
}
