package naqel

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// Service implements real-time rate fetching for Naqel
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    interfaces.HTTPClient
	config        *Config
	isInitialized bool
}

// NewService creates a new Naqel service instance
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
	return "Naqel Express"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
	s.logger.Info("Initializing Naqel service", "config_keys", len(config))

	// Override defaults with provided config
	if err := s.config.LoadFromMap(config); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s.isInitialized = true
	s.logger.Info("Naqel service initialized successfully",
		"base_url", s.config.BaseURL,
		"client_id", s.config.ClientID,
		"version", s.config.Version)
	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       s.config.BaseURL,
		"client_id":      s.config.ClientID,
		"version":        s.config.Version,
		"shipper_name":   s.config.ShipperName,
		"is_initialized": s.isInitialized,
		"provider_type":  "real_time",
	}
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.logger.Info("Closing Naqel service")
	s.isInitialized = false
	return nil
}

// GetRates fetches rates from Naqel API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("naqel service not initialized")
	}

	// Validate request before making API call
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	startTime := time.Now()
	
	// Map country codes
	originCountry := s.mapCountryCode(request.OriginCountry)
	destCountry := s.mapCountryCode(request.DestCountry)
	
	s.logger.Info("Fetching rates from Naqel API",
		"request_id", request.RequestID,
		"origin", request.OriginCity,
		"origin_country", originCountry,
		"destination", request.DestCity,
		"dest_country", destCountry,
		"weight", request.Weight)

	// Determine all possible LoadTypeIDs based on route
	loadTypeIDs := s.getLoadTypeIDsForRoute(originCountry, destCountry)
	
	s.logger.Info("Fetching rates for multiple LoadTypeIDs", 
		"load_type_ids", loadTypeIDs,
		"total_load_types", len(loadTypeIDs))

	response := &dtos.RateCalculationResponse{
		RequestID:    request.RequestID,
		Status:       "success",
		Message:      "Aggregated rates retrieved from Naqel",
		Quotes:       []dtos.RateQuote{},
		Timestamp:    time.Now(),
		CacheHit:     false,
	}

	// Fetch rates in parallel using goroutines
	quotes, successfulCount := s.fetchRatesParallel(ctx, request, loadTypeIDs, startTime)
	response.Quotes = quotes
	response.TotalQuotes = len(response.Quotes)
	if response.TotalQuotes > 0 {
		response.BestQuote = &response.Quotes[0]
	}

	s.metrics.IncrementCounter("naqel_api_success", map[string]string{
		"total_quotes": fmt.Sprintf("%d", len(response.Quotes)),
	})
	s.metrics.RecordTimer("naqel_api_total_duration", time.Since(startTime), map[string]string{
		"total_successful": fmt.Sprintf("%d", successfulCount),
	})

	if successfulCount == 0 {
		response.Status = "failed"
		response.Message = "No successful rate retrieved from Naqel"
	}

	return response, nil
}

// getLoadTypeIDsForRoute determines all possible LoadTypeIDs based on route
func (s *Service) getLoadTypeIDsForRoute(originCountry, destCountry string) []int {
	isInternational := !strings.EqualFold(originCountry, destCountry)
	
	if isInternational {
		// International routes - try multiple LoadTypeIDs
		loadTypeIDs := []int{
			33, // Document Int'l
			34, // Non Document Int'l
		}
		
		// Add IRC (International Road Courier) for GCC neighbors
		gccCountries := map[string]bool{"AE": true, "BH": true, "KW": true, "OM": true, "QA": true, "SA": true}
		if gccCountries[originCountry] && gccCountries[destCountry] {
			loadTypeIDs = append(loadTypeIDs, 65) // IRC - International Road Courier
		}
		
		return loadTypeIDs
	}
	
	// Domestic routes
	return []int{
		36, // Non Document (Domestic Courier)
		39, // Express Domestic
	}
}

// fetchRatesParallel fetches rates for multiple LoadTypeIDs in parallel
func (s *Service) fetchRatesParallel(ctx context.Context, request *dtos.RateCalculationRequest, loadTypeIDs []int, startTime time.Time) ([]dtos.RateQuote, int) {
	// Create channels for results
	quoteChan := make(chan dtos.RateQuote, len(loadTypeIDs))
	
	// Create semaphore for concurrency control (max 4 concurrent requests)
	sem := make(chan struct{}, 4)
	
	var wg sync.WaitGroup
	
	for _, loadTypeID := range loadTypeIDs {
		wg.Add(1)
		go func(ltID int) {
			defer wg.Done()
			
			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()
			
			// Check context cancellation
			select {
			case <-ctx.Done():
				s.logger.Warn("Context cancelled during rate fetching", "load_type_id", ltID)
				return
			default:
			}
			
			s.logger.Info("Fetching Naqel rate", "load_type_id", ltID)
			
			// Convert request with specific LoadTypeID
			naqelRequest, err := s.convertToNaqelRequestWithLoadType(request, ltID)
			if err != nil {
				s.logger.Error("Failed to convert request", "load_type_id", ltID, "error", err)
				return
			}
			
			s.logger.Debug("Naqel request prepared", "load_type_id", ltID)
			
			// Call Naqel API
			naqelResponse, err := s.callNaqelAPI(ctx, naqelRequest)
			if err != nil {
				s.logger.Warn("Naqel API call failed", "load_type_id", ltID, "error", err)
				return
			}
			
			// Create quote and send to channel
			quote := s.createRateQuoteFromResponse(naqelResponse, request, ltID, time.Since(startTime))
			if quote.TotalPrice > 0 {
				quoteChan <- quote
				s.logger.Info("Successfully fetched Naqel rate", "load_type_id", ltID, "price", quote.TotalPrice)
			} else {
				s.logger.Warn("Naqel rate returned zero or invalid price", "load_type_id", ltID)
			}
		}(loadTypeID)
	}
	
	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(quoteChan)
	}()
	
	// Collect results from channel
	var quotes []dtos.RateQuote
	for quote := range quoteChan {
		quotes = append(quotes, quote)
	}
	
	// Sort quotes by price (lowest first)
	sort.Slice(quotes, func(i, j int) bool {
		return quotes[i].TotalPrice < quotes[j].TotalPrice
	})
	
	// Mark the first (cheapest) quote as recommended
	if len(quotes) > 0 {
		quotes[0].IsRecommended = true
	}
	
	successfulCount := len(quotes)
	s.logger.Info("Parallel rate fetching completed",
		"total_load_type_ids", len(loadTypeIDs),
		"successful_count", successfulCount,
		"duration_ms", time.Since(startTime).Milliseconds())
	
	return quotes, successfulCount
}

// convertToNaqelRequest converts our request format to Naqel SOAP XML format
func (s *Service) convertToNaqelRequest(req *dtos.RateCalculationRequest) (*NaqelGetRateRequest, error) {
	// Use default LoadTypeID (will be overridden by convertToNaqelRequestWithLoadType)
	loadTypeID := 34 // Default to Non Document Int'l
	return s.convertToNaqelRequestWithLoadType(req, loadTypeID)
}

// convertToNaqelRequestWithLoadType converts our request format to Naqel SOAP XML format with specific LoadTypeID
func (s *Service) convertToNaqelRequestWithLoadType(req *dtos.RateCalculationRequest, loadTypeID int) (*NaqelGetRateRequest, error) {
	if len(req.Packages) == 0 {
		return nil, fmt.Errorf("at least one package is required")
	}

	// Use first package for weight
	pkg := req.Packages[0]
	weightKg := utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)

	// Format FromDate as ISO 8601
	fromDate := time.Now().Format(time.RFC3339)

	// Map country codes (KSA -> SA, etc.)
	originCountry := s.mapCountryCode(req.OriginCountry)

	// Use city codes for Origin and Destination (assuming they're already in Naqel format)
	originCode := req.OriginCity
	destCode := req.DestCity

	// Create SOAP request
	naqelRequest := &NaqelGetRateRequest{
		SoapNS: "http://schemas.xmlsoap.org/soap/envelope/",
		XsiNS:  "http://www.w3.org/2001/XMLSchema-instance",
		XsdNS:  "http://www.w3.org/2001/XMLSchema",
		Body: NaqelGetRateBody{
			GetRate: NaqelGetRateBodyContent{
				Namespace: "http://tempuri.org/",
				ClientInfo: NaqelClientInfo{
					ClientAddress: NaqelClientAddress{
						ShipperName:  s.config.ShipperName,
						FirstAddress: "Test Address", // Can be made configurable
						Location:     req.OriginCity,
						CountryCode:  originCountry, // Use mapped country code
						CityCode:     originCode,
					},	
					ClientContact: NaqelClientContact{
						Name:        "Test Client",
						Email:       "test@example.com",
						PhoneNumber: "0500000000",
						MobileNo:    "0500000000",
					},
					ClientID: s.config.ClientID,
					Password: s.config.Password,
					Version:  s.config.Version,
				},
				Weight:       weightKg,
				LoadTypeID:   loadTypeID,
				FromDate:     fromDate,
				Origin:       originCode,
				Destination:  destCode,
			},
		},
	}

	return naqelRequest, nil
}

// mapServiceToLoadTypeID maps our service types to Naqel LoadTypeID
func (s *Service) mapServiceToLoadTypeID(serviceType, originCountry, destCountry string) int {
	isInternational := !strings.EqualFold(originCountry, destCountry)
	serviceTypeLower := strings.ToLower(serviceType)

	if isInternational {
		// --- International routes ---
		switch serviceTypeLower {
		case "express", "priority":
			return 33 // Document Int'l – International Courier
		case "standard", "economy":
			return 34 // Non Document Int'l – International Courier
		default:
			// GCC neighbors (road courier)
			gccCountries := map[string]bool{"AE": true, "BH": true, "KW": true, "OM": true, "QA": true}
			if gccCountries[destCountry] {
				return 65 // IRC – International Road Courier
			}
			return 34 // Default to Non Document Int'l
		}
	}

	// --- Domestic routes ---
	switch serviceTypeLower {
	case "express", "priority":
		return 39 // Express Domestic
	case "standard", "economy":
		return 36 // Non Document – Domestic Courier
	default:
		return 36 // Default fallback
	}
}



// mapCountryCode maps country codes to Naqel format
func (s *Service) mapCountryCode(countryCode string) string {
	countryCodeUpper := strings.ToUpper(countryCode)
	
	// Map common country code variations to ISO 3166-1 alpha-2	
	countryMap := map[string]string{
		"KSA": "SA", // Saudi Arabia
		"UAE": "AE", // United Arab Emirates
		"BHR": "BH", // Bahrain
		"KWT": "KW", // Kuwait
		"JOR": "JO", // Jordan
		"OMN": "OM", // Oman
		"LBN": "LB", // Lebanon
		"EGY": "EG", // Egypt
		"QAT": "QA", // Qatar
		"MAR": "MA", // Morocco
		"IRQ": "IQ", // Iraq
	}

	if mapped, exists := countryMap[countryCodeUpper]; exists {
		s.logger.Debug("Mapped country code", "original", countryCode, "mapped", mapped)
		return mapped
	}
	
	// If already 2 characters, return as-is
	if len(countryCodeUpper) == 2 {
		return countryCodeUpper
	}
	
	// Return original if no mapping found
	return countryCodeUpper
}

// callNaqelAPI makes the actual SOAP API call to Naqel
func (s *Service) callNaqelAPI(ctx context.Context, request *NaqelGetRateRequest) (*NaqelGetRateResponse, error) {
	// Use standard Go http.Client for SOAP requests
	client := &http.Client{
		Timeout: time.Duration(s.config.TimeoutMs) * time.Millisecond,
	}

	endpoint := s.config.BaseURL

	// Marshal request to XML
	xmlData, err := xml.MarshalIndent(request, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP request: %w", err)
	}

	// Create SOAP envelope with proper XML declaration
	soapRequest := `<?xml version="1.0" encoding="utf-8"?>` + "\n" + string(xmlData)

	// Log the full SOAP request (redact password)
	safeRequest := strings.ReplaceAll(soapRequest, s.config.Password, "[REDACTED]")
	s.logger.Info("Naqel SOAP Request",
		"endpoint", endpoint,
		"request_length", len(soapRequest),
		"soap_request", safeRequest)
	s.logger.Debug("Naqel API Request (full)",
		"endpoint", endpoint,
		"request_length", len(soapRequest))

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(soapRequest))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set SOAP headers
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", `"http://tempuri.org/GetRate"`)

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Log the full response
	s.logger.Info("Naqel SOAP Response",
		"status_code", resp.StatusCode,
		"body_length", len(body),
		"response_body", string(body))
	s.logger.Debug("Naqel API Response (full)",
		"status_code", resp.StatusCode,
		"body_length", len(body))

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("Naqel API returned non-200 status",
			"status_code", resp.StatusCode,
			"response_body", string(body))
		return nil, fmt.Errorf("naqel API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SOAP response
	var response NaqelGetRateResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		s.logger.Warn("Failed to unmarshal Naqel response directly, trying custom parser", "error", err)
		// Try to parse as plain XML if SOAP envelope parsing fails
		if err2 := s.parseNaqelResponse(body, &response); err2 != nil {
			s.logger.Error("Failed to unmarshal Naqel response with custom parser", "error", err2, "body", string(body))
			return nil, fmt.Errorf("failed to unmarshal SOAP response: %w (original: %v)", err2, err)
		}
	}

	// Log parsed response details
	result := response.Body.GetRateResponse.GetRateResult
	s.logger.Info("Successfully parsed Naqel API response",
		"price", result.Price,
		"rate", result.Rate,
		"currency", result.Currency,
		"message", result.Message)

	return &response, nil
}

// parseNaqelResponse attempts to parse Naqel response XML
func (s *Service) parseNaqelResponse(body []byte, response *NaqelGetRateResponse) error {
	// First, try to find the GetRateResult or Price element directly
	decoder := xml.NewDecoder(bytes.NewReader(body))
	
	var inGetRateResult bool
	var result NaqelGetRateResult

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "GetRateResult" {
				inGetRateResult = true
				// Decode the entire GetRateResult element
				if err := decoder.DecodeElement(&result, &se); err != nil {
					s.logger.Warn("Failed to decode GetRateResult element", "error", err)
					// Continue to try finding Price directly
				} else {
					response.Body.GetRateResponse.GetRateResult = result
					return nil
				}
			} else if se.Name.Local == "Price" && inGetRateResult {
				// Found Price element, read its value
				var priceValue string
				if err := decoder.DecodeElement(&priceValue, &se); err != nil {
					s.logger.Warn("Failed to decode Price element", "error", err)
				} else {
					// Parse price as float
					var price float64
					if _, err := fmt.Sscanf(priceValue, "%f", &price); err == nil {
						result.Price = price
						response.Body.GetRateResponse.GetRateResult = result
						return nil
					}
				}
			}
		case xml.EndElement:
			if se.Name.Local == "GetRateResult" {
				inGetRateResult = false
			}
		}
	}

	// If we can't find GetRateResult, try to unmarshal the whole body
	return xml.Unmarshal(body, response)
}

// createRateQuoteFromResponse creates a single rate quote from Naqel API response
func (s *Service) createRateQuoteFromResponse(
	naqelResp *NaqelGetRateResponse,
	originalReq *dtos.RateCalculationRequest,
	loadTypeID int,
	duration time.Duration,
) dtos.RateQuote {
	// Extract rate from response (API returns Price field, not Rate)
	result := naqelResp.Body.GetRateResponse.GetRateResult
	
	// Use Price field first (as per actual API response), fallback to Rate
	rateValue := result.Price
	if rateValue <= 0 {
		rateValue = result.Rate
	}

	// Determine currency (default to SAR if not provided)
	currency := result.Currency
	if currency == "" {
		currency = "SAR" // Saudi Riyal as default
	}

	// Map LoadTypeID to service type for display
	originCountry := s.mapCountryCode(originalReq.OriginCountry)
	destCountry := s.mapCountryCode(originalReq.DestCountry)
	isInternational := !strings.EqualFold(originCountry, destCountry)
	
	serviceType, serviceLevel := s.mapLoadTypeIDToServiceType(loadTypeID, isInternational)
	
	// Calculate estimated days
	estimatedDays := s.getEstimatedDaysForLoadType(loadTypeID, isInternational)

	quote := dtos.RateQuote{
		QuoteID:         fmt.Sprintf("naqel_%d_%d", loadTypeID, time.Now().UnixNano()),
		PartnerID:       "naqel",
		PartnerName:     "Naqel Express",
		ProviderType:    dtos.ProviderTypeRealTime,
		BasePrice:       rateValue,
		TotalPrice:      rateValue,
		Currency:        currency,
		ServiceType:     serviceType,
		ServiceLevel:    serviceLevel,
		EstimatedDays:   estimatedDays,
		ValidUntil:      time.Now().Add(24 * time.Hour), // 24 hour validity
		Confidence:      0.90,
		IsRecommended:   false, // Will be set to true for best quote later
		Source:          "real_time",
		ResponseTimeMs:  duration.Milliseconds(),
		ExternalQuoteID: fmt.Sprintf("naqel_%d_%.2f", loadTypeID, rateValue),
	}

	if result.Message != "" {
		quote.Notes = result.Message
	}

	return quote
}

// mapLoadTypeIDToServiceType maps Naqel LoadTypeID to our service type and service level
func (s *Service) mapLoadTypeIDToServiceType(loadTypeID int, isInternational bool) (string, string) {
	switch loadTypeID {
	case 33: // Document Int'l
		if isInternational {
			return "express", "Document International"
		}
		return "standard", "Document"
	case 34: // Non Document Int'l
		if isInternational {
			return "standard", "Non-Document International"
		}
		return "standard", "Non-Document"
	case 36: // Non Document (Domestic)
		return "standard", "Non-Document Domestic"
	case 39: // Express Domestic
		return "express", "Express Domestic"
	case 65: // IRC - International Road Courier
		return "economy", "International Road Courier"
	default:
		if isInternational {
			return "standard", "International Courier"
		}
		return "standard", "Domestic Courier"
	}
}

// getEstimatedDaysForLoadType provides estimated delivery days based on LoadTypeID
func (s *Service) getEstimatedDaysForLoadType(loadTypeID int, isInternational bool) int {
	switch loadTypeID {
	case 33: // Document Int'l
		if isInternational {
			return 2
		}
		return 1
	case 34: // Non Document Int'l
		if isInternational {
			return 4
		}
		return 2
	case 36: // Non Document (Domestic)
		return 3
	case 39: // Express Domestic
		return 1
	case 65: // IRC - International Road Courier
		return 5
	default:
		if isInternational {
			return 4
		}
		return 2
	}
}

// convertFromNaqelResponse converts Naqel response to our format (kept for backward compatibility)
func (s *Service) convertFromNaqelResponse(
	naqelResp *NaqelGetRateResponse,
	originalReq *dtos.RateCalculationRequest,
	responseTime time.Duration,
) *dtos.RateCalculationResponse {
	// Use default LoadTypeID for single response
	defaultLoadTypeID := 34
	quote := s.createRateQuoteFromResponse(naqelResp, originalReq, defaultLoadTypeID, responseTime)

	response := &dtos.RateCalculationResponse{
		RequestID:    originalReq.RequestID,
		Status:       "success",
		Message:      "Rates retrieved from Naqel",
		Quotes:       []dtos.RateQuote{quote},
		TotalQuotes:  1,
		BestQuote:    &quote,
		ResponseTime: responseTime.Milliseconds(),
		CacheHit:     false,
		Timestamp:    time.Now(),
	}

	return response
}

// mapServiceTypeToOurType maps Naqel service type to our standard service types
func (s *Service) mapServiceTypeToOurType(serviceType, originCountry, destCountry string) string {
	isInternational := originCountry != destCountry
	serviceTypeLower := strings.ToLower(serviceType)

	if isInternational {
		switch serviceTypeLower {
		case "express":
			return "express"
		case "standard":
			return "standard"
		case "economy", "premium":
			return "economy"
		default:
			return "standard"
		}
	} else {
		switch serviceTypeLower {
		case "express":
			return "express"
		case "standard", "economy", "premium":
			return "standard"
		default:
			return "standard"
		}
	}
}

// getServiceLevelName returns a human-readable service level name
func (s *Service) getServiceLevelName(serviceType, originCountry, destCountry string) string {
	isInternational := originCountry != destCountry
	serviceTypeLower := strings.ToLower(serviceType)

	if isInternational {
		switch serviceTypeLower {
		case "express":
			return "International Express"
		case "standard":
			return "International Standard"
		case "economy":
			return "International Economy"
		default:
			return "International Courier"
		}
	} else {
		switch serviceTypeLower {
		case "express":
			return "Domestic Express"
		case "standard":
			return "Domestic Standard"
		case "economy":
			return "Domestic Economy"
		default:
			return "Domestic Courier"
		}
	}
}

// getEstimatedDays provides estimated delivery days based on service type and route
func (s *Service) getEstimatedDays(serviceType, originCountry, destCountry string) int {
	isInternational := originCountry != destCountry
	serviceTypeLower := strings.ToLower(serviceType)

	if isInternational {
		switch serviceTypeLower {
		case "express":
			return 2
		case "standard":
			return 4
		case "economy":
			return 7
		default:
			return 4
		}
	} else {
		switch serviceTypeLower {
		case "express":
			return 1
		case "standard":
			return 2
		case "economy":
			return 3
		default:
			return 2
		}
	}
}

// IsHealthy performs health check for Naqel API
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("naqel service not initialized")
	}

	s.logger.Info("Performing Naqel API health check")

	// Create a minimal test request for health check
	testRequest := &dtos.RateCalculationRequest{
		RequestID:     "health_check",
		CustomerID:    "test",
		OriginCity:    "RUH007", // Riyadh
		OriginCountry: "SA",
		DestCity:      "BAH0063", // Bahrain
		DestCountry:   "BH",
		Weight:        1.0,
		Distance:      100,
		ServiceType:   "standard",
		PickupDate:    time.Now(),
		DeliveryDate:  time.Now().Add(24 * time.Hour),
		Priority:      "normal",
		Currency:      "SAR",
		Packages: []dtos.PackageDetails{
			{
				Weight:     1.0,
				WeightUnit: "kg",
				Length:     10,
				Width:      10,
				Height:     10,
				DimUnit:    "cm",
			},
		},
		Source: "health_check",
	}

	// Try to make a health check call with shorter timeout
	healthCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	startTime := time.Now()
	_, err := s.GetRates(healthCtx, testRequest)

	s.metrics.RecordTimer("naqel_health_check_duration", time.Since(startTime), map[string]string{
		"status": func() string {
			if err != nil {
				return "failed"
			}
			return "success"
		}(),
	})

	if err != nil {
		s.logger.Warn("Naqel API health check failed", "error", err)
		return fmt.Errorf("naqel API health check failed: %w", err)
	}

	s.logger.Info("Naqel API health check successful")
	return nil
}

