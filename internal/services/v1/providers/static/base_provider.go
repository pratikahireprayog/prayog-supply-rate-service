package static

import (
	"context"
	"fmt"
	"time"

	constants "github.com/prayog/prayog-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-rate-service/internal/shared/interfaces/v1"
	models "github.com/prayog/prayog-rate-service/internal/shared/models/v1"
)

// BaseStaticProvider provides a base implementation for static rate providers
type BaseStaticProvider struct {
	partner       *models.Partner
	config        map[string]interface{}
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	rateRepo      interfaces.RateRepository
	cacheManager  interfaces.CacheManager
	isInitialized bool

	// Cache information
	cacheInfo *dtos.CacheInfo
}

// NewBaseStaticProvider creates a new base static provider
func NewBaseStaticProvider(
	partner *models.Partner,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	rateRepo interfaces.RateRepository,
	cacheManager interfaces.CacheManager,
) *BaseStaticProvider {
	return &BaseStaticProvider{
		partner:      partner,
		logger:       logger,
		metrics:      metrics,
		rateRepo:     rateRepo,
		cacheManager: cacheManager,
		cacheInfo: &dtos.CacheInfo{
			Enabled:      true,
			LastRefresh:  time.Now(),
			NextRefresh:  time.Now().Add(time.Hour),
			TotalEntries: 0,
			HitRate:      0.0,
			MissRate:     0.0,
		},
	}
}

// GetProviderType returns the provider type
func (p *BaseStaticProvider) GetProviderType() dtos.ProviderType {
	return dtos.ProviderTypeStatic
}

// GetProviderName returns the provider name
func (p *BaseStaticProvider) GetProviderName() string {
	return p.partner.Name
}

// Initialize initializes the provider with configuration
func (p *BaseStaticProvider) Initialize(config map[string]interface{}) error {
	if p.partner == nil {
		return fmt.Errorf("partner cannot be nil")
	}

	p.config = config

	// Load initial rate data
	if err := p.loadRateData(context.Background()); err != nil {
		return fmt.Errorf("failed to load initial rate data: %w", err)
	}

	p.isInitialized = true

	p.logger.Info("Static provider initialized",
		"partner_id", p.partner.ID,
		"partner_name", p.partner.Name,
		"total_rates", p.cacheInfo.TotalEntries)

	return nil
}

// IsHealthy performs a health check
func (p *BaseStaticProvider) IsHealthy(ctx context.Context) error {
	if !p.isInitialized {
		return constants.ErrProviderInitFail
	}

	if !p.partner.IsActive {
		return constants.ErrPartnerInactive
	}

	// Check if rate repository is accessible
	if err := p.performRepositoryHealthCheck(ctx); err != nil {
		return fmt.Errorf("repository health check failed: %w", err)
	}

	return nil
}

// GetConfiguration returns the provider configuration
func (p *BaseStaticProvider) GetConfiguration() map[string]interface{} {
	return p.config
}

// Close gracefully shuts down the provider
func (p *BaseStaticProvider) Close() error {
	p.isInitialized = false
	p.logger.Info("Static provider closed", "partner_id", p.partner.ID)
	return nil
}

// GetRates calculates rates from static data
func (p *BaseStaticProvider) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	startTime := time.Now()

	if !p.isInitialized {
		return nil, constants.ErrProviderInitFail
	}

	// Build rate criteria from request
	criteria := p.buildRateCriteria(request)

	// Get matching rates from repository
	rates, err := p.rateRepo.GetByCriteria(ctx, criteria)
	if err != nil {
		p.metrics.IncrementCounter("static_provider_error", map[string]string{
			"partner_id": p.partner.ID.String(),
			"error_type": "repository_error",
		})
		return nil, fmt.Errorf("failed to get rates from repository: %w", err)
	}

	// Filter and calculate quotes
	quotes := p.calculateQuotesFromRates(request, rates)

	// Build response
	response := p.buildStandardResponse(request, quotes)
	response.ResponseTime = time.Since(startTime).Milliseconds()

	// Update metrics
	p.metrics.IncrementCounter("static_provider_calculation", map[string]string{
		"partner_id":   p.partner.ID.String(),
		"total_quotes": fmt.Sprintf("%d", len(quotes)),
	})
	p.metrics.RecordTimer("static_provider_calculation_time", time.Since(startTime), map[string]string{
		"partner_id": p.partner.ID.String(),
	})

	p.logger.Debug("Static rate calculation completed",
		"partner_id", p.partner.ID,
		"request_id", request.RequestID,
		"matching_rates", len(rates),
		"generated_quotes", len(quotes),
		"duration_ms", response.ResponseTime)

	return response, nil
}

// RefreshRates refreshes the static rate cache
func (p *BaseStaticProvider) RefreshRates(ctx context.Context) error {
	p.logger.Info("Starting rate refresh", "partner_id", p.partner.ID)

	if err := p.loadRateData(ctx); err != nil {
		p.metrics.IncrementCounter("static_provider_refresh_failed", map[string]string{
			"partner_id": p.partner.ID.String(),
		})
		return fmt.Errorf("failed to refresh rates: %w", err)
	}

	p.cacheInfo.LastRefresh = time.Now()
	p.cacheInfo.NextRefresh = time.Now().Add(time.Hour)

	p.metrics.IncrementCounter("static_provider_refresh_success", map[string]string{
		"partner_id": p.partner.ID.String(),
	})

	p.logger.Info("Rate refresh completed",
		"partner_id", p.partner.ID,
		"total_rates", p.cacheInfo.TotalEntries)

	return nil
}

// GetCacheInfo returns information about the cache status
func (p *BaseStaticProvider) GetCacheInfo() *dtos.CacheInfo {
	return p.cacheInfo
}

// ValidateRateData validates the static rate data
func (p *BaseStaticProvider) ValidateRateData(rates []*models.Rate) error {
	if len(rates) == 0 {
		return fmt.Errorf("no rates provided for validation")
	}

	var validationErrors []string

	for i, rate := range rates {
		// Validate partner ID matches
		if rate.PartnerID != p.partner.ID.String() {
			validationErrors = append(validationErrors,
				fmt.Sprintf("rate %d: partner ID mismatch, expected %s, got %s",
					i, p.partner.ID, rate.PartnerID))
		}

		// Validate price values
		if rate.BasePrice < 0 {
			validationErrors = append(validationErrors,
				fmt.Sprintf("rate %d: base price cannot be negative", i))
		}

		if rate.PricePerKm < 0 {
			validationErrors = append(validationErrors,
				fmt.Sprintf("rate %d: price per km cannot be negative", i))
		}

		if rate.PricePerKg < 0 {
			validationErrors = append(validationErrors,
				fmt.Sprintf("rate %d: price per kg cannot be negative", i))
		}

		// Validate weight and distance ranges
		if rate.WeightMax <= rate.WeightMin {
			validationErrors = append(validationErrors,
				fmt.Sprintf("rate %d: weight max must be greater than weight min", i))
		}

		if rate.DistanceMax <= rate.DistanceMin {
			validationErrors = append(validationErrors,
				fmt.Sprintf("rate %d: distance max must be greater than distance min", i))
		}

		// Validate date ranges
		if rate.ValidTo != nil && rate.ValidTo.Before(rate.ValidFrom) {
			validationErrors = append(validationErrors,
				fmt.Sprintf("rate %d: valid to date must be after valid from date", i))
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation errors: %v", validationErrors)
	}

	return nil
}

// Private helper methods

func (p *BaseStaticProvider) loadRateData(ctx context.Context) error {
	// Get all rates for this partner
	rates, err := p.rateRepo.GetByPartner(ctx, p.partner.ID.String())
	if err != nil {
		return fmt.Errorf("failed to get partner rates: %w", err)
	}

	// Validate the rates
	if err := p.ValidateRateData(rates); err != nil {
		p.logger.Warn("Rate data validation warnings",
			"partner_id", p.partner.ID,
			"error", err)
		// Continue with loading despite validation warnings
	}

	// Update cache info
	p.cacheInfo.TotalEntries = len(rates)
	p.cacheInfo.CacheSize = int64(len(rates) * 500) // Rough estimate

	p.logger.Debug("Rate data loaded",
		"partner_id", p.partner.ID,
		"total_rates", len(rates))

	return nil
}

func (p *BaseStaticProvider) performRepositoryHealthCheck(ctx context.Context) error {
	// Try to get a count of rates for this partner
	rates, err := p.rateRepo.GetByPartner(ctx, p.partner.ID.String())
	if err != nil {
		return fmt.Errorf("repository is not accessible: %w", err)
	}

	p.logger.Debug("Repository health check passed",
		"partner_id", p.partner.ID,
		"available_rates", len(rates))

	return nil
}

func (p *BaseStaticProvider) buildRateCriteria(request *dtos.RateCalculationRequest) *dtos.RateCriteria {
	// Build criteria to find matching rates
	now := time.Now()

	return &dtos.RateCriteria{
		PartnerIDs:   []string{p.partner.ID.String()},
		ServiceTypes: []string{request.ServiceType},
		OriginCity:   request.OriginCity,
		DestCity:     request.DestCity,
		WeightMin:    &request.Weight,
		WeightMax:    &request.Weight,
		DistanceMin:  &request.Distance,
		DistanceMax:  &request.Distance,
		ValidDate:    &now,
		IsActive:     &[]bool{true}[0],
		Currency:     request.Currency,
		OrderBy:      "base_price",
		OrderDir:     "asc",
		Limit:        100, // Reasonable limit
	}
}

func (p *BaseStaticProvider) calculateQuotesFromRates(request *dtos.RateCalculationRequest, rates []*models.Rate) []dtos.RateQuote {
	var quotes []dtos.RateQuote

	for _, rate := range rates {
		// Check if rate is valid for the request parameters
		if !p.isRateApplicable(rate, request) {
			continue
		}

		// Calculate the price using the rate
		totalPrice := rate.CalculatePrice(request.Weight, request.Distance)

		// Create quote
		quote := p.createQuote(request, rate, totalPrice)
		quotes = append(quotes, quote)
	}

	return quotes
}

func (p *BaseStaticProvider) isRateApplicable(rate *models.Rate, request *dtos.RateCalculationRequest) bool {
	// Check if rate is valid for the pickup date
	if !rate.IsValidForDate(request.PickupDate) {
		return false
	}

	// Check weight range
	if !rate.IsValidForWeight(request.Weight) {
		return false
	}

	// Check distance range
	if !rate.IsValidForDistance(request.Distance) {
		return false
	}

	// Check service type
	if rate.ServiceType != request.ServiceType {
		return false
	}

	// Check origin and destination
	if rate.OriginCity != request.OriginCity || rate.DestCity != request.DestCity {
		return false
	}

	return true
}

func (p *BaseStaticProvider) createQuote(request *dtos.RateCalculationRequest, rate *models.Rate, totalPrice float64) dtos.RateQuote {
	quoteID := fmt.Sprintf("%s-static-%d", p.partner.Code, time.Now().UnixNano())

	// Calculate estimated delivery time based on service type
	estimatedDays := p.calculateEstimatedDays(request.ServiceType, request.Distance)

	// Create price breakdown
	priceBreakdown := dtos.PriceBreakdown{
		BasePrice:      rate.BasePrice,
		WeightCharge:   rate.PricePerKg * request.Weight,
		DistanceCharge: rate.PricePerKm * request.Distance,
		TotalPrice:     totalPrice,
	}

	return dtos.RateQuote{
		QuoteID:      quoteID,
		PartnerID:    p.partner.ID.String(),
		PartnerName:  p.partner.Name,
		ProviderType: dtos.ProviderTypeStatic,

		BasePrice:      rate.BasePrice,
		TotalPrice:     totalPrice,
		Currency:       rate.Currency,
		PriceBreakdown: priceBreakdown,

		ServiceType:    rate.ServiceType,
		ServiceLevel:   "standard",
		EstimatedDays:  estimatedDays,
		EstimatedHours: estimatedDays * 24,

		ValidUntil: time.Now().Add(24 * time.Hour), // 24 hour validity
		Confidence: 0.95,                           // Higher confidence for static rates

		InsuranceAvailable: true,
		TrackingAvailable:  true,
		SignatureAvailable: true,

		Source:         "static",
		ResponseTimeMs: 0, // Will be set by caller

		Terms: fmt.Sprintf("Rate valid from %s", rate.ValidFrom.Format("2006-01-02")),
	}
}

func (p *BaseStaticProvider) calculateEstimatedDays(serviceType string, distance float64) int {
	// Simple estimation logic based on service type and distance
	baseDays := 1

	switch serviceType {
	case "same_day":
		return 1
	case "express":
		baseDays = 2
	case "premium":
		baseDays = 3
	case "standard":
		baseDays = 5
	case "economy":
		baseDays = 7
	}

	// Add days based on distance
	if distance > 500 {
		baseDays += 2
	} else if distance > 200 {
		baseDays += 1
	}

	return baseDays
}

func (p *BaseStaticProvider) buildStandardResponse(request *dtos.RateCalculationRequest, quotes []dtos.RateQuote) *dtos.RateCalculationResponse {
	var bestQuote *dtos.RateQuote
	if len(quotes) > 0 {
		// Find the quote with the best price
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
