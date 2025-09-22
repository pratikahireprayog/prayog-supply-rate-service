package dynamic

import (
	"context"
	"fmt"
	"time"

	constants "github.com/prayog/prayog-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-rate-service/internal/shared/interfaces/v1"
	models "github.com/prayog/prayog-rate-service/internal/shared/models/v1"
)

// BaseDynamicProvider provides a base implementation for dynamic rate providers
type BaseDynamicProvider struct {
	partner          *models.Partner
	config           map[string]interface{}
	logger           interfaces.Logger
	metrics          interfaces.MetricsCollector
	httpClient       interfaces.HTTPClient
	lastResponseTime time.Duration
	isInitialized    bool
}

// NewBaseDynamicProvider creates a new base dynamic provider
func NewBaseDynamicProvider(
	partner *models.Partner,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	httpClient interfaces.HTTPClient,
) *BaseDynamicProvider {
	return &BaseDynamicProvider{
		partner:    partner,
		logger:     logger,
		metrics:    metrics,
		httpClient: httpClient,
	}
}

// GetProviderType returns the provider type
func (p *BaseDynamicProvider) GetProviderType() dtos.ProviderType {
	return dtos.ProviderTypeDynamic
}

// GetProviderName returns the provider name
func (p *BaseDynamicProvider) GetProviderName() string {
	return p.partner.Name
}

// Initialize initializes the provider with configuration
func (p *BaseDynamicProvider) Initialize(config map[string]interface{}) error {
	if p.partner == nil {
		return fmt.Errorf("partner cannot be nil")
	}

	if !p.partner.IsAPIBased() {
		return fmt.Errorf("partner %s is not configured for API-based operations", p.partner.ID)
	}

	p.config = config

	// Validate required configuration
	if err := p.validateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	p.isInitialized = true

	p.logger.Info("Dynamic provider initialized",
		"partner_id", p.partner.ID,
		"partner_name", p.partner.Name,
		"api_endpoint", p.partner.GetAPIEndpoint())

	return nil
}

// IsHealthy performs a health check
func (p *BaseDynamicProvider) IsHealthy(ctx context.Context) error {
	if !p.isInitialized {
		return constants.ErrProviderInitFail
	}

	if !p.partner.IsHealthy() {
		return constants.ErrPartnerUnhealthy
	}

	// Perform API health check
	return p.performAPIHealthCheck(ctx)
}

// GetConfiguration returns the provider configuration
func (p *BaseDynamicProvider) GetConfiguration() map[string]interface{} {
	return p.config
}

// Close gracefully shuts down the provider
func (p *BaseDynamicProvider) Close() error {
	p.isInitialized = false
	p.logger.Info("Dynamic provider closed", "partner_id", p.partner.ID)
	return nil
}

// GetRates is the main interface method - must be implemented by concrete providers
func (p *BaseDynamicProvider) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	return nil, constants.ErrNotImplemented
}

// GetRealTimeQuote gets a real-time quote from the partner API
func (p *BaseDynamicProvider) GetRealTimeQuote(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	// This is the same as GetRates for dynamic providers
	return p.GetRates(ctx, request)
}

// ValidateAPICredentials validates the API credentials
func (p *BaseDynamicProvider) ValidateAPICredentials(ctx context.Context) error {
	if !p.isInitialized {
		return constants.ErrProviderInitFail
	}

	// Create a simple health check request
	return p.performAPIHealthCheck(ctx)
}

// GetAPILimits returns the API rate limits and usage
func (p *BaseDynamicProvider) GetAPILimits(ctx context.Context) (*dtos.APILimits, error) {
	// This should be implemented by concrete providers
	// Return default limits for now
	return &dtos.APILimits{
		Limit:     1000,
		Remaining: 950,
		Reset:     time.Now().Add(time.Hour),
		Window:    "1h",
	}, nil
}

// GetLastResponseTime returns the last API response time
func (p *BaseDynamicProvider) GetLastResponseTime() time.Duration {
	return p.lastResponseTime
}

// Protected helper methods for concrete implementations

// ValidateConfiguration validates the provider configuration
func (p *BaseDynamicProvider) validateConfiguration() error {
	// Check API endpoint
	if p.partner.APIEndpoint == nil || *p.partner.APIEndpoint == "" {
		return fmt.Errorf("API endpoint is required for dynamic provider")
	}

	// Check API key if required
	if p.requiresAPIKey() && (p.partner.APIKey == nil || *p.partner.APIKey == "") {
		return fmt.Errorf("API key is required for partner %s", p.partner.Name)
	}

	// Validate timeout
	if p.partner.TimeoutMs <= 0 {
		p.partner.TimeoutMs = constants.DefaultPartnerTimeout
	}

	return nil
}

// RequiresAPIKey determines if the provider requires an API key
func (p *BaseDynamicProvider) requiresAPIKey() bool {
	// Most dynamic providers require API keys
	// Concrete implementations can override this
	return true
}

// PerformAPIHealthCheck performs a health check against the partner API
func (p *BaseDynamicProvider) performAPIHealthCheck(ctx context.Context) error {
	if p.httpClient == nil {
		return fmt.Errorf("HTTP client not configured")
	}

	endpoint := p.partner.GetAPIEndpoint()
	if endpoint == "" {
		return fmt.Errorf("API endpoint not configured")
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(p.partner.TimeoutMs)*time.Millisecond)
	defer cancel()

	// Perform health check (this should be implemented by concrete providers)
	startTime := time.Now()

	// For base implementation, just validate the endpoint is reachable
	err := p.pingEndpoint(timeoutCtx, endpoint)

	p.lastResponseTime = time.Since(startTime)

	// Record metrics
	status := "success"
	if err != nil {
		status = "failure"
	}

	p.metrics.IncrementCounter("provider_health_check", map[string]string{
		"partner_id": p.partner.ID.String(),
		"status":     status,
	})
	p.metrics.RecordTimer("provider_health_check_time", p.lastResponseTime, map[string]string{
		"partner_id": p.partner.ID.String(),
	})

	return err
}

// PingEndpoint performs a basic connectivity check to the API endpoint
func (p *BaseDynamicProvider) pingEndpoint(ctx context.Context, endpoint string) error {
	// This is a simplified implementation
	// In a real implementation, you would make an actual HTTP request
	// to a health check endpoint or perform a basic connectivity test

	if endpoint == "" {
		return fmt.Errorf("endpoint is empty")
	}

	// For now, just return success if endpoint is configured
	// Concrete implementations should override this with actual API calls
	return nil
}

// BuildStandardResponse builds a standard rate calculation response
func (p *BaseDynamicProvider) buildStandardResponse(request *dtos.RateCalculationRequest, quotes []dtos.RateQuote) *dtos.RateCalculationResponse {
	var bestQuote *dtos.RateQuote
	if len(quotes) > 0 {
		// Simple best quote selection (lowest price)
		best := quotes[0]
		for _, quote := range quotes {
			if quote.TotalPrice < best.TotalPrice {
				best = quote
			}
		}
		best.IsRecommended = true
		bestQuote = &best
	}

	status := "success"
	message := constants.MessageRateCalculated
	if len(quotes) == 0 {
		status = "no_rates"
		message = constants.MessageRateNotFound
	}

	return &dtos.RateCalculationResponse{
		RequestID:   request.RequestID,
		Status:      status,
		Message:     message,
		Quotes:      quotes,
		BestQuote:   bestQuote,
		TotalQuotes: len(quotes),
		CacheHit:    false,
		Timestamp:   time.Now(),
	}
}

// CreateQuote creates a quote with standard fields populated
func (p *BaseDynamicProvider) createQuote(request *dtos.RateCalculationRequest, basePrice float64, estimatedDays int) dtos.RateQuote {
	quoteID := fmt.Sprintf("%s-%d", p.partner.Code, time.Now().UnixNano())

	// Calculate additional charges (these should be implemented by concrete providers)
	fuelSurcharge := basePrice * 0.05 // 5% fuel surcharge
	handlingCharge := 50.0            // Fixed handling charge
	taxAmount := basePrice * 0.18     // 18% tax

	totalPrice := basePrice + fuelSurcharge + handlingCharge + taxAmount

	return dtos.RateQuote{
		QuoteID:      quoteID,
		PartnerID:    p.partner.ID.String(),
		PartnerName:  p.partner.Name,
		ProviderType: dtos.ProviderTypeDynamic,

		BasePrice:  basePrice,
		TotalPrice: totalPrice,
		Currency:   request.Currency,

		PriceBreakdown: dtos.PriceBreakdown{
			BasePrice:      basePrice,
			FuelSurcharge:  fuelSurcharge,
			HandlingCharge: handlingCharge,
			TaxAmount:      taxAmount,
			TotalPrice:     totalPrice,
		},

		ServiceType:    request.ServiceType,
		ServiceLevel:   "standard",
		EstimatedDays:  estimatedDays,
		EstimatedHours: estimatedDays * 24,

		ValidUntil: time.Now().Add(24 * time.Hour), // 24 hour validity
		Confidence: 0.85,                           // Default confidence

		InsuranceAvailable: true,
		TrackingAvailable:  true,
		SignatureAvailable: true,

		Source:         "dynamic",
		ResponseTimeMs: p.lastResponseTime.Milliseconds(),
	}
}

// LogAPIRequest logs API requests for debugging and monitoring
func (p *BaseDynamicProvider) logAPIRequest(method, url string, requestBody interface{}, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	p.logger.Info("API request completed",
		"partner_id", p.partner.ID,
		"method", method,
		"url", url,
		"status", status,
		"duration_ms", duration.Milliseconds(),
		"error", err)

	p.metrics.IncrementCounter("api_request", map[string]string{
		"partner_id": p.partner.ID.String(),
		"method":     method,
		"status":     status,
	})
	p.metrics.RecordTimer("api_request_time", duration, map[string]string{
		"partner_id": p.partner.ID.String(),
		"method":     method,
	})
}
