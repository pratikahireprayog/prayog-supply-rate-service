package unified_rate

import (
	"fmt"
	"time"
)

// Config holds the configuration for Unified Rate service
type Config struct {
	// API Configuration
	BaseURL                string `json:"base_url"`
	CalculateRatesEndpoint string `json:"calculate_rates_endpoint"`

	// Authentication Configuration
	LoginURL             string `json:"login_url"`
	Username             string `json:"username"`
	Password             string `json:"password"`
	SigninType           string `json:"signin_type"`
	RefreshBufferMinutes int    `json:"refresh_buffer_minutes"`

	// HTTP Configuration
	TimeoutMs  int `json:"timeout_ms"`
	RetryCount int `json:"retry_count"`

	// Cache Configuration
	CacheTTLMinutes int `json:"cache_ttl_minutes"`

	// Service Configuration
	DefaultCurrency       string   `json:"default_currency"`
	SupportedCurrencies   []string `json:"supported_currencies"`
	MaxPackagesPerRequest int      `json:"max_packages_per_request"`

	// Rate Calculation Configuration
	DefaultServiceTypes []string `json:"default_service_types"`
	EnableAllServices   bool     `json:"enable_all_services"`

	// Feature Flags
	EnableHealthCheck  bool `json:"enable_health_check"`
	EnableMetrics      bool `json:"enable_metrics"`
	EnableDetailedLogs bool `json:"enable_detailed_logs"`
}

// NewDefaultConfig creates a default configuration for Unified Rate service
func NewDefaultConfig() *Config {
	return &Config{
		// API Configuration
		BaseURL:                "https://sandbox-apis.prayog.io/gateway/ure/api",
		CalculateRatesEndpoint: "/external-rate-calculation/calculate",

		// Authentication Configuration
		LoginURL:             "https://sandbox-apis.prayog.io/auth/login",
		Username:             "avinash.singh@prayog.io",
		Password:             "Prayog@Avinash539",
		SigninType:           "EMAIL",
		RefreshBufferMinutes: 30,

		// HTTP Configuration
		TimeoutMs:  30000, // 30 seconds
		RetryCount: 2,

		// Cache Configuration
		CacheTTLMinutes: 15, // 15 minutes cache

		// Service Configuration
		DefaultCurrency:       "INR",
		SupportedCurrencies:   []string{"INR", "USD", "EUR", "GBP"},
		MaxPackagesPerRequest: 10,

		// Rate Calculation Configuration
		DefaultServiceTypes: []string{"standard", "express"},
		EnableAllServices:   true,

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
	if baseURL, ok := configMap["base_url"].(string); ok {
		c.BaseURL = baseURL
	}
	if endpoint, ok := configMap["calculate_rates_endpoint"].(string); ok {
		c.CalculateRatesEndpoint = endpoint
	}

	// Update Authentication configuration
	if loginURL, ok := configMap["login_url"].(string); ok {
		c.LoginURL = loginURL
	}
	if username, ok := configMap["username"].(string); ok {
		c.Username = username
	}
	if password, ok := configMap["password"].(string); ok {
		c.Password = password
	}
	if signinType, ok := configMap["signin_type"].(string); ok {
		c.SigninType = signinType
	}
	if refreshBuffer, ok := configMap["refresh_buffer_minutes"].(int); ok {
		c.RefreshBufferMinutes = refreshBuffer
	}
	if refreshBuffer, ok := configMap["refresh_buffer_minutes"].(float64); ok {
		c.RefreshBufferMinutes = int(refreshBuffer)
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

	// Update service configuration
	if currency, ok := configMap["default_currency"].(string); ok {
		c.DefaultCurrency = currency
	}
	if currencies, ok := configMap["supported_currencies"].([]string); ok {
		c.SupportedCurrencies = currencies
	}
	if currencies, ok := configMap["supported_currencies"].([]interface{}); ok {
		c.SupportedCurrencies = make([]string, len(currencies))
		for i, curr := range currencies {
			if currStr, ok := curr.(string); ok {
				c.SupportedCurrencies[i] = currStr
			}
		}
	}
	if maxPackages, ok := configMap["max_packages_per_request"].(int); ok {
		c.MaxPackagesPerRequest = maxPackages
	}
	if maxPackages, ok := configMap["max_packages_per_request"].(float64); ok {
		c.MaxPackagesPerRequest = int(maxPackages)
	}

	// Update rate calculation configuration
	if serviceTypes, ok := configMap["default_service_types"].([]string); ok {
		c.DefaultServiceTypes = serviceTypes
	}
	if serviceTypes, ok := configMap["default_service_types"].([]interface{}); ok {
		c.DefaultServiceTypes = make([]string, len(serviceTypes))
		for i, st := range serviceTypes {
			if stStr, ok := st.(string); ok {
				c.DefaultServiceTypes[i] = stStr
			}
		}
	}
	if enableAll, ok := configMap["enable_all_services"].(bool); ok {
		c.EnableAllServices = enableAll
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

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate required fields
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if c.CalculateRatesEndpoint == "" {
		return fmt.Errorf("calculate_rates_endpoint is required")
	}

	// Validate authentication configuration
	if c.LoginURL == "" {
		return fmt.Errorf("login_url is required")
	}
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}
	if c.SigninType == "" {
		return fmt.Errorf("signin_type is required")
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

	// Validate refresh buffer
	if c.RefreshBufferMinutes < 0 {
		return fmt.Errorf("refresh_buffer_minutes cannot be negative")
	}
	if c.RefreshBufferMinutes > 120 { // 2 hours
		return fmt.Errorf("refresh_buffer_minutes should not exceed 120 minutes")
	}

	// Validate currency
	if c.DefaultCurrency == "" {
		return fmt.Errorf("default_currency is required")
	}
	if len(c.DefaultCurrency) != 3 {
		return fmt.Errorf("default_currency must be a 3-letter ISO code")
	}

	// Validate supported currencies
	if len(c.SupportedCurrencies) == 0 {
		return fmt.Errorf("at least one supported currency is required")
	}
	for _, currency := range c.SupportedCurrencies {
		if len(currency) != 3 {
			return fmt.Errorf("all supported currencies must be 3-letter ISO codes")
		}
	}

	// Validate max packages
	if c.MaxPackagesPerRequest <= 0 {
		return fmt.Errorf("max_packages_per_request must be positive")
	}
	if c.MaxPackagesPerRequest > 50 {
		return fmt.Errorf("max_packages_per_request should not exceed 50")
	}

	// Validate default service types
	if len(c.DefaultServiceTypes) == 0 {
		return fmt.Errorf("at least one default service type is required")
	}
	validServiceTypes := map[string]bool{
		"standard": true,
		"express":  true,
		"premium":  true,
	}
	for _, serviceType := range c.DefaultServiceTypes {
		if !validServiceTypes[serviceType] {
			return fmt.Errorf("invalid service type: %s", serviceType)
		}
	}

	return nil
}

// ToMap converts configuration to map for JSON serialization
func (c *Config) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"base_url":                 c.BaseURL,
		"calculate_rates_endpoint": c.CalculateRatesEndpoint,
		"login_url":                c.LoginURL,
		"username":                 c.Username,
		"password":                 "***HIDDEN***", // Don't expose password in serialization
		"signin_type":              c.SigninType,
		"refresh_buffer_minutes":   c.RefreshBufferMinutes,
		"timeout_ms":               c.TimeoutMs,
		"retry_count":              c.RetryCount,
		"cache_ttl_minutes":        c.CacheTTLMinutes,
		"default_currency":         c.DefaultCurrency,
		"supported_currencies":     c.SupportedCurrencies,
		"max_packages_per_request": c.MaxPackagesPerRequest,
		"default_service_types":    c.DefaultServiceTypes,
		"enable_all_services":      c.EnableAllServices,
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

// IsCurrencySupported checks if a currency is supported
func (c *Config) IsCurrencySupported(currency string) bool {
	for _, supportedCurrency := range c.SupportedCurrencies {
		if supportedCurrency == currency {
			return true
		}
	}
	return false
}

// IsServiceTypeSupported checks if a service type is supported
func (c *Config) IsServiceTypeSupported(serviceType string) bool {
	if c.EnableAllServices {
		validServiceTypes := map[string]bool{
			"standard": true,
			"express":  true,
			"premium":  true,
		}
		return validServiceTypes[serviceType]
	}

	for _, supportedType := range c.DefaultServiceTypes {
		if supportedType == serviceType {
			return true
		}
	}
	return false
}
