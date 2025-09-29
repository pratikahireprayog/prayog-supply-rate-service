package fedex

// Config holds FedEx-specific configuration
type Config struct {
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
	SecretKey   string `json:"secret_key"`
	AccountID   string `json:"account_id"`
	TimeoutMs   int    `json:"timeout_ms"`
	RetryCount  int    `json:"retry_count"`
	Environment string `json:"environment"` // "test" or "production"
}

// NewDefaultConfig creates a new default configuration for FedEx
func NewDefaultConfig() *Config {
	return &Config{
		BaseURL:     "https://apis-sandbox.fedex.com", // Sandbox URL
		TimeoutMs:   30000,                            // 30 seconds
		RetryCount:  2,
		Environment: "test",
	}
}

// LoadFromMap loads configuration from a map
func (c *Config) LoadFromMap(config map[string]interface{}) error {
	// Implementation placeholder
	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Implementation placeholder
	return nil
}
