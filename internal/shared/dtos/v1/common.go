package v1

import (
	"time"

	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
)

// ProviderType is an alias for models.PartnerType for consistency
type ProviderType = models.PartnerType

// Provider type constants - using the same values as models
const (
	ProviderTypePreDefined = models.PartnerTypePreDefined
	ProviderTypeRealTime   = models.PartnerTypeRealTime
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool              `json:"success"`
	Message   string            `json:"message"`
	Data      interface{}       `json:"data,omitempty"`
	Error     *ErrorDetail      `json:"error,omitempty"`
	Meta      *ResponseMetadata `json:"meta,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ResponseMetadata represents metadata for API responses
type ResponseMetadata struct {
	RequestID    string      `json:"request_id,omitempty"`
	ResponseTime int64       `json:"response_time_ms"`
	Version      string      `json:"version"`
	Pagination   *Pagination `json:"pagination,omitempty"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// CacheInfo represents cache information
type CacheInfo struct {
	Enabled      bool      `json:"enabled"`
	LastRefresh  time.Time `json:"last_refresh"`
	NextRefresh  time.Time `json:"next_refresh"`
	TotalEntries int       `json:"total_entries"`
	CacheSize    int64     `json:"cache_size_bytes"`
	HitRate      float64   `json:"hit_rate"`
	MissRate     float64   `json:"miss_rate"`
}

// APILimits represents API rate limits
type APILimits struct {
	Limit      int       `json:"limit"`
	Remaining  int       `json:"remaining"`
	Reset      time.Time `json:"reset"`
	Window     string    `json:"window"`
	RetryAfter *int      `json:"retry_after,omitempty"`
}

// RateCriteria represents criteria for rate searching
type RateCriteria struct {
	PartnerIDs   []string               `json:"partner_ids,omitempty"`
	ServiceTypes []string               `json:"service_types,omitempty"`
	OriginCity   string                 `json:"origin_city,omitempty"`
	DestCity     string                 `json:"dest_city,omitempty"`
	WeightMin    *float64               `json:"weight_min,omitempty"`
	WeightMax    *float64               `json:"weight_max,omitempty"`
	DistanceMin  *float64               `json:"distance_min,omitempty"`
	DistanceMax  *float64               `json:"distance_max,omitempty"`
	ValidDate    *time.Time             `json:"valid_date,omitempty"`
	IsActive     *bool                  `json:"is_active,omitempty"`
	Currency     string                 `json:"currency,omitempty"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	OrderBy      string                 `json:"order_by,omitempty"`
	OrderDir     string                 `json:"order_dir,omitempty"`
	Limit        int                    `json:"limit,omitempty"`
	Offset       int                    `json:"offset,omitempty"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64         `json:"hits"`
	Misses      int64         `json:"misses"`
	HitRate     float64       `json:"hit_rate"`
	MissRate    float64       `json:"miss_rate"`
	TotalKeys   int           `json:"total_keys"`
	TotalMemory int64         `json:"total_memory_bytes"`
	Uptime      time.Duration `json:"uptime"`
	LastFlush   *time.Time    `json:"last_flush,omitempty"`
}

// ProviderHealthResponse represents health status of all providers
type ProviderHealthResponse struct {
	Status             string                 `json:"status"`
	TotalProviders     int                    `json:"total_providers"`
	HealthyProviders   int                    `json:"healthy_providers"`
	UnhealthyProviders int                    `json:"unhealthy_providers"`
	Providers          []ProviderHealthStatus `json:"providers"`
	CheckedAt          time.Time              `json:"checked_at"`
	ResponseTime       int64                  `json:"response_time_ms"`
}

// ProviderHealthStatus represents health status of a single provider
type ProviderHealthStatus struct {
	PartnerID    string                 `json:"partner_id"`
	PartnerName  string                 `json:"partner_name"`
	ProviderType ProviderType           `json:"provider_type"`
	Status       string                 `json:"status"`
	IsActive     bool                   `json:"is_active"`
	LastChecked  time.Time              `json:"last_checked"`
	ResponseTime time.Duration          `json:"response_time"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// PartnerRequest represents request for partner operations
type PartnerRequest struct {
	Name        string                 `json:"name" validate:"required,min=2,max=100"`
	Code        string                 `json:"code" validate:"required,min=2,max=20,alphanum"`
	Type        ProviderType           `json:"type" validate:"required,oneof=pre_defined real_time"`
	IsActive    bool                   `json:"is_active"`
	Priority    int                    `json:"priority" validate:"required,min=1,max=10"`
	Config      map[string]interface{} `json:"config,omitempty"`
	APIEndpoint *string                `json:"api_endpoint,omitempty"`
	APIKey      *string                `json:"api_key,omitempty"`
	TimeoutMs   int                    `json:"timeout_ms" validate:"required,min=1000,max=30000"`
	RetryCount  int                    `json:"retry_count" validate:"required,min=0,max=5"`
	Description string                 `json:"description,omitempty"`
}

// PartnerResponse represents response for partner operations
type PartnerResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Code         string                 `json:"code"`
	Type         ProviderType           `json:"type"`
	IsActive     bool                   `json:"is_active"`
	Priority     int                    `json:"priority"`
	Config       map[string]interface{} `json:"config,omitempty"`
	APIEndpoint  *string                `json:"api_endpoint,omitempty"`
	TimeoutMs    int                    `json:"timeout_ms"`
	RetryCount   int                    `json:"retry_count"`
	HealthStatus string                 `json:"health_status"`
	LastChecked  *time.Time             `json:"last_checked,omitempty"`
	Description  string                 `json:"description,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// RateRequest represents request for rate operations
type RateRequest struct {
	PartnerID   string     `json:"partner_id" validate:"required"`
	ServiceType string     `json:"service_type" validate:"required,oneof=standard premium express same_day economy"`
	OriginCity  string     `json:"origin_city" validate:"required,min=2,max=100"`
	DestCity    string     `json:"dest_city" validate:"required,min=2,max=100"`
	WeightMin   float64    `json:"weight_min" validate:"required,min=0"`
	WeightMax   float64    `json:"weight_max" validate:"required,gtfield=WeightMin"`
	DistanceMin float64    `json:"distance_min" validate:"required,min=0"`
	DistanceMax float64    `json:"distance_max" validate:"required,gtfield=DistanceMin"`
	BasePrice   float64    `json:"base_price" validate:"required,min=0"`
	PricePerKm  float64    `json:"price_per_km" validate:"required,min=0"`
	PricePerKg  float64    `json:"price_per_kg" validate:"required,min=0"`
	Currency    string     `json:"currency" validate:"required,oneof=INR USD EUR"`
	ValidFrom   time.Time  `json:"valid_from" validate:"required"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
	IsActive    bool       `json:"is_active"`
}

// RateResponse represents response for rate operations
type RateResponse struct {
	ID          string     `json:"id"`
	PartnerID   string     `json:"partner_id"`
	ServiceType string     `json:"service_type"`
	OriginCity  string     `json:"origin_city"`
	DestCity    string     `json:"dest_city"`
	WeightMin   float64    `json:"weight_min"`
	WeightMax   float64    `json:"weight_max"`
	DistanceMin float64    `json:"distance_min"`
	DistanceMax float64    `json:"distance_max"`
	BasePrice   float64    `json:"base_price"`
	PricePerKm  float64    `json:"price_per_km"`
	PricePerKg  float64    `json:"price_per_kg"`
	Currency    string     `json:"currency"`
	ValidFrom   time.Time  `json:"valid_from"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// HealthCheckRequest represents request for health check
type HealthCheckRequest struct {
	Deep      bool     `json:"deep,omitempty"`
	Services  []string `json:"services,omitempty"`
	TimeoutMs int      `json:"timeout_ms,omitempty"`
}

// HealthCheckResponse represents response for health check
type HealthCheckResponse struct {
	Status    string                   `json:"status"`
	Message   string                   `json:"message"`
	Services  map[string]ServiceHealth `json:"services"`
	Timestamp time.Time                `json:"timestamp"`
	Uptime    time.Duration            `json:"uptime"`
	Version   string                   `json:"version"`
}

// ServiceHealth represents health status of a service component
type ServiceHealth struct {
	Status       string                 `json:"status"`
	ResponseTime time.Duration          `json:"response_time"`
	LastChecked  time.Time              `json:"last_checked"`
	Error        string                 `json:"error,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}
