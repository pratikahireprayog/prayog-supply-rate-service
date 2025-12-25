package delhivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// Service implements real-time rate fetching for Delhivery
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    interfaces.HTTPClient
	config        *Config
	authService   *AuthService
	isInitialized bool
}

// NewService creates a new Delhivery service instance
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
	return "Delhivery"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
	s.logger.Info("Initializing Delhivery service", "config_keys", len(config))

	// Override defaults with provided config
	if err := s.config.LoadFromMap(config); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize auth service
	s.authService = NewAuthService(s.httpClient, s.logger)
	
	// Update auth config if provided
	authConfigMap := make(map[string]interface{})
	if username, ok := config["username"].(string); ok {
		authConfigMap["username"] = username
	}
	if password, ok := config["password"].(string); ok {
		authConfigMap["password"] = password
	}
	if loginURL, ok := config["login_url"].(string); ok {
		authConfigMap["login_url"] = loginURL
	}
	if signinType, ok := config["signin_type"].(string); ok {
		authConfigMap["signin_type"] = signinType
	}
	if tenantID, ok := config["tenant_id"].(string); ok {
		authConfigMap["tenant_id"] = tenantID
	}
	if len(authConfigMap) > 0 {
		if err := s.authService.UpdateConfig(authConfigMap); err != nil {
			s.logger.Warn("Failed to update auth configuration", "error", err)
		}
	}

	s.isInitialized = true

	s.logger.Info("Delhivery service initialized successfully",
		"base_url", s.config.BaseURL,
		"username", s.config.Username)

	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       s.config.BaseURL,
		"username":       s.config.Username,
		"is_initialized": s.isInitialized,
		"provider_type":  "real_time",
	}
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.logger.Info("Closing Delhivery service")
	s.isInitialized = false
	return nil
}

// IsHealthy checks if the service is healthy and operational
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("Delhivery service not initialized")
	}

	s.logger.Info("Performing Delhivery API health check")

	// Check if we can get a valid token
	startTime := time.Now()
	_, err := s.authService.GetBearerToken(ctx)
	if err != nil {
		s.logger.Warn("Delhivery API health check failed - authentication", "error", err, "duration_ms", time.Since(startTime).Milliseconds())
		return fmt.Errorf("Delhivery API authentication failed: %w", err)
	}

	s.logger.Info("Delhivery API health check passed", "duration_ms", time.Since(startTime).Milliseconds())
	return nil
}

// GetRates fetches rates from Delhivery API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("Delhivery service not initialized")
	}

	// Validate request
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	startTime := time.Now()
	s.logger.Info("Fetching rates from Delhivery API",
		"request_id", request.RequestID,
		"origin_pin", request.OriginCity,
		"dest_pin", request.DestCity,
		"packages", len(request.Packages))

	// Convert our request to Delhivery format
	estimateReq, err := s.convertToDelhiveryRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Fetch rate
	quote, err := s.fetchRate(ctx, estimateReq, request)
	if err != nil {
		s.metrics.IncrementCounter("delhivery_api_error", map[string]string{
			"error_type": "api_call_failed",
		})
		return nil, fmt.Errorf("failed to fetch rate: %w", err)
	}

	// Build response
	var quotes []dtos.RateQuote
	if quote != nil {
		quotes = append(quotes, *quote)
	}

	response := s.buildResponse(request, quotes, time.Since(startTime))

	s.metrics.IncrementCounter("delhivery_api_success", map[string]string{
		"total_quotes": fmt.Sprintf("%d", len(quotes)),
	})
	s.metrics.RecordTimer("delhivery_api_duration", time.Since(startTime), map[string]string{
		"endpoint": "get_rates",
	})

	return response, nil
}

// fetchRate fetches a rate from Delhivery API
func (s *Service) fetchRate(ctx context.Context, estimateReq *FreightEstimateRequest, originalReq *dtos.RateCalculationRequest) (*dtos.RateQuote, error) {
	// Get authentication token
	bearerToken, err := s.authService.GetBearerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	// Prepare headers
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": bearerToken,
	}

	// Make API call
	startTime := time.Now()
	httpResponse, err := s.httpClient.Post(ctx, s.config.GetEstimateURL(), estimateReq, headers)
	if err != nil {
		return nil, fmt.Errorf("Delhivery API call failed: %w", err)
	}

	duration := time.Since(startTime)

	s.logger.Debug("Delhivery API response received",
		"status_code", httpResponse.StatusCode,
		"duration_ms", duration.Milliseconds())

	// Handle non-200 responses
	if httpResponse.StatusCode != http.StatusOK {
		// Try to parse error response
		var errResp FreightEstimateResponse
		if err := json.Unmarshal(httpResponse.Body, &errResp); err == nil {
			if errResp.Error != nil {
				s.logger.Error("Delhivery API returned error",
					"status_code", httpResponse.StatusCode,
					"error_code", errResp.Error.Code,
					"error_message", errResp.Error.Message)
			} else {
				s.logger.Error("Delhivery API returned error",
					"status_code", httpResponse.StatusCode,
					"message", errResp.Message)
			}
		}

		// If 401, clear token and retry once
		if httpResponse.StatusCode == http.StatusUnauthorized {
			s.logger.Info("Received 401, clearing token")
			s.authService.ClearToken()
		}

		return nil, fmt.Errorf("Delhivery API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response
	var estimateResp FreightEstimateResponse
	if err := json.Unmarshal(httpResponse.Body, &estimateResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if calculation was successful
	if !estimateResp.Success || estimateResp.Data.TotalFreight == 0 {
		return nil, fmt.Errorf("freight estimate failed: %s", estimateResp.Message)
	}

	// Convert to our quote format
	quote := s.convertToQuote(&estimateResp.Data, originalReq, duration)

	return &quote, nil
}

// convertToDelhiveryRequest converts our request to Delhivery format
func (s *Service) convertToDelhiveryRequest(req *dtos.RateCalculationRequest) (*FreightEstimateRequest, error) {
	// Calculate total weight in grams
	totalWeightG := 0.0
	var dimensions []Dimension

	for _, pkg := range req.Packages {
		// Convert weight to grams
		weightKg := utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)
		weightG := weightKg * 1000
		totalWeightG += weightG

		// Convert dimensions to cm
		lengthCM := utils.ConvertDimensionToCm(pkg.Length, pkg.DimUnit)
		widthCM := utils.ConvertDimensionToCm(pkg.Width, pkg.DimUnit)
		heightCM := utils.ConvertDimensionToCm(pkg.Height, pkg.DimUnit)

		dimensions = append(dimensions, Dimension{
			LengthCM: lengthCM,
			WidthCM:  widthCM,
			HeightCM: heightCM,
			BoxCount: 1, // Default to 1 box per package
		})
	}

	// Determine payment mode and freight mode
	// Delhivery API expects: 'fop' (Freight on Prepaid) or 'fod' (Freight on Delivery)
	// Note: B2B accounts typically require 'fod' (Freight on Delivery)
	paymentMode := "prepaid"
	freightMode := "fod" // Default to FOD for B2B accounts (FoP not allowed for B2B)
	if req.Metadata != nil {
		if pm, ok := req.Metadata["payment_mode"].(string); ok && pm == "cod" {
			paymentMode = "cod"
			freightMode = "fod" // Freight on Delivery
		}
		// Allow explicit freight_mode override
		if fm, ok := req.Metadata["freight_mode"].(string); ok && (fm == "fop" || fm == "fod") {
			freightMode = fm
		}
	}

	// Get invoice amount from metadata or default to 1 (API requires > 0)
	invAmount := 1.0
	if req.Metadata != nil {
		if amt, ok := req.Metadata["inv_amount"].(float64); ok && amt > 0 {
			invAmount = amt
		}
	}

	// Determine ROV insurance (Reverse of Value insurance)
	rovInsurance := req.InsuranceRequired

	estimateReq := &FreightEstimateRequest{
		Dimensions:    dimensions,
		WeightG:        int(totalWeightG),
		ChequePayment:  false,
		SourcePin:      req.OriginCity,    // OriginCity contains postal code
		ConsigneePin:   req.DestCity,      // DestCity contains postal code
		PaymentMode:    paymentMode,
		InvAmount:      invAmount,
		FreightMode:    freightMode,
		ROVInsurance:   rovInsurance,
	}

	return estimateReq, nil
}

// convertToQuote converts Delhivery response to our quote format
func (s *Service) convertToQuote(data *FreightEstimateData, originalReq *dtos.RateCalculationRequest, responseTime time.Duration) dtos.RateQuote {
	quoteID := fmt.Sprintf("delhivery_%s_%s", uuid.New().String()[:8], time.Now().Format("20060102150405"))

	// Calculate tax amount
	taxAmount := data.Taxes.Total
	if taxAmount == 0 {
		// Fallback to sum of individual taxes
		taxAmount = data.Taxes.CGST + data.Taxes.SGST + data.Taxes.IGST
	}

	// Build price breakdown
	priceBreakdown := dtos.PriceBreakdown{
		BasePrice:       data.BaseFreight,
		FuelSurcharge:   data.FuelSurcharge,
		InsuranceCharge: data.InsuranceCharge,
		TaxAmount:       taxAmount,
		TotalPrice:      data.TotalFreight,
	}

	// Build metadata
	metadata := map[string]interface{}{
		"response_time_ms": responseTime.Milliseconds(),
		"service_type":     data.ServiceType,
		"freight_mode":     originalReq.Metadata["freight_mode"],
	}

	if len(data.Breakdown) > 0 {
		metadata["charge_breakdown"] = data.Breakdown
	}

	// Set validity (7 days from now)
	validUntil := time.Now().Add(7 * 24 * time.Hour)

	serviceType := data.ServiceType
	if serviceType == "" {
		serviceType = "standard"
	}

	quote := dtos.RateQuote{
		QuoteID:            quoteID,
		PartnerID:          "delhivery",
		PartnerName:        "Delhivery",
		ProviderType:      "real_time",
		BasePrice:          data.BaseFreight,
		TotalPrice:         data.TotalFreight,
		Currency:           "INR",
		PriceBreakdown:     priceBreakdown,
		ServiceType:        serviceType,
		ServiceLevel:       "standard",
		EstimatedDays:     data.EstimatedDays,
		ValidUntil:         validUntil,
		Confidence:         1.0,
		IsRecommended:     false,
		InsuranceAvailable: data.InsuranceCharge > 0,
		TrackingAvailable:  true,
		SignatureAvailable: false,
		Source:             "delhivery_api",
		Description:       fmt.Sprintf("Delhivery %s service", serviceType),
		Metadata:           metadata,
	}

	return quote
}

// buildResponse builds the rate calculation response
func (s *Service) buildResponse(request *dtos.RateCalculationRequest, quotes []dtos.RateQuote, responseTime time.Duration) *dtos.RateCalculationResponse {
	response := &dtos.RateCalculationResponse{
		RequestID:    request.RequestID,
		Status:       "success",
		Message:      "Rates retrieved successfully",
		Quotes:       quotes,
		TotalQuotes:  len(quotes),
		ResponseTime: responseTime.Milliseconds(),
		CacheHit:     false,
		Timestamp:    time.Now(),
	}

	// Set best quote (lowest price)
	if len(quotes) > 0 {
		bestQuote := &quotes[0]
		for i := 1; i < len(quotes); i++ {
			if quotes[i].TotalPrice < bestQuote.TotalPrice {
				bestQuote = &quotes[i]
			}
		}
		response.BestQuote = bestQuote
	}

	return response
}

// validateRequest validates the rate calculation request
func (s *Service) validateRequest(request *dtos.RateCalculationRequest) error {
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	if len(request.Packages) == 0 {
		return fmt.Errorf("at least one package is required")
	}

	if request.OriginCity == "" {
		return fmt.Errorf("origin postal code is required")
	}

	if request.DestCity == "" {
		return fmt.Errorf("destination postal code is required")
	}

	// Validate postal codes are 6 digits (Indian format)
	if len(request.OriginCity) != 6 {
		return fmt.Errorf("origin postal code must be 6 digits")
	}

	if len(request.DestCity) != 6 {
		return fmt.Errorf("destination postal code must be 6 digits")
	}

	return nil
}

