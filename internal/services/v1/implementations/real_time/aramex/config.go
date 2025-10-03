package aramex

import (
	"os"
)

// Config holds ARAMEX -specific configuration
type Config struct {
    BaseURL           string
    Username          string
    Password          string
    AccountNumber     string
    AccountPin        string
    AccountEntity     string
    AccountCountryCode string
    Source            int
    Version           string
}

// NewDefaultConfig creates a new default configuration for ARAMEX
// SECURITY: Credentials should be loaded from environment variables
func NewDefaultConfig() *Config {
    return &Config{
        BaseURL:           getEnv("ARAMEX_BASE_URL", "https://ws.aramex.net/ShippingAPI.V2/RateCalculator/Service_1_0.svc/json"),
        Username:          getEnv("ARAMEX_USERNAME", "test.api@aramex.com"),
        Password:          getEnv("ARAMEX_PASSWORD", "Aramex@12345"),
        Version:           getEnv("ARAMEX_VERSION", "v1.0"),
        AccountNumber:     getEnv("ARAMEX_ACCOUNT_NUMBER", "60531487"),
        AccountPin:        getEnv("ARAMEX_ACCOUNT_PIN", "654654"),
        AccountEntity:     getEnv("ARAMEX_ACCOUNT_ENTITY", "BOM"),
        AccountCountryCode:getEnv("ARAMEX_ACCOUNT_COUNTRY_CODE", "IN"),
        Source:            24,
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

// NewProductionConfig creates a production configuration for ARAMEX (deprecated)
// Use NewDefaultConfig() instead - it loads from environment variables
func NewProductionConfig() *Config {
	return NewDefaultConfig()
}

// LoadFromMap loads configuration from a map
func (c *Config) LoadFromMap(config map[string]interface{}) error {
    if baseURL, ok := config["base_url"].(string); ok {
        c.BaseURL = baseURL
    }
    if username, ok := config["username"].(string); ok {
        c.Username = username
    }
    if password, ok := config["password"].(string); ok {
        c.Password = password
    }
    if accountNumber, ok := config["account_number"].(string); ok {
        c.AccountNumber = accountNumber
    }
    if accountPin, ok := config["account_pin"].(string); ok {
        c.AccountPin = accountPin
    }
    if accountEntity, ok := config["account_entity"].(string); ok {
        c.AccountEntity = accountEntity
    }
    if accountCountryCode, ok := config["account_country_code"].(string); ok {
        c.AccountCountryCode = accountCountryCode
    }
    if source, ok := config["source"].(int); ok {
        c.Source = source
    }
    if version, ok := config["version"].(string); ok {
        c.Version = version
    }

	return nil;
}
