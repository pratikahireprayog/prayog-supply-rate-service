package fedex

import (
	"os"
	"strconv"
	"strings"
)

// Config holds FedEx service configuration
type Config struct {
    BaseURL       string
    ClientID      string
    ClientSecret  string
    AccountNumber string
    Environment   string
    TimeoutMs     int
    RetryCount    int
}

// NewDefaultConfig creates default FedEx configuration with environment variables
func NewDefaultConfig() *Config {
    return &Config{
        BaseURL:       getEnv("FEDEX_BASE_URL", ""),
        ClientID:      getEnv("FEDEX_CLIENT_ID", ""),
        ClientSecret:  getEnv("FEDEX_CLIENT_SECRET", ""),
        AccountNumber: getEnv("FEDEX_ACCOUNT_NUMBER", ""),
        Environment:   getEnv("FEDEX_ENVIRONMENT", ""),
        TimeoutMs:     getEnvAsInt("FEDEX_TIMEOUT_MS", 30000),
        RetryCount:    getEnvAsInt("FEDEX_RETRY_COUNT", 2),
    }
}

// LoadFromMap loads configuration from map (overrides env vars)
func (c *Config) LoadFromMap(config map[string]interface{}) error {
    if baseURL, ok := config["base_url"].(string); ok && baseURL != "" {
        c.BaseURL = baseURL
    }
    if clientID, ok := config["client_id"].(string); ok && clientID != "" {
        c.ClientID = clientID
    }
    if clientSecret, ok := config["client_secret"].(string); ok && clientSecret != "" {
        c.ClientSecret = clientSecret
    }
    if accountNumber, ok := config["account_number"].(string); ok && accountNumber != "" {
        c.AccountNumber = accountNumber
    }
    if environment, ok := config["environment"].(string); ok && environment != "" {
        c.Environment = environment
    }
    if timeoutMs, ok := config["timeout_ms"].(int); ok && timeoutMs > 0 {
        c.TimeoutMs = timeoutMs
    }
    if retryCount, ok := config["retry_count"].(int); ok && retryCount > 0 {
        c.RetryCount = retryCount
    }
    
    return nil
}

// Helper function to get environment variable with default
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return strings.TrimSpace(value)
    }
    return defaultValue
}

// Helper function to get environment variable as int with default
func getEnvAsInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
            return intValue
        }
    }
    return defaultValue
}