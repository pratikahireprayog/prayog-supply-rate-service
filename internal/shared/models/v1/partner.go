package v1

import (
	"time"

	"github.com/google/uuid"
)

// PartnerType represents the type of partner
type PartnerType string

const (
	PartnerTypePreDefined PartnerType = "pre_defined"
	PartnerTypeRealTime   PartnerType = "real_time"
)

// String returns the string representation of PartnerType
func (pt PartnerType) String() string {
	return string(pt)
}

// Partner represents a logistics partner that provides rates
type Partner struct {
	ID           uuid.UUID              `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string                 `json:"name" gorm:"uniqueIndex;not null" validate:"required,min=2,max=100"`
	Code         string                 `json:"code" gorm:"uniqueIndex;not null" validate:"required,min=2,max=20,alphanum"`
	Type         PartnerType            `json:"type" gorm:"not null;index" validate:"required,oneof=pre_defined real_time"`
	IsActive     bool                   `json:"is_active" gorm:"not null;default:true"`
	Priority     int                    `json:"priority" gorm:"not null;default:1" validate:"required,min=1,max=10"`
	Config       map[string]interface{} `json:"config,omitempty" gorm:"type:jsonb"`
	APIEndpoint  *string                `json:"api_endpoint,omitempty" gorm:""`
	APIKey       *string                `json:"api_key,omitempty" gorm:""`
	TimeoutMs    int                    `json:"timeout_ms" gorm:"not null;default:5000" validate:"required,min=1000,max=30000"`
	RetryCount   int                    `json:"retry_count" gorm:"not null;default:3" validate:"required,min=0,max=5"`
	HealthStatus string                 `json:"health_status" gorm:"not null;default:unknown" validate:"required,oneof=healthy unhealthy unknown"`
	LastChecked  *time.Time             `json:"last_checked,omitempty" gorm:""`
	Description  string                 `json:"description,omitempty" gorm:"type:text"`
	CreatedAt    time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (Partner) TableName() string {
	return "partners"
}

// IsHealthy checks if the partner is healthy and active
func (p *Partner) IsHealthy() bool {
	return p.IsActive && p.HealthStatus == "healthy"
}

// IsAPIBased checks if the partner uses API-based rate fetching
func (p *Partner) IsAPIBased() bool {
	return p.Type == PartnerTypeRealTime && p.APIEndpoint != nil && *p.APIEndpoint != ""
}

// GetAPIEndpoint safely returns the API endpoint
func (p *Partner) GetAPIEndpoint() string {
	if p.APIEndpoint == nil {
		return ""
	}
	return *p.APIEndpoint
}

// GetAPIKey safely returns the API key
func (p *Partner) GetAPIKey() string {
	if p.APIKey == nil {
		return ""
	}
	return *p.APIKey
}

// UpdateHealthStatus updates the health status and last checked time
func (p *Partner) UpdateHealthStatus(status string) {
	p.HealthStatus = status
	now := time.Now()
	p.LastChecked = &now
}
