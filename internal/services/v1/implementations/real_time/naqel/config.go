package naqel

import (
	"os"
)

// Config holds Naqel-specific configuration
type Config struct {
	BaseURL    string
	ClientID   string
	Password   string
	Version    string
	ShipperName string
	TimeoutMs  int
	RetryCount int
}

// NewDefaultConfig creates a new default configuration for Naqel
// SECURITY: Credentials should be loaded from environment variables
func NewDefaultConfig() *Config {
	return &Config{
		BaseURL:     getEnv("NAQEL_BASE_URL", ""),
		ClientID:    getEnv("NAQEL_CLIENT_ID", ""),
		Password:    getEnv("NAQEL_PASSWORD", ""),
		Version:     getEnv("NAQEL_VERSION", "9.0"),
		ShipperName: getEnv("NAQEL_SHIPPER_NAME", ""),
		TimeoutMs:   30000,
		RetryCount:  2,
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

// LoadFromMap loads configuration from a map
func (c *Config) LoadFromMap(config map[string]interface{}) error {
	if baseURL, ok := config["base_url"].(string); ok {
		c.BaseURL = baseURL
	}
	if clientID, ok := config["client_id"].(string); ok {
		c.ClientID = clientID
	}
	if password, ok := config["password"].(string); ok {
		c.Password = password
	}
	if version, ok := config["version"].(string); ok {
		c.Version = version
	}
	if shipperName, ok := config["shipper_name"].(string); ok {
		c.ShipperName = shipperName
	}
	if timeoutMs, ok := config["timeout_ms"].(int); ok {
		c.TimeoutMs = timeoutMs
	}
	if retryCount, ok := config["retry_count"].(int); ok {
		c.RetryCount = retryCount
	}

	return nil
}

