package mover

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
	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// Service implements real-time rate fetching for Mover
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    interfaces.HTTPClient
	config        *Config
	isInitialized bool
}

// NewService creates a new Mover service instance
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
	return "Mover"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
	s.logger.Info("Initializing Mover service", "config_keys", len(config))

	// Override defaults with provided config
	if err := s.config.LoadFromMap(config); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s.isInitialized = true

	s.logger.Info("Mover service initialized successfully",
		"base_url", s.config.BaseURL)

	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       s.config.BaseURL,
		"is_initialized": s.isInitialized,
		"provider_type": "real_time",
	}
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.logger.Info("Closing Mover service")
	s.isInitialized = false
	return nil
}

// IsHealthy checks if the service is healthy and operational
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("Mover service not initialized")
	}

	s.logger.Info("Performing Mover API health check")

	// For now, just check if service is initialized
	// In production, you might want to make a lightweight API call
	s.logger.Info("Mover API health check passed")
	return nil
}

// GetRates fetches rates from Mover API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("Mover service not initialized")
	}

	// Validate request
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	startTime := time.Now()
	s.logger.Info("Fetching rates from Mover API",
		"request_id", request.RequestID,
		"origin", request.OriginCity,
		"dest", request.DestCity,
		"packages", len(request.Packages))

	// Convert our request to Mover format
	estimateReq, err := s.convertToMoverRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Fetch rates (returns multiple vehicle options)
	quotes, err := s.fetchRates(ctx, estimateReq, request)
	if err != nil {
		s.metrics.IncrementCounter("mover_api_error", map[string]string{
			"error_type": "api_call_failed",
		})
		return nil, fmt.Errorf("failed to fetch rates: %w", err)
	}

	response := s.buildResponse(request, quotes, time.Since(startTime))

	s.metrics.IncrementCounter("mover_api_success", map[string]string{
		"total_quotes": fmt.Sprintf("%d", len(quotes)),
	})
	s.metrics.RecordTimer("mover_api_duration", time.Since(startTime), map[string]string{
		"endpoint": "get_rates",
	})

	return response, nil
}

// fetchRates fetches rates from Mover API (returns multiple vehicle options)
func (s *Service) fetchRates(ctx context.Context, estimateReq *OrderEstimateRequest, originalReq *dtos.RateCalculationRequest) ([]dtos.RateQuote, error) {
	// Prepare headers with Basic Auth
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Basic %s", s.config.AuthToken),
	}

	// Make API call
	startTime := time.Now()
	httpResponse, err := s.httpClient.Post(ctx, s.config.GetEstimateURL(), estimateReq, headers)
	if err != nil {
		return nil, fmt.Errorf("Mover API call failed: %w", err)
	}

	duration := time.Since(startTime)

	s.logger.Debug("Mover API response received",
		"status_code", httpResponse.StatusCode,
		"duration_ms", duration.Milliseconds())

	// Handle non-200 responses
	if httpResponse.StatusCode != http.StatusOK {
		s.logger.Error("Mover API returned error",
			"status_code", httpResponse.StatusCode,
			"response", string(httpResponse.Body))
		return nil, fmt.Errorf("Mover API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response
	var estimateResp OrderEstimateResponse
	if err := json.Unmarshal(httpResponse.Body, &estimateResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if we have vehicle options
	if len(estimateResp.List) == 0 {
		return nil, fmt.Errorf("no vehicle options available")
	}

	// Convert each vehicle option to a quote
	var quotes []dtos.RateQuote
	for _, vehicle := range estimateResp.List {
		// Only include available vehicles (status == 1)
		if vehicle.Status == 1 {
			quote := s.convertToQuote(&vehicle, originalReq, duration, estimateResp.EstimateID, estimateResp.ValidTill)
			quotes = append(quotes, quote)
		}
	}

	if len(quotes) == 0 {
		return nil, fmt.Errorf("no available vehicle options")
	}

	return quotes, nil
}

// convertToMoverRequest converts our request to Mover API format
func (s *Service) convertToMoverRequest(req *dtos.RateCalculationRequest) (*OrderEstimateRequest, error) {
	// Calculate total weight in kg
	totalWeightKg := 0.0
	for _, pkg := range req.Packages {
		weightKg := utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)
		totalWeightKg += weightKg
	}

	// Get lat/lon from metadata or use defaults
	// Default coordinates for Indian cities (can be enhanced with geocoding)
	pickupLat := 17.410767683570757  // Default Hyderabad
	pickupLon := 78.40520492132076
	dropLat := 17.476274545643385
	dropLon := 78.49034896159837

	if req.Metadata != nil {
		if lat, ok := req.Metadata["pickup_lat"].(float64); ok {
			pickupLat = lat
		}
		if lon, ok := req.Metadata["pickup_lon"].(float64); ok {
			pickupLon = lon
		}
		if lat, ok := req.Metadata["drop_lat"].(float64); ok {
			dropLat = lat
		}
		if lon, ok := req.Metadata["drop_lon"].(float64); ok {
			dropLon = lon
		}
	}

	// Get contact info from metadata or use defaults
	contactName := s.config.DefaultContactName
	contactMobile := s.config.DefaultContactMobile
	if req.Metadata != nil {
		if name, ok := req.Metadata["contact_name"].(string); ok && name != "" {
			contactName = name
		}
		if mobile, ok := req.Metadata["contact_mobile"].(string); ok && mobile != "" {
			contactMobile = mobile
		}
	}

	// Determine payment method (0 = prepaid, 1 = COD)
	paymentMethod := 0
	if req.Metadata != nil {
		if pm, ok := req.Metadata["payment_mode"].(string); ok && pm == "cod" {
			paymentMethod = 1
		}
	}

	// Calculate distance and duration (use from request or calculate)
	distance := req.Distance * 1000 // Convert km to meters
	duration := int(req.Distance * 60) // Rough estimate: 1km = 1 minute

	// Get goods worth from metadata or default to 0
	goodsWorth := 0.0
	if req.Metadata != nil {
		if worth, ok := req.Metadata["goods_worth"].(float64); ok {
			goodsWorth = worth
		}
	}

	estimateReq := &OrderEstimateRequest{
		Type:          "parcel",
		ShipMode:      "on_demand",
		Desc:          fmt.Sprintf("Shipment from %s to %s", req.OriginCity, req.DestCity),
		WeightKg:      totalWeightKg,
		GoodsWorth:    goodsWorth,
		OptimiseRoute: false,
		BoxRequired:   false,
		PaymentMethod: paymentMethod,
		RouteInfo: RouteInfo{
			Distance: distance,
			Duration: duration,
		},
		Pickup: Location{
			Lat:           pickupLat,
			Lon:           pickupLon,
			Address:       req.OriginCity, // Use postal code as address
			DeliveryNote:  "",
			ContactMobile: contactMobile,
			ContactName:   contactName,
			Udf1:          "",
		},
		Drop: Location{
			Lat:           dropLat,
			Lon:           dropLon,
			Address:       req.DestCity, // Use postal code as address
			DeliveryNote:  "",
			ContactMobile: contactMobile,
			ContactName:   contactName,
			Udf1:          "",
		},
		Waypoints: []string{},
	}

	return estimateReq, nil
}

// convertToQuote converts Mover vehicle option to our quote format
func (s *Service) convertToQuote(vehicle *VehicleOption, originalReq *dtos.RateCalculationRequest, responseTime time.Duration, estimateID, validTill string) dtos.RateQuote {
	quoteID := fmt.Sprintf("mover_%d_%s", vehicle.ID, uuid.New().String()[:8])

	// Use discounted amount if available, otherwise use base amount
	totalPrice := vehicle.DiscountedAmount
	if totalPrice == 0 {
		totalPrice = vehicle.Amount
	}

	// Build price breakdown
	priceBreakdown := dtos.PriceBreakdown{
		BasePrice:      vehicle.Amount,
		DiscountAmount: vehicle.Discount,
		TotalPrice:      totalPrice,
	}

	// Calculate estimated days from duration (duration is in seconds)
	estimatedDays := 1 // Default
	if vehicle.Duration > 0 {
		// Convert seconds to days (rough estimate: 8 hours per day)
		estimatedDays = (vehicle.Duration / (8 * 3600)) + 1
		if estimatedDays < 1 {
			estimatedDays = 1
		}
	}

	// Parse validTill timestamp if provided
	validUntil := time.Now().Add(7 * 24 * time.Hour) // Default to 7 days
	if validTill != "" {
		if parsedTime, err := time.Parse(time.RFC3339, validTill); err == nil {
			validUntil = parsedTime
		}
	}

	// Build metadata with vehicle details
	metadata := map[string]interface{}{
		"response_time_ms":    responseTime.Milliseconds(),
		"distance":            vehicle.Distance,
		"duration":            vehicle.Duration,
		"vehicle_id":          vehicle.ID,
		"vehicle_name":        vehicle.Name,
		"capacity_kg":         vehicle.CapacityInKg,
		"helper_applicable":   vehicle.HelperApplicable == 1,
		"helper_amount":       vehicle.HelperAmount,
		"amount_with_helper":  vehicle.AmountWithHelper,
		"is_eco_friendly":     vehicle.IsEcoFriendly == 1,
		"estimate_id":         estimateID,
	}

	// Determine service type based on vehicle name
	serviceType := originalReq.ServiceType
	if vehicle.Name != "" {
		// Map vehicle names to service types
		switch {
		case contains(vehicle.Name, "Bike", "bike"):
			serviceType = "express"
		case contains(vehicle.Name, "3 Wheeler", "3 wheeler"):
			serviceType = "standard"
		case contains(vehicle.Name, "Canter", "canter", "Truck", "truck"):
			serviceType = "premium"
		}
	}

	quote := dtos.RateQuote{
		QuoteID:            quoteID,
		PartnerID:          "mover",
		PartnerName:        "Mover",
		ProviderType:      models.PartnerTypeRealTime,
		BasePrice:          vehicle.Amount,
		TotalPrice:         totalPrice,
		Currency:           "INR",
		PriceBreakdown:     priceBreakdown,
		ServiceType:        serviceType,
		ServiceLevel:       "standard",
		EstimatedDays:     estimatedDays,
		ValidUntil:         validUntil,
		Confidence:         1.0,
		IsRecommended:     false,
		InsuranceAvailable: false,
		TrackingAvailable:  true,
		SignatureAvailable: false,
		Source:             "mover_api",
		Description:       fmt.Sprintf("Mover %s - %s", vehicle.Name, serviceType),
		Metadata:           metadata,
		ExternalQuoteID:   estimateID,
	}

	return quote
}

// contains checks if a string contains any of the given substrings (case-insensitive)
func contains(str string, substrings ...string) bool {
	strLower := strings.ToLower(str)
	for _, substr := range substrings {
		if strings.Contains(strLower, strings.ToLower(substr)) {
			return true
		}
	}
	return false
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

	return nil
}

