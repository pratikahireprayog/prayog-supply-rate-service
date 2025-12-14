package mover

import (
	"os"
	"strconv"
)

// Config holds the configuration for Mover API
type Config struct {
	// API Configuration
	BaseURL          string `json:"base_url"`
	EstimateEndpoint string `json:"estimate_endpoint"`

	// Authentication - Hardcoded Basic Auth token
	AuthToken string `json:"auth_token"`

	// HTTP Configuration
	TimeoutMs  int `json:"timeout_ms"`
	RetryCount int `json:"retry_count"`

	// Default contact information
	DefaultContactName   string `json:"default_contact_name"`
	DefaultContactMobile string `json:"default_contact_mobile"`
}

// NewDefaultConfig creates a default configuration for Mover service
func NewDefaultConfig() *Config {
	return &Config{
		// API Configuration
		BaseURL:          getEnv("MOVER_BASE_URL", "https://api.boxnmove.com"),
		EstimateEndpoint: getEnv("MOVER_ESTIMATE_ENDPOINT", "/business/order-estimate"),

		// Hardcoded Basic Auth token as per requirements
		AuthToken: "cHJheW9nLWNsaWVudEBtb3Zlci5kZWxpdmVyeTpFeUdXREFpNmd1SnJhSGlM",

		// HTTP Configuration
		TimeoutMs:  getEnvAsInt("MOVER_TIMEOUT_MS", 30000), // 30 seconds
		RetryCount: getEnvAsInt("MOVER_RETRY_COUNT", 2),

		// Default contact information
		DefaultContactName:   getEnv("MOVER_DEFAULT_CONTACT_NAME", "Jhon Doe"),
		DefaultContactMobile: getEnv("MOVER_DEFAULT_CONTACT_MOBILE", "9999999999"),
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
	if val, ok := configMap["auth_token"].(string); ok && val != "" {
		c.AuthToken = val
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
	if val, ok := configMap["default_contact_name"].(string); ok && val != "" {
		c.DefaultContactName = val
	}
	if val, ok := configMap["default_contact_mobile"].(string); ok && val != "" {
		c.DefaultContactMobile = val
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

