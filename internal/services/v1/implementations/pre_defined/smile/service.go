package smile

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// Service implements Smile Rate API rate fetching for Prayog platform
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    interfaces.HTTPClient
	config        *Config
	authService   *AuthService
	isInitialized bool
}

// NewService creates a new smile service instance
func NewService(
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	httpClient interfaces.HTTPClient,
) *Service {
	return &Service{
		logger:      logger,
		metrics:     metrics,
		httpClient:  httpClient,
		config:      NewDefaultConfig(),
		authService: NewAuthService(httpClient, logger),
	}
}

// GetImplementationType returns the implementation type
func (s *Service) GetImplementationType() dtos.ProviderType {
	return dtos.ProviderTypePreDefined
}

// GetImplementationName returns the implementation name
func (s *Service) GetImplementationName() string {
	return "Smile Rate"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
	s.logger.Info("Initializing Smile Rate service")

	// Update configuration from provided config
	if err := s.config.UpdateFromMap(config); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	// Validate configuration
	if err := s.config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Update auth service config if provided, otherwise use defaults from NewAuthService
	// Auth config can be provided at root level or nested under "auth"
	authConfigMap := make(map[string]interface{})
	if authConfig, ok := config["auth"].(map[string]interface{}); ok {
		authConfigMap = authConfig
	} else {
		// Check if auth fields are at root level
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
	}
	
	// Always pass tenant ID from service config to auth service
	if s.config.TenantID != "" {
		authConfigMap["tenant_id"] = s.config.TenantID
	}
	
	// Only update auth config if we have auth-related fields
	if len(authConfigMap) > 0 {
		if err := s.authService.UpdateConfig(authConfigMap); err != nil {
			s.logger.Warn("Failed to update auth configuration", "error", err)
		}
	}

	// Set HTTP client timeout
	s.httpClient.SetTimeout(time.Duration(s.config.TimeoutMs) * time.Millisecond)
	s.httpClient.SetRetryCount(s.config.RetryCount)

	s.isInitialized = true

	s.logger.Info("Smile Rate service initialized successfully",
		"base_url", s.config.BaseURL,
		"timeout_ms", s.config.TimeoutMs,
		"retry_count", s.config.RetryCount,
		"tenant_id", s.config.TenantID,
		"has_bearer_token", s.config.BearerToken != "")

	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"provider_type":     "pre_defined",
		"base_url":          s.config.BaseURL,
		"timeout_ms":        s.config.TimeoutMs,
		"retry_count":       s.config.RetryCount,
		"cache_ttl_minutes": s.config.CacheTTLMinutes,
		"tenant_id":         s.config.TenantID,
		"is_initialized":    s.isInitialized,
	}
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.isInitialized = false
	s.logger.Info("Smile Rate service closed")
	return nil
}

// GetRates fetches rates using Smile Rate API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	startTime := time.Now()

	if !s.isInitialized {
		return nil, fmt.Errorf("smile service not initialized")
	}

	s.logger.Info("Fetching rates from Smile Rate API",
		"request_id", request.RequestID,
		"origin", request.OriginCity,
		"destination", request.DestCity,
		"weight", request.Weight,
		"service_type", request.ServiceType)

	// Smile supports SURFACE and AIR service types
	// We need to fetch rates for both service types
	serviceTypes := []string{"SURFACE", "AIR"}

	var allQuotes []dtos.RateQuote
	var errors []dtos.ProviderError

	// Fetch rates for both SURFACE and AIR
	for quoteIndex, serviceType := range serviceTypes {
		// Create a copy of the request with the specific service type
		serviceRequest := s.createServiceRequest(request, serviceType)
		
		// Convert our request to smile API format
		smileRequest, err := s.convertToSmileAPIRequest(serviceRequest, serviceType)
		if err != nil {
			s.logger.Warn("Failed to convert request to smile API format",
				"service_type", serviceType,
				"error", err)
			errors = append(errors, dtos.ProviderError{
				PartnerID:    "smile",
				PartnerName:  "Smile Rate",
				ErrorCode:    "CONVERSION_ERROR",
				ErrorMessage: fmt.Sprintf("Failed to convert request for %s: %v", serviceType, err),
				Timestamp:    time.Now(),
			})
			continue
		}

		// Call smile API
		requestJSON, _ := json.Marshal(smileRequest)
		s.logger.Debug("Calling smile API", 
			"service_type", serviceType, 
			"request", string(requestJSON))
		smileResponse, err := s.callSmileAPI(ctx, smileRequest)
		if err != nil {
			s.logger.Warn("Smile API call failed",
				"service_type", serviceType,
				"error", err)
			errors = append(errors, dtos.ProviderError{
				PartnerID:    "smile",
				PartnerName:  "Smile Rate",
				ErrorCode:    "API_CALL_FAILED",
				ErrorMessage: fmt.Sprintf("API call failed for %s: %v", serviceType, err),
				Timestamp:    time.Now(),
			})
			continue
		}

		// Convert smile API response to our format
		quotes := s.convertFromSmileAPIResponse(smileResponse, request, serviceType, quoteIndex, time.Since(startTime))
		allQuotes = append(allQuotes, quotes...)
	}

	// Create response with all quotes
	response := &dtos.RateCalculationResponse{
		RequestID:    request.RequestID,
		Status:       "success",
		Message:      "Rates retrieved from Smile Rate API",
		Quotes:       allQuotes,
		TotalQuotes:  len(allQuotes),
		ResponseTime: time.Since(startTime).Milliseconds(),
		CacheHit:     false,
		Timestamp:    time.Now(),
		Errors:       errors,
	}

	// Set best quote (first one if available)
	if len(allQuotes) > 0 {
		response.BestQuote = &allQuotes[0]
	}

	// Record metrics
	s.metrics.IncrementCounter("smile_api_calculation_success", map[string]string{
		"quotes_found": fmt.Sprintf("%d", len(allQuotes)),
	})
	s.metrics.RecordTimer("smile_api_calculation_time", time.Since(startTime), map[string]string{
		"service_type": request.ServiceType,
	})

	s.logger.Info("Smile API rate calculation completed",
		"request_id", request.RequestID,
		"quotes_found", len(allQuotes),
		"duration_ms", response.ResponseTime)

	return response, nil
}

// IsHealthy performs health check for smile API service
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("smile service not initialized")
	}

	// Perform a simple health check by calling a lightweight endpoint
	return s.performHealthCheck(ctx)
}

// RefreshRates refreshes the rate cache (for pre-defined implementation interface)
func (s *Service) RefreshRates(ctx context.Context) error {
	s.logger.Info("Refreshing smile API service cache")
	// For smile API, we don't maintain local cache as it's real-time
	// But we can validate the API connectivity
	return s.IsHealthy(ctx)
}

// createServiceRequest creates a service request for a specific service type
func (s *Service) createServiceRequest(req *dtos.RateCalculationRequest, smileServiceType string) *dtos.RateCalculationRequest {
	// Create a copy of the request
	serviceRequest := *req
	// The service type mapping will be handled in convertToSmileAPIRequest
	return &serviceRequest
}

// convertToSmileAPIRequest converts our request format to smile API format
func (s *Service) convertToSmileAPIRequest(req *dtos.RateCalculationRequest, smileServiceType string) (*SmileRateRequest, error) {
	// Convert origin and destination cities (pincodes) to integers
	fromPincode, err := strconv.Atoi(req.OriginCity)
	if err != nil {
		return nil, fmt.Errorf("invalid origin pincode: %s", req.OriginCity)
	}

	toPincode, err := strconv.Atoi(req.DestCity)
	if err != nil {
		return nil, fmt.Errorf("invalid destination pincode: %s", req.DestCity)
	}

	// Use the provided smile service type (SURFACE or AIR)
	serviceType := smileServiceType

	// Get dimensions from first package or use defaults
	var length, width, height float64 = 1.0, 1.0, 1.0
	if len(req.Packages) > 0 {
		pkg := req.Packages[0]
		length = pkg.Length
		width = pkg.Width
		height = pkg.Height
		// Convert dimensions to cm if needed
		if pkg.DimUnit != "cm" {
			switch pkg.DimUnit {
			case "m":
				length *= 100
				width *= 100
				height *= 100
			case "mm":
				length /= 10
				width /= 10
				height /= 10
			case "in":
				length *= 2.54
				width *= 2.54
				height *= 2.54
			}
		}
	}

	// Convert weight to kg if needed
	weight := req.Weight
	if len(req.Packages) > 0 && req.Packages[0].WeightUnit != "kg" {
		switch req.Packages[0].WeightUnit {
		case "g":
			weight /= 1000
		case "lb":
			weight *= 0.453592
		}
	}

	// Build user options - COD should always be off for Smile
	userOptions := &UserOptionsRequest{
		COD: false, // COD is always disabled for Smile
	}

	if req.InsuranceRequired {
		userOptions.Insurance = &InsuranceOption{
			Enabled: true,
			Amount:  0, // Will be calculated by API
		}
	}

	// Get product type from config, default to empty string if not configured
	productType := s.config.DefaultProductType
	
	// Get rate card ID from config or use default from mapping
	rateCardID := s.config.RateCardID
	if rateCardID == "" {
		rateCardID = GetRateCardID("smile_ecomm")
	}
	
	smileRequest := &SmileRateRequest{
		RateCardID:          rateCardID,
		FromPincode:         fromPincode,
		ToPincode:           toPincode,
		ServiceType:         serviceType,
		ProductType:         productType,
		Weight:              weight,
		Length:              length,
		Height:              height,
		Width:               width,
		IncludeDefaultCharges: false,
		UserOptions:         userOptions,
	}

	return smileRequest, nil
}

// convertFromSmileAPIResponse converts smile API response to our format
func (s *Service) convertFromSmileAPIResponse(
	smileResp *SmileRateResponse,
	originalReq *dtos.RateCalculationRequest,
	smileServiceType string,
	quoteIndex int,
	responseTime time.Duration,
) []dtos.RateQuote {
	quotes := []dtos.RateQuote{}

	if !smileResp.Success() {
		s.logger.Warn("Smile API response not successful",
			"status", smileResp.Status,
			"message", smileResp.Message,
			"service_type", smileServiceType)
		return quotes
	}

	if smileResp.Data == nil {
		s.logger.Warn("Smile API response data is nil",
			"status", smileResp.Status,
			"message", smileResp.Message,
			"service_type", smileServiceType)
		return quotes
	}

	// Smile returns a flat response structure, so we create a single quote per response
	// Map service type and get description
	serviceType := s.mapServiceTypeFromSmile(smileServiceType)
	description := s.getServiceDescription(smileServiceType)

	quote := dtos.RateQuote{
		QuoteID:         fmt.Sprintf("smile_%s_%d_%d", smileServiceType, quoteIndex, time.Now().UnixNano()),
		PartnerID:       "smile",
		PartnerName:     "Smile Rate",
		ProviderType:    dtos.ProviderTypePreDefined,
		BasePrice:       smileResp.Data.BaseRate,
		TotalPrice:      smileResp.Data.TotalAmount,
		Currency:        "INR", // Default currency for Smile
		ServiceType:     serviceType,
		ServiceLevel:    smileServiceType,
		Description:     description,
		ValidUntil:      time.Now().Add(24 * time.Hour), // 24 hour validity
		Confidence:      0.95,
		IsRecommended:   quoteIndex == 0, // First quote (SURFACE) is recommended
		Source:          "pre_defined",
		ResponseTimeMs:  responseTime.Milliseconds(),
		PriceBreakdown:  s.convertPriceBreakdown(smileResp.Data.Charges),
	}

	// Get estimated days from calculation details if available
	if smileResp.Data.Calculation != nil {
		// Estimated days can be inferred from service type
		if smileServiceType == "SURFACE" {
			quote.EstimatedDays = 3 // Standard delivery
		} else if smileServiceType == "AIR" {
			quote.EstimatedDays = 1 // Express delivery
		}
	}

	quotes = append(quotes, quote)
	return quotes
}

// callSmileAPI makes the actual API call to smile rate calculation endpoint
func (s *Service) callSmileAPI(ctx context.Context, request *SmileRateRequest) (*SmileRateResponse, error) {
	// Get bearer token - prefer static token from config, otherwise use auth service
	var bearerToken string
	var tenantID string
	var err error
	
	if s.config.BearerToken != "" {
		// Use static bearer token from config
		bearerToken = s.config.BearerToken
		if !strings.HasPrefix(bearerToken, "Bearer ") {
			bearerToken = "Bearer " + bearerToken
		}
		s.logger.Debug("Using static bearer token from config")
		tenantID = s.config.TenantID // Use tenant ID from config when using static token
	} else {
		// Try to get token from auth service
		bearerToken, err = s.authService.GetBearerToken(ctx)
		if err != nil {
			s.logger.Error("Failed to get bearer token from auth service", "error", err)
			return nil, fmt.Errorf("authentication required: failed to get bearer token: %w", err)
		}
		s.logger.Debug("Using bearer token from auth service")
		
		// Always use tenant ID from config (not from auth token) for smile service
		// This allows using a specific tenant ID regardless of which user authenticated
		tenantID = s.config.TenantID
		if tenantID == "" {
			// Fallback to token tenant ID only if config doesn't have one
			if tokenInfo := s.authService.GetTokenInfo(); tokenInfo != nil && tokenInfo.TenantID != "" {
				tenantID = tokenInfo.TenantID
				s.logger.Debug("Using tenant ID from auth token as fallback", "tenant_id", tenantID)
			}
		} else {
			s.logger.Debug("Using tenant ID from config", "tenant_id", tenantID)
		}
	}

	// Ensure we have a bearer token
	if bearerToken == "" {
		return nil, fmt.Errorf("authentication required: bearer token is not configured")
	}

	// Prepare headers
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json, text/plain, */*",
		"Authorization": bearerToken,
	}

	// Add tenant ID if available (from token or config) - use X-Tenant-Id header
	if tenantID != "" {
		headers["X-Tenant-Id"] = tenantID
	}

	// Make HTTP request to the rate calculation endpoint
	url := fmt.Sprintf("%s%s", s.config.BaseURL, s.config.CalculateRatesEndpoint)

	s.logger.Debug("Calling smile API",
		"url", url,
		"method", "POST",
		"tenant_id", tenantID)

	httpResponse, err := s.httpClient.Post(ctx, url, request, headers)
	if err != nil {
		s.logger.Error("HTTP request to smile API failed", "error", err, "url", url)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if httpResponse.StatusCode != 200 {
		s.logger.Error("Smile API returned non-200 status",
			"status_code", httpResponse.StatusCode,
			"response_body", string(httpResponse.Body),
			"url", url)
		return nil, fmt.Errorf("smile API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response
	var response SmileRateResponse
	if err := json.Unmarshal(httpResponse.Body, &response); err != nil {
		s.logger.Error("Failed to unmarshal Smile API response", 
			"error", err,
			"response_body", string(httpResponse.Body))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// performHealthCheck performs a simple health check
func (s *Service) performHealthCheck(ctx context.Context) error {
	// Create a minimal test request to check API connectivity
	rateCardID := s.config.RateCardID
	if rateCardID == "" {
		rateCardID = GetRateCardID("smile_ecomm")
	}
	
	testRequest := &SmileRateRequest{
		RateCardID:          rateCardID,
		FromPincode:         411015,
		ToPincode:           411001,
		ServiceType:         "SURFACE",
		ProductType:         s.config.DefaultProductType,
		Weight:              1.0,
		Length:              1.0,
		Height:              1.0,
		Width:               1.0,
		IncludeDefaultCharges: false,
		UserOptions:         &UserOptionsRequest{
			Insurance: &InsuranceOption{
				Enabled: false,
				Amount:  0,
			},
			COD: false,
		},
	}

	// Try to make a health check call (with shorter timeout)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.callSmileAPI(ctx, testRequest)
	return err
}

// Helper methods

// mapServiceTypeToSmile maps our service type to smile API service type
// Note: For Smile, we use SURFACE and AIR directly
func (s *Service) mapServiceTypeToSmile(serviceType string) string {
	switch serviceType {
	case "standard", "SURFACE":
		return "SURFACE"
	case "premium", "AIR":
		return "AIR"
	case "express":
		return "AIR"
	case "same_day":
		return "AIR"
	default:
		return "SURFACE"
	}
}

// mapServiceTypeFromSmile maps smile API service type back to our format
func (s *Service) mapServiceTypeFromSmile(serviceType string) string {
	switch serviceType {
	case "SURFACE":
		return "standard"
	case "AIR":
		return "premium"
	default:
		return "standard"
	}
}

// convertPriceBreakdown converts charge details to price breakdown
func (s *Service) convertPriceBreakdown(charges []ChargeDetail) dtos.PriceBreakdown {
	breakdown := dtos.PriceBreakdown{}

	for _, charge := range charges {
		chargeCode := charge.ChargeCode()
		switch chargeCode {
		case "FUEL_SURCHARGE":
			breakdown.FuelSurcharge += charge.Amount
		case "GST":
			breakdown.TaxAmount += charge.Amount
		case "INSURANCE_CHARGES":
			breakdown.InsuranceCharge += charge.Amount
		case "COD_CHARGE":
			breakdown.HandlingCharge += charge.Amount
		case "DCC_DELIVERY", "KYC_DELIVERY":
			breakdown.HandlingCharge += charge.Amount
		default:
			// Add other charges to handling charge
			breakdown.HandlingCharge += charge.Amount
		}
		breakdown.TotalPrice += charge.Amount
	}

	return breakdown
}

// getServiceDescription returns a description for the service type
func (s *Service) getServiceDescription(serviceType string) string {
	switch serviceType {
	case "SURFACE":
		return "Standard Delivery"
	case "AIR":
		return "Express Delivery"
	default:
		return "Standard Delivery"
	}
}

