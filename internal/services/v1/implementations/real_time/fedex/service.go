package fedex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// Service implements real-time rate fetching for FedEx
type Service struct {
    logger        interfaces.Logger
    metrics       interfaces.MetricsCollector
    httpClient    interfaces.HTTPClient
    config        *Config
    isInitialized bool
    accessToken   string
    tokenExpiry   time.Time
}

// NewService creates a new FedEx service instance
func NewService(
    logger interfaces.Logger,
    metrics interfaces.MetricsCollector,
    httpClient interfaces.HTTPClient,
) *Service {

    return &Service{
        logger:     logger,
        metrics:    metrics,
        httpClient: httpClient,
        config:     NewDefaultConfig(),
        isInitialized: true,
    }
}

// GetImplementationType returns the implementation type
func (s *Service) GetImplementationType() dtos.ProviderType {
    return dtos.ProviderTypeRealTime
}

// GetImplementationName returns the implementation name
func (s *Service) GetImplementationName() string {
    return "FedEx Express"
}

// Initialize initializes the service with given configuration
func (s *Service) Initialize(config map[string]interface{}) error {
    s.logger.Info("Initializing FedEx service", "config_keys", len(config))

    // Override defaults with provided config
    if err := s.config.LoadFromMap(config); err != nil {
		s.logger.Info("=====================", err)
        return fmt.Errorf("failed to load configuration: %w", err)
    }
    // Validate required configuration
    if s.config.ClientID == "" || s.config.ClientSecret == "" {
				s.logger.Info("===================== client id and secrete issue")

        return fmt.Errorf("FedEx client ID and client secret are required")
    }
    s.isInitialized = true
    s.logger.Info("FedEx service initialized successfully", 
        "base_url", s.config.BaseURL,
        "environment", s.config.Environment,
        "account_number", s.config.AccountNumber)
    return nil
}

// GetConfiguration returns the service configuration
func (s *Service) GetConfiguration() map[string]interface{} {
    return map[string]interface{}{
        "base_url":        s.config.BaseURL,
        "environment":     s.config.Environment,
        "account_number":  s.config.AccountNumber,
        "is_initialized":  s.isInitialized,
        "provider_type":   "real_time",
    }
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
    s.logger.Info("Closing FedEx service")
    s.isInitialized = false
    s.accessToken = ""
    return nil
}

// GetRates fetches rates from FedEx API
func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	s.logger.Info("INSIDE FEDEX SERVICE =======================")
    if !s.isInitialized {
        return nil, fmt.Errorf("FedEx service not initialized")
    }

    // Validate request before making API call
    if err := s.validateRequest(request); err != nil {
        return nil, fmt.Errorf("request validation failed: %w", err)
    }

    startTime := time.Now()
    s.logger.Info("Fetching rates from FedEx API",
        "request_id", request.RequestID,
        "origin", request.OriginCity,
        "origin_country", request.OriginCountry,
        "destination", request.DestCity,
        "dest_country", request.DestCountry,
        "weight", request.Weight,
        "packages", len(request.Packages))

    // Get access token
    if err := s.ensureAccessToken(ctx); err != nil {
        s.metrics.IncrementCounter("fedex_api_error", map[string]string{
            "error_type": "auth_failed",
        })
        return nil, fmt.Errorf("failed to get access token: %w", err)
    }

    // Convert our request to FedEx format
    fedexRequest, err := s.convertToFedexRequest(request)
    if err != nil {
        return nil, fmt.Errorf("failed to convert request: %w", err)
    }

    // Make API call
    fedexResponse, err := s.callFedexAPI(ctx, fedexRequest)
    if err != nil {
        s.metrics.IncrementCounter("fedex_api_error", map[string]string{
            "error_type": "api_call_failed",
        })
        return nil, fmt.Errorf("FedEx API call failed: %w", err)
    }

    // Convert FedEx response to our format
    response := s.convertFromFedexResponse(fedexResponse, request, time.Since(startTime))

    s.metrics.IncrementCounter("fedex_api_success", map[string]string{
        "total_quotes": fmt.Sprintf("%d", len(response.Quotes)),
    })
    s.metrics.RecordTimer("fedex_api_duration", time.Since(startTime), map[string]string{
        "endpoint": "get_rates",
    })

    return response, nil
}

// convertToFedexRequest converts our request format to FedEx API format
func (s *Service) convertToFedexRequest(req *dtos.RateCalculationRequest) (*FedexRateRequest, error) {
    if len(req.Packages) == 0 {
        return nil, fmt.Errorf("at least one package is required")
    }

    // Use first package for dimensions and weight (FedEx API can handle multiple but we'll use first for simplicity)
    pkg := req.Packages[0]
    
    // Convert weight to kg
    weightKg :=  utils.ConvertWeightToKg(pkg.Weight, pkg.WeightUnit)    

    // Convert dimensions to cm and round to integers
    lengthCm := int(utils.ConvertDimensionToCm(pkg.Length, pkg.DimUnit))
    widthCm := int(utils.ConvertDimensionToCm(pkg.Width, pkg.DimUnit))
    heightCm := int(utils.ConvertDimensionToCm(pkg.Height, pkg.DimUnit))

    fedexRequest := &FedexRateRequest{
        AccountNumber: struct {
            Value string `json:"value"`
        }{
            Value: s.config.AccountNumber,
        },
        RequestedShipment: struct {
            Shipper struct {
                Address struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                } `json:"address"`
            } `json:"shipper"`
            Recipient struct {
                Address struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                } `json:"address"`
            } `json:"recipient"`
            PickupType    string `json:"pickupType"`
            RateRequestType []string `json:"rateRequestType"`
            PreferredCurrency string `json:"preferredCurrency"`
            PackageCount   int `json:"packageCount"`
            RequestedPackageLineItems []RequestedPackageLineItem `json:"requestedPackageLineItems"`
        }{
            Shipper: struct {
                Address struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                } `json:"address"`
            }{
                Address: struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                }{
                    PostalCode:  req.OriginCity,
                    CountryCode: req.OriginCountry,
                },
            },
            Recipient: struct {
                Address struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                } `json:"address"`
            }{
                Address: struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                }{
                    PostalCode:  req.DestCity,
                    CountryCode: req.DestCountry,
                },
            },
            PickupType:        "USE_SCHEDULED_PICKUP",
            RateRequestType:   []string{"LIST"},
            PreferredCurrency: "INR",
            PackageCount:      1,
            RequestedPackageLineItems: []RequestedPackageLineItem{
                {
                    GroupPackageCount: 1,
                    Weight: struct {
                        Units string `json:"units"`
                        Value float64 `json:"value"`
                    }{
                        Units: "KG",
                        Value: weightKg,
                    },
                    Dimensions: struct {
                        Length int `json:"length"`
                        Width  int `json:"width"`
                        Height int `json:"height"`
                        Units  string `json:"units"`
                    }{
                        Length: lengthCm,
                        Width:  widthCm,
                        Height: heightCm,
                        Units:  "CM",
                    },
                },
            },
        },
    }

    return fedexRequest, nil
}

// ensureAccessToken ensures we have a valid access token
func (s *Service) ensureAccessToken(ctx context.Context) error {
    if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
        return nil
    }

    s.logger.Info("🔑 Fetching new FedEx access token using standard http.Client")

    // Use standard Go http.Client directly - this will work for sure
    client := &http.Client{
        Timeout: 30 * time.Second,
    }

    // Create the form data exactly like curl does
    formData := url.Values{}
    formData.Set("grant_type", "client_credentials")
    formData.Set("client_id", s.config.ClientID)
    formData.Set("client_secret", s.config.ClientSecret)

    tokenURL := s.config.BaseURL + "/oauth/token"
    
    s.logger.Info("📤 Sending FedEx auth request with standard http.Client",
        "url", tokenURL,
        "client_id", s.config.ClientID,
        "client_secret_prefix", s.getSecretPrefix(s.config.ClientSecret))

    // Create the request
    req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(formData.Encode()))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    // Set the proper headers
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("Accept", "application/json")

    // Make the request
    resp, err := client.Do(req)
    if err != nil {
        s.logger.Error("❌ FedEx auth HTTP request failed", "error", err)
        return fmt.Errorf("failed to get access token: %w", err)
    }
    defer resp.Body.Close()

    // Read the response
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response: %w", err)
    }

    s.logger.Info("📨 FedEx auth response",
        "status_code", resp.StatusCode,
        "body", string(body))

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("FedEx auth returned status %d: %s", resp.StatusCode, string(body))
    }

    var tokenResponse FedexTokenResponse
    if err := json.Unmarshal(body, &tokenResponse); err != nil {
        s.logger.Error("❌ Failed to parse FedEx token response", "error", err)
        return fmt.Errorf("failed to parse token response: %w", err)
    }

    if tokenResponse.AccessToken == "" {
        return fmt.Errorf("empty access token in response")
    }

    s.accessToken = tokenResponse.AccessToken
    // Set expiry with 5-minute buffer
    s.tokenExpiry = time.Now().Add(time.Duration(tokenResponse.ExpiresIn-300) * time.Second)

    s.logger.Info("✅ FedEx access token obtained successfully with standard http.Client",
        "expires_in", tokenResponse.ExpiresIn,
        "token_type", tokenResponse.TokenType)
    return nil
}

// Helper to safely log secret prefix
func (s *Service) getSecretPrefix(secret string) string {
    if len(secret) == 0 {
        return "EMPTY"
    }
    if len(secret) <= 4 {
        return secret
    }
    return secret[:4] + "..."
}

// callFedexAPI makes the actual API call to FedEx
func (s *Service) callFedexAPI(ctx context.Context, request *FedexRateRequest) (*FedexRateResponse, error) {
    // Use standard Go http.Client for consistency
    client := &http.Client{
        Timeout: 30 * time.Second,
    }

    endpoint := s.config.BaseURL + "/rate/v1/rates/quotes"

    // Convert request to JSON
    requestBody, err := json.Marshal(request)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    s.logger.Debug("FedEx API Request",
        "endpoint", endpoint,
        "request_body_length", len(requestBody))

    // Create the request
    req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(requestBody)))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+s.accessToken)
    req.Header.Set("X-locale", "en_US")

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

    s.logger.Debug("FedEx API Response",
        "status_code", resp.StatusCode,
        "body_length", len(body))

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("FedEx API returned status %d: %s", resp.StatusCode, string(body))
    }

    // Parse response into our struct
    var response FedexRateResponse
    if err := json.Unmarshal(body, &response); err != nil {
        s.logger.Error("❌ Failed to unmarshal FedEx response", "error", err)
        return nil, fmt.Errorf("failed to unmarshal response: %w", err)
    }

    s.logger.Info("✅ Successfully parsed FedEx API response", 
        "total_services", len(response.Output.RateReplyDetails))
    
    return &response, nil
}

// Helper function to get keys from map
func getKeys(m map[string]interface{}) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}

// Helper function for min
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

// convertFromFedexResponse converts FedEx response to our format
func (s *Service) convertFromFedexResponse(
    fedexResp *FedexRateResponse,
    originalReq *dtos.RateCalculationRequest,
    responseTime time.Duration,
) *dtos.RateCalculationResponse {

    response := &dtos.RateCalculationResponse{
        RequestID:    originalReq.RequestID,
        Status:       "success",
        Message:      "Rates retrieved from FedEx",
        Quotes:       []dtos.RateQuote{},
        ResponseTime: responseTime.Milliseconds(),
        CacheHit:     false,
        Timestamp:    time.Now(),
    }

    if fedexResp.Output.RateReplyDetails == nil {
        response.Status = "error"
        response.Message = "No rate details found in FedEx response"
        return response
    }

    s.logger.Info("📊 Processing FedEx rate response", 
        "total_services", len(fedexResp.Output.RateReplyDetails))

    for i, rateDetail := range fedexResp.Output.RateReplyDetails {
        if len(rateDetail.RatedShipmentDetails) == 0 {
            s.logger.Debug("Skipping rate detail with no shipment details", "service_type", rateDetail.ServiceType)
            continue
        }

        // Use the first rated shipment detail
        shipmentDetail := rateDetail.RatedShipmentDetails[0]

        // Map FedEx service type to our service type
        // serviceType := s.mapServiceTypeToOurType(rateDetail.ServiceType)

        // Calculate estimated days (simplified)
        estimatedDays := s.getEstimatedDays(rateDetail.ServiceType, originalReq.OriginCountry, originalReq.DestCountry)

        // Get currency from shipment detail or package detail
        currency := shipmentDetail.Currency
        if currency == "" && len(shipmentDetail.RatedPackages) > 0 {
            currency = shipmentDetail.RatedPackages[0].PackageRateDetail.Currency
        }
        if currency == "" {
            currency = "INR" // Default fallback
        }

        // Use TotalNetCharge as the price
        totalPrice := shipmentDetail.TotalNetCharge
        basePrice := shipmentDetail.TotalBaseCharge

        // If prices are zero, try to get from package details
        if totalPrice == 0 && len(shipmentDetail.RatedPackages) > 0 {
            totalPrice = shipmentDetail.RatedPackages[0].PackageRateDetail.NetCharge
            basePrice = shipmentDetail.RatedPackages[0].PackageRateDetail.BaseCharge
        }

        // Skip if no valid price
        if totalPrice == 0 {
            s.logger.Debug("Skipping service with zero price", "service_type", rateDetail.ServiceType)
            continue
        }

        quote := dtos.RateQuote{
            QuoteID:         fmt.Sprintf("fedex_%s_%d", strings.ToLower(rateDetail.ServiceType), i),
            PartnerID:       "fedex",
            PartnerName:     "FedEx Express",
            ProviderType:    dtos.ProviderTypeRealTime,
            BasePrice:       basePrice,
            TotalPrice:      totalPrice,
            Currency:        currency,
            ServiceType:     rateDetail.ServiceType,
            ServiceLevel:    rateDetail.ServiceName,
            EstimatedDays:   estimatedDays,
            ValidUntil:      time.Now().Add(24 * time.Hour), // 24 hour validity
            Confidence:      0.92,
            IsRecommended:   i == 0, // First quote as recommended
            Source:          "real_time",
            ResponseTimeMs:  responseTime.Milliseconds(),
            ExternalQuoteID: rateDetail.ServiceType,
            Description: fmt.Sprintf("%s – %s", rateDetail.ServiceName, s.mapFedexServiceDescription(rateDetail.ServiceType)),
        }

        s.logger.Debug("FedEx rate quote",
            "service_type", rateDetail.ServiceType,
            "service_name", rateDetail.ServiceName,
            "base_price", basePrice,
            "total_price", totalPrice,
            "currency", currency,
            "estimated_days", estimatedDays)

        response.Quotes = append(response.Quotes, quote)
    }

    response.TotalQuotes = len(response.Quotes)
    if response.TotalQuotes > 0 {
        response.BestQuote = &response.Quotes[0]
        s.logger.Info("✅ Successfully processed FedEx rates", 
            "total_quotes", response.TotalQuotes,
            "best_quote_price", response.BestQuote.TotalPrice,
            "best_quote_currency", response.BestQuote.Currency,
            "best_quote_service", response.BestQuote.ServiceType)
    } else {
        s.logger.Warn("⚠️ No valid quotes found in FedEx response")
        response.Status = "partial_success"
        response.Message = "FedEx API returned services but no valid pricing"
    }

    return response
}

// mapServiceTypeToOurType maps FedEx service types to our service types
func (s *Service) mapServiceTypeToOurType(fedexServiceType string) string {
    switch strings.ToUpper(fedexServiceType) {
    case "FEDEX_EXPRESS_SAVER", "FEDEX_GROUND":
        return "economy"
    case "FEDEX_2_DAY", "FEDEX_2_DAY_AM":
        return "standard"
    case "FEDEX_OVERNIGHT", "FEDEX_FIRST_OVERNIGHT", "FEDEX_PRIORITY_OVERNIGHT":
        return "express"
    case "INTERNATIONAL_ECONOMY":
        return "economy"
    case "INTERNATIONAL_PRIORITY":
        return "express"
    default:
        return "standard"
    }
}

// getEstimatedDays provides estimated delivery days based on service type and route
func (s *Service) getEstimatedDays(serviceType, originCountry, destCountry string) int {
    isInternational := originCountry != destCountry
    
    switch strings.ToUpper(serviceType) {
    case "FEDEX_FIRST_OVERNIGHT", "FEDEX_PRIORITY_OVERNIGHT":
        return 1
    case "FEDEX_OVERNIGHT":
        if isInternational {
            return 2
        }
        return 1
    case "FEDEX_2_DAY", "FEDEX_2_DAY_AM":
        return 2
    case "FEDEX_EXPRESS_SAVER":
        return 3
    case "INTERNATIONAL_PRIORITY":
        return 2
    case "INTERNATIONAL_ECONOMY":
        return 5
    case "FEDEX_GROUND":
        if isInternational {
            return 7
        }
        return 4
    default:
        return 3
    }
}

// IsHealthy performs health check for FedEx API
func (s *Service) IsHealthy(ctx context.Context) error {
    if !s.isInitialized {
        return fmt.Errorf("FedEx service not initialized")
    }

    s.logger.Info("Performing FedEx API health check")

    // Test authentication
    if err := s.ensureAccessToken(ctx); err != nil {
        s.logger.Warn("FedEx API health check failed - authentication", "error", err)
        return fmt.Errorf("FedEx API authentication failed: %w", err)
    }

    // Create a minimal test request for health check
    testRequest := &FedexRateRequest{
        AccountNumber: struct {
            Value string `json:"value"`
        }{
            Value: s.config.AccountNumber,
        },
        RequestedShipment: struct {
            Shipper struct {
                Address struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                } `json:"address"`
            } `json:"shipper"`
            Recipient struct {
                Address struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                } `json:"address"`
            } `json:"recipient"`
            PickupType    string `json:"pickupType"`
            RateRequestType []string `json:"rateRequestType"`
            PreferredCurrency string `json:"preferredCurrency"`
            PackageCount   int `json:"packageCount"`
            RequestedPackageLineItems []RequestedPackageLineItem `json:"requestedPackageLineItems"`
        }{
            Shipper: struct {
                Address struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                } `json:"address"`
            }{
                Address: struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                }{
                    PostalCode:  "10001",
                    CountryCode: "US",
                },
            },
            Recipient: struct {
                Address struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                } `json:"address"`
            }{
                Address: struct {
                    PostalCode  string `json:"postalCode"`
                    CountryCode string `json:"countryCode"`
                }{
                    PostalCode:  "90001",
                    CountryCode: "US",
                },
            },
            PickupType:        "USE_SCHEDULED_PICKUP",
            RateRequestType:   []string{"LIST"},
            PreferredCurrency: "INR",
            PackageCount:      1,
            RequestedPackageLineItems: []RequestedPackageLineItem{
                {
                    GroupPackageCount: 1,
                    Weight: struct {
                        Units string `json:"units"`
                        Value float64 `json:"value"`
                    }{
                        Units: "KG",
                        Value: 1.0,
                    },
                    Dimensions: struct {
                        Length int `json:"length"`
                        Width  int `json:"width"`
                        Height int `json:"height"`
                        Units  string `json:"units"`
                    }{
                        Length: 10,
                        Width:  10,
                        Height: 10,
                        Units:  "CM",
                    },
                },
            },
        },
    }

    // Try to make a health check call with shorter timeout
    healthCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()

    startTime := time.Now()
    _, err := s.callFedexAPI(healthCtx, testRequest)

    s.metrics.RecordTimer("fedex_health_check_duration", time.Since(startTime), map[string]string{
        "status": func() string {
            if err != nil {
                return "failed"
            }
            return "success"
        }(),
    })

    if err != nil {
        s.logger.Warn("FedEx API health check failed", "error", err)
        return fmt.Errorf("FedEx API health check failed: %w", err)
    }

    s.logger.Info("FedEx API health check successful")
    return nil
}

func (s *Service) mapFedexServiceDescription(serviceType string) string {
	switch strings.ToUpper(serviceType) {
	case "FIRST_OVERNIGHT":
		return "Earliest next-day delivery by 8 AM."
	case "PRIORITY_OVERNIGHT":
		return "Next-day delivery by 10:30 AM to most addresses."
	case "FEDEX_2_DAY":
		return "Delivery within 2 business days."
	case "FEDEX_EXPRESS_SAVER":
		return "Delivery within 3 business days."
	case "FEDEX_GROUND":
		return "Day-definite delivery within 1–5 business days."
	default:
		return "FedEx shipping service."
	}
}
