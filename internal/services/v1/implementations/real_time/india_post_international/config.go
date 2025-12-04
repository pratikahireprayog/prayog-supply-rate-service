package india_post_international

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for India Post International API
type Config struct {
	BaseURL           string
	LoginURL          string
	TariffURL         string
	Username          string
	Password          string
	Timeout           time.Duration
	TokenExpiryBuffer time.Duration
	MaxRetries        int
	RetryDelay        time.Duration
	Enabled           bool
	Rating            float64
}

// NewDefaultConfig creates a default India Post International configuration from environment variables
func NewDefaultConfig() *Config {
	config := &Config{
		BaseURL:           getEnvOrDefault("INDIA_POST_INTL_BASE_URL", "https://test.cept.gov.in/beextcustomer"),
		Username:          getEnvOrDefault("INDIA_POST_INTL_USERNAME", "9999999999"),
		Password:          getEnvOrDefault("INDIA_POST_INTL_PASSWORD", "Dop@1234"),
		LoginURL:          getEnvOrDefault("INDIA_POST_INTL_LOGIN_URL", "/v1/access/login"),
		TariffURL:         getEnvOrDefault("INDIA_POST_INTL_TARIFF_URL", "/v1/international-tariff/itps"),
		Timeout:           getEnvAsDurationOrDefault("INDIA_POST_INTL_TIMEOUT", 30*time.Second),
		TokenExpiryBuffer: getEnvAsDurationOrDefault("INDIA_POST_INTL_TOKEN_EXPIRY_BUFFER", 5*time.Minute),
		MaxRetries:        getEnvAsIntOrDefault("INDIA_POST_INTL_MAX_RETRIES", 3),
		RetryDelay:        getEnvAsDurationOrDefault("INDIA_POST_INTL_RETRY_DELAY", 1*time.Second),
		Enabled:           getEnvAsBoolOrDefault("INDIA_POST_INTL_ENABLED", true),
		Rating:            4.0,
	}

	return config
}

// LoadFromMap loads configuration from a map
func (c *Config) LoadFromMap(configMap map[string]interface{}) error {
	if baseURL, ok := configMap["base_url"].(string); ok && baseURL != "" {
		c.BaseURL = baseURL
	}

	if loginURL, ok := configMap["login_url"].(string); ok && loginURL != "" {
		c.LoginURL = loginURL
	}

	if tariffURL, ok := configMap["tariff_url"].(string); ok && tariffURL != "" {
		c.TariffURL = tariffURL
	}

	if username, ok := configMap["username"].(string); ok && username != "" {
		c.Username = username
	}

	if password, ok := configMap["password"].(string); ok && password != "" {
		c.Password = password
	}

	if timeoutMs, ok := configMap["timeout_ms"].(float64); ok && timeoutMs > 0 {
		c.Timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	if timeout, ok := configMap["timeout"].(string); ok && timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil {
			c.Timeout = parsed
		}
	}

	if tokenExpiryBuffer, ok := configMap["token_expiry_buffer"].(string); ok && tokenExpiryBuffer != "" {
		if parsed, err := time.ParseDuration(tokenExpiryBuffer); err == nil {
			c.TokenExpiryBuffer = parsed
		}
	}

	if maxRetries, ok := configMap["max_retries"].(float64); ok && maxRetries > 0 {
		c.MaxRetries = int(maxRetries)
	}

	if retryDelay, ok := configMap["retry_delay"].(string); ok && retryDelay != "" {
		if parsed, err := time.ParseDuration(retryDelay); err == nil {
			c.RetryDelay = parsed
		}
	}

	if enabled, ok := configMap["enabled"].(bool); ok {
		c.Enabled = enabled
	}

	if rating, ok := configMap["rating"].(float64); ok && rating > 0 {
		c.Rating = rating
	}

	return c.Validate()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	if c.Username == "" {
		return fmt.Errorf("username is required")
	}

	if c.Password == "" {
		return fmt.Errorf("password is required")
	}

	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	if c.TokenExpiryBuffer <= 0 {
		c.TokenExpiryBuffer = 5 * time.Minute
	}

	if c.MaxRetries < 0 {
		c.MaxRetries = 3
	}

	if c.RetryDelay <= 0 {
		c.RetryDelay = 1 * time.Second
	}

	return nil
}

// GetLoginURL returns the full login URL
func (c *Config) GetLoginURL() string {
	return c.BaseURL + c.LoginURL
}

// GetTariffURL returns the full tariff calculation URL
func (c *Config) GetTariffURL() string {
	return c.BaseURL + c.TariffURL
}

// GetRefreshTokenURL returns the refresh token URL
func (c *Config) GetRefreshTokenURL() string {
	return c.BaseURL + "/beextcustomer/v1/access/TokenWithRtoken"
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsBoolOrDefault(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

