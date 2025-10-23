package indiapost

import (
	"fmt"
)

// Config holds the configuration for India Post API
type Config struct {
	BaseURL         string
	LoginEndpoint   string
	TariffEndpoint  string
	Username        string
	Password        string
	TimeoutMs       int
	Environment     string
	TokenExpirySec  int
}

// NewDefaultConfig creates a default India Post configuration
func NewDefaultConfig() *Config {
	return &Config{
		BaseURL:         "https://test.cept.gov.in/beextcustomer/v1",
		LoginEndpoint:   "/access/login",
		TariffEndpoint:  "/international-tariff/calculate",
		Username:        "",
		Password:        "",
		TimeoutMs:       30000,
		Environment:     "test",
		TokenExpirySec:  900, // 15 minutes
	}
}

// LoadFromMap loads configuration from a map
func (c *Config) LoadFromMap(configMap map[string]interface{}) error {
	if baseURL, ok := configMap["base_url"].(string); ok && baseURL != "" {
		c.BaseURL = baseURL
	}

	if loginEndpoint, ok := configMap["login_endpoint"].(string); ok && loginEndpoint != "" {
		c.LoginEndpoint = loginEndpoint
	}

	if tariffEndpoint, ok := configMap["tariff_endpoint"].(string); ok && tariffEndpoint != "" {
		c.TariffEndpoint = tariffEndpoint
	}

	if username, ok := configMap["username"].(string); ok && username != "" {
		c.Username = username
	}

	if password, ok := configMap["password"].(string); ok && password != "" {
		c.Password = password
	}

	if timeoutMs, ok := configMap["timeout_ms"].(float64); ok && timeoutMs > 0 {
		c.TimeoutMs = int(timeoutMs)
	}

	if environment, ok := configMap["environment"].(string); ok && environment != "" {
		c.Environment = environment
	}

	if tokenExpiry, ok := configMap["token_expiry_sec"].(float64); ok && tokenExpiry > 0 {
		c.TokenExpirySec = int(tokenExpiry)
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

	if c.TimeoutMs <= 0 {
		c.TimeoutMs = 30000
	}

	if c.TokenExpirySec <= 0 {
		c.TokenExpirySec = 900
	}

	return nil
}

// GetLoginURL returns the full login URL
func (c *Config) GetLoginURL() string {
	return c.BaseURL + c.LoginEndpoint
}

// GetTariffURL returns the full tariff calculation URL
func (c *Config) GetTariffURL() string {
	return c.BaseURL + c.TariffEndpoint
}

