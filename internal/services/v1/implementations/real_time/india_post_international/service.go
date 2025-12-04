package india_post_international

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// Service implements real-time rate fetching for India Post International
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    interfaces.HTTPClient
	config        *Config
	authManager   *AuthManager
	isInitialized bool
}

// NewService creates a new India Post International service instance
func NewService(
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	httpClient interfaces.HTTPClient,
) *Service {
	config := NewDefaultConfig()

	service := &Service{
		logger:        logger,
		metrics:       metrics,
		httpClient:    httpClient,
		config:        config,
		isInitialized: false,
	}

	return service
}

// GetImplementationType returns the implementation type
func (s *Service) GetImplementationType() dtos.ProviderType {
	return dtos.ProviderTypeRealTime
}

// GetImplementationName returns the implementation name
func (s *Service) GetImplementationName() string {
	return "India Post International"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
	s.logger.Info("Initializing India Post International service", "config_keys", len(config))

	// Override defaults with provided config
	if err := s.config.LoadFromMap(config); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize auth manager
	s.authManager = NewAuthManager(s.config, s.httpClient, s.logger)

	// Pre-authenticate to verify credentials
	if err := s.authManager.Authenticate(context.Background()); err != nil {
		s.logger.Warn("Initial authentication failed, will retry on first request", "error", err)
		// Don't fail initialization, authentication will happen lazily
	}

	s.isInitialized = true

	s.logger.Info("India Post International service initialized successfully",
		"base_url", s.config.BaseURL,
		"enabled", s.config.Enabled)

	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       s.config.BaseURL,
		"enabled":        s.config.Enabled,
		"is_initialized": s.isInitialized,
		"provider_type":  "real_time",
	}
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.logger.Info("Closing India Post International service")
	s.isInitialized = false
	return nil
}

// GetRates fetches rates from India Post International API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("India Post International service not initialized")
	}

	if !s.config.Enabled {
		return nil, fmt.Errorf("India Post International service is disabled")
	}

	// Validate request before making API call
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	startTime := time.Now()
	s.logger.Info("Fetching rates from India Post International API",
		"request_id", request.RequestID,
		"origin", request.OriginCity,
		"origin_country", request.OriginCountry,
		"destination", request.DestCity,
		"dest_country", request.DestCountry,
		"packages", len(request.Packages))

	// Convert our request to India Post format
	tariffReq, err := s.convertToIndiaPostRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Fetch rate
	quote, err := s.fetchRate(ctx, tariffReq, request)
	if err != nil {
		s.metrics.IncrementCounter("india_post_intl_api_error", map[string]string{
			"error_type": "api_call_failed",
		})
		return nil, fmt.Errorf("failed to fetch rate: %w", err)
	}

	// Build response
	quotes := []dtos.RateQuote{}
	if quote != nil {
		quotes = append(quotes, *quote)
	}

	response := s.buildResponse(request, quotes, time.Since(startTime))

	s.metrics.IncrementCounter("india_post_intl_api_success", map[string]string{
		"total_quotes": fmt.Sprintf("%d", len(quotes)),
	})
	s.metrics.RecordTimer("india_post_intl_api_duration", time.Since(startTime), map[string]string{
		"endpoint": "get_rates",
	})

	return response, nil
}

// fetchRate fetches a rate from India Post International API
func (s *Service) fetchRate(ctx context.Context, tariffReq *TariffRequest, originalReq *dtos.RateCalculationRequest) (*dtos.RateQuote, error) {
	// Get authentication token
	token, err := s.authManager.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Prepare headers
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}

	// Make API call
	startTime := time.Now()
	httpResponse, err := s.httpClient.Post(ctx, s.config.GetTariffURL(), tariffReq, headers)
	if err != nil {
		s.metrics.IncrementCounter("india_post_intl_api_error", map[string]string{
			"error_type": "api_call_failed",
		})
		return nil, fmt.Errorf("India Post International API call failed: %w", err)
	}

	duration := time.Since(startTime)

	s.logger.Debug("India Post International API response received",
		"status_code", httpResponse.StatusCode,
		"duration_ms", duration.Milliseconds())

	// Handle non-200 responses
	if httpResponse.StatusCode != http.StatusOK {
		// Try to parse error response
		var errResp ErrorResponse
		if err := json.Unmarshal(httpResponse.Body, &errResp); err == nil {
			s.logger.Error("India Post International API returned error",
				"status_code", httpResponse.StatusCode,
				"error", errResp.Error,
				"message", errResp.Message)
		}

		// If 401, invalidate token and retry once
		if httpResponse.StatusCode == http.StatusUnauthorized {
			s.logger.Info("Received 401, invalidating token")
			s.authManager.InvalidateToken()
		}

		return nil, fmt.Errorf("India Post International API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response - handle wrapped format with statusCode and data
	var wrappedResp struct {
		StatusCode int                    `json:"statusCode"`
		Data       map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(httpResponse.Body, &wrappedResp); err != nil {
		s.logger.Error("Failed to parse India Post International response", "error", err, "response", string(httpResponse.Body))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract data from wrapped response
	tariffResp := TariffResponse{}
	dataMap := wrappedResp.Data

	// Extract totalAmount (this is the tariff amount)
	if totalAmount, ok := dataMap["totalAmount"].(float64); ok {
		tariffResp.TariffAmount = totalAmount
	}

	// Extract success and convert to status string
	if success, ok := dataMap["success"].(bool); ok {
		tariffResp.Success = success
		if success {
			tariffResp.Status = "success"
		} else {
			tariffResp.Status = "failed"
		}
	}

	// Currency is typically INR for India Post, but check if provided
	if currency, ok := dataMap["currency"].(string); ok && currency != "" {
		tariffResp.Currency = currency
	} else {
		// Default to INR for India Post International
		tariffResp.Currency = "INR"
	}

	// Extract delivery time if available
	if deliveryTime, ok := dataMap["deliveryTime"].(string); ok && deliveryTime != "" {
		tariffResp.DeliveryTime = deliveryTime
	} else if estimatedDays, ok := dataMap["estimatedDays"].(string); ok && estimatedDays != "" {
		tariffResp.DeliveryTime = estimatedDays
	}

	// Extract message if available
	if message, ok := dataMap["message"].(string); ok {
		tariffResp.Message = message
	}

	// Check if calculation was successful
	if !tariffResp.Success || tariffResp.Status != "success" {
		return nil, fmt.Errorf("tariff calculation failed: status=%s, message=%s", tariffResp.Status, tariffResp.Message)
	}

	// Convert to our quote format
	quote := s.convertToQuote(&tariffResp, originalReq, duration)

	return &quote, nil
}

// convertToIndiaPostRequest converts our request to India Post format
func (s *Service) convertToIndiaPostRequest(req *dtos.RateCalculationRequest) (*TariffRequest, error) {
	// Calculate total weight (convert to grams for India Post)
	totalWeightGrams := 0.0
	for _, pkg := range req.Packages {
		weightKg := utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)
		totalWeightGrams += weightKg * 1000 // Convert to grams
	}

	// Default to 50g if no weight provided
	if totalWeightGrams == 0 {
		totalWeightGrams = 50
	}

	// Use origin city as source pincode (assuming it's a pincode)
	sourcePincode := req.OriginCity
	// If origin city is not a pincode, we might need to extract it from metadata
	// For now, we'll use it as-is

	// Use destination country code
	destCountryCode := strings.ToUpper(req.DestCountry)

	tariffReq := &TariffRequest{
		Weight:        int(totalWeightGrams),
		CountryCode:   destCountryCode,
		SourcePincode: sourcePincode,
	}

	return tariffReq, nil
}

// convertToQuote converts India Post response to our quote format
func (s *Service) convertToQuote(tariffResp *TariffResponse, originalReq *dtos.RateCalculationRequest, responseTime time.Duration) dtos.RateQuote {
	quoteID := fmt.Sprintf("india_post_intl_%d", time.Now().UnixNano())

	// Estimate delivery days (default to 7 for international)
	deliveryDays := 7
	if tariffResp.DeliveryTime != "" {
		// Try to parse delivery time string (e.g., "7-10 days")
		// For now, use default
	}

	// Create price breakdown
	priceBreakdown := dtos.PriceBreakdown{
		BasePrice:      tariffResp.TariffAmount,
		FuelSurcharge:  0, // India Post includes in base price
		HandlingCharge: 0,
		InsuranceCharge: 0,
		TaxAmount:      0,
		TotalPrice:     tariffResp.TariffAmount,
	}

	quote := dtos.RateQuote{
		QuoteID:      quoteID,
		PartnerID:    "india_post_international",
		PartnerName:  "India Post International",
		ProviderType: dtos.ProviderTypeRealTime,

		BasePrice:      tariffResp.TariffAmount,
		TotalPrice:     tariffResp.TariffAmount,
		Currency:       tariffResp.Currency,
		PriceBreakdown: priceBreakdown,

		ServiceType:    originalReq.ServiceType,
		ServiceLevel:   "International",
		EstimatedDays:  deliveryDays,
		EstimatedHours: deliveryDays * 24,

		ValidUntil: time.Now().Add(24 * time.Hour), // 24 hour validity
		Confidence: 0.95,                            // High confidence for government service

		InsuranceAvailable: true,
		TrackingAvailable:  true,
		SignatureAvailable: true,

		Source:         "real_time",
		ResponseTimeMs: responseTime.Milliseconds(),
		ExternalQuoteID: tariffResp.Status,

		Metadata: map[string]interface{}{
			"status":        tariffResp.Status,
			"message":       tariffResp.Message,
			"delivery_time": tariffResp.DeliveryTime,
			"currency":      tariffResp.Currency,
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
	message := "Rates retrieved from India Post International"
	if len(quotes) == 0 {
		status = "no_rates"
		message = "No rates available from India Post International"
	}

	return &dtos.RateCalculationResponse{
		RequestID:    request.RequestID,
		Status:       status,
		Message:      message,
		Quotes:       quotes,
		BestQuote:    bestQuote,
		TotalQuotes:  len(quotes),
		ResponseTime: responseTime.Milliseconds(),
		CacheHit:     false,
		Timestamp:    time.Now(),
	}
}

// validateRequest validates the request before making API call
func (s *Service) validateRequest(request *dtos.RateCalculationRequest) error {
	// India Post International requires source pincode and destination country
	if request.OriginCity == "" {
		return fmt.Errorf("origin city (source pincode) is required for India Post International shipments")
	}

	if request.DestCountry == "" {
		return fmt.Errorf("destination country code is required for India Post International shipments")
	}

	// Validate country code format (should be 2-letter ISO code)
	if len(request.DestCountry) != 2 {
		return fmt.Errorf("destination country code must be a 2-letter ISO code")
	}

	// Validate weight
	if len(request.Packages) == 0 {
		return fmt.Errorf("at least one package is required")
	}

	return nil
}

// IsHealthy performs health check for India Post International API
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("India Post International service not initialized")
	}

	if !s.config.Enabled {
		return fmt.Errorf("India Post International service is disabled")
	}

	s.logger.Info("Performing India Post International API health check")

	// Check if we can get a valid token
	startTime := time.Now()
	_, err := s.authManager.GetAccessToken(ctx)

	duration := time.Since(startTime)

	s.metrics.RecordTimer("india_post_intl_health_check_duration", duration, map[string]string{
		"status": func() string {
			if err != nil {
				return "failed"
			}
			return "success"
		}(),
	})

	if err != nil {
		s.logger.Warn("India Post International API health check failed", "error", err)
		return fmt.Errorf("India Post International API health check failed: %w", err)
	}

	s.logger.Info("India Post International API health check successful", "duration_ms", duration.Milliseconds())
	return nil
}

