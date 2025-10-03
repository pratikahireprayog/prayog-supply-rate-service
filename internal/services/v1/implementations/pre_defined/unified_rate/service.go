package unified_rate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// Service implements unified API rate fetching for Prayog platform
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    interfaces.HTTPClient
	config        *Config
	rateCardRepo  interfaces.UnifiedRateCardRepository
	isInitialized bool
}

// NewService creates a new unified service instance
func NewService(
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	httpClient interfaces.HTTPClient,
	rateCardRepo interfaces.UnifiedRateCardRepository,
) *Service {
	return &Service{
		logger:       logger,
		metrics:      metrics,
		httpClient:   httpClient,
		rateCardRepo: rateCardRepo,
		config:       NewDefaultConfig(),
	}
}

// GetImplementationType returns the implementation type
func (s *Service) GetImplementationType() dtos.ProviderType {
	return dtos.ProviderTypePreDefined
}

// GetImplementationName returns the implementation name
func (s *Service) GetImplementationName() string {
	return "Unified Rate"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
	s.logger.Info("Initializing Unified Rate service")

	// Update configuration from provided config
	if err := s.config.UpdateFromMap(config); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	// Set HTTP client timeout
	s.httpClient.SetTimeout(time.Duration(s.config.TimeoutMs) * time.Millisecond)
	s.httpClient.SetRetryCount(s.config.RetryCount)

	s.isInitialized = true

	s.logger.Info("Unified Rate service initialized successfully",
		"base_url", s.config.BaseURL,
		"timeout_ms", s.config.TimeoutMs,
		"retry_count", s.config.RetryCount)

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
		"is_initialized":    s.isInitialized,
	}
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.isInitialized = false
	s.logger.Info("Unified Rate service closed")
	return nil
}

// GetRates fetches rates using Unified Rate API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	startTime := time.Now()

	if !s.isInitialized {
		return nil, fmt.Errorf("unified service not initialized")
	}

	s.logger.Info("Fetching rates from Unified Rate API",
		"request_id", request.RequestID,
		"origin", request.OriginCity,
		"destination", request.DestCity,
		"weight", request.Weight,
		"service_type", request.ServiceType)

	// Convert our request to unified API format
	unifiedRequest, err := s.convertToUnifiedAPIRequest(request)
	if err != nil {
		s.metrics.IncrementCounter("unified_api_convert_request_failed", map[string]string{
			"error": "request_conversion",
		})
		return nil, fmt.Errorf("failed to convert request to unified API format: %w", err)
	}

	// Call unified API with partner code - for unified rate, we use "unified" as default
	partnerCode := "unified"

	unifiedResponse, err := s.callUnifiedAPI(ctx, unifiedRequest, partnerCode)
	if err != nil {
		s.metrics.IncrementCounter("unified_api_call_failed", map[string]string{
			"error": "api_call",
		})
		return nil, fmt.Errorf("unified API call failed: %w", err)
	}

	// Convert unified API response to our format
	response := s.convertFromUnifiedAPIResponse(unifiedResponse, request, time.Since(startTime))

	// Record metrics
	s.metrics.IncrementCounter("unified_api_calculation_success", map[string]string{
		"quotes_found": fmt.Sprintf("%d", len(response.Quotes)),
	})
	s.metrics.RecordTimer("unified_api_calculation_time", time.Since(startTime), map[string]string{
		"service_type": request.ServiceType,
	})

	s.logger.Info("Unified API rate calculation completed",
		"request_id", request.RequestID,
		"quotes_found", len(response.Quotes),
		"duration_ms", response.ResponseTime)

	return response, nil
}

// IsHealthy performs health check for unified API service
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("unified service not initialized")
	}

	// Perform a simple health check by calling a lightweight endpoint
	return s.performHealthCheck(ctx)
}

// RefreshRates refreshes the rate cache (for pre-defined implementation interface)
func (s *Service) RefreshRates(ctx context.Context) error {
	s.logger.Info("Refreshing unified API service cache")
	// For unified API, we don't maintain local cache as it's real-time
	// But we can validate the API connectivity
	return s.IsHealthy(ctx)
}

// convertToUnifiedAPIRequest converts our request format to unified API format
func (s *Service) convertToUnifiedAPIRequest(req *dtos.RateCalculationRequest) (*UnifiedRateRequest, error) {
	// Extract location information from postal codes
	sourceLocation := &LocationInfo{PostalCode: req.OriginCity, CountryCode: "IN"}
	destLocation := &LocationInfo{PostalCode: req.DestCity, CountryCode: "IN"}

	// Convert weight to grams (unified API uses grams)
	weightInGrams := s.convertWeightToGrams(req.Weight, "kg") // Assuming input is in kg

	unifiedRequest := &UnifiedRateRequest{
		SourceLocation: LocationRequest{
			PostalCode:  sourceLocation.PostalCode,
			CountryCode: sourceLocation.CountryCode,
		},
		DestinationLocation: LocationRequest{
			PostalCode:  destLocation.PostalCode,
			CountryCode: destLocation.CountryCode,
		},
		Packages: []PackageRequest{
			{
				Weight: WeightRequest{
					Value: weightInGrams,
					Unit:  "GRAMS",
				},
				Dimensions: DimensionsRequest{
					Length: 10, // Default dimensions
					Width:  10,
					Height: 10,
					Unit:   "cm",
				},
			},
		},
		ServiceTypes: s.mapServiceTypesToUnified(req.ServiceType),
		Currency:     req.Currency,
		Metadata: map[string]interface{}{
			"request_id":    req.RequestID,
			"pickup_date":   req.PickupDate.Format(time.RFC3339),
			"delivery_date": req.DeliveryDate.Format(time.RFC3339),
			"priority":      req.Priority,
			"source":        req.Source,
		},
	}

	return unifiedRequest, nil
}

// convertFromUnifiedAPIResponse converts unified API response to our format
func (s *Service) convertFromUnifiedAPIResponse(
	unifiedResp *UnifiedRateResponse,
	originalReq *dtos.RateCalculationRequest,
	responseTime time.Duration,
) *dtos.RateCalculationResponse {
	response := &dtos.RateCalculationResponse{
		RequestID:    originalReq.RequestID,
		Status:       "success",
		Message:      "Rates retrieved from Unified Rate API",
		Quotes:       []dtos.RateQuote{},
		ResponseTime: responseTime.Milliseconds(),
		CacheHit:     false,
		Timestamp:    time.Now(),
	}

	if !unifiedResp.Success {
		response.Status = "error"
		response.Message = unifiedResp.Message
		return response
	}

	// Convert rate calculations to quotes
	for i, rateCalc := range unifiedResp.Data.RateCalculations {
		for j, serviceRate := range rateCalc.ServiceRates {
			quote := dtos.RateQuote{
				QuoteID:         fmt.Sprintf("unified_%s_%d_%d", rateCalc.ServiceType, i, j),
				PartnerID:       "unified_api",
				PartnerName:     "Unified Rate",
				ProviderType:    dtos.ProviderTypePreDefined,
				BasePrice:       serviceRate.BaseRate,
				TotalPrice:      serviceRate.TotalRate,
				Currency:        serviceRate.Currency,
				ServiceType:     rateCalc.ServiceType,
				ServiceLevel:    serviceRate.ServiceName,
				EstimatedDays:   serviceRate.EstimatedDays,
				ValidUntil:      time.Now().Add(24 * time.Hour), // 24 hour validity
				Confidence:      0.95,
				IsRecommended:   i == 0 && j == 0, // First quote as recommended
				Source:          "pre_defined",
				ResponseTimeMs:  responseTime.Milliseconds(),
				ExternalQuoteID: serviceRate.RateID,
				PriceBreakdown:  s.convertPriceBreakdown(serviceRate.Charges),
			}

			response.Quotes = append(response.Quotes, quote)
		}
	}

	response.TotalQuotes = len(response.Quotes)
	return response
}

// callUnifiedAPI makes the actual API call to unified rate calculation endpoint
func (s *Service) callUnifiedAPI(ctx context.Context, request *UnifiedRateRequest, partnerCode string) (*UnifiedRateResponse, error) {
	// Get API configuration for the partner
	_, apiKey, _, err := s.rateCardRepo.GetConfigByPartnerCode(ctx, partnerCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get partner configuration: %w", err)
	}

	// Prepare headers with API key authentication
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "*/*",
		"api-key":      apiKey,
	}

	// Make HTTP request to the rate calculation endpoint
	url := fmt.Sprintf("%s%s", s.config.BaseURL, s.config.CalculateRatesEndpoint)

	s.logger.Debug("Calling unified API",
		"url", url,
		"method", "POST",
		"partner_code", partnerCode)

	httpResponse, err := s.httpClient.Post(ctx, url, request, headers)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if httpResponse.StatusCode != 200 {
		return nil, fmt.Errorf("unified API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response
	var response UnifiedRateResponse
	if err := json.Unmarshal(httpResponse.Body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// performHealthCheck performs a simple health check
func (s *Service) performHealthCheck(ctx context.Context) error {
	// We can create a minimal test request to check API connectivity
	testRequest := &UnifiedRateRequest{
		SourceLocation: LocationRequest{
			PostalCode:  "560001",
			CountryCode: "IN",
		},
		DestinationLocation: LocationRequest{
			PostalCode:  "110001",
			CountryCode: "IN",
		},
		Packages: []PackageRequest{
			{
				Weight: WeightRequest{Value: 500, Unit: "GRAMS"},
				Dimensions: DimensionsRequest{
					Length: 10, Width: 10, Height: 10, Unit: "cm",
				},
			},
		},
		ServiceTypes: []string{"standard"},
		Currency:     "INR",
	}

	// Try to make a health check call (with shorter timeout)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use "unified" as default partner code for health check
	_, err := s.callUnifiedAPI(ctx, testRequest, "unified")
	return err
}

// Helper methods

func (s *Service) convertWeightToGrams(weight float64, unit string) float64 {
	switch unit {
	case "kg":
		return weight * 1000
	case "g", "grams":
		return weight
	case "lb":
		return weight * 453.592
	default:
		return weight // assume grams
	}
}

func (s *Service) mapServiceTypesToUnified(serviceType string) []string {
	switch serviceType {
	case "express":
		return []string{"express", "premium"}
	case "standard":
		return []string{"standard"}
	case "premium":
		return []string{"premium"}
	default:
		return []string{"standard", "express"}
	}
}

func (s *Service) convertPriceBreakdown(charges []ChargeDetail) dtos.PriceBreakdown {
	breakdown := dtos.PriceBreakdown{}

	for _, charge := range charges {
		switch charge.ChargeCode {
		case "BASE_RATE", "WEIGHT_HANDLING_FEE":
			breakdown.BasePrice += charge.Amount
		case "WEIGHT_CHARGE":
			breakdown.WeightCharge += charge.Amount
		case "DISTANCE_CHARGE":
			breakdown.DistanceCharge += charge.Amount
		case "FUEL_SURCHARGE":
			breakdown.FuelSurcharge += charge.Amount
		case "HANDLING_CHARGE", "FRAGILE_CHARGE", "SIGNATURE_CHARGE":
			breakdown.HandlingCharge += charge.Amount
		case "INSURANCE_CHARGES":
			breakdown.InsuranceCharge += charge.Amount
		case "GST":
			if charge.IsTax {
				breakdown.TaxAmount += charge.Amount
			}
		default:
			// Add other charges to base price
			breakdown.BasePrice += charge.Amount
		}
		breakdown.TotalPrice += charge.Amount
	}

	return breakdown
}
