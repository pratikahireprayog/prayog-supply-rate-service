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
	ID   string `json:"id"` // Optional - can be empty, will use code instead
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
	RateID       string `json:"rate_id"`
	Service      string `json:"service"`
	Price        Price  `json:"price"`
	DeliveryDays *int   `json:"delivery_days,omitempty"`
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
