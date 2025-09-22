package v1

import (
	"context"
	"time"

	dtos "github.com/prayog/prayog-rate-service/internal/shared/dtos/v1"
	models "github.com/prayog/prayog-rate-service/internal/shared/models/v1"
)

// RateProvider defines the interface for all rate providers
type RateProvider interface {
	// GetRates retrieves rates based on the given criteria
	GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error)

	// GetProviderType returns the type of this provider
	GetProviderType() dtos.ProviderType

	// GetProviderName returns the name of this provider
	GetProviderName() string

	// IsHealthy checks if the provider is healthy and operational
	IsHealthy(ctx context.Context) error

	// GetConfiguration returns the provider's configuration
	GetConfiguration() map[string]interface{}

	// Initialize initializes the provider with given configuration
	Initialize(config map[string]interface{}) error

	// Close gracefully shuts down the provider
	Close() error
}

// StaticRateProvider defines additional methods for static rate providers
type StaticRateProvider interface {
	RateProvider

	// RefreshRates refreshes the static rate cache
	RefreshRates(ctx context.Context) error

	// GetCacheInfo returns information about the cache status
	GetCacheInfo() *dtos.CacheInfo

	// ValidateRateData validates the static rate data
	ValidateRateData(rates []*models.Rate) error
}

// DynamicRateProvider defines additional methods for dynamic rate providers
type DynamicRateProvider interface {
	RateProvider

	// GetRealTimeQuote gets a real-time quote from the partner API
	GetRealTimeQuote(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error)

	// ValidateAPICredentials validates the API credentials
	ValidateAPICredentials(ctx context.Context) error

	// GetAPILimits returns the API rate limits and usage
	GetAPILimits(ctx context.Context) (*dtos.APILimits, error)

	// GetLastResponseTime returns the last API response time
	GetLastResponseTime() time.Duration
}

// RateProviderFactory defines the interface for the factory pattern
type RateProviderFactory interface {
	// CreateProvider creates a provider instance by type and partner ID
	CreateProvider(providerType dtos.ProviderType, partnerID string) (RateProvider, error)

	// RegisterProvider registers a new provider implementation
	RegisterProvider(providerType dtos.ProviderType, creator ProviderCreator) error

	// GetRegisteredProviders returns all registered provider types
	GetRegisteredProviders() []dtos.ProviderType

	// IsProviderRegistered checks if a provider type is registered
	IsProviderRegistered(providerType dtos.ProviderType) bool

	// GetProviderInstance gets an existing provider instance
	GetProviderInstance(partnerID string) (RateProvider, error)

	// RemoveProviderInstance removes a provider instance
	RemoveProviderInstance(partnerID string) error

	// GetAllInstances returns all active provider instances
	GetAllInstances() map[string]RateProvider

	// HealthCheckAll performs health check on all provider instances
	HealthCheckAll(ctx context.Context) map[string]error
}

// ProviderCreator defines the function signature for creating provider instances
type ProviderCreator func(partner *models.Partner) (RateProvider, error)

// RateService defines the main service interface for rate calculations
type RateService interface {
	// CalculateRates calculates rates from all available providers
	CalculateRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error)

	// CalculateRatesByProvider calculates rates from a specific provider
	CalculateRatesByProvider(ctx context.Context, request *dtos.RateCalculationRequest, providerID string) (*dtos.RateCalculationResponse, error)

	// GetBestRate returns the best rate from all available providers
	GetBestRate(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateQuote, error)

	// CompareRates compares rates from multiple providers
	CompareRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateComparisonResponse, error)

	// GetProviderHealth returns health status of all providers
	GetProviderHealth(ctx context.Context) (*dtos.ProviderHealthResponse, error)

	// RefreshProviders refreshes all provider configurations
	RefreshProviders(ctx context.Context) error
}

// PartnerRepository defines the interface for partner data access
type PartnerRepository interface {
	// GetByID retrieves a partner by ID
	GetByID(ctx context.Context, id string) (*models.Partner, error)

	// GetByType retrieves partners by type
	GetByType(ctx context.Context, partnerType models.PartnerType) ([]*models.Partner, error)

	// GetActive retrieves all active partners
	GetActive(ctx context.Context) ([]*models.Partner, error)

	// Create creates a new partner
	Create(ctx context.Context, partner *models.Partner) error

	// Update updates an existing partner
	Update(ctx context.Context, partner *models.Partner) error

	// Delete deletes a partner
	Delete(ctx context.Context, id string) error

	// UpdateHealthStatus updates the health status of a partner
	UpdateHealthStatus(ctx context.Context, id string, status string) error
}

// RateRepository defines the interface for rate data access
type RateRepository interface {
	// GetByPartner retrieves rates by partner ID
	GetByPartner(ctx context.Context, partnerID string) ([]*models.Rate, error)

	// GetByCriteria retrieves rates matching the given criteria
	GetByCriteria(ctx context.Context, criteria *dtos.RateCriteria) ([]*models.Rate, error)

	// Create creates a new rate
	Create(ctx context.Context, rate *models.Rate) error

	// Update updates an existing rate
	Update(ctx context.Context, rate *models.Rate) error

	// Delete deletes a rate
	Delete(ctx context.Context, id string) error

	// BulkCreate creates multiple rates in a single transaction
	BulkCreate(ctx context.Context, rates []*models.Rate) error

	// GetValidRates retrieves rates valid for the given date
	GetValidRates(ctx context.Context, date time.Time) ([]*models.Rate, error)
}

// CacheManager defines the interface for caching operations
type CacheManager interface {
	// Set stores a value in the cache
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Get retrieves a value from the cache
	Get(ctx context.Context, key string, dest interface{}) error

	// Delete removes a key from the cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// Clear clears all cache entries
	Clear(ctx context.Context) error

	// GetStats returns cache statistics
	GetStats(ctx context.Context) (*dtos.CacheStats, error)
}

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	// IncrementCounter increments a counter metric
	IncrementCounter(name string, tags map[string]string)

	// RecordTimer records a timer metric
	RecordTimer(name string, duration time.Duration, tags map[string]string)

	// RecordGauge records a gauge metric
	RecordGauge(name string, value float64, tags map[string]string)

	// RecordHistogram records a histogram metric
	RecordHistogram(name string, value float64, tags map[string]string)
}

// Logger defines the interface for logging operations
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...interface{})

	// Info logs an info message
	Info(msg string, fields ...interface{})

	// Warn logs a warning message
	Warn(msg string, fields ...interface{})

	// Error logs an error message
	Error(msg string, fields ...interface{})

	// Fatal logs a fatal message and exits
	Fatal(msg string, fields ...interface{})

	// With returns a logger with additional fields
	With(fields ...interface{}) Logger
}

// HTTPClient defines the interface for HTTP operations
type HTTPClient interface {
	// Get performs a GET request
	Get(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error)

	// Post performs a POST request
	Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*HTTPResponse, error)

	// Put performs a PUT request
	Put(ctx context.Context, url string, body interface{}, headers map[string]string) (*HTTPResponse, error)

	// Delete performs a DELETE request
	Delete(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error)

	// SetTimeout sets the default timeout for requests
	SetTimeout(timeout time.Duration)

	// SetRetryCount sets the number of retry attempts
	SetRetryCount(count int)
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
	Duration   time.Duration
}
