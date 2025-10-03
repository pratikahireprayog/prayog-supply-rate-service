package v1

import (
	"fmt"
	"time"

	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	v1 "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
)

// RateCalculationRequest represents the request for rate calculation
type RateCalculationRequest struct {
	RequestID    string    `json:"request_id" validate:"required"`
	CustomerID   string    `json:"customer_id" validate:"required"`
	OriginCity   string    `json:"origin_city" validate:"required,min=2,max=100"`
	DestCity     string    `json:"dest_city" validate:"required,min=2,max=100"`
	Weight       float64   `json:"weight" validate:"required,min=0.1,max=10000"`
	Distance     float64   `json:"distance" validate:"required,min=0.1,max=10000"`
	ServiceType  string    `json:"service_type" validate:"required,oneof=standard premium express same_day economy"`
	PickupDate   time.Time `json:"pickup_date" validate:"required"`
	DeliveryDate time.Time `json:"delivery_date" validate:"required,gtfield=PickupDate"`
	Priority     string    `json:"priority" validate:"required,oneof=low normal high urgent"`
	Currency     string    `json:"currency" validate:"required,oneof=INR USD EUR"`

	// Location details (for international shipping)
	OriginCountry string `json:"origin_country" validate:"required,len=2"` // ISO 3166-1 alpha-2
	DestCountry   string `json:"dest_country" validate:"required,len=2"`   // ISO 3166-1 alpha-2

	// Package information (for accurate pricing)
	Packages []PackageDetails `json:"packages" validate:"required,dive"`

	// Optional filters
	PartnerIDs    []string `json:"partner_ids,omitempty"`
	ProviderTypes []string `json:"provider_types,omitempty" validate:"dive,oneof=pre_defined real_time"`
	MaxPrice      *float64 `json:"max_price,omitempty" validate:"omitempty,min=0"`
	MaxDays       *int     `json:"max_days,omitempty" validate:"omitempty,min=1,max=30"`

	// Additional information
	InsuranceRequired bool                   `json:"insurance_required,omitempty"`
	TrackingRequired  bool                   `json:"tracking_required,omitempty"`
	SignatureRequired bool                   `json:"signature_required,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`

	// Request context
	Source    string `json:"source" validate:"required,oneof=api web mobile batch"`
	UserAgent string `json:"user_agent,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// PackageDetails represents detailed package information
type PackageDetails struct {
	Weight     float64 `json:"weight" validate:"required,min=0.1"`
	WeightUnit string  `json:"weight_unit" validate:"required,oneof=kg g lb"`
	Length     float64 `json:"length" validate:"required,min=0.1"`
	Width      float64 `json:"width" validate:"required,min=0.1"`
	Height     float64 `json:"height" validate:"required,min=0.1"`
	DimUnit    string  `json:"dimension_unit" validate:"required,oneof=cm in mm"`
}

// RateCalculationResponse represents the response from rate calculation
type RateCalculationResponse struct {
	RequestID    string      `json:"request_id"`
	Status       string      `json:"status"`
	Message      string      `json:"message"`
	Quotes       []RateQuote `json:"quotes"`
	BestQuote    *RateQuote  `json:"best_quote,omitempty"`
	TotalQuotes  int         `json:"total_quotes"`
	ResponseTime int64       `json:"response_time_ms"`
	CacheHit     bool        `json:"cache_hit"`
	Timestamp    time.Time   `json:"timestamp"`

	// Aggregated information
	PriceRange    *PriceRange `json:"price_range,omitempty"`
	DeliveryRange *TimeRange  `json:"delivery_range,omitempty"`

	// Errors from providers
	Errors []ProviderError `json:"errors,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RateQuote represents a single rate quote from a provider
type RateQuote struct {
	QuoteID      string         `json:"quote_id"`
	PartnerID    string         `json:"partner_id"`
	PartnerName  string         `json:"partner_name"`
	ProviderType v1.PartnerType `json:"provider_type"`

	// Pricing information
	BasePrice      float64        `json:"base_price"`
	TotalPrice     float64        `json:"total_price"`
	Currency       string         `json:"currency"`
	PriceBreakdown PriceBreakdown `json:"price_breakdown"`

	// Service information
	ServiceType    string `json:"service_type"`
	ServiceLevel   string `json:"service_level"`
	EstimatedDays  int    `json:"estimated_days"`
	EstimatedHours int    `json:"estimated_hours"`

	// Validity and confidence
	ValidUntil      time.Time `json:"valid_until"`
	Confidence      float64   `json:"confidence"`
	IsRecommended   bool      `json:"is_recommended"`
	RecommendReason string    `json:"recommend_reason,omitempty"`

	// Additional services
	InsuranceAvailable bool `json:"insurance_available"`
	TrackingAvailable  bool `json:"tracking_available"`
	SignatureAvailable bool `json:"signature_available"`

	// Metadata
	Source          string                 `json:"source"`
	ResponseTimeMs  int64                  `json:"response_time_ms"`
	ExternalQuoteID string                 `json:"external_quote_id,omitempty"`
	Terms           string                 `json:"terms,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// PriceBreakdown provides detailed price information
type PriceBreakdown struct {
	BasePrice       float64 `json:"base_price"`
	WeightCharge    float64 `json:"weight_charge"`
	DistanceCharge  float64 `json:"distance_charge"`
	FuelSurcharge   float64 `json:"fuel_surcharge"`
	HandlingCharge  float64 `json:"handling_charge"`
	InsuranceCharge float64 `json:"insurance_charge"`
	TaxAmount       float64 `json:"tax_amount"`
	DiscountAmount  float64 `json:"discount_amount"`
	TotalPrice      float64 `json:"total_price"`
}

// PriceRange represents the price range across all quotes
type PriceRange struct {
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
	AvgPrice float64 `json:"avg_price"`
	Currency string  `json:"currency"`
}

// TimeRange represents the delivery time range
type TimeRange struct {
	MinDays int `json:"min_days"`
	MaxDays int `json:"max_days"`
	AvgDays int `json:"avg_days"`
}

// ProviderError represents errors from individual providers
type ProviderError struct {
	PartnerID    string    `json:"partner_id"`
	PartnerName  string    `json:"partner_name"`
	ErrorCode    string    `json:"error_code"`
	ErrorMessage string    `json:"error_message"`
	Timestamp    time.Time `json:"timestamp"`
}

// RateComparisonRequest represents request for comparing rates
type RateComparisonRequest struct {
	RateCalculationRequest

	// Comparison specific options
	SortBy    string   `json:"sort_by" validate:"omitempty,oneof=price delivery_time confidence partner_priority"`
	SortOrder string   `json:"sort_order" validate:"omitempty,oneof=asc desc"`
	GroupBy   string   `json:"group_by" validate:"omitempty,oneof=partner service_type delivery_time"`
	Filters   []Filter `json:"filters,omitempty"`
	Limit     int      `json:"limit" validate:"omitempty,min=1,max=100"`
}

// RateComparisonResponse represents response for rate comparison
type RateComparisonResponse struct {
	RequestID    string                 `json:"request_id"`
	Comparison   RateComparison         `json:"comparison"`
	Quotes       []RateQuote            `json:"quotes"`
	Summary      ComparisonSummary      `json:"summary"`
	ResponseTime int64                  `json:"response_time_ms"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RateComparison provides detailed comparison information
type RateComparison struct {
	BestPrice       *RateQuote `json:"best_price,omitempty"`
	FastestDelivery *RateQuote `json:"fastest_delivery,omitempty"`
	MostReliable    *RateQuote `json:"most_reliable,omitempty"`
	Recommended     *RateQuote `json:"recommended,omitempty"`
}

// ComparisonSummary provides summary statistics
type ComparisonSummary struct {
	TotalQuotes       int        `json:"total_quotes"`
	PriceRange        PriceRange `json:"price_range"`
	DeliveryRange     TimeRange  `json:"delivery_range"`
	ProviderTypes     []string   `json:"provider_types"`
	ServiceTypes      []string   `json:"service_types"`
	AverageConfidence float64    `json:"average_confidence"`
}

// Filter represents a filter for rate comparison
type Filter struct {
	Field    string      `json:"field" validate:"required"`
	Operator string      `json:"operator" validate:"required,oneof=eq neq gt gte lt lte in nin"`
	Value    interface{} `json:"value" validate:"required"`
}

// Validate validates the rate calculation request
func (r *RateCalculationRequest) Validate() error {
	// Custom validation logic can be added here
	if r.DeliveryDate.Before(r.PickupDate) {
		return constants.ErrInvalidDateRange
	}

	if r.Weight <= 0 {
		return constants.ErrInvalidWeight
	}

	if r.Distance <= 0 {
		return constants.ErrInvalidDistance
	}

	return nil
}

// GetCacheKey generates a cache key for the request
func (r *RateCalculationRequest) GetCacheKey() string {
	return fmt.Sprintf("rate:%s:%s:%s:%.2f:%.2f:%s:%s",
		r.OriginCity, r.DestCity, r.ServiceType,
		r.Weight, r.Distance, r.Currency,
		r.PickupDate.Format("2006-01-02"))
}

// IsUrgent checks if the request has urgent priority
func (r *RateCalculationRequest) IsUrgent() bool {
	return r.Priority == "urgent" || r.Priority == "high"
}
