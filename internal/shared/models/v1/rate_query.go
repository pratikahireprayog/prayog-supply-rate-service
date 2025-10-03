package v1

import (
	"time"

	"github.com/google/uuid"
)

// RateQuery represents a rate calculation request
type RateQuery struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID   string    `json:"customer_id" gorm:"index;not null" validate:"required"`
	OriginCity   string    `json:"origin_city" gorm:"not null" validate:"required,min=2,max=100"`
	DestCity     string    `json:"dest_city" gorm:"not null" validate:"required,min=2,max=100"`
	Weight       float64   `json:"weight" gorm:"not null" validate:"required,min=0.1,max=10000"`
	Distance     float64   `json:"distance" gorm:"not null" validate:"required,min=0.1,max=10000"`
	ServiceType  string    `json:"service_type" gorm:"not null" validate:"required,oneof=standard premium express same_day"`
	PickupDate   time.Time `json:"pickup_date" gorm:"not null" validate:"required"`
	DeliveryDate time.Time `json:"delivery_date" gorm:"not null" validate:"required,gtfield=PickupDate"`
	Priority     string    `json:"priority" gorm:"not null;default:normal" validate:"required,oneof=low normal high urgent"`

	// Metadata
	RequestID string                 `json:"request_id" gorm:"index" validate:"required"`
	Source    string                 `json:"source" gorm:"not null;default:api" validate:"required,oneof=api web mobile batch"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (RateQuery) TableName() string {
	return "rate_queries"
}

// IsUrgent checks if the query has urgent priority
func (rq *RateQuery) IsUrgent() bool {
	return rq.Priority == "urgent" || rq.Priority == "high"
}

// GetDeliveryDurationHours returns the delivery duration in hours
func (rq *RateQuery) GetDeliveryDurationHours() float64 {
	return rq.DeliveryDate.Sub(rq.PickupDate).Hours()
}

// IsValidTimeframe checks if pickup and delivery dates are valid
func (rq *RateQuery) IsValidTimeframe() bool {
	now := time.Now()

	// Pickup should be in the future (with some grace period)
	gracePeriod := 30 * time.Minute
	if rq.PickupDate.Before(now.Add(-gracePeriod)) {
		return false
	}

	// Delivery should be after pickup
	if !rq.DeliveryDate.After(rq.PickupDate) {
		return false
	}

	// Maximum delivery window (e.g., 30 days)
	maxDeliveryWindow := 30 * 24 * time.Hour
	if rq.DeliveryDate.After(rq.PickupDate.Add(maxDeliveryWindow)) {
		return false
	}

	return true
}

// GetWeightCategory returns the weight category for the shipment
func (rq *RateQuery) GetWeightCategory() string {
	switch {
	case rq.Weight <= 1.0:
		return "light"
	case rq.Weight <= 5.0:
		return "medium"
	case rq.Weight <= 25.0:
		return "heavy"
	default:
		return "bulk"
	}
}

// GetDistanceCategory returns the distance category for the shipment
func (rq *RateQuery) GetDistanceCategory() string {
	switch {
	case rq.Distance <= 10.0:
		return "local"
	case rq.Distance <= 100.0:
		return "regional"
	case rq.Distance <= 500.0:
		return "interstate"
	default:
		return "national"
	}
}
