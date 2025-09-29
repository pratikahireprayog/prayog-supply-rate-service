package dhl

import (
	"fmt"
)

// Config holds DHL-specific configuration
type Config struct {
	BaseURL       string `json:"base_url"`
	Credentials   string `json:"credentials"` // Base64 encoded username:password
	AccountNumber string `json:"account_number"`
	TimeoutMs     int    `json:"timeout_ms"`
	RetryCount    int    `json:"retry_count"`
	Environment   string `json:"environment"` // "test" or "production"
}

// NewDefaultConfig creates a new default configuration for DHL
func NewDefaultConfig() *Config {
	return &Config{
		BaseURL:       "https://express.api.dhl.com/mydhlapi/test/rates",
		Credentials:   "c2hyZWVtYXJ1dDhJTjpJITBwTV40c1IjNG5KJDF1", // Base64: shreemarut8IN:I!0pM^4sR#4nJ$1u
		AccountNumber: "533748932",
		TimeoutMs:     30000, // 30 seconds
		RetryCount:    2,
		Environment:   "test",
	}
}

// NewProductionConfig creates a production configuration for DHL
func NewProductionConfig() *Config {
	return &Config{
		BaseURL:       "https://express.api.dhl.com/mydhlapi/rates",
		Credentials:   "", // Should be set via environment variables
		AccountNumber: "", // Should be set via environment variables
		TimeoutMs:     30000,
		RetryCount:    2,
		Environment:   "production",
	}
}

// LoadFromMap loads configuration from a map
func (c *Config) LoadFromMap(config map[string]interface{}) error {
	if baseURL, ok := config["base_url"].(string); ok && baseURL != "" {
		c.BaseURL = baseURL
	}

	if credentials, ok := config["credentials"].(string); ok && credentials != "" {
		c.Credentials = credentials
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

	if environment, ok := config["environment"].(string); ok && environment != "" {
		c.Environment = environment

		// Auto-update base URL based on environment
		if environment == "production" {
			c.BaseURL = "https://express.api.dhl.com/mydhlapi/rates"
		} else if environment == "test" {
			c.BaseURL = "https://express.api.dhl.com/mydhlapi/test/rates"
		}
	}

	return c.Validate()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	if c.Credentials == "" {
		return fmt.Errorf("credentials are required")
	}

	if c.AccountNumber == "" {
		return fmt.Errorf("account_number is required")
	}

	if c.TimeoutMs <= 0 {
		return fmt.Errorf("timeout_ms must be positive")
	}

	if c.RetryCount < 0 {
		return fmt.Errorf("retry_count cannot be negative")
	}

	if c.Environment != "test" && c.Environment != "production" {
		return fmt.Errorf("environment must be 'test' or 'production'")
	}

	return nil
}

// ToMap converts configuration to a map
func (c *Config) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       c.BaseURL,
		"account_number": c.AccountNumber,
		"timeout_ms":     c.TimeoutMs,
		"retry_count":    c.RetryCount,
		"environment":    c.Environment,
		// Note: We don't include credentials in the map for security
	}
}
