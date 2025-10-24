package india_post_domestic

import (
    "fmt"
    "os"
)

// Config holds India Post Domestic specific configuration
type Config struct {
    BaseURL         string `json:"base_url"`
    AccessTokenURL  string `json:"access_token_url"`
    RefreshTokenURL string `json:"refresh_token_url"`
    Username        string `json:"username"`
    Password        string `json:"password"`
    TimeoutMs       int    `json:"timeout_ms"`
    RetryCount      int    `json:"retry_count"`
    Enabled         bool   `json:"enabled"`
}

func NewDefaultConfig() *Config {
    return &Config{
        BaseURL:         getEnv("INDIA_POST_DOMESTIC_BASE_URL", ""),
        AccessTokenURL:  getEnv("INDIA_POST_DOMESTIC_ACCESS_TOKEN_URL", ""),
        RefreshTokenURL: getEnv("INDIA_POST_DOMESTIC_REFRESH_TOKEN_URL", ""),
        Username:        getEnv("INDIA_POST_DOMESTIC_USERNAME", ""),
        Password:        getEnv("INDIA_POST_DOMESTIC_PASSWORD", ""),
        TimeoutMs:       30000,
        RetryCount:      2,
        Enabled:         true,
    }
}

func getEnv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}

func (c *Config) LoadFromMap(m map[string]interface{}) error {
    if m == nil {
        return c.Validate()
    }
    if v, ok := m["base_url"].(string); ok && v != "" { c.BaseURL = v }
    if v, ok := m["access_token_url"].(string); ok && v != "" { c.AccessTokenURL = v }
    if v, ok := m["refresh_token_url"].(string); ok && v != "" { c.RefreshTokenURL = v }
    if v, ok := m["username"].(string); ok { c.Username = v }
    if v, ok := m["password"].(string); ok { c.Password = v }
    if v, ok := m["timeout_ms"].(int); ok && v > 0 { c.TimeoutMs = v }
    if v, ok := m["retry_count"].(int); ok && v >= 0 { c.RetryCount = v }
    if v, ok := m["enabled"].(bool); ok { c.Enabled = v }
    return c.Validate()
}

func (c *Config) Validate() error {
    if c.BaseURL == "" { return fmt.Errorf("base_url is required") }
    if c.AccessTokenURL == "" { return fmt.Errorf("access_token_url is required") }
    if c.RefreshTokenURL == "" { return fmt.Errorf("refresh_token_url is required") }
    if c.TimeoutMs <= 0 { return fmt.Errorf("timeout_ms must be positive") }
    if c.RetryCount < 0 { return fmt.Errorf("retry_count cannot be negative") }
    return nil
}


