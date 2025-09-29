# Implementation Guide: Simplified Quotes Endpoint

## Overview
Implement a single, simplified endpoint `POST /supply-rate/v1/quotes` that retrieves rates from multiple partners with a minimal, flexible response structure.

## Changes Summary
1. Single endpoint `/quotes` instead of multiple rate endpoints
2. Simplified partner object (only id and code)
3. Flexible price structure to accommodate different partner types
4. Data source limited to `pre_defined` and `real_time`
5. No price breakdown for now (simplified pricing)

---

## 1. Update Routes Structure

### File: `internal/infrastructure/api/http/v1/routes/rate_routes.go`

Replace the entire routes setup with:

```go
package routes

import (
	"github.com/gofiber/fiber/v2"

	handlersv1 "github.com/prayog/prayog-supply-rate-service/internal/infrastructure/api/http/v1/handlers"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// SetupRateRoutes sets up the simplified quote route
func SetupRateRoutes(
	router fiber.Router,
	rateService interfaces.RateService,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) {
	// Create rate handler
	rateHandler := handlersv1.NewRateHandler(rateService, logger, metrics)

	// Single quotes endpoint
	router.Post("/quotes", rateHandler.GetQuotes)
	
	// Health endpoint for implementations
	// Implementation health is now available via deep health check: /supply-rate/health?deep=true
}
```

---

## 2. Create New DTOs

### File: `internal/shared/dtos/v1/quotes.go` (NEW FILE)

```go
package v1

import (
	"time"
)

// QuoteRequest represents the request for getting quotes from partners
type QuoteRequest struct {
	SourceLocation      Location               `json:"source_location" validate:"required"`
	DestinationLocation Location               `json:"destination_location" validate:"required"`
	Packages            []Package              `json:"packages" validate:"required,dive"`
	Partners            []Partner              `json:"partners" validate:"required,dive"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// Location represents a geographical location
type Location struct {
	PostalCode  string  `json:"postal_code" validate:"required"`
	CountryCode string  `json:"country_code" validate:"required,len=2"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
}

// Package represents a package to be shipped
type Package struct {
	Weight     Weight     `json:"weight" validate:"required"`
	Dimensions Dimensions `json:"dimensions" validate:"required"`
}

// Weight represents package weight
type Weight struct {
	Value float64 `json:"value" validate:"required,min=0.1"`
	Unit  string  `json:"unit" validate:"required,oneof=kg g lb"`
}

// Dimensions represents package dimensions
type Dimensions struct {
	Length float64 `json:"length" validate:"required,min=0.1"`
	Width  float64 `json:"width" validate:"required,min=0.1"`
	Height float64 `json:"height" validate:"required,min=0.1"`
	Unit   string  `json:"unit" validate:"required,oneof=cm in mm"`
}

// Partner represents a shipping partner
type Partner struct {
	ID   string `json:"id" validate:"required"`
	Code string `json:"code" validate:"required"`
}

// QuoteResponse represents the response from quote request
type QuoteResponse struct {
	RequestID    string              `json:"request_id"`
	PartnerRates []PartnerRateResult `json:"partner_rates"`
	Summary      QuoteSummary        `json:"summary"`
	RetrievedAt  time.Time           `json:"retrieved_at"`
}

// PartnerRateResult represents rates from a single partner
type PartnerRateResult struct {
	Partner        PartnerInfo `json:"partner"`
	Success        bool        `json:"success"`
	AvailableRates []Rate      `json:"available_rates,omitempty"`
	Error          *RateError  `json:"error,omitempty"`
	DataSource     string      `json:"data_source"` // "pre_defined" or "real_time"
	ResponseTimeMs int64       `json:"response_time_ms"`
}

// PartnerInfo represents partner information
type PartnerInfo struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

// Rate represents a single rate quote
type Rate struct {
	RateID       string                 `json:"rate_id"`
	Service      string                 `json:"service"`
	Price        Price                  `json:"price"`
	DeliveryDays *int                   `json:"delivery_days,omitempty"`
}

// Price represents pricing information
type Price struct {
	Currency string                 `json:"currency"`
	Amount   float64                `json:"amount"`
	Type     string                 `json:"type,omitempty"`
	Criteria map[string]interface{} `json:"criteria,omitempty"`
}

// RateError represents an error from a partner
type RateError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// QuoteSummary represents summary of quote results
type QuoteSummary struct {
	TotalPartners      int `json:"total_partners"`
	SuccessfulPartners int `json:"successful_partners"`
	TotalRatesFound    int `json:"total_rates_found"`
}
```

---

## 3. Update Handler

### File: `internal/infrastructure/api/http/v1/handlers/rate_handler.go`

Add this new method to the existing RateHandler:

```go
// GetQuotes handles POST /quotes
func (h *RateHandler) GetQuotes(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := c.Get(constants.HeaderRequestID)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Parse request body
	var req dtos.QuoteRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warn("Invalid request body",
			"error", err,
			"request_id", requestID,
			"path", c.Path())

		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			constants.MessageInvalidRequest,
			constants.CodeInvalidRequest,
			err.Error(),
		))
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.logger.Warn("Request validation failed",
			"error", err,
			"request_id", requestID)

		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidationErrorResponse(err))
	}

	h.logger.Info("Processing quote request",
		"request_id", requestID,
		"source_postal_code", req.SourceLocation.PostalCode,
		"destination_postal_code", req.DestinationLocation.PostalCode,
		"partners_count", len(req.Partners))

	// Call service
	response, err := h.rateService.GetQuotes(c.Context(), &req, requestID)
	if err != nil {
		h.logger.Error("Get quotes failed",
			"error", err,
			"request_id", requestID)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	duration := time.Since(startTime)
	h.metrics.IncrementCounter("quotes_success", map[string]string{
		"endpoint":       "get_quotes",
		"total_partners": fmt.Sprintf("%d", response.Summary.TotalPartners),
	})
	h.metrics.RecordTimer("quotes_duration", duration, map[string]string{
		"endpoint": "get_quotes",
	})

	h.logger.Info("Quote request completed",
		"request_id", requestID,
		"total_partners", response.Summary.TotalPartners,
		"successful_partners", response.Summary.SuccessfulPartners,
		"total_rates", response.Summary.TotalRatesFound,
		"duration_ms", duration.Milliseconds())

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(response))
}
```

---

## 4. Update Service Interface

### File: `internal/shared/interfaces/v1/rate.go`

Add this method to the RateService interface:

```go
// RateService defines the main service interface for rate operations
type RateService interface {
	// GetQuotes retrieves quotes from multiple partners
	GetQuotes(ctx context.Context, request *dtos.QuoteRequest, requestID string) (*dtos.QuoteResponse, error)
	
	// Keep existing methods
	GetImplementationHealth(ctx context.Context) (*dtos.ProviderHealthResponse, error)
	RefreshImplementations(ctx context.Context) error
}
```

---

## 5. Update Service Implementation

### File: `internal/services/v1/rate_service.go`

Add this method to the existing RateService struct:

```go
// GetQuotes retrieves quotes from multiple partners
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
		
		partnerResult := dtos.PartnerRateResult{
			Partner: dtos.PartnerInfo{
				ID:   partner.ID,
				Code: partner.Code,
			},
			Success:        false,
			AvailableRates: []dtos.Rate{},
		}

		// Get implementation for this partner
		implementation, err := s.factory.GetImplementationInstance(partner.ID)
		if err != nil {
			partnerResult.Error = &dtos.RateError{
				Code:    "PARTNER_NOT_FOUND",
				Message: "Partner implementation not found",
				Details: err.Error(),
			}
			partnerResult.ResponseTimeMs = time.Since(partnerStartTime).Milliseconds()
			response.PartnerRates = append(response.PartnerRates, partnerResult)
			continue
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

// Helper method to convert quote request to internal rate request
func (s *RateService) convertQuoteRequestToRateRequest(req *dtos.QuoteRequest) *dtos.RateCalculationRequest {
	// Extract metadata values with defaults
	currency := "INR"
	serviceType := "standard"
	
	if req.Metadata != nil {
		if curr, ok := req.Metadata["currency"].(string); ok {
			currency = curr
		}
		if svc, ok := req.Metadata["service_type"].(string); ok {
			serviceType = svc
		}
	}

	// Calculate total weight (simplified - sum all packages)
	totalWeight := 0.0
	for _, pkg := range req.Packages {
		totalWeight += pkg.Weight.Value
	}

	return &dtos.RateCalculationRequest{
		RequestID:    uuid.New().String(),
		OriginCity:   req.SourceLocation.PostalCode,
		DestCity:     req.DestinationLocation.PostalCode,
		Weight:       totalWeight,
		ServiceType:  serviceType,
		Currency:     currency,
		PickupDate:   time.Now(),
		DeliveryDate: time.Now().AddDate(0, 0, 1), // Default next day
		Priority:     "normal",
		Source:       "api",
	}
}

// Helper method to convert internal rate response to simplified rates
func (s *RateService) convertRateResponseToRates(resp *dtos.RateCalculationResponse) []dtos.Rate {
	rates := make([]dtos.Rate, 0, len(resp.Quotes))
	
	for _, quote := range resp.Quotes {
		rate := dtos.Rate{
			RateID:  quote.QuoteID,
			Service: quote.PartnerName,
			Price: dtos.Price{
				Currency: quote.Currency,
				Amount:   quote.TotalPrice,
				Type:     "standard",
			},
		}
		
		if quote.EstimatedDays > 0 {
			rate.DeliveryDays = &quote.EstimatedDays
		}
		
		rates = append(rates, rate)
	}
	
	return rates
}

// Helper method to calculate quote summary
func (s *RateService) calculateQuoteSummary(partnerRates []dtos.PartnerRateResult) dtos.QuoteSummary {
	summary := dtos.QuoteSummary{
		TotalPartners: len(partnerRates),
	}
	
	for _, partnerRate := range partnerRates {
		if partnerRate.Success {
			summary.SuccessfulPartners++
			summary.TotalRatesFound += len(partnerRate.AvailableRates)
		}
	}
	
	return summary
}
```

---

## 6. Update Main Server

### File: `cmd/server/main.go`

Ensure the route setup calls the updated function:

```go
// In your server setup, make sure you're calling:
routes.SetupRateRoutes(app.Group("/supply-rate/v1"), rateService, logger, metrics)
```

---

## Implementation Steps

1. **Create** the new DTO file `internal/shared/dtos/v1/quotes.go`
2. **Update** the routes file to use the single `/quotes` endpoint
3. **Add** the `GetQuotes` method to the rate handler
4. **Add** the `GetQuotes` method to the service interface
5. **Implement** the `GetQuotes` method in the rate service
6. **Test** the endpoint with sample requests
7. **Remove** old unused endpoints and DTOs (after testing)

---

## Sample Request for Testing

```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "source_location": {
      "postal_code": "560001",
      "country_code": "IN"
    },
    "destination_location": {
      "postal_code": "110001",
      "country_code": "IN"
    },
    "packages": [
      {
        "weight": {
          "value": 1.0,
          "unit": "kg"
        },
        "dimensions": {
          "length": 10.0,
          "width": 10.0,
          "height": 10.0,
          "unit": "cm"
        }
      }
    ],
    "partners": [
      {
        "id": "partner-1",
        "code": "DELHIVERY"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "standard"
    }
  }'
```

---

## Expected Response Format

```json
{
  "success": true,
  "message": "Rate quotes retrieved successfully.",
  "metadata": {
    "request_id": "req-123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2024-01-15T10:30:00Z",
    "response_time_ms": 120,
    "partners_queried": 1,
    "partners_succeeded": 1,
    "partners_failed": 0,
    "total_rates_found": 1
  },
  "data": {
    "successful_responses": [
      {
        "partner": {
          "code": "delhivery",
          "name": "Delhivery"
        },
        "source": "pre_defined",
        "available_rates": [
          {
            "rate_id": "del-surface-standard-001",
            "service": "Surface Delivery",
            "price": {
              "currency": "INR",
              "amount": 75.50,
              "type": "weight_distance_based",
              "criteria": {
                "weight_range": "0-5 kg",
                "distance_range": "100-500 km"
              }
            },
            "delivery_days": 3
          }
        ],
        "response_time_ms": 45
      }
    ],
    "failed_responses": []
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Partner-Specific Implementations

### Porter (Distance-based pricing per city)
```json
{
  "rate_id": "PORTER-BLR-001",
  "service": "Average Fare",
  "price": {
    "currency": "INR",
    "amount": 50.0,
    "type": "distance_based",
    "criteria": {
      "city": "Bangalore",
      "distance_range": "0-1 km"
    }
  }
}
```

### DHL (Complex international)
```json
{
  "success": true,
  "message": "Rate quotes retrieved successfully.",
  "metadata": {
    "request_id": "req-a1b2c3d4-e5f6-7890-1234-567890abcdef",
    "timestamp": "2025-09-30T02:20:00Z",
    "response_time_ms": 1450,
    "partners_queried": 1,
    "partners_succeeded": 1,
    "partners_failed": 0,
    "total_rates_found": 2
  },
  "data": {
    "successful_responses": [
      {
        "partner": {
          "code": "dhl",
          "name": "DHL Express"
        },
        "source": "real_time",
        "available_rates": [
          {
            "rate_id": "dhl-express-worldwide-123",
            "service": "EXPRESS WORLDWIDE",
            "price": {
              "currency": "INR",
              "amount": 15724.80,
              "type": "real_time_international"
            },
            "delivery_days": 7
          },
          {
            "rate_id": "dhl-economy-select-456",
            "service": "ECONOMY SELECT",
            "price": {
              "currency": "INR",
              "amount": 12500.00,
              "type": "real_time_international"
            },
            "delivery_days": 10
          }
        ],
        "response_time_ms": 1250
      }
    ],
    "failed_responses": []
  },
  "timestamp": "2025-09-30T02:20:00Z"
}
```

### Multi-Partner Response with Success and Failure
```json
{
  "success": true,
  "message": "Rate quotes retrieved successfully.",
  "metadata": {
    "request_id": "req-a1b2c3d4-e5f6-7890-1234-567890abcdef",
    "timestamp": "2025-09-30T02:20:00Z",
    "response_time_ms": 1450,
    "partners_queried": 3,
    "partners_succeeded": 2,
    "partners_failed": 1,
    "total_rates_found": 3
  },
  "data": {
    "successful_responses": [
      {
        "partner": {
          "code": "dhl",
          "name": "DHL Express"
        },
        "source": "real_time",
        "available_rates": [
          {
            "rate_id": "dhl-express-worldwide-123",
            "service": "EXPRESS WORLDWIDE",
            "price": {
              "currency": "INR",
              "amount": 15724.80,
              "type": "real_time_international"
            }
          },
          {
            "rate_id": "dhl-economy-select-456",
            "service": "ECONOMY SELECT",
            "price": {
              "currency": "INR",
              "amount": 12500.00,
              "type": "real_time_international"
            }
          }
        ]
      },
      {
        "partner": {
          "code": "delhivery",
          "name": "Delhivery"
        },
        "source": "pre_defined",
        "available_rates": [
          {
            "rate_id": "del-surface-standard-789",
            "service": "Surface Standard",
            "price": {
              "currency": "INR",
              "amount": 150.00,
              "type": "weight_distance_based"
            }
          }
        ]
      }
    ],
    "failed_responses": [
      {
        "partner": {
          "code": "fedex",
          "name": "FedEx"
        },
        "source": "real_time",
        "error": {
          "code": "PARTNER_TIMEOUT",
          "message": "The request to the partner API timed out after 3000ms."
        }
      }
    ]
  }
}
```

---

## Benefits of This Implementation

1. **Minimal Core Structure**: Just partner id/code + rate id/service/price
2. **Flexible Price Types**: Can handle any pricing model via `type` and `criteria`
3. **Partner Agnostic**: Works with Porter, DHL, Delhivery, and future partners
4. **Extensible**: Add new fields without breaking existing structure
5. **Simple to Start**: Can begin with basic fields, add complexity later
6. **Generic Response**: Same response format for all partner types

This implementation provides a clean foundation that can evolve as requirements grow while maintaining backward compatibility.
