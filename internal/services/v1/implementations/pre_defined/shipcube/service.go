package shipcube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// Service implements real-time rate fetching for ShipCube
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    *http.Client
	config        *Config
	isInitialized bool
}

// NewDefaultConfig returns default configuration
func NewDefaultConfig() *Config {
	return &Config{
		BaseURL:  getEnv("UNIFIED_CALCULATE_RATE_URL", ""),
		TenantID: getEnv("SHIPCUBE_TENANT_ID", ""),

	}
}

// LoadFromMap loads configuration from a map
func (c *Config) LoadFromMap(config map[string]interface{}) error {
	if baseURL, ok := config["base_url"].(string); ok {
		c.BaseURL = baseURL
	}
	if tenantID, ok := config["tenant_id"].(string); ok {
		c.TenantID = tenantID
	}
	// if apiKey, ok := config["api_key"].(string); ok {
	// 	c.APIKey = apiKey
	// }
	return nil
}

// NewService creates a new ShipCube service instance
func NewService(
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	httpClient interfaces.HTTPClient,
) *Service {
	// Create a standard http.Client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &Service{
		logger:        logger,
		metrics:       metrics,
		httpClient:    client,
		config:        NewDefaultConfig(),
		isInitialized: true,
	}
}

// GetImplementationType returns the implementation type
func (s *Service) GetImplementationType() dtos.ProviderType {
	return dtos.ProviderTypeRealTime
}

// GetImplementationName returns the implementation name
func (s *Service) GetImplementationName() string {
	return "ShipCube"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
	s.logger.Info("Initializing ShipCube service", "config_keys", len(config))

	// Override defaults with provided config
	if err := s.config.LoadFromMap(config); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate required configuration
	if s.config.TenantID == "" {
		return fmt.Errorf("ShipCube tenant ID is required")
	}

	s.isInitialized = true
	s.logger.Info("ShipCube service initialized successfully", 
		"base_url", s.config.BaseURL,
		"tenant_id", s.config.TenantID)
	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       s.config.BaseURL,
		"tenant_id":      s.config.TenantID,
		"is_initialized": s.isInitialized,
		"provider_type":  "real_time",
	}
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.logger.Info("Closing ShipCube service")
	s.isInitialized = false
	return nil
}

// GetRates fetches rates from ShipCube API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	s.logger.Info("INSIDE SHIPCUBE SERVICE =======================")

	if !s.isInitialized {
		return nil, fmt.Errorf("ShipCube service not initialized")
	}

	// Validate request before making API call
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	startTime := time.Now()
	s.logger.Info("Fetching rates from ShipCube API",
		"request_id", request.RequestID,
		"origin", request.OriginCity,
		"origin_country", request.OriginCountry,
		"destination", request.DestCity,
		"dest_country", request.DestCountry,
		"weight", request.Weight,
		"packages", len(request.Packages))

	// Convert our request to ShipCube format
	shipCubeRequest, err := s.convertToShipCubeRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	s.logger.Info("ShipCube Request", "request", shipCubeRequest)

	// Make API call
	shipCubeResponse, err := s.callShipCubeAPI(ctx, shipCubeRequest)
	if err != nil {
		s.metrics.IncrementCounter("shipcube_api_error", map[string]string{
			"error_type": "api_call_failed",
		})
		return nil, fmt.Errorf("ShipCube API call failed: %w", err)
	}

	// Convert ShipCube response to our format
	response := s.convertFromShipCubeResponse(shipCubeResponse, request, time.Since(startTime))

	s.metrics.IncrementCounter("shipcube_api_success", map[string]string{
		"total_quotes": fmt.Sprintf("%d", len(response.Quotes)),
	})
	s.metrics.RecordTimer("shipcube_api_duration", time.Since(startTime), map[string]string{
		"endpoint": "get_rates",
	})

	return response, nil
}

// convertToShipCubeRequest converts our request format to ShipCube API format
func (s *Service) convertToShipCubeRequest(req *dtos.RateCalculationRequest) (*ShipCubeRateRequest, error) {
	if len(req.Packages) == 0 {
		return nil, fmt.Errorf("at least one package is required")
	}

	// Use first package for weight (ShipCube API uses weight in grams)
	pkg := req.Packages[0]
	
	// Convert weight to grams
	weightGrams := utils.ConvertWeightToGrams(pkg.Weight, pkg.WeightUnit)

	shipCubeRequest := &ShipCubeRateRequest{
		ProductType: "standard",
		ServiceType: "weight_based_rates",
		Location:    "INDIA",
		Filters:     map[string]interface{}{},
		Dimensions: ShipCubeDimensions{
			Weight: weightGrams,
		},
		UserOptions:          map[string]interface{}{},
		IncludeDefaultCharges: false,
	}

	s.logger.Debug("Converted to ShipCube request",
		"product_type", shipCubeRequest.ProductType,
		"service_type", shipCubeRequest.ServiceType,
		"location", shipCubeRequest.Location,
		"weight_grams", weightGrams)

	return shipCubeRequest, nil
}

// callShipCubeAPI makes the actual API call to ShipCube
func (s *Service) callShipCubeAPI(ctx context.Context, request *ShipCubeRateRequest) (*ShipCubeRateResponse, error) {
	endpoint := s.config.BaseURL + "/calculate"

	// Convert request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	s.logger.Debug("ShipCube API Request",
		"endpoint", endpoint,
		"request_body", string(requestBody))

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Tenant-Id", s.config.TenantID)

	// Add API key if available
	// if s.config.APIKey != "" {
	// 	req.Header.Set("Authorization", "Bearer "+s.config.APIKey)
	// }

	// Make the request using the concrete http.Client
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	s.logger.Debug("ShipCube API Response",
		"status_code", resp.StatusCode,
		"body", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ShipCube API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response into our struct
	var response ShipCubeRateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		s.logger.Error("Failed to unmarshal ShipCube response", "error", err, "body", string(body))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	s.logger.Info("Successfully parsed ShipCube API response", 
		"status", response.Status,
		"message", response.Message,
		"total_amount", response.Data.TotalAmount)
	
	return &response, nil
}

// convertFromShipCubeResponse converts ShipCube response to our format
func (s *Service) convertFromShipCubeResponse(
	shipCubeResp *ShipCubeRateResponse,
	originalReq *dtos.RateCalculationRequest,
	responseTime time.Duration,
) *dtos.RateCalculationResponse {

	response := &dtos.RateCalculationResponse{
		RequestID:    originalReq.RequestID,
		Status:       "success",
		Message:      "Rates retrieved from ShipCube",
		Quotes:       []dtos.RateQuote{},
		ResponseTime: responseTime.Milliseconds(),
		CacheHit:     false,
		Timestamp:    time.Now(),
	}

	// Check if the response was successful
	if shipCubeResp.Status != "success" || shipCubeResp.Data.TotalAmount <= 0 {
		response.Status = "error"
		response.Message = fmt.Sprintf("No valid rates found in ShipCube response: %s", shipCubeResp.Message)
		return response
	}

	s.logger.Info("Processing ShipCube rate response", 
		"total_amount", shipCubeResp.Data.TotalAmount,
		"base_rate", shipCubeResp.Data.BaseRate)

	// Create a single quote from the ShipCube response
	// Since ShipCube returns a single rate, we create one quote
	serviceType := "Standard"
	currency := "INR" 

	quote := dtos.RateQuote{
		QuoteID:         fmt.Sprintf("shipcube_%s_%d", strings.ToLower(serviceType), 0),
		PartnerID:       "83c5a4ac-b297-466a-9b14-9f2602103737",
		PartnerName:     "ShipCube",
		ProviderType:    dtos.ProviderTypeRealTime,
		BasePrice:       shipCubeResp.Data.BaseRate,
		TotalPrice:      shipCubeResp.Data.TotalAmount,
		Currency:        currency,
		ServiceType:     serviceType,
		ServiceLevel:    shipCubeResp.Data.Request.ServiceType,
		ValidUntil:      time.Now().Add(24 * time.Hour), // 24 hour validity
		Confidence:      0.90,
		IsRecommended:   true,
		Source:          "predefined",
		ResponseTimeMs:  responseTime.Milliseconds(),
		ExternalQuoteID: "shipcube_rate",
	}

	s.logger.Debug("ShipCube rate quote",
		"service_type", serviceType,
		"base_price", shipCubeResp.Data.BaseRate,
		"total_price", shipCubeResp.Data.TotalAmount,
		"currency", currency)

	response.Quotes = append(response.Quotes, quote)
	response.TotalQuotes = len(response.Quotes)
	
	if response.TotalQuotes > 0 {
		response.BestQuote = &response.Quotes[0]
		s.logger.Info("Successfully processed ShipCube rates", 
			"total_quotes", response.TotalQuotes,
			"best_quote_price", response.BestQuote.TotalPrice,
			"best_quote_currency", response.BestQuote.Currency,
			"best_quote_service", response.BestQuote.ServiceType)
	} else {
		s.logger.Warn("No valid quotes found in ShipCube response")
		response.Status = "partial_success"
		response.Message = "ShipCube API returned data but no valid pricing"
	}

	return response
}
// validateRequest validates the rate calculation request
func (s *Service) validateRequest(request *dtos.RateCalculationRequest) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}
	
	if len(request.Packages) == 0 {
		return fmt.Errorf("at least one package is required")
	}
	
	if request.OriginCountry == "" {
		return fmt.Errorf("origin country is required")
	}
	
	if request.DestCountry == "" {
		return fmt.Errorf("destination country is required")
	}
	
	return nil
}

// IsHealthy performs health check for ShipCube API
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("ShipCube service not initialized")
	}

	s.logger.Info("Performing ShipCube API health check")

	// Create a minimal test request for health check
	testRequest := &ShipCubeRateRequest{
		ProductType: "standard",
		ServiceType: "weight_based_rates",
		Location:    "INDIA",
		Filters:     map[string]interface{}{},
		Dimensions: ShipCubeDimensions{
			Weight: 1000, // 1kg in grams
		},
		UserOptions:          map[string]interface{}{},
		IncludeDefaultCharges: false,
	}

	// Try to make a health check call with shorter timeout
	healthCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	startTime := time.Now()
	_, err := s.callShipCubeAPI(healthCtx, testRequest)

	s.metrics.RecordTimer("shipcube_health_check_duration", time.Since(startTime), map[string]string{
		"status": func() string {
			if err != nil {
				return "failed"
			}
			return "success"
		}(),
	})

	if err != nil {
		s.logger.Warn("ShipCube API health check failed", "error", err)
		return fmt.Errorf("ShipCube API health check failed: %w", err)
	}

	s.logger.Info("ShipCube API health check successful")
	return nil
}