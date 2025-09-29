package dhl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// Service implements real-time rate fetching for DHL Express
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    interfaces.HTTPClient
	config        *Config
	isInitialized bool
}

// NewService creates a new DHL service instance
func NewService(
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	httpClient interfaces.HTTPClient,
) *Service {
	return &Service{
		logger:        logger,
		metrics:       metrics,
		httpClient:    httpClient,
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
	return "DHL Express"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
	s.logger.Info("Initializing DHL service", "config_keys", len(config))

	// Override defaults with provided config
	if err := s.config.LoadFromMap(config); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s.isInitialized = true
	s.logger.Info("DHL service initialized successfully")
	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       s.config.BaseURL,
		"account_number": s.config.AccountNumber,
		"is_initialized": s.isInitialized,
		"provider_type":  "real_time",
	}
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.logger.Info("Closing DHL service")
	s.isInitialized = false
	return nil
}

// GetRates fetches rates from DHL API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("DHL service not initialized")
	}

	startTime := time.Now()
	s.logger.Info("Fetching rates from DHL API",
		"request_id", request.RequestID,
		"origin", request.OriginCity,
		"destination", request.DestCity,
		"weight", request.Weight)

	// Convert our request to DHL format
	dhlRequest, err := s.convertToDHLRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make API call
	dhlResponse, err := s.callDHLAPI(ctx, dhlRequest)
	if err != nil {
		s.metrics.IncrementCounter("dhl_api_error", map[string]string{
			"error_type": "api_call_failed",
		})
		return nil, fmt.Errorf("DHL API call failed: %w", err)
	}

	// Convert DHL response to our format
	response := s.convertFromDHLResponse(dhlResponse, request, time.Since(startTime))

	s.metrics.IncrementCounter("dhl_api_success", map[string]string{
		"total_quotes": fmt.Sprintf("%d", len(response.Quotes)),
	})
	s.metrics.RecordTimer("dhl_api_duration", time.Since(startTime), map[string]string{
		"endpoint": "get_rates",
	})

	return response, nil
}

// convertToDHLRequest converts our request format to DHL API format
func (s *Service) convertToDHLRequest(req *dtos.RateCalculationRequest) (map[string]interface{}, error) {
	// Map service type to DHL product codes
	productCode := s.mapServiceToProductCode(req.ServiceType)

	dhlRequest := map[string]interface{}{
		"customerDetails": map[string]interface{}{
			"shipperDetails": map[string]interface{}{
				"postalCode":  req.OriginCity,
				"cityName":    s.getCityNameFromPostalCode(req.OriginCity),
				"countryCode": "IN", // Default - should be extracted from request
			},
			"receiverDetails": map[string]interface{}{
				"postalCode":  req.DestCity,
				"cityName":    s.getCityNameFromPostalCode(req.DestCity),
				"countryCode": "CN", // Default - should be extracted from request
			},
		},
		"accounts": []map[string]interface{}{
			{
				"typeCode": "shipper",
				"number":   s.config.AccountNumber,
			},
		},
		"productsAndServices": []map[string]interface{}{
			{
				"productCode":      productCode,
				"localProductCode": productCode,
			},
		},
		"payerCountryCode":           "IN",
		"plannedShippingDateAndTime": req.PickupDate.Format("2006-01-02T15:04:05GMT-07:00"),
		"unitOfMeasurement":          "metric",
		"isCustomsDeclarable":        true,
		"estimatedDeliveryDate": map[string]interface{}{
			"isRequested": true,
			"typeCode":    "QDDC",
		},
		"returnStandardProductsOnly": true,
		"packages": []map[string]interface{}{
			{
				"weight": req.Weight,
				"dimensions": map[string]interface{}{
					"length": 30, // Default dimensions - should be from packages
					"width":  20,
					"height": 15,
				},
			},
		},
	}

	return dhlRequest, nil
}

// getCityNameFromPostalCode returns a city name for the postal code (simplified mapping)
func (s *Service) getCityNameFromPostalCode(postalCode string) string {
	// This is a simplified mapping - in production, use a proper postal code service
	cityMap := map[string]string{
		"560001": "Bangalore",
		"560086": "Bangalore",
		"110001": "New Delhi",
		"400001": "Mumbai",
		"600001": "Chennai",
		"700001": "Kolkata",
		"266001": "Qing Dao",
	}

	if city, exists := cityMap[postalCode]; exists {
		return city
	}

	return postalCode // Fallback to postal code itself
}

// mapServiceToProductCode maps our service types to DHL product codes
func (s *Service) mapServiceToProductCode(serviceType string) string {
	switch strings.ToLower(serviceType) {
	case "express":
		return "P" // EXPRESS WORLDWIDE
	case "standard":
		return "N" // DOMESTIC EXPRESS
	case "economy":
		return "U" // EXPRESS WORLDWIDE NONDOC
	default:
		return "P" // Default to EXPRESS WORLDWIDE
	}
}

// callDHLAPI makes the actual API call to DHL
func (s *Service) callDHLAPI(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	// Prepare headers
	headers := map[string]string{
		"accept":                           "application/json",
		"Authorization":                    fmt.Sprintf("Basic %s", s.config.Credentials),
		"Content-Type":                     "application/json",
		"x-version":                        "2.12.0",
		"Message-Reference":                uuid.New().String(),
		"Message-Reference-Date":           time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT"),
		"Plugin-Name":                      "PrayogRateService",
		"Plugin-Version":                   "1.0.0",
		"Shipping-System-Platform-Name":    "Prayog",
		"Shipping-System-Platform-Version": "1.0.0",
	}

	// Make HTTP request
	httpResponse, err := s.httpClient.Post(ctx, s.config.BaseURL+"?strictValidation=false", request, headers)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DHL API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(httpResponse.Body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

// convertFromDHLResponse converts DHL response to our format
func (s *Service) convertFromDHLResponse(
	dhlResp map[string]interface{},
	originalReq *dtos.RateCalculationRequest,
	responseTime time.Duration,
) *dtos.RateCalculationResponse {

	response := &dtos.RateCalculationResponse{
		RequestID:    originalReq.RequestID,
		Status:       "success",
		Message:      "Rates retrieved from DHL",
		Quotes:       []dtos.RateQuote{},
		ResponseTime: responseTime.Milliseconds(),
		CacheHit:     false,
		Timestamp:    time.Now(),
	}

	// Extract products from DHL response
	products, ok := dhlResp["products"].([]interface{})
	if !ok {
		response.Status = "error"
		response.Message = "No products found in DHL response"
		return response
	}

	for i, product := range products {
		productMap, ok := product.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract product details
		productName, _ := productMap["productName"].(string)
		productCode, _ := productMap["productCode"].(string)

		// Extract pricing
		var totalPrice float64
		var currency string = "INR" // Default
		if totalPriceArray, ok := productMap["totalPrice"].([]interface{}); ok && len(totalPriceArray) > 0 {
			if firstPrice, ok := totalPriceArray[0].(map[string]interface{}); ok {
				if price, ok := firstPrice["price"].(float64); ok {
					totalPrice = price
				}
				if curr, ok := firstPrice["priceCurrency"].(string); ok {
					currency = curr
				}
			}
		}

		// Extract delivery information
		var deliveryDays int
		if deliveryCap, ok := productMap["deliveryCapabilities"].(map[string]interface{}); ok {
			if transitDays, ok := deliveryCap["totalTransitDays"].(float64); ok {
				deliveryDays = int(transitDays)
			}
		}

		// Create rate quote
		quote := dtos.RateQuote{
			QuoteID:         fmt.Sprintf("dhl_%s_%d", strings.ToLower(productCode), i),
			PartnerID:       "dhl",
			PartnerName:     "DHL Express",
			ProviderType:    dtos.ProviderTypeRealTime,
			BasePrice:       totalPrice,
			TotalPrice:      totalPrice,
			Currency:        currency,
			ServiceType:     s.mapProductCodeToServiceType(productCode),
			ServiceLevel:    productName,
			EstimatedDays:   deliveryDays,
			ValidUntil:      time.Now().Add(24 * time.Hour), // 24 hour validity
			Confidence:      0.95,
			IsRecommended:   i == 0, // First product as recommended
			Source:          "real_time",
			ResponseTimeMs:  responseTime.Milliseconds(),
			ExternalQuoteID: productCode,
		}

		response.Quotes = append(response.Quotes, quote)
	}

	response.TotalQuotes = len(response.Quotes)
	if response.TotalQuotes > 0 {
		// Set best quote as the first one
		response.BestQuote = &response.Quotes[0]
	}

	return response
}

// mapProductCodeToServiceType maps DHL product codes to our service types
func (s *Service) mapProductCodeToServiceType(productCode string) string {
	switch productCode {
	case "P":
		return "express"
	case "N":
		return "standard"
	case "U":
		return "economy"
	default:
		return "express"
	}
}

// IsHealthy performs health check for DHL API
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("DHL service not initialized")
	}

	s.logger.Info("Performing DHL API health check")

	// Create a minimal test request for health check
	testRequest := map[string]interface{}{
		"customerDetails": map[string]interface{}{
			"shipperDetails": map[string]interface{}{
				"postalCode":  "560001",
				"cityName":    "Bangalore",
				"countryCode": "IN",
			},
			"receiverDetails": map[string]interface{}{
				"postalCode":  "110001",
				"cityName":    "New Delhi",
				"countryCode": "IN",
			},
		},
		"accounts": []map[string]interface{}{
			{
				"typeCode": "shipper",
				"number":   s.config.AccountNumber,
			},
		},
		"productsAndServices": []map[string]interface{}{
			{
				"productCode":      "N",
				"localProductCode": "N",
			},
		},
		"payerCountryCode":           "IN",
		"plannedShippingDateAndTime": time.Now().AddDate(0, 0, 1).Format("2006-01-02T15:04:05GMT-07:00"),
		"unitOfMeasurement":          "metric",
		"isCustomsDeclarable":        false,
		"estimatedDeliveryDate": map[string]interface{}{
			"isRequested": true,
			"typeCode":    "QDDC",
		},
		"returnStandardProductsOnly": true,
		"packages": []map[string]interface{}{
			{
				"weight": 1.0,
				"dimensions": map[string]interface{}{
					"length": 10,
					"width":  10,
					"height": 10,
				},
			},
		},
	}

	// Try to make a health check call with shorter timeout
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	startTime := time.Now()
	_, err := s.callDHLAPI(healthCtx, testRequest)

	s.metrics.RecordTimer("dhl_health_check_duration", time.Since(startTime), map[string]string{
		"status": func() string {
			if err != nil {
				return "failed"
			}
			return "success"
		}(),
	})

	if err != nil {
		s.logger.Warn("DHL API health check failed", "error", err)
		return fmt.Errorf("DHL API health check failed: %w", err)
	}

	s.logger.Info("DHL API health check successful")
	return nil
}
