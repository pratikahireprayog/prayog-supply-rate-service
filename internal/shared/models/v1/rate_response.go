package v1

import (
	"time"

	"github.com/google/uuid"
)

// RateResponse represents the response from a rate calculation
type RateResponse struct {
	ID          uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	QueryID     uuid.UUID   `json:"query_id" gorm:"type:uuid;not null;index" validate:"required"`
	PartnerID   string      `json:"partner_id" gorm:"not null;index" validate:"required"`
	PartnerName string      `json:"partner_name" gorm:"not null" validate:"required"`
	PartnerType PartnerType `json:"partner_type" gorm:"not null" validate:"required"`

	// Rate Information
	BasePrice  float64 `json:"base_price" gorm:"not null" validate:"required,min=0"`
	TotalPrice float64 `json:"total_price" gorm:"not null" validate:"required,min=0"`
	Currency   string  `json:"currency" gorm:"not null;default:INR" validate:"required,oneof=INR USD EUR"`

	// Additional Charges
	FuelSurcharge   float64 `json:"fuel_surcharge" gorm:"not null;default:0" validate:"min=0"`
	HandlingCharge  float64 `json:"handling_charge" gorm:"not null;default:0" validate:"min=0"`
	InsuranceCharge float64 `json:"insurance_charge" gorm:"not null;default:0" validate:"min=0"`
	TaxAmount       float64 `json:"tax_amount" gorm:"not null;default:0" validate:"min=0"`
	DiscountAmount  float64 `json:"discount_amount" gorm:"not null;default:0" validate:"min=0"`

	// Service Information
	ServiceType    string `json:"service_type" gorm:"not null" validate:"required"`
	EstimatedDays  int    `json:"estimated_days" gorm:"not null" validate:"required,min=1,max=30"`
	EstimatedHours int    `json:"estimated_hours" gorm:"not null" validate:"required,min=1,max=720"`
	ServiceLevel   string `json:"service_level" gorm:"not null;default:standard" validate:"required,oneof=economy standard premium express"`

	// Validity and Status
	ValidUntil    time.Time `json:"valid_until" gorm:"not null" validate:"required"`
	Status        string    `json:"status" gorm:"not null;default:active" validate:"required,oneof=active expired invalid"`
	IsRecommended bool      `json:"is_recommended" gorm:"not null;default:false"`
	Confidence    float64   `json:"confidence" gorm:"not null;default:1.0" validate:"required,min=0,max=1"`

	// Source and Metadata
	Source          string     `json:"source" gorm:"not null" validate:"required,oneof=static dynamic cached"`
	ResponseTimeMs  int64      `json:"response_time_ms" gorm:"not null" validate:"required,min=0"`
	CacheHit        bool       `json:"cache_hit" gorm:"not null;default:false"`
	RateRuleID      *uuid.UUID `json:"rate_rule_id,omitempty" gorm:"type:uuid;index"`
	ExternalQuoteID *string    `json:"external_quote_id,omitempty" gorm:""`

	// Additional Data
	Terms    *string                `json:"terms,omitempty" gorm:"type:text"`
	Notes    *string                `json:"notes,omitempty" gorm:"type:text"`
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (RateResponse) TableName() string {
	return "rate_responses"
}

// IsValid checks if the rate response is still valid
func (rr *RateResponse) IsValid() bool {
	return rr.Status == "active" && time.Now().Before(rr.ValidUntil)
}

// IsExpired checks if the rate response has expired
func (rr *RateResponse) IsExpired() bool {
	return rr.Status == "expired" || time.Now().After(rr.ValidUntil)
}

// GetTotalWithTax returns the total price including tax
func (rr *RateResponse) GetTotalWithTax() float64 {
	return rr.TotalPrice + rr.TaxAmount
}

// GetNetPrice returns the price after discount and before tax
func (rr *RateResponse) GetNetPrice() float64 {
	return rr.TotalPrice - rr.DiscountAmount
}

// GetSavingsAmount returns the discount amount as savings
func (rr *RateResponse) GetSavingsAmount() float64 {
	return rr.DiscountAmount
}

// GetPriceBreakdown returns a detailed breakdown of the price
func (rr *RateResponse) GetPriceBreakdown() map[string]float64 {
	return map[string]float64{
		"base_price":       rr.BasePrice,
		"fuel_surcharge":   rr.FuelSurcharge,
		"handling_charge":  rr.HandlingCharge,
		"insurance_charge": rr.InsuranceCharge,
		"tax_amount":       rr.TaxAmount,
		"discount_amount":  -rr.DiscountAmount, // Negative because it's a reduction
		"total_price":      rr.TotalPrice,
	}
}

// MarkAsExpired marks the response as expired
func (rr *RateResponse) MarkAsExpired() {
	rr.Status = "expired"
}

// MarkAsInvalid marks the response as invalid
func (rr *RateResponse) MarkAsInvalid() {
	rr.Status = "invalid"
}

// SetRecommended marks this response as recommended
func (rr *RateResponse) SetRecommended(recommended bool) {
	rr.IsRecommended = recommended
}

// GetEstimatedDeliveryTime returns estimated delivery time in hours
func (rr *RateResponse) GetEstimatedDeliveryTime() time.Duration {
	return time.Duration(rr.EstimatedHours) * time.Hour
}
