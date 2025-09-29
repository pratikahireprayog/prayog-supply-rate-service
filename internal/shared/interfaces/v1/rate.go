package v1

import (
	"context"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
)

// RateImplementation defines the interface for all rate implementations
type RateImplementation interface {
	// GetRates retrieves rates based on the given criteria
	GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error)

	// GetImplementationType returns the type of this implementation
	GetImplementationType() dtos.ProviderType

	// GetImplementationName returns the name of this implementation
	GetImplementationName() string

	// IsHealthy checks if the implementation is healthy and operational
	IsHealthy(ctx context.Context) error

	// GetConfiguration returns the implementation's configuration
	GetConfiguration() map[string]interface{}

	// Initialize initializes the implementation with given configuration
	Initialize(config map[string]interface{}) error

	// Close gracefully shuts down the implementation
	Close() error
}

// PreDefinedRate defines additional methods for pre-defined rates
type PreDefinedRate interface {
	RateImplementation

	// RefreshRates refreshes the pre-defined rate cache
	RefreshRates(ctx context.Context) error

	// GetCacheInfo returns information about the cache status
	GetCacheInfo() *dtos.CacheInfo

	// ValidateRateData validates the pre-defined rate data
	ValidateRateData(rates []*models.Rate) error
}

// RealTimeRate defines additional methods for real-time rates
type RealTimeRate interface {
	RateImplementation

	// GetRealTimeQuote gets a real-time quote from the partner API
	GetRealTimeQuote(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error)

	// ValidateAPICredentials validates the API credentials
	ValidateAPICredentials(ctx context.Context) error

	// GetAPILimits returns the API rate limits and usage
	GetAPILimits(ctx context.Context) (*dtos.APILimits, error)

	// GetLastResponseTime returns the last API response time
	GetLastResponseTime() time.Duration
}

// RateFactory defines the interface for the factory pattern
type RateFactory interface {
	// CreateImplementation creates an implementation instance by type and partner ID
	CreateImplementation(implementationType dtos.ProviderType, partnerID string) (RateImplementation, error)

	// RegisterImplementation registers a new implementation creator
	RegisterImplementation(implementationType dtos.ProviderType, creator ImplementationCreator) error

	// GetRegisteredImplementations returns all registered implementation types
	GetRegisteredImplementations() []dtos.ProviderType

	// IsImplementationRegistered checks if an implementation type is registered
	IsImplementationRegistered(implementationType dtos.ProviderType) bool

	// GetImplementationInstance gets an existing implementation instance
	GetImplementationInstance(partnerID string) (RateImplementation, error)

	// RemoveImplementationInstance removes an implementation instance
	RemoveImplementationInstance(partnerID string) error

	// GetAllInstances returns all active implementation instances
	GetAllInstances() map[string]RateImplementation

	// HealthCheckAll performs health check on all implementation instances
	HealthCheckAll(ctx context.Context) map[string]error
}

// ImplementationCreator defines the function signature for creating implementation instances
type ImplementationCreator func(partner *models.Partner) (RateImplementation, error)

// RateService defines the main service interface for rate operations
type RateService interface {
	// GetQuotes retrieves quotes from multiple partners
	GetQuotes(ctx context.Context, request *dtos.QuoteRequest, requestID string) (*dtos.QuoteResponse, error)

	// GetImplementationHealth returns health status of all implementations
	GetImplementationHealth(ctx context.Context) (*dtos.ProviderHealthResponse, error)

	// RefreshImplementations refreshes all implementation configurations
	RefreshImplementations(ctx context.Context) error
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
