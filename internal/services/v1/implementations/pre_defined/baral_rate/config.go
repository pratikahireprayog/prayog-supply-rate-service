package baral_rate

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for Baral Rate Card service
type Config struct {
	// API Configuration
	BaseURL                 string `json:"base_url"`
	AllRateCardsEndpoint    string `json:"all_rate_cards_endpoint"`
	PartnerCode             string `json:"partner_code"`

	// HTTP Configuration
	TimeoutMs  int `json:"timeout_ms"`
	RetryCount int `json:"retry_count"`

	// Cache Configuration
	CacheTTLMinutes int  `json:"cache_ttl_minutes"`
	EnableCache     bool `json:"enable_cache"`

	// Service Configuration
	DefaultCurrency string `json:"default_currency"`

	// Feature Flags
	EnableHealthCheck  bool `json:"enable_health_check"`
	EnableMetrics      bool `json:"enable_metrics"`
	EnableDetailedLogs bool `json:"enable_detailed_logs"`
}

// NewDefaultConfig creates a default configuration for Baral Rate Card service
func NewDefaultConfig() *Config {
	return &Config{
		// API Configuration - loaded from environment or defaults
		BaseURL:              getEnv("BARAL_BASE_URL", "https://apis2.delcaper.com/rate-card-api"),
		AllRateCardsEndpoint: getEnv("BARAL_ENDPOINT", "/allratecards"),
		PartnerCode:          getEnv("BARAL_PARTNER_CODE", "sunil_baral"),

		// HTTP Configuration
		TimeoutMs:  getEnvAsInt("BARAL_TIMEOUT_MS", 30000), // 30 seconds
		RetryCount: getEnvAsInt("BARAL_RETRY_COUNT", 2),

		// Cache Configuration
		CacheTTLMinutes: getEnvAsInt("BARAL_CACHE_TTL_MINUTES", 60), // 60 minutes cache
		EnableCache:     true,

		// Service Configuration
		DefaultCurrency: getEnv("BARAL_DEFAULT_CURRENCY", "INR"),

		// Feature Flags
		EnableHealthCheck:  true,
		EnableMetrics:      true,
		EnableDetailedLogs: false,
	}
}

// UpdateFromMap updates configuration from a map of key-value pairs
func (c *Config) UpdateFromMap(configMap map[string]interface{}) error {
	if configMap == nil {
		return nil
	}

	// Update API configuration
	if baseURL, ok := configMap["base_url"].(string); ok && baseURL != "" {
		c.BaseURL = baseURL
	}
	if endpoint, ok := configMap["all_rate_cards_endpoint"].(string); ok && endpoint != "" {
		c.AllRateCardsEndpoint = endpoint
	}
	if partnerCode, ok := configMap["partner_code"].(string); ok && partnerCode != "" {
		c.PartnerCode = partnerCode
	}

	// Update HTTP configuration
	if timeoutMs, ok := configMap["timeout_ms"].(int); ok {
		c.TimeoutMs = timeoutMs
	}
	if timeoutMs, ok := configMap["timeout_ms"].(float64); ok {
		c.TimeoutMs = int(timeoutMs)
	}
	if retryCount, ok := configMap["retry_count"].(int); ok {
		c.RetryCount = retryCount
	}
	if retryCount, ok := configMap["retry_count"].(float64); ok {
		c.RetryCount = int(retryCount)
	}

	// Update cache configuration
	if cacheTTL, ok := configMap["cache_ttl_minutes"].(int); ok {
		c.CacheTTLMinutes = cacheTTL
	}
	if cacheTTL, ok := configMap["cache_ttl_minutes"].(float64); ok {
		c.CacheTTLMinutes = int(cacheTTL)
	}
	if enableCache, ok := configMap["enable_cache"].(bool); ok {
		c.EnableCache = enableCache
	}

	// Update service configuration
	if currency, ok := configMap["default_currency"].(string); ok && currency != "" {
		c.DefaultCurrency = currency
	}

	// Update feature flags
	if enableHealth, ok := configMap["enable_health_check"].(bool); ok {
		c.EnableHealthCheck = enableHealth
	}
	if enableMetrics, ok := configMap["enable_metrics"].(bool); ok {
		c.EnableMetrics = enableMetrics
	}
	if enableLogs, ok := configMap["enable_detailed_logs"].(bool); ok {
		c.EnableDetailedLogs = enableLogs
	}

	return c.Validate()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate required fields
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if c.AllRateCardsEndpoint == "" {
		return fmt.Errorf("all_rate_cards_endpoint is required")
	}
	if c.PartnerCode == "" {
		return fmt.Errorf("partner_code is required")
	}

	// Validate timeout
	if c.TimeoutMs <= 0 {
		return fmt.Errorf("timeout_ms must be positive")
	}
	if c.TimeoutMs < 1000 {
		return fmt.Errorf("timeout_ms must be at least 1000ms")
	}
	if c.TimeoutMs > 120000 {
		return fmt.Errorf("timeout_ms must not exceed 120000ms (2 minutes)")
	}

	// Validate retry count
	if c.RetryCount < 0 {
		return fmt.Errorf("retry_count cannot be negative")
	}
	if c.RetryCount > 5 {
		return fmt.Errorf("retry_count should not exceed 5")
	}

	// Validate cache TTL
	if c.CacheTTLMinutes < 0 {
		return fmt.Errorf("cache_ttl_minutes cannot be negative")
	}
	if c.CacheTTLMinutes > 1440 { // 24 hours
		return fmt.Errorf("cache_ttl_minutes should not exceed 1440 minutes (24 hours)")
	}

	// Validate currency
	if c.DefaultCurrency == "" {
		return fmt.Errorf("default_currency is required")
	}
	if len(c.DefaultCurrency) != 3 {
		return fmt.Errorf("default_currency must be a 3-letter ISO code")
	}

	return nil
}

// GetAllRateCardsURL returns the full URL for the all rate cards endpoint
func (c *Config) GetAllRateCardsURL() string {
	return fmt.Sprintf("%s%s?partnerCode=%s", c.BaseURL, c.AllRateCardsEndpoint, c.PartnerCode)
}

// ToMap converts configuration to map for JSON serialization
func (c *Config) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"base_url":                 c.BaseURL,
		"all_rate_cards_endpoint":  c.AllRateCardsEndpoint,
		"partner_code":             c.PartnerCode,
		"timeout_ms":               c.TimeoutMs,
		"retry_count":              c.RetryCount,
		"cache_ttl_minutes":        c.CacheTTLMinutes,
		"enable_cache":             c.EnableCache,
		"default_currency":         c.DefaultCurrency,
		"enable_health_check":      c.EnableHealthCheck,
		"enable_metrics":           c.EnableMetrics,
		"enable_detailed_logs":     c.EnableDetailedLogs,
	}
}

// GetTimeout returns timeout as duration
func (c *Config) GetTimeout() time.Duration {
	return time.Duration(c.TimeoutMs) * time.Millisecond
}

// GetCacheTTL returns cache TTL as duration
func (c *Config) GetCacheTTL() time.Duration {
	return time.Duration(c.CacheTTLMinutes) * time.Minute
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt reads an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
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

