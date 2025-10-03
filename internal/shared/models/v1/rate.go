package v1

import (
	"time"

	"github.com/google/uuid"
)

// Rate represents a rate card entry with pricing information
type Rate struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PartnerID   string     `json:"partner_id" gorm:"index;not null" validate:"required"`
	ServiceType string     `json:"service_type" gorm:"index;not null" validate:"required,oneof=standard premium express same_day"`
	OriginCity  string     `json:"origin_city" gorm:"index;not null" validate:"required"`
	DestCity    string     `json:"dest_city" gorm:"index;not null" validate:"required"`
	WeightMin   float64    `json:"weight_min" gorm:"not null" validate:"required,min=0"`
	WeightMax   float64    `json:"weight_max" gorm:"not null" validate:"required,gtfield=WeightMin"`
	DistanceMin float64    `json:"distance_min" gorm:"not null" validate:"required,min=0"`
	DistanceMax float64    `json:"distance_max" gorm:"not null" validate:"required,gtfield=DistanceMin"`
	BasePrice   float64    `json:"base_price" gorm:"not null" validate:"required,min=0"`
	PricePerKm  float64    `json:"price_per_km" gorm:"not null" validate:"required,min=0"`
	PricePerKg  float64    `json:"price_per_kg" gorm:"not null" validate:"required,min=0"`
	Currency    string     `json:"currency" gorm:"not null;default:INR" validate:"required,oneof=INR USD EUR"`
	ValidFrom   time.Time  `json:"valid_from" gorm:"not null" validate:"required"`
	ValidTo     *time.Time `json:"valid_to,omitempty" gorm:""`
	IsActive    bool       `json:"is_active" gorm:"not null;default:true"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (Rate) TableName() string {
	return "rates"
}

// IsValidForDate checks if the rate is valid for the given date
func (r *Rate) IsValidForDate(date time.Time) bool {
	if !r.IsActive {
		return false
	}

	if date.Before(r.ValidFrom) {
		return false
	}

	if r.ValidTo != nil && date.After(*r.ValidTo) {
		return false
	}

	return true
}

// IsValidForWeight checks if the rate applies to the given weight
func (r *Rate) IsValidForWeight(weight float64) bool {
	return weight >= r.WeightMin && weight <= r.WeightMax
}

// IsValidForDistance checks if the rate applies to the given distance
func (r *Rate) IsValidForDistance(distance float64) bool {
	return distance >= r.DistanceMin && distance <= r.DistanceMax
}

// CalculatePrice calculates the total price based on weight and distance
func (r *Rate) CalculatePrice(weight, distance float64) float64 {
	return r.BasePrice + (r.PricePerKm * distance) + (r.PricePerKg * weight)
}
