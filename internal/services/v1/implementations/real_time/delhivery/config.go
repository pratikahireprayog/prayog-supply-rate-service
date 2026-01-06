package delhivery

import (
	"os"
	"strconv"
)

// Config holds the configuration for Delhivery API
type Config struct {
	// API Configuration
	BaseURL          string `json:"base_url"`
	EstimateEndpoint string `json:"estimate_endpoint"`

	// Authentication
	APIKey   string `json:"api_key"`
	Username string `json:"username"`
	Password string `json:"password"`

	// HTTP Configuration
	TimeoutMs  int `json:"timeout_ms"`
	RetryCount int `json:"retry_count"`
}

// NewDefaultConfig creates a default configuration for Delhivery service
func NewDefaultConfig() *Config {
	return &Config{
		// API Configuration - loaded from environment or defaults
		BaseURL:          getEnv("DELHIVERY_BASE_URL", "https://track.delhivery.com"),
		EstimateEndpoint: getEnv("DELHIVERY_ESTIMATE_ENDPOINT", "/api/kinko/v1/invoice/charges/.json"),

		// Authentication - loaded from environment
		APIKey:   getEnv("DELHIVERY_API_KEY", "7882000764f1aa847f8e0addadb7262eb7ad8de6"),
		Username: getEnv("DELHIVERY_USERNAME", ""),
		Password: getEnv("DELHIVERY_PASSWORD", ""),

		// HTTP Configuration
		TimeoutMs:  getEnvAsInt("DELHIVERY_TIMEOUT_MS", 30000), // 30 seconds
		RetryCount: getEnvAsInt("DELHIVERY_RETRY_COUNT", 2),
	}
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
	if val, ok := configMap["estimate_endpoint"].(string); ok && val != "" {
		c.EstimateEndpoint = val
	}
	if val, ok := configMap["api_key"].(string); ok && val != "" {
		c.APIKey = val
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
