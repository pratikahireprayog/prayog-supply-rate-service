# Real-Time API Integration Guide

## Overview
This guide covers integrating real-time APIs for shipping partners like DHL, FedEx, UPS, etc. Real-time partners provide live rates through API calls instead of pre-defined rate tables.

## Architecture Overview

```
Quote Request → Partner Code Normalization → Real-Time Implementation → Partner API → Response Mapping → Quote Response
```

## Partner Code Handling

### Code Normalization
Partner codes are normalized to snake_case for consistent internal handling:

```go
// Examples:
"DHL" -> "dhl"
"FedEx" -> "fedex"  
"Blue Dart" -> "blue_dart"
"Delhivery Express" -> "delhivery_express"
```

### Partner Resolution
Partners can be identified by either `id` or `code`. If `id` is empty, the system will use `code`:

```json
{
  "partners": [
    {
      "id": "",           // Optional - can be empty
      "code": "DHL"       // Required - will be normalized to "dhl"
    }
  ]
}
```

## Implementation Structure

Real-time integrations follow a modular approach with dedicated partner directories:

```
internal/services/v1/implementations/real_time/
├── dhl/                     # DHL Express module (✅ FULLY IMPLEMENTED)
│   ├── service.go           # Main DHL service implementation
│   ├── config.go            # DHL-specific configuration management
│   └── models.go            # DHL API request/response models
├── fedex/                   # FedEx module (placeholder)
│   ├── service.go           # Service skeleton
│   └── config.go            # Configuration skeleton
├── ups/                     # UPS module (placeholder)
│   ├── service.go           # Service skeleton  
│   └── config.go            # Configuration skeleton
└── base.go                  # Base real-time implementation
```

**Benefits:**
- ✅ Partner isolation and maintainability
- ✅ Easy addition of new partners
- ✅ Clear separation of concerns
- ✅ Independent testing and deployment
- ✅ Partner-specific configuration management

---

## DHL Integration Implementation

### 1. DHL API Configuration

**Base URL:** `https://express.api.dhl.com/mydhlapi/test/rates`

**Required Headers:**
- `Authorization: Basic {base64_encoded_credentials}`
- `Content-Type: application/json`
- `accept: application/json`
- `x-version: 2.12.0`
- `Message-Reference: {unique_message_id}`
- `Message-Reference-Date: {current_timestamp}`

### 2. Request Mapping

#### Our Quote Request → DHL API Request

```json
// Our Request Format
{
  "source_location": {
    "postal_code": "560086",
    "country_code": "IN"
  },
  "destination_location": {
    "postal_code": "266001", 
    "country_code": "CN"
  },
  "packages": [
    {
      "weight": {"value": 0.5, "unit": "kg"},
      "dimensions": {"length": 121, "width": 20, "height": 30, "unit": "cm"}
    }
  ],
  "partners": [{"id": "", "code": "DHL"}],
  "metadata": {"currency": "INR"}
}

// DHL API Request Format
{
  "customerDetails": {
    "shipperDetails": {
      "postalCode": "560086",
      "cityName": "Bangalore",
      "countryCode": "IN"
    },
    "receiverDetails": {
      "postalCode": "266001",
      "cityName": "QING DAO", 
      "countryCode": "CN"
    }
  },
  "accounts": [
    {
      "typeCode": "shipper",
      "number": "533748932"
    }
  ],
  "productsAndServices": [
    {
      "productCode": "P",
      "localProductCode": "P"
    }
  ],
  "payerCountryCode": "IN",
  "plannedShippingDateAndTime": "2025-06-16T28:00:00GMT+05:30",
  "unitOfMeasurement": "metric",
  "isCustomsDeclarable": true,
  "estimatedDeliveryDate": {
    "isRequested": true,
    "typeCode": "QDDC"
  },
  "returnStandardProductsOnly": true,
  "packages": [
    {
      "weight": 0.5,
      "dimensions": {
        "length": 121,
        "width": 20,
        "height": 30
      }
    }
  ]
}
```

### 3. Response Mapping

#### DHL API Response → Our Quote Response

```json
// DHL API Response Format
{
  "products": [
    {
      "productName": "EXPRESS WORLDWIDE",
      "productCode": "P",
      "localProductCode": "P",
      "localProductCountryCode": "IN",
      "networkTypeCode": "TD",
      "isCustomerAgreement": false,
      "weight": {
        "volumetric": 14.52,
        "provided": 15,
        "unitOfMeasurement": "metric"
      },
      "totalPrice": [
        {
          "currencyType": "BILLC",
          "priceCurrency": "INR",
          "price": 15724.8
        }
      ],
      "deliveryCapabilities": {
        "deliveryTypeCode": "QDDC",
        "estimatedDeliveryDateAndTime": "2025-06-23T23:59:00",
        "totalTransitDays": 7
      }
    }
  ]
}

// Our Quote Response Format  
{
  "partner_rates": [
    {
      "partner": {
        "id": "dhl_partner_uuid",
        "code": "dhl"
      },
      "success": true,
      "available_rates": [
        {
          "rate_id": "dhl_express_worldwide_001",
          "service": "EXPRESS WORLDWIDE",
          "price": {
            "currency": "INR",
            "amount": 15724.8,
            "type": "international_express",
            "criteria": {
              "product_code": "P",
              "weight_charged": 14.52,
              "weight_provided": 15,
              "route": "international"
            }
          },
          "delivery_days": 7
        }
      ],
      "data_source": "real_time",
      "response_time_ms": 1250
    }
  ]
}
```

---

## Implementation Files

### 1. DHL Real-Time Implementation

**File:** `internal/services/v1/implementations/real_time/dhl_implementation.go`

```go
package real_time

import (
	"bytes"
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

// DHLImplementation implements real-time rate fetching for DHL
type DHLImplementation struct {
	*BaseRealTime
	baseURL     string
	credentials string
	accountNumber string
}

// NewDHLImplementation creates a new DHL implementation
func NewDHLImplementation(
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	httpClient interfaces.HTTPClient,
) *DHLImplementation {
	base := NewBaseRealTime(logger, metrics, httpClient)
	
	return &DHLImplementation{
		BaseRealTime: base,
		baseURL:     "https://express.api.dhl.com/mydhlapi/test/rates",
		credentials: "c2hyZWVtYXJ1dDhJTjpJITBwTV40c1IjNG5KJDF1", // Base64 encoded
		accountNumber: "533748932",
	}
}

// GetImplementationType returns the implementation type
func (d *DHLImplementation) GetImplementationType() dtos.ProviderType {
	return dtos.ProviderTypeRealTime
}

// GetImplementationName returns the implementation name
func (d *DHLImplementation) GetImplementationName() string {
	return "DHL Express"
}

// GetRates fetches rates from DHL API
func (d *DHLImplementation) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	startTime := time.Now()
	
	// Convert our request to DHL format
	dhlRequest, err := d.convertToDHLRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make API call
	dhlResponse, err := d.callDHLAPI(ctx, dhlRequest)
	if err != nil {
		return nil, fmt.Errorf("DHL API call failed: %w", err)
	}

	// Convert DHL response to our format
	response := d.convertFromDHLResponse(dhlResponse, request, time.Since(startTime))
	
	return response, nil
}

// convertToDHLRequest converts our request format to DHL API format
func (d *DHLImplementation) convertToDHLRequest(req *dtos.RateCalculationRequest) (map[string]interface{}, error) {
	// Generate unique message reference
	messageRef := uuid.New().String()
	
	// Map service type to DHL product codes
	productCode := d.mapServiceToProductCode(req.ServiceType)
	
	dhlRequest := map[string]interface{}{
		"customerDetails": map[string]interface{}{
			"shipperDetails": map[string]interface{}{
				"postalCode":  req.OriginCity, // Using postal code from our structure
				"cityName":    req.OriginCity,
				"countryCode": "IN", // Default or extract from metadata
			},
			"receiverDetails": map[string]interface{}{
				"postalCode":  req.DestCity,
				"cityName":    req.DestCity,
				"countryCode": "CN", // Default or extract from metadata
			},
		},
		"accounts": []map[string]interface{}{
			{
				"typeCode": "shipper",
				"number":   d.accountNumber,
			},
		},
		"productsAndServices": []map[string]interface{}{
			{
				"productCode":      productCode,
				"localProductCode": productCode,
			},
		},
		"payerCountryCode":              "IN",
		"plannedShippingDateAndTime":   req.PickupDate.Format("2006-01-02T15:04:05GMT-07:00"),
		"unitOfMeasurement":            "metric",
		"isCustomsDeclarable":          true,
		"estimatedDeliveryDate": map[string]interface{}{
			"isRequested": true,
			"typeCode":   "QDDC",
		},
		"returnStandardProductsOnly": true,
		"packages": []map[string]interface{}{
			{
				"weight": req.Weight,
				"dimensions": map[string]interface{}{
					"length": 30, // Default dimensions or extract from packages
					"width":  20,
					"height": 15,
				},
			},
		},
	}

	return dhlRequest, nil
}

// mapServiceToProductCode maps our service types to DHL product codes
func (d *DHLImplementation) mapServiceToProductCode(serviceType string) string {
	switch strings.ToLower(serviceType) {
	case "express":
		return "P" // EXPRESS WORLDWIDE
	case "standard":
		return "N" // DOMESTIC EXPRESS
	default:
		return "P" // Default to EXPRESS WORLDWIDE
	}
}

// callDHLAPI makes the actual API call to DHL
func (d *DHLImplementation) callDHLAPI(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	// Convert request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Prepare headers
	headers := map[string]string{
		"accept":                    "application/json",
		"Authorization":             fmt.Sprintf("Basic %s", d.credentials),
		"Content-Type":              "application/json",
		"x-version":                 "2.12.0",
		"Message-Reference":         uuid.New().String(),
		"Message-Reference-Date":    time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT"),
	}

	// Make HTTP request
	httpResponse, err := d.httpClient.Post(ctx, d.baseURL+"?strictValidation=false", request, headers)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if httpResponse.StatusCode != 200 {
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
func (d *DHLImplementation) convertFromDHLResponse(
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
		productMap := product.(map[string]interface{})
		
		// Extract product details
		productName := productMap["productName"].(string)
		productCode := productMap["productCode"].(string)
		
		// Extract pricing
		var totalPrice float64
		var currency string
		if totalPriceArray, ok := productMap["totalPrice"].([]interface{}); ok && len(totalPriceArray) > 0 {
			firstPrice := totalPriceArray[0].(map[string]interface{})
			totalPrice = firstPrice["price"].(float64)
			currency = firstPrice["priceCurrency"].(string)
		}

		// Extract delivery information
		var deliveryDays int
		if deliveryCap, ok := productMap["deliveryCapabilities"].(map[string]interface{}); ok {
			if transitDays, exists := deliveryCap["totalTransitDays"]; exists {
				deliveryDays = int(transitDays.(float64))
			}
		}

		// Extract weight information
		var chargedWeight float64
		var providedWeight float64
		if weight, ok := productMap["weight"].(map[string]interface{}); ok {
			if vol, exists := weight["volumetric"]; exists {
				chargedWeight = vol.(float64)
			}
			if prov, exists := weight["provided"]; exists {
				providedWeight = prov.(float64)
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
			ServiceType:     d.mapProductCodeToServiceType(productCode),
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
	return response
}

// mapProductCodeToServiceType maps DHL product codes to our service types
func (d *DHLImplementation) mapProductCodeToServiceType(productCode string) string {
	switch productCode {
	case "P":
		return "express"
	case "N":
		return "standard"
	default:
		return "express"
	}
}

// IsHealthy performs health check for DHL API
func (d *DHLImplementation) IsHealthy(ctx context.Context) error {
	// Create a minimal test request
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
				"number":   d.accountNumber,
			},
		},
		"productsAndServices": []map[string]interface{}{
			{
				"productCode":      "N",
				"localProductCode": "N",
			},
		},
		"payerCountryCode":              "IN",
		"plannedShippingDateAndTime":   time.Now().AddDate(0, 0, 1).Format("2006-01-02T15:04:05GMT-07:00"),
		"unitOfMeasurement":            "metric",
		"isCustomsDeclarable":          false,
		"estimatedDeliveryDate": map[string]interface{}{
			"isRequested": true,
			"typeCode":   "QDDC",
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

	// Try to make a health check call
	_, err := d.callDHLAPI(ctx, testRequest)
	return err
}
```

### 2. Partner Code Normalization Utility

**File:** `internal/shared/utils/v1/partner_utils.go`

```go
package v1

import (
	"regexp"
	"strings"
)

// NormalizePartnerCode converts partner codes to snake_case format
func NormalizePartnerCode(code string) string {
	if code == "" {
		return ""
	}

	// Convert to lowercase
	normalized := strings.ToLower(code)
	
	// Replace spaces, hyphens, and special characters with underscores
	re := regexp.MustCompile(`[^a-z0-9]+`)
	normalized = re.ReplaceAllString(normalized, "_")
	
	// Remove leading/trailing underscores
	normalized = strings.Trim(normalized, "_")
	
	// Replace multiple consecutive underscores with single underscore
	re = regexp.MustCompile(`_+`)
	normalized = re.ReplaceAllString(normalized, "_")
	
	return normalized
}

// Example conversions:
// "DHL" -> "dhl"
// "FedEx Express" -> "fedex_express"
// "Blue Dart" -> "blue_dart"
// "DHL-Express" -> "dhl_express"
// "UPS Ground" -> "ups_ground"
```

### 3. Updated Partner Resolution in Service

**File:** `internal/services/v1/rate_service.go` (Update GetQuotes method)

```go
// GetQuotes retrieves quotes from multiple partners (Updated)
func (s *RateService) GetQuotes(ctx context.Context, request *dtos.QuoteRequest, requestID string) (*dtos.QuoteResponse, error) {
	startTime := time.Now()

	response := &dtos.QuoteResponse{
		RequestID:    requestID,
		PartnerRates: make([]dtos.PartnerRateResult, 0, len(request.Partners)),
		RetrievedAt:  startTime,
	}

	// Process each partner
	for _, partner := range request.Partners {
		partnerStartTime := time.Now()
		
		// Normalize partner code
		normalizedCode := utils.NormalizePartnerCode(partner.Code)
		
		partnerResult := dtos.PartnerRateResult{
			Partner: dtos.PartnerInfo{
				ID:   partner.ID,
				Code: normalizedCode, // Use normalized code
			},
			Success:        false,
			AvailableRates: []dtos.Rate{},
		}

		// Use partner.ID if provided, otherwise use normalized code
		partnerIdentifier := partner.ID
		if partnerIdentifier == "" {
			partnerIdentifier = normalizedCode
		}

		// Get implementation for this partner
		implementation, err := s.factory.GetImplementationInstance(partnerIdentifier)
		if err != nil {
			// Try to create implementation based on partner code
			implementation, err = s.createImplementationByCode(normalizedCode)
			if err != nil {
				partnerResult.Error = &dtos.RateError{
					Code:    "PARTNER_NOT_FOUND",
					Message: "Partner implementation not found",
					Details: fmt.Sprintf("No implementation found for partner code: %s", normalizedCode),
				}
				partnerResult.ResponseTimeMs = time.Since(partnerStartTime).Milliseconds()
				response.PartnerRates = append(response.PartnerRates, partnerResult)
				continue
			}
		}

		// Convert request to internal format
		internalRequest := s.convertQuoteRequestToRateRequest(request)

		// Get rates from implementation
		rateResponse, err := implementation.GetRates(ctx, internalRequest)
		if err != nil {
			partnerResult.Error = &dtos.RateError{
				Code:    "RATE_FETCH_FAILED",
				Message: "Failed to fetch rates from partner",
				Details: err.Error(),
			}
		} else {
			// Convert response to simplified format
			partnerResult.AvailableRates = s.convertRateResponseToRates(rateResponse)
			partnerResult.Success = true
		}

		// Set data source based on implementation type
		if implementation.GetImplementationType() == dtos.ProviderTypePreDefined {
			partnerResult.DataSource = "pre_defined"
		} else {
			partnerResult.DataSource = "real_time"
		}

		partnerResult.ResponseTimeMs = time.Since(partnerStartTime).Milliseconds()
		response.PartnerRates = append(response.PartnerRates, partnerResult)
	}

	// Calculate summary
	response.Summary = s.calculateQuoteSummary(response.PartnerRates)

	return response, nil
}

// createImplementationByCode creates implementation based on partner code
func (s *RateService) createImplementationByCode(normalizedCode string) (interfaces.RateImplementation, error) {
	switch normalizedCode {
	case "dhl":
		return NewDHLImplementation(s.logger, s.metrics, s.httpClient), nil
	case "fedex":
		// return NewFedExImplementation(...), nil
		return nil, fmt.Errorf("FedEx implementation not yet available")
	case "ups":
		// return NewUPSImplementation(...), nil
		return nil, fmt.Errorf("UPS implementation not yet available")
	default:
		return nil, fmt.Errorf("unknown partner code: %s", normalizedCode)
	}
}
```

---

## Testing the Integration

### 1. Sample Request

```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "source_location": {
      "postal_code": "560086",
      "country_code": "IN"
    },
    "destination_location": {
      "postal_code": "266001",
      "country_code": "CN"
    },
    "packages": [
      {
        "weight": {
          "value": 0.5,
          "unit": "kg"
        },
        "dimensions": {
          "length": 121.0,
          "width": 20.0,
          "height": 30.0,
          "unit": "cm"
        }
      }
    ],
    "partners": [
      {
        "id": "",
        "code": "DHL"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "express"
    }
  }'
```

### 2. Expected Response

```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": {
    "request_id": "req-12345",
    "partner_rates": [
      {
        "partner": {
          "id": "",
          "code": "dhl"
        },
        "success": true,
        "available_rates": [
          {
            "rate_id": "dhl_p_0",
            "service": "EXPRESS WORLDWIDE",
            "price": {
              "currency": "INR",
              "amount": 15724.8,
              "type": "international_express",
              "criteria": {
                "product_code": "P",
                "weight_charged": 14.52,
                "weight_provided": 15,
                "route": "international"
              }
            },
            "delivery_days": 7
          }
        ],
        "data_source": "real_time",
        "response_time_ms": 1250
      }
    ],
    "summary": {
      "total_partners": 1,
      "successful_partners": 1,
      "total_rates_found": 1
    },
    "retrieved_at": "2025-09-23T13:15:00.000Z"
  },
  "meta": {
    "request_id": "req-12345",
    "response_time_ms": 1350,
    "version": "v1"
  },
  "timestamp": "2025-09-23T13:15:00.000Z"
}
```

---

## Error Handling

### 1. DHL API Errors

```json
{
  "partner_rates": [
    {
      "partner": {
        "id": "",
        "code": "dhl"
      },
      "success": false,
      "available_rates": [],
      "error": {
        "code": "RATE_FETCH_FAILED",
        "message": "Failed to fetch rates from partner",
        "details": "DHL API returned status 400: Invalid postal code"
      },
      "data_source": "real_time",
      "response_time_ms": 500
    }
  ]
}
```

### 2. Network/Timeout Errors

```json
{
  "error": {
    "code": "PARTNER_TIMEOUT",
    "message": "Partner API request timed out",
    "details": "DHL API did not respond within 30 seconds"
  }
}
```

---

## Configuration

### Environment Variables

```bash
# DHL Configuration
DHL_API_BASE_URL=https://express.api.dhl.com/mydhlapi/test/rates
DHL_API_USERNAME=shreemarut8IN
DHL_API_PASSWORD=I!0pM^4sR#4nJ$1u
DHL_ACCOUNT_NUMBER=533748932
DHL_TIMEOUT_MS=30000

# General Real-time Configuration  
REALTIME_MAX_CONCURRENT=5
REALTIME_DEFAULT_TIMEOUT_MS=30000
REALTIME_RETRY_COUNT=2
```

### Partner Configuration (Database)

```sql
INSERT INTO partners (id, code, name, type, is_active, config) VALUES
(
  uuid_generate_v4(),
  'dhl',
  'DHL Express',
  'real_time',
  true,
  '{
    "api_base_url": "https://express.api.dhl.com/mydhlapi/test/rates",
    "account_number": "533748932",
    "timeout_ms": 30000,
    "retry_count": 2,
    "supported_services": ["express", "standard"],
    "supported_countries": ["IN", "CN", "US", "GB", "DE"]
  }'
);
```

---

## Future Partner Integrations

### Adding New Partners

1. Create implementation file: `internal/services/v1/implementations/real_time/{partner}_implementation.go`
2. Implement the `RateImplementation` interface
3. Add partner code mapping in `createImplementationByCode` method
4. Add configuration and testing
5. Update documentation

### Supported Partner List

- ✅ **DHL Express** - International express delivery
- 🔄 **FedEx** - International and domestic express
- 🔄 **UPS** - Global logistics and package delivery  
- 🔄 **Blue Dart** - Domestic express delivery (India)
- 🔄 **Aramex** - Middle East and international express

### Partner Priority

Partners can be prioritized based on:
- Response time
- Success rate
- Pricing competitiveness
- Service coverage
- API reliability

---

## Performance Considerations

### Concurrent Requests
- Maximum 5 concurrent API calls per partner
- Timeout after 30 seconds
- Retry failed requests up to 2 times

### Caching Strategy
- Cache successful responses for 15 minutes
- Cache API failures for 5 minutes to avoid repeated calls
- Invalidate cache on partner configuration changes

### Rate Limiting
- Implement partner-specific rate limits
- Use exponential backoff for failed requests
- Monitor API quota usage

This integration guide provides a complete framework for adding real-time partners to the rate service, with DHL as the first implementation example.
