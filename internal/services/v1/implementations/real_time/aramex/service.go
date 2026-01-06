package aramex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
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
 
    // List of all Aramex product types to query
    productTypes := []string{"PDX", "PPX", "PLX", "DDX", "DPX", "GDX", "GPX", "EPX"}

    response := &dtos.RateCalculationResponse{
        RequestID:    request.RequestID,
        Status:       "success",
        Message:      "Aggregated rates retrieved from Aramex",
        Quotes:       []dtos.RateQuote{},
        Timestamp:    time.Now(),
        CacheHit:     false,
    }

    // Fetch rates in parallel using goroutines
    quotes, successfulCount := s.fetchRatesParallel(ctx, request, productTypes, startTime)
    response.Quotes = quotes
    response.TotalQuotes = len(response.Quotes)
    if response.TotalQuotes > 0 {
        response.BestQuote = &response.Quotes[0]
    }

    s.metrics.IncrementCounter("aramex_api_success", map[string]string{
        "total_quotes": fmt.Sprintf("%d", len(response.Quotes)),
    })
    s.metrics.RecordTimer("aramex_api_total_duration", time.Since(startTime), map[string]string{
        "total_successful": fmt.Sprintf("%d", successfulCount),
    })

    if successfulCount == 0 {
        response.Status = "failed"
        response.Message = "No successful rate retrieved from Aramex"
    }

    return response, nil
}

// fetchRatesParallel fetches rates for multiple product types in parallel
func (s *Service) fetchRatesParallel(ctx context.Context, request *dtos.RateCalculationRequest, productTypes []string, startTime time.Time) ([]dtos.RateQuote, int) {
    // Create channels for results
    quoteChan := make(chan dtos.RateQuote, len(productTypes))
    
    // Create semaphore for concurrency control (max 4 concurrent requests to avoid overwhelming the API)
    sem := make(chan struct{}, 4)
    
    var wg sync.WaitGroup
    
    for _, productType := range productTypes {
        wg.Add(1)
        go func(pt string) {
            defer wg.Done()
            
            // Acquire semaphore
            sem <- struct{}{}
            defer func() { <-sem }()
            
            // Check context cancellation
            select {
            case <-ctx.Done():
                s.logger.Warn("Context cancelled during rate fetching", "product_type", pt)
                return
            default:
            }
            
            s.logger.Info("Fetching Aramex rate", "product_type", pt)
            
            // Convert request
            aramexRequest, err := s.convertToAramexRequestWithProduct(request, pt)
            if err != nil {
                s.logger.Error("Failed to convert request", "product_type", pt, "error", err)
                return
            }
            
            s.logger.Debug("Aramex request prepared", "product_type", pt)
            
            // Call Aramex API
            aramexResponse, err := s.callAramexAPI(ctx, aramexRequest)
            if err != nil {
                s.logger.Warn("Aramex API call failed", "product_type", pt, "error", err)
                return
            }
            
            // Create quote and send to channel
            quote := s.createRateQuoteFromResponse(aramexResponse, request, pt, time.Since(startTime))
            quoteChan <- quote
            
            s.logger.Info("Successfully fetched Aramex rate", "product_type", pt, "price", quote.TotalPrice)
        }(productType)
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
    
    successfulCount := len(quotes)
    s.logger.Info("Parallel rate fetching completed", 
        "total_product_types", len(productTypes),
        "successful_count", successfulCount,
        "duration_ms", time.Since(startTime).Milliseconds())
    
    return quotes, successfulCount
}

// convertToAramexRequest converts our request format to Aramex API format
func (s *Service) convertToAramexRequest(req *dtos.RateCalculationRequest) (*AramexRateRequest, error) {
    if len(req.Packages) == 0 {
        return nil, fmt.Errorf("at least one package is required")
    }

    // Use first package for dimensions and weight (Aramex API typically handles single package per request)
    pkg := req.Packages[0]
    
    // Convert weight to kg
    weightKg := utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)
    
    // Convert dimensions to cm
    lengthCm := utils.ConvertDimensionToCm(pkg.Length, pkg.DimUnit)
    widthCm := utils.ConvertDimensionToCm(pkg.Width, pkg.DimUnit)
    heightCm := utils.ConvertDimensionToCm(pkg.Height, pkg.DimUnit)

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

// mapProductTypeToServiceLevel maps product type to service level name
func (s *Service) mapProductTypeToServiceLevel(productType string) string {
	switch strings.ToUpper(productType) {
	case "PDX":
		return "Priority Document Express"
	case "PPX":
		return "Priority Parcel Express"
	case "PLX":
		return "Priority Letter Express"
	case "DDX":
		return "Deferred Document Express"
	case "DPX":
		return "Deferred Parcel Express"
	case "GDX":
		return "Ground Document Express"
	case "GPX":
		return "Ground Parcel Express"
	case "EPX":
		return "Economy Parcel Express"
	default:
		return "Priority Parcel Express"
	}
}

// getEstimatedDays provides estimated delivery days based on product type and route
func (s *Service) getEstimatedDays(productType, originCountry, destCountry string) int {
	isInternational := originCountry != destCountry
	
	switch strings.ToUpper(productType) {
	case "PDX", "PPX", "PLX": // Priority services
		if isInternational {
			return 2
		}
		return 1
	case "DDX", "DPX": // Deferred (2nd Day Delivery)
		return 2
	case "GDX", "GPX": // Ground services
		if isInternational {
			return 7
		}
		return 4
	case "EPX": // Economy
		if isInternational {
			return 7
		}
		return 5
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

func (s *Service) convertToAramexRequestWithProduct(req *dtos.RateCalculationRequest, productType string) (*AramexRateRequest, error) {
    if len(req.Packages) == 0 {
        return nil, fmt.Errorf("at least one package is required")
    }

    pkg := req.Packages[0]
    weightKg := utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)
    lengthCm := utils.ConvertDimensionToCm(pkg.Length, pkg.DimUnit)
    widthCm := utils.ConvertDimensionToCm(pkg.Width, pkg.DimUnit)
    heightCm := utils.ConvertDimensionToCm(pkg.Height, pkg.DimUnit)

    // Infer product group based on product type prefix
    productGroup := "EXP"

    return &AramexRateRequest{
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
            Line1:       "N/A",
            City:        req.OriginCity,
            PostCode:    req.OriginCity,
            CountryCode: req.OriginCountry,
        },
        DestinationAddress: Address{
            Line1:       "N/A",
            City:        req.DestCity,
            PostCode:    req.DestCity,
            CountryCode: req.DestCountry,
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
            DescriptionOfGoods: "General Goods",
            GoodsOriginCountry: req.OriginCountry,
            NumberOfPieces:     1,
            ProductGroup:       productGroup,
            ProductType:        productType,
            PaymentType:        "P",
        },
    }, nil
}


func (s *Service) createRateQuoteFromResponse(resp *AramexRateResponse, req *dtos.RateCalculationRequest, productType string, duration time.Duration) dtos.RateQuote {
    return dtos.RateQuote{
        QuoteID:         fmt.Sprintf("aramex_%s_%d", strings.ToLower(productType), time.Now().UnixNano()),
        PartnerID:       "aramex",
        PartnerName:     "Aramex",
        ProviderType:    dtos.ProviderTypeRealTime,
        BasePrice:       resp.RateDetails.Amount,
        TotalPrice:      resp.TotalAmount.Value,
        Currency:        resp.TotalAmount.CurrencyCode,
        ServiceType:     productType,
        ServiceLevel:    s.mapProductTypeToServiceLevel(productType),
        EstimatedDays:   s.getEstimatedDays(productType, req.OriginCountry, req.DestCountry),
        ValidUntil:      time.Now().Add(24 * time.Hour),
        Confidence:      0.9,
        IsRecommended:   productType == "PPX", // usually PPX is priority
        Source:          "real_time",
        ResponseTimeMs:  duration.Milliseconds(),
        ExternalQuoteID: fmt.Sprintf("%s_%s", productType, resp.TotalAmount.CurrencyCode),
        Description:     s.fetchDescriptionFromProductType(productType),
    }
}

// fetchDescriptionFromProductType returns a human-readable description
// for a given Aramex product type code.
func (s *Service) fetchDescriptionFromProductType(productType string) string {
	switch strings.ToUpper(productType) {
	case "PDX":
		return "Priority Document Express – Urgent, time-sensitive consignments containing printed matter or document material."
	case "PPX":
		return "Priority Parcel Express – Urgent, time-sensitive consignments containing non-printed matter or non-document material."
	case "PLX":
		return "Priority Letter Express – Urgent, time-sensitive consignments containing printed matter of weight less than 0.5 kg."
	case "DDX":
		return "Deferred Document Express – 2nd Day Delivery consignments containing printed matter or document material."
	case "DPX":
		return "Deferred Parcel Express – 2nd Day Delivery consignments containing non-printed matter or non-document material."
	case "GDX":
		return "Ground Document Express – Ground delivery consignments containing printed matter or document material."
	case "GPX":
		return "Ground Parcel Express – Ground delivery consignments containing non-printed matter or non-document material."
	case "EPX":
		return "Economy Parcel Express – Non-document shipments for commercial use, including online sales, delivered locally or globally."
	default:
		return "Unknown product type – description not available."
	}
}
