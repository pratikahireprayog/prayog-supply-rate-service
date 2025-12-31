package delhivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// Service implements real-time rate fetching for Delhivery Kinko API
type Service struct {
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
	httpClient    interfaces.HTTPClient
	config        *Config
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
	s.logger.Info("Initializing Delhivery Kinko service", "config_keys", len(config))

	// Override defaults with provided config
	if err := s.config.LoadFromMap(config); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s.isInitialized = true

	s.logger.Info("Delhivery Kinko service initialized successfully",
		"base_url", s.config.BaseURL,
		"estimate_endpoint", s.config.EstimateEndpoint)

	return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       s.config.BaseURL,
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

// IsHealthy checks if the service is healthy
func (s *Service) IsHealthy(ctx context.Context) error {
	if !s.isInitialized {
		return fmt.Errorf("Delhivery service not initialized")
	}
	// Simple health check could be a mock call or just checking initialization
	return nil
}

// GetRates fetches rates from Delhivery Kinko API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	if !s.isInitialized {
		return nil, fmt.Errorf("Delhivery service not initialized")
	}

	// Validate request
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	startTime := time.Now()
	s.logger.Info("Fetching rates from Delhivery Kinko API",
		"request_id", request.RequestID,
		"origin_pin", request.OriginCity,
		"dest_pin", request.DestCity)

	var allQuotes []dtos.RateQuote
	
	// Fetch for both Surface (S) and Express (E) modes
	modes := []string{"S", "E"}
	for _, mode := range modes {
		quote, err := s.fetchKinkoRate(ctx, mode, request)
		if err != nil {
			s.logger.Warn("Failed to fetch Delhivery Kinko rate", "mode", mode, "error", err)
			continue
		}
		if quote != nil {
			allQuotes = append(allQuotes, *quote)
		}
	}

	if len(allQuotes) == 0 {
		return nil, fmt.Errorf("no rates available from Delhivery")
	}

	response := s.buildResponse(request, allQuotes, time.Since(startTime))

	s.metrics.IncrementCounter("delhivery_api_success", map[string]string{
		"total_quotes": fmt.Sprintf("%d", len(allQuotes)),
	})

	return response, nil
}

// fetchKinkoRate fetches a single rate for a specific mode
func (s *Service) fetchKinkoRate(ctx context.Context, mode string, req *dtos.RateCalculationRequest) (*dtos.RateQuote, error) {
	// Prepare Query Params
	// https://track.delhivery.com/api/kinko/v1/invoice/charges/.json?md=S&ss=Delivered&d_pin=411014&o_pin=411014&cgm=5000&pt=Pre-paid
	
	params := url.Values{}
	params.Add("md", mode)
	params.Add("ss", "Delivered")
	params.Add("d_pin", req.DestCity)
	params.Add("o_pin", req.OriginCity)
	
	// Calculate total weight in grams
	totalWeightG := 0.0
	for _, pkg := range req.Packages {
		weightKg := utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)
		totalWeightG += (weightKg * 1000)
	}
	params.Add("cgm", fmt.Sprintf("%.0f", totalWeightG))
	
	paymentType := "Pre-paid"
	if req.Metadata != nil {
		if pt, ok := req.Metadata["payment_type"].(string); ok {
			paymentType = pt
		}
	}
	params.Add("pt", paymentType)

	fullURL := fmt.Sprintf("%s?%s", s.config.GetEstimateURL(), params.Encode())
	
	headers := map[string]string{
		"Authorization": fmt.Sprintf("TOKEN %s", s.config.APIKey),
		"Content-Type":  "application/json",
	}

	s.logger.Debug("Making Delhivery Kinko API call", "url", fullURL)

	httpResponse, err := s.httpClient.Get(ctx, fullURL, headers)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	var kinkoResp KinkoRateResponse
	if err := json.Unmarshal(httpResponse.Body, &kinkoResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(kinkoResp) == 0 {
		return nil, nil // No results
	}

	return s.convertToQuote(&kinkoResp[0], mode, req), nil
}

func (s *Service) convertToQuote(item *KinkoRateItem, mode string, originalReq *dtos.RateCalculationRequest) *dtos.RateQuote {
	quoteID := fmt.Sprintf("delhivery_%s_%s", mode, uuid.New().String()[:8])
	
	serviceTypeName := "Surface"
	if mode == "E" {
		serviceTypeName = "Express"
	}

	priceBreakdown := dtos.PriceBreakdown{
		BasePrice:  item.ChargeDL,
		TaxAmount:  item.TaxData.CGST + item.TaxData.SGST + item.TaxData.IGST,
		TotalPrice: item.TotalAmount,
	}

	quote := &dtos.RateQuote{
		QuoteID:            quoteID,
		PartnerID:          "delhivery",
		PartnerName:        "Delhivery",
		ProviderType:      "real_time",
		BasePrice:          item.ChargeDL,
		TotalPrice:         item.TotalAmount,
		Currency:           "INR",
		PriceBreakdown:     priceBreakdown,
		ServiceType:        serviceTypeName,
		ServiceLevel:       serviceTypeName,
		ValidUntil:         time.Now().Add(7 * 24 * time.Hour),
		Confidence:         1.0,
		Source:             "delhivery_kinko_api",
		Description:       fmt.Sprintf("Delhivery %s service", serviceTypeName),
		Metadata: map[string]interface{}{
			"mode":           mode,
			"zone":           item.Zone,
			"charged_weight": item.ChargedWeight,
			"tax_details":    item.TaxData,
		},
	}

	return quote
}

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

func (s *Service) validateRequest(request *dtos.RateCalculationRequest) error {
	if request == nil {
		return fmt.Errorf("request is nil")
	}
	if len(request.Packages) == 0 {
		return fmt.Errorf("at least one package is required")
	}
	if request.OriginCity == "" || request.DestCity == "" {
		return fmt.Errorf("origin and destination postal codes are required")
	}
	return nil
}
