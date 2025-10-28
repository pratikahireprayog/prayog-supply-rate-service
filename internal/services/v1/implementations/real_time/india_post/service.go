package indiapost

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

// NewService creates a new India Post service instance
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

	s.isInitialized = true

	s.logger.Info("India Post International service initialized successfully",
		"base_url", s.config.BaseURL,
		"environment", s.config.Environment)
	
	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       s.config.BaseURL,
		"environment":    s.config.Environment,
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
	tariffRequests, err := s.convertToIndiaPostRequests(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Fetch rates for all product types
	var quotes []dtos.RateQuote
	for _, tariffReq := range tariffRequests {
		quote, err := s.fetchSingleRate(ctx, tariffReq, request)
		if err != nil {
			s.logger.Warn("Failed to fetch rate for product type",
				"product_type", tariffReq.ProductType,
				"error", err)
			continue
		}
		
		if quote != nil {
			quotes = append(quotes, *quote)
		}
	}

	// Build response
	response := s.buildResponse(request, quotes, time.Since(startTime))

	s.metrics.IncrementCounter("india_post_api_success", map[string]string{
		"total_quotes": fmt.Sprintf("%d", len(quotes)),
	})
	s.metrics.RecordTimer("india_post_api_duration", time.Since(startTime), map[string]string{
		"endpoint": "get_rates",
	})

	return response, nil
}

// fetchSingleRate fetches a rate for a single product type
func (s *Service) fetchSingleRate(ctx context.Context, tariffReq *TariffRequest, originalReq *dtos.RateCalculationRequest) (*dtos.RateQuote, error) {
	// Get authentication token
	token, err := s.authManager.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	// Validate tariff request
	if err := s.validateTariffRequest(tariffReq); err != nil {
		return nil, fmt.Errorf("tariff request validation failed: %w", err)
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
		s.metrics.IncrementCounter("india_post_api_error", map[string]string{
			"error_type": "api_call_failed",
		})
		return nil, fmt.Errorf("India Post API call failed: %w", err)
	}

	duration := time.Since(startTime)

	s.logger.Debug("India Post API response received",
		"status_code", httpResponse.StatusCode,
		"duration_ms", duration.Milliseconds())

	// Handle non-200 responses
	if httpResponse.StatusCode != http.StatusOK {
		// Try to parse error response
		var errResp ErrorResponse
		if err := json.Unmarshal(httpResponse.Body, &errResp); err == nil {
			s.logger.Error("India Post API returned error",
				"status_code", httpResponse.StatusCode,
				"error", errResp.Error,
				"message", errResp.Message)
		}
		
		// If 401, invalidate token and retry once
		if httpResponse.StatusCode == http.StatusUnauthorized {
			s.logger.Info("Received 401, invalidating token and retrying")
			s.authManager.InvalidateToken()
			// Could implement retry logic here if needed
		}
		
		return nil, fmt.Errorf("India Post API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response
	var tariffResp TariffResponse
	if err := json.Unmarshal(httpResponse.Body, &tariffResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if calculation was successful
	if !tariffResp.Data.Success {
		return nil, fmt.Errorf("tariff calculation failed: product_code=%s", tariffResp.Data.ProductCode)
	}

	// Convert to our quote format
	quote := s.convertToQuote(&tariffResp.Data, originalReq, duration)

	return &quote, nil
}

// convertToIndiaPostRequests converts our request to India Post format
// Returns multiple requests for different product types/services
func (s *Service) convertToIndiaPostRequests(req *dtos.RateCalculationRequest) ([]*TariffRequest, error) {
	var requests []*TariffRequest

	// Calculate total weight (convert to grams for India Post)
	totalWeightKg := 0.0
	for _, pkg := range req.Packages {
		weightKg := utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)
		totalWeightKg += weightKg
	}
	
	// Convert to grams
	totalWeightGrams := totalWeightKg * 1000

	// Determine product types to quote based on weight and service type
	productTypes := s.determineProductTypes(totalWeightKg, req.ServiceType)
	
	// Get transmission mode based on service type
	transmissionMode := string(MapServiceTypeToTransmissionMode(req.ServiceType))

	// Create request for each product type
	for _, productType := range productTypes {
		tariffReq := &TariffRequest{
			ProductType:        string(productType),
			Weight:             totalWeightGrams,
			CountryCode:        strings.ToUpper(req.DestCountry),
			Registration:       true,  // Always include registration for tracking
			Insurance:          totalWeightKg > 2, // Insurance for valuable shipments
			InsAmount:          calculateInsuranceAmount(totalWeightKg),
			AdviceOfDelivery:   false, // Optional service
			ModeOfTransmission: transmissionMode,
			SourcePincode:      req.OriginCity,
		}
		
		requests = append(requests, tariffReq)
	}

	return requests, nil
}

// determineProductTypes determines which product types to quote based on weight and service
func (s *Service) determineProductTypes(weightKg float64, serviceType string) []ProductType {
	var productTypes []ProductType

	// Based on weight, determine eligible product types
	if weightKg <= 2.0 {
		// Can send as letter or small packet
		productTypes = append(productTypes, ProductTypeFGNLetter)
		if serviceType == "express" {
			productTypes = append(productTypes, ProductTypeEMS)
		}
	} else if weightKg <= 30.0 {
		// Send as parcel
		productTypes = append(productTypes, ProductTypeFGNParcel)
		if serviceType == "express" {
			productTypes = append(productTypes, ProductTypeEMS)
		}
	} else {
		// Only EMS for heavy items
		productTypes = append(productTypes, ProductTypeEMS)
	}

	return productTypes
}

// calculateInsuranceAmount calculates insurance amount based on weight
func calculateInsuranceAmount(weightKg float64) float64 {
	// Simple calculation: 1000 INR per kg (can be adjusted)
	return weightKg * 1000
}

// convertToQuote converts India Post response to our quote format
func (s *Service) convertToQuote(data *TariffData, originalReq *dtos.RateCalculationRequest, responseTime time.Duration) dtos.RateQuote {
	quoteID := fmt.Sprintf("india_post_%s_%d", strings.ToLower(data.ProductCode), time.Now().UnixNano())

	// Map product code to service type
	serviceType := s.mapProductCodeToServiceType(data.ProductCode)
	
	// Estimate delivery days
	deliveryDays := estimateDeliveryDays(ModeOfTransmission(data.ModeOfTransmission), data.CountryCode)

	// Create price breakdown
	priceBreakdown := dtos.PriceBreakdown{
		BasePrice:       data.BasicTariff.TotalBasicPrice,
		FuelSurcharge:   0, // India Post includes in base price
		HandlingCharge:  data.VASCharges.Total,
		InsuranceCharge: data.VASCharges.Insurance,
		TaxAmount:       data.TaxCalculation.TotalGST,
		TotalPrice:      data.TotalAmount,
	}

	quote := dtos.RateQuote{
		QuoteID:      quoteID,
		PartnerID:    "india_post_international",
		PartnerName:  "India Post International",
		ProviderType: dtos.ProviderTypeRealTime,

		BasePrice:      data.BasicTariff.TotalBasicPrice,
		TotalPrice:     data.TotalAmount,
		Currency:       originalReq.Currency,
		PriceBreakdown: priceBreakdown,

		ServiceType:    serviceType,
		ServiceLevel:   data.ProductName,
		EstimatedDays:  deliveryDays,
		EstimatedHours: deliveryDays * 24,

		ValidUntil: time.Now().Add(24 * time.Hour), // 24 hour validity
		Confidence: 0.95,                            // High confidence for government service

		InsuranceAvailable: true,
		TrackingAvailable:  data.VASCharges.Registration > 0, // Registration includes tracking
		SignatureAvailable: data.VASCharges.AdviceOfDelivery > 0,

		Source:          "real_time",
		ResponseTimeMs:  responseTime.Milliseconds(),
		ExternalQuoteID: data.ProductCode,
		
		Metadata: map[string]interface{}{
			"product_code":        data.ProductCode,
			"tariff_group":        data.TariffGroup,
			"mode_transmission":   data.ModeOfTransmission,
			"weight_grams":        data.Weight,
			"calculated_at":       data.CalculatedAt.Format(time.RFC3339),
			"gst_type":            data.TaxCalculation.GSTType,
			"applicable_gst_rate": data.TaxCalculation.ApplicableRate,
			"custom_charges": map[string]float64{
				"base_price":      data.BasicTariff.BasePrice,
				"ams_charge":      data.BasicTariff.AmsCharge,
				"sal_charge":      data.BasicTariff.SalCharge,
				"registration":    data.VASCharges.Registration,
				"insurance":       data.VASCharges.Insurance,
				"advice_delivery": data.VASCharges.AdviceOfDelivery,
			},
		},
	}

	return quote
}

// mapProductCodeToServiceType maps India Post product codes to our service types
func (s *Service) mapProductCodeToServiceType(productCode string) string {
	switch productCode {
	case "EMS":
		return "express"
	case "FGN_LETTER", "SMALL_PACKET":
		return "standard"
	case "FGN_PARCEL":
		return "standard"
	default:
		return "standard"
	}
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

// IsHealthy performs health check for India Post API
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("India Post International service not initialized")
	}

	s.logger.Info("Performing India Post International API health check")

	// Check if we can get a valid token
	startTime := time.Now()
	_, err := s.authManager.GetToken(ctx)
	
	duration := time.Since(startTime)
	
	s.metrics.RecordTimer("india_post_health_check_duration", duration, map[string]string{
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

