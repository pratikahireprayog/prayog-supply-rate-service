package delhivery

import (
	"os"
	"strconv"
)

// Config holds the configuration for Delhivery API
type Config struct {
	// API Configuration
	BaseURL           string `json:"base_url"`
	LoginEndpoint     string `json:"login_endpoint"`
	EstimateEndpoint  string `json:"estimate_endpoint"`

	// Authentication
	Username string `json:"username"`
	Password string `json:"password"`

	// HTTP Configuration
	TimeoutMs  int `json:"timeout_ms"`
	RetryCount int `json:"retry_count"`

	// Token Configuration
	TokenExpirySec int `json:"token_expiry_sec"` // Token expiry in seconds (default 7 days)
}

// NewDefaultConfig creates a default configuration for Delhivery service
func NewDefaultConfig() *Config {
	return &Config{
		// API Configuration - loaded from environment or defaults
		BaseURL:          getEnv("DELHIVERY_BASE_URL", "https://ltl-clients-api-dev.delhivery.com"),
		LoginEndpoint:    getEnv("DELHIVERY_LOGIN_ENDPOINT", "/ums/login"),
		EstimateEndpoint: getEnv("DELHIVERY_ESTIMATE_ENDPOINT", "/freight/estimate"),

		// Authentication - loaded from environment
		Username: getEnv("DELHIVERY_USERNAME", "SHREEMARUTIINTEGRAT6B2BC-B2B"),
		Password: getEnv("DELHIVERY_PASSWORD", "Welcome@1234"),

		// HTTP Configuration
		TimeoutMs:  getEnvAsInt("DELHIVERY_TIMEOUT_MS", 30000), // 30 seconds
		RetryCount: getEnvAsInt("DELHIVERY_RETRY_COUNT", 2),

		// Token Configuration - JWT tokens typically expire in 7 days
		TokenExpirySec: getEnvAsInt("DELHIVERY_TOKEN_EXPIRY_SEC", 604800), // 7 days default
	}
}

// GetLoginURL returns the full login URL
func (c *Config) GetLoginURL() string {
	return c.BaseURL + c.LoginEndpoint
}

// GetEstimateURL returns the full estimate URL
func (c *Config) GetEstimateURL() string {
	return c.BaseURL + c.EstimateEndpoint
}

// LoadFromMap loads configuration from a map (for database config)
func (c *Config) LoadFromMap(configMap map[string]interface{}) error {
	if configMap == nil {
		return nil
	}

	if val, ok := configMap["base_url"].(string); ok && val != "" {
		c.BaseURL = val
	}
	if val, ok := configMap["login_endpoint"].(string); ok && val != "" {
		c.LoginEndpoint = val
	}
	if val, ok := configMap["estimate_endpoint"].(string); ok && val != "" {
		c.EstimateEndpoint = val
	}
	if val, ok := configMap["username"].(string); ok && val != "" {
		c.Username = val
	}
	if val, ok := configMap["password"].(string); ok && val != "" {
		c.Password = val
	}
	if val, ok := configMap["timeout_ms"].(int); ok && val > 0 {
		c.TimeoutMs = val
	}
	if val, ok := configMap["timeout_ms"].(float64); ok && val > 0 {
		c.TimeoutMs = int(val)
	}
	if val, ok := configMap["retry_count"].(int); ok && val >= 0 {
		c.RetryCount = val
	}
	if val, ok := configMap["retry_count"].(float64); ok && val >= 0 {
		c.RetryCount = int(val)
	}
	if val, ok := configMap["token_expiry_sec"].(int); ok && val > 0 {
		c.TokenExpirySec = val
	}
	if val, ok := configMap["token_expiry_sec"].(float64); ok && val > 0 {
		c.TokenExpirySec = int(val)
	}

	return nil
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

