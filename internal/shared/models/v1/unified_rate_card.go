package v1

import (
	"time"

	"github.com/google/uuid"
)

// UnifiedRateCard represents a rate card stored for unified rate API
type UnifiedRateCard struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PartnerCode   string     `json:"partner_code" gorm:"index;not null" validate:"required,min=2,max=20"`
	Name          string     `json:"name" gorm:"not null" validate:"required,min=2,max=100"`
	ProductType   string     `json:"product_type" gorm:"not null" validate:"required"`
	IsActive      bool       `json:"is_active" gorm:"not null;default:true"`
	IsDefault     bool       `json:"is_default" gorm:"not null;default:false"`
	EffectiveFrom time.Time  `json:"effective_from" gorm:"not null" validate:"required"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty" gorm:""`

	// Unified Rate API specific fields
	UnifiedRateCardID string `json:"unified_rate_card_id" gorm:"index" validate:"required"` // ID from unified API
	TenantID          string `json:"tenant_id" gorm:"not null" validate:"required"`
	APIKey            string `json:"api_key" gorm:"not null" validate:"required"`

	// Configuration
	Config map[string]interface{} `json:"config,omitempty" gorm:"type:jsonb"`

	// Audit fields
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedBy *string   `json:"created_by,omitempty" gorm:""`
	UpdatedBy *string   `json:"updated_by,omitempty" gorm:""`
}

// TableName specifies the table name for GORM
func (UnifiedRateCard) TableName() string {
	return "unified_rate_cards"
}

// IsValidForDate checks if the rate card is valid for the given date
func (urc *UnifiedRateCard) IsValidForDate(date time.Time) bool {
	if !urc.IsActive {
		return false
	}

	if date.Before(urc.EffectiveFrom) {
		return false
	}

	if urc.EffectiveTo != nil && date.After(*urc.EffectiveTo) {
		return false
	}

	return true
}

// IsCurrentlyValid checks if the rate card is valid for the current time
func (urc *UnifiedRateCard) IsCurrentlyValid() bool {
	return urc.IsValidForDate(time.Now())
}

// UnifiedRateCardRequest represents request for creating/updating rate cards
type UnifiedRateCardRequest struct {
	PartnerCode   string                 `json:"partner_code" validate:"required,min=2,max=20"`
	Name          string                 `json:"name" validate:"required,min=2,max=100"`
	ProductType   string                 `json:"product_type" validate:"required"`
	IsActive      bool                   `json:"is_active"`
	IsDefault     bool                   `json:"is_default"`
	EffectiveFrom time.Time              `json:"effective_from" validate:"required"`
	EffectiveTo   *time.Time             `json:"effective_to,omitempty"`
	TenantID      string                 `json:"tenant_id" validate:"required"`
	APIKey        string                 `json:"api_key" validate:"required"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// UnifiedRateCardResponse represents response for rate card operations
type UnifiedRateCardResponse struct {
	ID                uuid.UUID              `json:"id"`
	PartnerCode       string                 `json:"partner_code"`
	Name              string                 `json:"name"`
	ProductType       string                 `json:"product_type"`
	IsActive          bool                   `json:"is_active"`
	IsDefault         bool                   `json:"is_default"`
	EffectiveFrom     time.Time              `json:"effective_from"`
	EffectiveTo       *time.Time             `json:"effective_to,omitempty"`
	UnifiedRateCardID string                 `json:"unified_rate_card_id"`
	TenantID          string                 `json:"tenant_id"`
	Config            map[string]interface{} `json:"config,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	CreatedBy         *string                `json:"created_by,omitempty"`
	UpdatedBy         *string                `json:"updated_by,omitempty"`
}

// ToResponse converts model to response DTO
func (urc *UnifiedRateCard) ToResponse() *UnifiedRateCardResponse {
	return &UnifiedRateCardResponse{
		ID:                urc.ID,
		PartnerCode:       urc.PartnerCode,
		Name:              urc.Name,
		ProductType:       urc.ProductType,
		IsActive:          urc.IsActive,
		IsDefault:         urc.IsDefault,
		EffectiveFrom:     urc.EffectiveFrom,
		EffectiveTo:       urc.EffectiveTo,
		UnifiedRateCardID: urc.UnifiedRateCardID,
		TenantID:          urc.TenantID,
		Config:            urc.Config,
		CreatedAt:         urc.CreatedAt,
		UpdatedAt:         urc.UpdatedAt,
		CreatedBy:         urc.CreatedBy,
		UpdatedBy:         urc.UpdatedBy,
	}
}

// FromRequest creates model from request DTO
func (urc *UnifiedRateCard) FromRequest(req *UnifiedRateCardRequest) {
	urc.PartnerCode = req.PartnerCode
	urc.Name = req.Name
	urc.ProductType = req.ProductType
	urc.IsActive = req.IsActive
	urc.IsDefault = req.IsDefault
	urc.EffectiveFrom = req.EffectiveFrom
	urc.EffectiveTo = req.EffectiveTo
	urc.TenantID = req.TenantID
	urc.APIKey = req.APIKey
	urc.Config = req.Config
}

// UnifiedRateCardFilter represents filters for rate card queries
type UnifiedRateCardFilter struct {
	PartnerCode string     `json:"partner_code,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
	IsDefault   *bool      `json:"is_default,omitempty"`
	ValidAt     *time.Time `json:"valid_at,omitempty"`
	TenantID    string     `json:"tenant_id,omitempty"`
}
