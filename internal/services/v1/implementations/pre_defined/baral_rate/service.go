package baral_rate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// Service implements Baral Rate Card API integration
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    interfaces.HTTPClient
	config        *Config
	isInitialized bool
	
	// Cache for rate cards
	cachedRates      []BaralRate
	cacheTimestamp   time.Time
}

// NewService creates a new Baral Rate Card service instance
func NewService(
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	httpClient interfaces.HTTPClient,
) *Service {
	config := NewDefaultConfig()

	return &Service{
		logger:        logger,
		metrics:       metrics,
		httpClient:    httpClient,
		config:        config,
		isInitialized: false,
	}
}

// GetImplementationType returns the implementation type
func (s *Service) GetImplementationType() dtos.ProviderType {
	return dtos.ProviderTypePreDefined
}

// GetImplementationName returns the implementation name
func (s *Service) GetImplementationName() string {
	return "Baral Rate Card"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
	s.logger.Info("Initializing Baral Rate Card service", "config_keys", len(config))

	// Override defaults with provided config
	if err := s.config.UpdateFromMap(config); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set HTTP client timeout and retry count
	s.httpClient.SetTimeout(s.config.GetTimeout())
	s.httpClient.SetRetryCount(s.config.RetryCount)

	s.isInitialized = true

	s.logger.Info("Baral Rate Card service initialized successfully",
		"base_url", s.config.BaseURL,
		"partner_code", s.config.PartnerCode,
		"cache_enabled", s.config.EnableCache,
		"cache_ttl_minutes", s.config.CacheTTLMinutes)

	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	config := s.config.ToMap()
	config["is_initialized"] = s.isInitialized
	config["provider_type"] = "pre_defined"
	config["implementation_name"] = s.GetImplementationName()
	return config
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.logger.Info("Closing Baral Rate Card service")
	s.isInitialized = false
	s.cachedRates = nil
	return nil
}

// GetRates fetches rates from Baral API and matches them with the request
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("Baral Rate Card service not initialized")
	}

	startTime := time.Now()
	s.logger.Info("Fetching rates from Baral Rate Card API",
		"request_id", request.RequestID,
		"origin", request.OriginCity,
		"destination", request.DestCity,
		"packages", len(request.Packages))

	// Validate request
	if err := ValidateRateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// Fetch all rate cards from Baral API
	baralRates, err := s.fetchRateCards(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rate cards: %w", err)
	}

	// Filter and match rates based on request
	matchedQuotes := s.matchRatesToRequest(request, baralRates, time.Since(startTime))

	// Build response
	response := s.buildResponse(request, matchedQuotes, time.Since(startTime))

	s.metrics.IncrementCounter("baral_api_success", map[string]string{
		"total_quotes": fmt.Sprintf("%d", len(matchedQuotes)),
	})
	s.metrics.RecordTimer("baral_api_duration", time.Since(startTime), map[string]string{
		"endpoint": "get_rates",
	})

	return response, nil
}

// fetchRateCards fetches all rate cards from Baral API with caching
func (s *Service) fetchRateCards(ctx context.Context) ([]BaralRate, error) {
	// Check cache if enabled
	if s.config.EnableCache && s.isCacheValid() {
		s.logger.Debug("Returning cached Baral rate cards",
			"cache_age_seconds", time.Since(s.cacheTimestamp).Seconds())
		s.metrics.IncrementCounter("baral_cache_hit", map[string]string{})
		return s.cachedRates, nil
	}

	// Fetch from API
	s.logger.Info("Fetching rate cards from Baral API",
		"url", s.config.GetAllRateCardsURL())

	headers := map[string]string{
		"Accept":       "*/*",
		"Content-Type": "application/json",
	}

	startTime := time.Now()
	httpResponse, err := s.httpClient.Get(ctx, s.config.GetAllRateCardsURL(), headers)
	if err != nil {
		s.metrics.IncrementCounter("baral_api_error", map[string]string{
			"error_type": "api_call_failed",
		})
		return nil, fmt.Errorf("Baral API call failed: %w", err)
	}

	duration := time.Since(startTime)
	s.logger.Debug("Baral API response received",
		"status_code", httpResponse.StatusCode,
		"duration_ms", duration.Milliseconds())

	// Handle non-200 responses
	if httpResponse.StatusCode != http.StatusOK {
		s.logger.Error("Baral API returned error",
			"status_code", httpResponse.StatusCode,
			"response", string(httpResponse.Body))
		return nil, fmt.Errorf("Baral API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response
	var apiResponse BaralAPIResponse
	if err := json.Unmarshal(httpResponse.Body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Baral API response: %w", err)
	}

	// Check if response was successful
	if apiResponse.Status != 200 {
		return nil, fmt.Errorf("Baral API returned non-success status: %d - %s", apiResponse.Status, apiResponse.Message)
	}

	// Extract rates from response
	var rates []BaralRate
	if apiResponse.Data != nil && len(apiResponse.Data.RateCard) > 0 {
		rates = apiResponse.Data.RateCard
	}

	s.logger.Info("Fetched rate cards from Baral API",
		"total_rates", len(rates),
		"duration_ms", duration.Milliseconds())

	// Update cache
	if s.config.EnableCache {
		s.cachedRates = rates
		s.cacheTimestamp = time.Now()
		s.metrics.IncrementCounter("baral_cache_update", map[string]string{})
	}

	return rates, nil
}

// isCacheValid checks if the cache is still valid
func (s *Service) isCacheValid() bool {
	if s.cachedRates == nil || len(s.cachedRates) == 0 {
		return false
	}
	
	cacheTTL := s.config.GetCacheTTL()
	return time.Since(s.cacheTimestamp) < cacheTTL
}

// matchRatesToRequest filters and matches rates based on the request criteria
func (s *Service) matchRatesToRequest(request *dtos.RateCalculationRequest, rates []BaralRate, responseTime time.Duration) []dtos.RateQuote {
	var quotes []dtos.RateQuote

	// Calculate total weight from packages
	totalWeight := 0.0
	for _, pkg := range request.Packages {
		weightKg := utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)
		totalWeight += weightKg
	}

	s.logger.Debug("Matching rates",
		"total_rates", len(rates),
		"total_weight_kg", totalWeight)

	// Convert all valid rates to quotes
	for _, rate := range rates {
		// Check if rate is valid and enabled
		if !rate.IsValid() {
			s.logger.Debug("Skipping invalid rate", "rate_id", rate.ID, "is_enable", rate.IsEnable)
			continue
		}

		// Calculate price for this weight
		// Note: Even if weight is less than minimum, we show the rate with minimum freight
		actualWeight := totalWeight
		minimumWeight := rate.GetMinimumWeightKg()
		if totalWeight < minimumWeight && minimumWeight > 0 {
			// Use minimum weight for calculation
			actualWeight = minimumWeight
			s.logger.Debug("Using minimum weight for rate",
				"rate_id", rate.ID,
				"requested_weight", totalWeight,
				"minimum_weight", minimumWeight)
		}

		// Convert to quote
		quote := s.convertToQuote(&rate, request, responseTime, actualWeight)
		quotes = append(quotes, quote)
	}

	s.logger.Info("Matched rates from Baral",
		"total_available", len(rates),
		"matched_quotes", len(quotes))

	return quotes
}

// convertToQuote converts a Baral rate to our quote format
func (s *Service) convertToQuote(rate *BaralRate, request *dtos.RateCalculationRequest, responseTime time.Duration, weightKg float64) dtos.RateQuote {
	quoteID := fmt.Sprintf("baral_%s_%d", s.config.PartnerCode, time.Now().UnixNano())

	currency := s.config.DefaultCurrency
	if request.Currency != "" {
		currency = request.Currency
	}

	// Calculate price
	totalPrice := rate.CalculatePrice(weightKg)
	basePrice := rate.GetMinimumFreightINR()
	fuelSurcharge := basePrice * (rate.GetFuelSurchargePercent() / 100)
	docketCharges := rate.GetDocketCharges()

	// Build price breakdown
	priceBreakdown := dtos.PriceBreakdown{
		BasePrice:       basePrice,
		FuelSurcharge:   fuelSurcharge,
		HandlingCharge:  docketCharges,
		TaxAmount:       0, // Tax calculation can be added later
		TotalPrice:      totalPrice,
		WeightCharge:    0,
		DistanceCharge:  0,
		InsuranceCharge: 0,
		DiscountAmount:  0,
	}

	// Estimate delivery days based on rate card type
	estimatedDays := 5 // Default for surface
	if rate.RateCardType == "air" {
		estimatedDays = 2
	}

	// Determine service type from rate card type
	serviceType := "standard"
	if rate.RateCardType == "air" || rate.RateCardType == "express" {
		serviceType = "express"
	}

	quote := dtos.RateQuote{
		QuoteID:      quoteID,
		PartnerID:    s.config.PartnerCode,
		PartnerName:  rate.PartnerDetails.CompanyName,
		ProviderType: dtos.ProviderTypePreDefined,

		BasePrice:      basePrice,
		TotalPrice:     totalPrice,
		Currency:       currency,
		PriceBreakdown: priceBreakdown,

		ServiceType:    serviceType,
		ServiceLevel:   rate.RateCardName,
		EstimatedDays:  estimatedDays,
		EstimatedHours: estimatedDays * 24,

		ValidUntil: time.Now().Add(24 * time.Hour), // 24 hour validity
		Confidence: 0.85, // Fixed confidence for pre-defined rates

		InsuranceAvailable: rate.Insurance.CarrierInsurance.InsurancePercentage > 0,
		TrackingAvailable:  true,
		SignatureAvailable: false,

		Source:          "pre_defined",
		ResponseTimeMs:  responseTime.Milliseconds(),
		ExternalQuoteID: rate.ID,

		Metadata: map[string]interface{}{
			"rate_id":         rate.ID,
			"rate_card_name":  rate.RateCardName,
			"partner_code":    rate.PartnerCode,
			"rate_card_type":  rate.RateCardType,
			"minimum_weight":  rate.MinimumWeight,
			"minimum_freight": rate.MinimumFreight,
			"weight_used_kg":  weightKg,
			"fuel_surcharge_percent": rate.GetFuelSurchargePercent(),
		},
	}

	return quote
}


// buildResponse builds the final response
func (s *Service) buildResponse(request *dtos.RateCalculationRequest, quotes []dtos.RateQuote, responseTime time.Duration) *dtos.RateCalculationResponse {
	var bestQuote *dtos.RateQuote
	if len(quotes) > 0 {
		// Find best quote (lowest price)
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
	message := "Rates retrieved from Baral Rate Card API"
	if len(quotes) == 0 {
		status = "no_rates"
		message = "No matching rates found in Baral Rate Card API"
	}

	return &dtos.RateCalculationResponse{
		RequestID:    request.RequestID,
		Status:       status,
		Message:      message,
		Quotes:       quotes,
		BestQuote:    bestQuote,
		TotalQuotes:  len(quotes),
		ResponseTime: responseTime.Milliseconds(),
		CacheHit:     s.config.EnableCache && s.isCacheValid(),
		Timestamp:    time.Now(),
	}
}

// IsHealthy performs health check for Baral API
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("Baral Rate Card service not initialized")
	}

	s.logger.Info("Performing Baral Rate Card API health check")

	startTime := time.Now()
	
	// Try to fetch rate cards as health check
	_, err := s.fetchRateCards(ctx)
	
	duration := time.Since(startTime)

	s.metrics.RecordTimer("baral_health_check_duration", duration, map[string]string{
		"status": func() string {
			if err != nil {
				return "failed"
			}
			return "success"
		}(),
	})

	if err != nil {
		s.logger.Warn("Baral Rate Card API health check failed", "error", err)
		return fmt.Errorf("Baral Rate Card API health check failed: %w", err)
	}

	s.logger.Info("Baral Rate Card API health check successful", "duration_ms", duration.Milliseconds())
	return nil
}

// RefreshRates refreshes the rate cache by fetching fresh data from the API
func (s *Service) RefreshRates(ctx context.Context) error {
	s.logger.Info("Refreshing Baral rate cards cache")
	
	// Invalidate current cache
	s.cachedRates = nil
	s.cacheTimestamp = time.Time{}
	
	// Fetch fresh data
	_, err := s.fetchRateCards(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh rate cards: %w", err)
	}
	
	s.logger.Info("Baral rate cards cache refreshed successfully",
		"total_rates", len(s.cachedRates))
	
	return nil
}

