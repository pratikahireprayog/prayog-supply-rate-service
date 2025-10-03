package unified_rate

import (
	"time"
)

// LocationInfo represents location information extracted from postal codes
type LocationInfo struct {
	PostalCode  string  `json:"postal_code"`
	CountryCode string  `json:"country_code"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
}

// UnifiedRateRequest represents the request format for Prayog Unified API rate calculation
type UnifiedRateRequest struct {
	SourceLocation      LocationRequest        `json:"sourceLocation"`
	DestinationLocation LocationRequest        `json:"destinationLocation"`
	Packages            []PackageRequest       `json:"packages"`
	ServiceTypes        []string               `json:"serviceTypes,omitempty"`
	Currency            string                 `json:"currency,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// LocationRequest represents location information in the unified API request
type LocationRequest struct {
	PostalCode  string  `json:"postalCode"`
	CountryCode string  `json:"countryCode"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
}

// PackageRequest represents package information in the unified API request
type PackageRequest struct {
	Weight     WeightRequest     `json:"weight"`
	Dimensions DimensionsRequest `json:"dimensions"`
}

// WeightRequest represents weight information in the unified API request
type WeightRequest struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"` // GRAMS, KG, LB
}

// DimensionsRequest represents dimension information in the unified API request
type DimensionsRequest struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"` // cm, in, mm
}

// UnifiedRateResponse represents the response format from Prayog Unified API
type UnifiedRateResponse struct {
	Success   bool                    `json:"success"`
	Message   string                  `json:"message"`
	Data      UnifiedRateResponseData `json:"data,omitempty"`
	Error     *UnifiedAPIError        `json:"error,omitempty"`
	Meta      *UnifiedResponseMeta    `json:"meta,omitempty"`
	Timestamp string                  `json:"timestamp"`
}

// UnifiedRateResponseData represents the data section of unified API response
type UnifiedRateResponseData struct {
	RequestID        string              `json:"requestId"`
	RateCalculations []RateCalculation   `json:"rateCalculations"`
	Summary          *CalculationSummary `json:"summary,omitempty"`
	ProcessingTime   int64               `json:"processingTimeMs"`
}

// RateCalculation represents rate calculation for a specific service type
type RateCalculation struct {
	ServiceType  string        `json:"serviceType"`
	ServiceRates []ServiceRate `json:"serviceRates"`
	Status       string        `json:"status"`
	Error        string        `json:"error,omitempty"`
}

// ServiceRate represents a specific rate quote for a service
type ServiceRate struct {
	RateID        string         `json:"rateId"`
	ServiceName   string         `json:"serviceName"`
	ServiceCode   string         `json:"serviceCode,omitempty"`
	BaseRate      float64        `json:"baseRate"`
	TotalRate     float64        `json:"totalRate"`
	Currency      string         `json:"currency"`
	EstimatedDays int            `json:"estimatedDays,omitempty"`
	Charges       []ChargeDetail `json:"charges,omitempty"`
	ValidUntil    string         `json:"validUntil,omitempty"`
	RateSource    string         `json:"rateSource,omitempty"` // "matrix", "api", "calculation"
	Confidence    float64        `json:"confidence,omitempty"`
}

// ChargeDetail represents individual charge details
type ChargeDetail struct {
	ChargeName  string  `json:"chargeName"`
	ChargeCode  string  `json:"chargeCode"`
	ChargeType  string  `json:"chargeType"` // FIXED, PERCENTAGE, FORMULA, SLAB
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	IsTax       bool    `json:"isTax"`
	Description string  `json:"description,omitempty"`
}

// CalculationSummary provides summary information about the rate calculation
type CalculationSummary struct {
	TotalServiceTypes      int     `json:"totalServiceTypes"`
	SuccessfulCalculations int     `json:"successfulCalculations"`
	TotalQuotesGenerated   int     `json:"totalQuotesGenerated"`
	AverageResponseTime    float64 `json:"averageResponseTimeMs"`
	CurrencyUsed           string  `json:"currencyUsed"`
}

// UnifiedAPIError represents error information from unified API
type UnifiedAPIError struct {
	Code     string                 `json:"code"`
	Message  string                 `json:"message"`
	Details  string                 `json:"details,omitempty"`
	Field    string                 `json:"field,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UnifiedResponseMeta represents metadata in unified API response
type UnifiedResponseMeta struct {
	RequestID          string `json:"requestId"`
	Version            string `json:"version"`
	ResponseTimeMs     int64  `json:"responseTimeMs"`
	RateLimitRemaining int    `json:"rateLimitRemaining,omitempty"`
	RateLimitReset     int64  `json:"rateLimitReset,omitempty"`
}

// RateCardRequest represents request for creating/managing rate cards
type RateCardRequest struct {
	Name          string         `json:"name"`
	ProductType   string         `json:"productType"`
	IsActive      bool           `json:"isActive"`
	IsDefault     bool           `json:"isDefault"`
	EffectiveFrom string         `json:"effectiveFrom"`
	EffectiveTo   string         `json:"effectiveTo"`
	Matrices      []RateMatrix   `json:"matrices"`
	Charges       []ChargeConfig `json:"charges"`
}

// RateMatrix represents rate matrix structure
type RateMatrix struct {
	MatrixKey string      `json:"matrixKey"`
	Rows      []MatrixRow `json:"rows"`
}

// MatrixRow represents a single row in rate matrix
type MatrixRow struct {
	Rate                   float64     `json:"rate"`
	Currency               string      `json:"currency"`
	Location               string      `json:"location"`
	ExcessCalculationUnit  string      `json:"excessCalculationUnit,omitempty"`
	ExcessCalculationValue float64     `json:"excessCalculationValue,omitempty"`
	IsLastSlab             bool        `json:"isLastSlab,omitempty"`
	Dimensions             []Dimension `json:"dimensions"`
}

// Dimension represents dimension criteria for rate calculation
type Dimension struct {
	DimensionKey string   `json:"dimensionKey"`
	MinValue     float64  `json:"minValue"`
	MaxValue     *float64 `json:"maxValue"` // null for open-ended ranges
	Unit         string   `json:"unit"`
}

// ChargeConfig represents charge configuration
type ChargeConfig struct {
	ChargeName        string              `json:"chargeName"`
	ChargeCode        string              `json:"chargeCode"`
	ChargeType        string              `json:"chargeType"`
	ChargeLevel       string              `json:"chargeLevel"`
	CalculationConfig CalculationConfig   `json:"calculationConfig"`
	TaxConfig         *TaxConfig          `json:"taxConfig,omitempty"`
	ApplicableFilters map[string][]string `json:"applicableFilters"`
	Priority          int                 `json:"priority"`
	IsActive          bool                `json:"isActive"`
	IsTax             bool                `json:"isTax"`
}

// CalculationConfig represents configuration for charge calculation
type CalculationConfig struct {
	FixedAmount   float64            `json:"fixedAmount,omitempty"`
	Percentage    float64            `json:"percentage,omitempty"`
	Formula       string             `json:"formula,omitempty"`
	Variables     map[string]string  `json:"variables,omitempty"`
	Slabs         []SlabConfig       `json:"slabs,omitempty"`
	BaseDimension string             `json:"baseDimension,omitempty"`
	UserOptions   *UserOptionsConfig `json:"userOptions,omitempty"`
}

// SlabConfig represents slab-based calculation configuration
type SlabConfig struct {
	From                   float64  `json:"from"`
	To                     *float64 `json:"to"` // null for open-ended
	Rate                   float64  `json:"rate"`
	IsLastSlab             bool     `json:"isLastSlab,omitempty"`
	ExcessCalculationUnit  string   `json:"excessCalculationUnit,omitempty"`
	ExcessCalculationValue float64  `json:"excessCalculationValue,omitempty"`
}

// UserOptionsConfig represents user options for charges
type UserOptionsConfig struct {
	Required []string `json:"required"`
}

// TaxConfig represents tax configuration for charges
type TaxConfig struct {
	IsTaxable   bool    `json:"isTaxable"`
	TaxRate     float64 `json:"taxRate"`
	TaxOnAmount string  `json:"taxOnAmount"` // BASE_RATE, CHARGE_AMOUNT
}

// HealthCheckRequest represents health check request
type HealthCheckRequest struct {
	ServiceType string                 `json:"serviceType,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// HealthCheckResponse represents health check response
type HealthCheckResponse struct {
	Success      bool                   `json:"success"`
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	ResponseTime int64                  `json:"responseTimeMs"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// APIUsageStats represents API usage statistics
type APIUsageStats struct {
	RequestsToday       int     `json:"requestsToday"`
	RequestsThisHour    int     `json:"requestsThisHour"`
	AverageResponseTime float64 `json:"averageResponseTimeMs"`
	SuccessRate         float64 `json:"successRate"`
	ErrorRate           float64 `json:"errorRate"`
	RateLimitRemaining  int     `json:"rateLimitRemaining"`
	RateLimitReset      int64   `json:"rateLimitReset"`
}

// ValidationError represents validation error details
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// APIErrorResponse represents structured error response from API
type APIErrorResponse struct {
	Success          bool              `json:"success"`
	Error            *UnifiedAPIError  `json:"error"`
	ValidationErrors []ValidationError `json:"validationErrors,omitempty"`
	Timestamp        time.Time         `json:"timestamp"`
	RequestID        string            `json:"requestId,omitempty"`
}

// RateCalculationContext represents context for rate calculation
type RateCalculationContext struct {
	RequestID      string                 `json:"requestId"`
	UserID         string                 `json:"userId,omitempty"`
	SessionID      string                 `json:"sessionId,omitempty"`
	IPAddress      string                 `json:"ipAddress,omitempty"`
	UserAgent      string                 `json:"userAgent,omitempty"`
	RequestTime    time.Time              `json:"requestTime"`
	Priority       string                 `json:"priority,omitempty"`
	Source         string                 `json:"source,omitempty"`
	AdditionalData map[string]interface{} `json:"additionalData,omitempty"`
}
