package aramex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// Service implements real-time rate fetching for Aramex
type Service struct {
    logger        interfaces.Logger
    metrics       interfaces.MetricsCollector
    httpClient    interfaces.HTTPClient
    config        *Config
    isInitialized bool
}

// NewService creates a new Aramex service instance
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
    return "Aramex"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
    s.logger.Info("Initializing Aramex service", "config_keys", len(config))

    // Override defaults with provided config
    if err := s.config.LoadFromMap(config); err != nil {
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    s.isInitialized = true
    s.logger.Info("Aramex service initialized successfully", 
        "base_url", s.config.BaseURL,
        "username", s.config.Username,
        "account_number", s.config.AccountNumber,
        "account_entity", s.config.AccountEntity)
    return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
    return map[string]interface{}{
        "base_url":               s.config.BaseURL,
        "username":               s.config.Username,
        "account_number":         s.config.AccountNumber,
        "account_entity":         s.config.AccountEntity,
        "account_country_code":   s.config.AccountCountryCode,
        "source":                 s.config.Source,
        "is_initialized":         s.isInitialized,
        "provider_type":          "real_time",
    }
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
    s.logger.Info("Closing Aramex service")
    s.isInitialized = false
    return nil
}

// GetRates fetches rates from Aramex API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
    if !s.isInitialized {
        return nil, fmt.Errorf("Aramex service not initialized")
    }

    // Validate request before making API call
    if err := s.validateRequest(request); err != nil {
        return nil, fmt.Errorf("request validation failed: %w", err)
    }

    startTime := time.Now()
    s.logger.Info("Fetching rates from Aramex API",
        "request_id", request.RequestID,
        "origin", request.OriginCity,
        "origin_country", request.OriginCountry,
        "destination", request.DestCity,
        "dest_country", request.DestCountry,
        "weight", request.Weight,
        "packages", len(request.Packages))

    // Convert our request to Aramex format
    aramexRequest, err := s.convertToAramexRequest(request)
    if err != nil {
        return nil, fmt.Errorf("failed to convert request: %w", err)
    }

    // Make API call
    aramexResponse, err := s.callAramexAPI(ctx, aramexRequest)
    if err != nil {
        s.metrics.IncrementCounter("aramex_api_error", map[string]string{
            "error_type": "api_call_failed",
        })
        return nil, fmt.Errorf("Aramex API call failed: %w", err)
    }

    // Convert Aramex response to our format
    response := s.convertFromAramexResponse(aramexResponse, request, time.Since(startTime))

    s.metrics.IncrementCounter("aramex_api_success", map[string]string{
        "total_quotes": fmt.Sprintf("%d", len(response.Quotes)),
    })
    s.metrics.RecordTimer("aramex_api_duration", time.Since(startTime), map[string]string{
        "endpoint": "calculate_rate",
    })

    return response, nil
}

// convertToAramexRequest converts our request format to Aramex API format
func (s *Service) convertToAramexRequest(req *dtos.RateCalculationRequest) (*AramexRateRequest, error) {
    if len(req.Packages) == 0 {
        return nil, fmt.Errorf("at least one package is required")
    }

    // Use first package for dimensions and weight (Aramex API typically handles single package per request)
    pkg := req.Packages[0]
    
    // Convert weight to kg
    weightKg := convertWeightToKg(pkg.Weight, pkg.WeightUnit)
    
    // Convert dimensions to cm
    lengthCm := convertDimensionToCm(pkg.Length, pkg.DimUnit)
    widthCm := convertDimensionToCm(pkg.Width, pkg.DimUnit)
    heightCm := convertDimensionToCm(pkg.Height, pkg.DimUnit)

    // Determine product group and type based on service type
    productGroup, productType := s.mapServiceToProductType(req.ServiceType)

    // Create the request with explicit null for ChargeableWeight
    aramexRequest := &AramexRateRequest{
        ClientInfo: ClientInfo{
            UserName:           s.config.Username,
            Password:           s.config.Password,
            Version:            s.config.Version,
            AccountNumber:      s.config.AccountNumber,
            AccountPin:         s.config.AccountPin,
            AccountEntity:      s.config.AccountEntity,
            AccountCountryCode: s.config.AccountCountryCode,
            Source:             s.config.Source,
        },
        OriginAddress: Address{
            Line1:              "N/A", // Placeholder as address lines are required
            Line2:              "",
            Line3:              "",
            City:               req.OriginCity,
            StateOrProvinceCode: "",
            PostCode:           req.OriginCity, // Using city as placeholder for postcode
            CountryCode:        req.OriginCountry,
        },
        DestinationAddress: Address{
            Line1:              "N/A", // Placeholder as address lines are required
            Line2:              "",
            Line3:              "",
            City:               req.DestCity,
            StateOrProvinceCode: "",
            PostCode:           req.DestCity, // Using city as placeholder for postcode
            CountryCode:        req.DestCountry,
        },
        ShipmentDetails: ShipmentDetails{
            Dimensions: Dimensions{
                Length: lengthCm,
                Width:  widthCm,
                Height: heightCm,
                Unit:   "cm",
            },
            ActualWeight: Weight{
                Unit:  "KG",
                Value: weightKg,
            },
            // Explicitly set ChargeableWeight to nil pointer
            ChargeableWeight:   nil,
            DescriptionOfGoods: "General Goods",
            GoodsOriginCountry: req.OriginCountry,
            NumberOfPieces:     1,
            ProductGroup:       productGroup,
            ProductType:        productType,
            PaymentType:        "P", // Prepaid
            PaymentOptions:     "",
            Services:           "",
        },
    }

    return aramexRequest, nil
}

// mapServiceToProductType maps our service types to Aramex product groups and types
func (s *Service) mapServiceToProductType(serviceType string) (string, string) {
    switch strings.ToLower(serviceType) {
    case "express":
        return "EXP", "PPX" // Express, Priority Parcel Express
    case "standard":
        return "EXP", "PLX" // Express, Parcel Express
    case "economy":
        return "DOM", "GPP" // Domestic, Ground Parcel
    default:
        return "EXP", "PPX" // Default to Express, Priority Parcel Express
    }
}

// callAramexAPI makes the actual API call to Aramex
func (s *Service) callAramexAPI(ctx context.Context, request *AramexRateRequest) (*AramexRateResponse, error) {
    // Construct full endpoint URL
    endpoint := s.config.BaseURL + "/CalculateRate"

    // Log the request for debugging (without sensitive data)
    safeRequest := *request
    safeRequest.ClientInfo.Password = "[REDACTED]"
    safeRequest.ClientInfo.AccountPin = "[REDACTED]"
    
    if requestJSON, err := json.MarshalIndent(safeRequest, "", "  "); err == nil {
        s.logger.Debug("Aramex API Request", "endpoint", endpoint, "request", string(requestJSON))
    }

    // Make HTTP request
    httpResponse, err := s.httpClient.Post(ctx, endpoint, request, map[string]string{
        "Content-Type": "application/json",
        "Accept":       "application/json",
    })
    if err != nil {
        return nil, fmt.Errorf("HTTP request failed: %w", err)
    }

    if httpResponse.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Aramex API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
    }

    // Parse response
    var response AramexRateResponse
    if err := json.Unmarshal(httpResponse.Body, &response); err != nil {
        return nil, fmt.Errorf("failed to unmarshal response: %w", err)
    }

    // Check if response has errors
    if response.HasErrors {
        return nil, fmt.Errorf("Aramex API returned error in response")
    }

    return &response, nil
}

// convertFromAramexResponse converts Aramex response to our format
func (s *Service) convertFromAramexResponse(
    aramexResp *AramexRateResponse,
    originalReq *dtos.RateCalculationRequest,
    responseTime time.Duration,
) *dtos.RateCalculationResponse {

    response := &dtos.RateCalculationResponse{
        RequestID:    originalReq.RequestID,
        Status:       "success",
        Message:      "Rates retrieved from Aramex",
        Quotes:       []dtos.RateQuote{},
        ResponseTime: responseTime.Milliseconds(),
        CacheHit:     false,
        Timestamp:    time.Now(),
    }

    // Create rate quote from Aramex response
    quote := dtos.RateQuote{
        QuoteID:         fmt.Sprintf("aramex_%s_%d", strings.ToLower(originalReq.ServiceType), 0),
        PartnerID:       "aramex",
        PartnerName:     "Aramex",
        ProviderType:    dtos.ProviderTypeRealTime,
        BasePrice:       aramexResp.RateDetails.Amount,
        TotalPrice:      aramexResp.TotalAmount.Value,
        Currency:        aramexResp.TotalAmount.CurrencyCode,
        ServiceType:     originalReq.ServiceType,
        ServiceLevel:    s.mapProductTypeToServiceLevel(originalReq.ServiceType),
        EstimatedDays:   s.getEstimatedDays(originalReq.ServiceType, originalReq.OriginCountry, originalReq.DestCountry),
        ValidUntil:      time.Now().Add(24 * time.Hour), 
        Confidence:      0.90,
        IsRecommended:   true,
        Source:          "real_time",
        ResponseTimeMs:  responseTime.Milliseconds(),
        ExternalQuoteID: fmt.Sprintf("%s_%s", originalReq.ServiceType, aramexResp.TotalAmount.CurrencyCode),
        // TaxAmount:       aramexResp.RateDetails.TaxAmount,
    }

    response.Quotes = append(response.Quotes, quote)
    response.TotalQuotes = len(response.Quotes)
    
    if response.TotalQuotes > 0 {
        response.BestQuote = &response.Quotes[0]
    }

    return response
}

// mapProductTypeToServiceLevel maps product type to service level name
func (s *Service) mapProductTypeToServiceLevel(serviceType string) string {
    switch strings.ToLower(serviceType) {
    case "express":
        return "Priority Parcel Express"
    case "standard":
        return "Parcel Express"
    case "economy":
        return "Ground Parcel"
    default:
        return "Priority Parcel Express"
    }
}

// getEstimatedDays provides estimated delivery days based on service type and route
func (s *Service) getEstimatedDays(serviceType, originCountry, destCountry string) int {
    isInternational := originCountry != destCountry
    
    switch strings.ToLower(serviceType) {
    case "express":
        if isInternational {
            return 2
        }
        return 1
    case "standard":
        if isInternational {
            return 4
        }
        return 2
    case "economy":
        if isInternational {
            return 7
        }
        return 4
    default:
        return 3
    }
}

// IsHealthy performs health check for Aramex API
func (s *Service) IsHealthy(ctx context.Context) error {
    if !s.isInitialized {
        return fmt.Errorf("Aramex service not initialized")
    }

    s.logger.Info("Performing Aramex API health check")

    // Create a minimal test request for health check
    testRequest := &AramexRateRequest{
        ClientInfo: ClientInfo{
            UserName:           s.config.Username,
            Password:           s.config.Password,
            Version:            s.config.Version,
            AccountNumber:      s.config.AccountNumber,
            AccountPin:         s.config.AccountPin,
            AccountEntity:      s.config.AccountEntity,
            AccountCountryCode: s.config.AccountCountryCode,
            Source:             s.config.Source,
        },
        OriginAddress: Address{
            Line1:              "Test Address",
            Line2:              "",
            Line3:              "",
            City:               "Mumbai",
            StateOrProvinceCode: "",
            PostCode:           "400001",
            CountryCode:        "IN",
        },
        DestinationAddress: Address{
            Line1:              "Test Address",
            Line2:              "",
            Line3:              "",
            City:               "Delhi",
            StateOrProvinceCode: "",
            PostCode:           "110001",
            CountryCode:        "IN",
        },
        ShipmentDetails: ShipmentDetails{
            Dimensions: Dimensions{
                Length: 10,
                Width:  10,
                Height: 10,
                Unit:   "cm",
            },
            ActualWeight: Weight{
                Unit:  "KG",
                Value: 1.0,
            },
            DescriptionOfGoods: "Test Shipment",
            GoodsOriginCountry: "IN",
            NumberOfPieces:     1,
            ProductGroup:       "DOM",
            ProductType:        "GPP",
            PaymentType:        "P",
        },
    }

    // Try to make a health check call with shorter timeout
    healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    startTime := time.Now()
    _, err := s.callAramexAPI(healthCtx, testRequest)

    s.metrics.RecordTimer("aramex_health_check_duration", time.Since(startTime), map[string]string{
        "status": func() string {
            if err != nil {
                return "failed"
            }
            return "success"
        }(),
    })

    if err != nil {
        s.logger.Warn("Aramex API health check failed", "error", err)
        return fmt.Errorf("Aramex API health check failed: %w", err)
    }

    s.logger.Info("Aramex API health check successful")
    return nil
}