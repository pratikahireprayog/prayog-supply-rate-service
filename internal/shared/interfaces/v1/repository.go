package v1

import (
	"context"

	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
)

// PartnerRepository defines database operations for partners
type PartnerRepository interface {
	// Create creates a new partner in the database
	Create(ctx context.Context, partner *models.Partner) error

	// GetByID retrieves a partner by ID
	GetByID(ctx context.Context, id string) (*models.Partner, error)

	// GetByCode retrieves a partner by code
	GetByCode(ctx context.Context, code string) (*models.Partner, error)

	// GetAll retrieves all partners with optional filtering
	GetAll(ctx context.Context, filters map[string]interface{}) ([]*models.Partner, error)

	// Update updates an existing partner
	Update(ctx context.Context, partner *models.Partner) error

	// Delete deletes a partner by ID
	Delete(ctx context.Context, id string) error

	// UpdateHealthStatus updates partner health status
	UpdateHealthStatus(ctx context.Context, id string, status string) error

	// GetActivePartners retrieves all active partners
	GetActivePartners(ctx context.Context) ([]*models.Partner, error)

	// GetPartnersByType retrieves partners by type
	GetPartnersByType(ctx context.Context, partnerType models.PartnerType) ([]*models.Partner, error)
}

// UnifiedRateCardRepository defines database operations for unified rate cards
type UnifiedRateCardRepository interface {
	// Create creates a new unified rate card in the database
	Create(ctx context.Context, rateCard *models.UnifiedRateCard) error

	// GetByID retrieves a unified rate card by ID
	GetByID(ctx context.Context, id string) (*models.UnifiedRateCard, error)

	// GetByPartnerCode retrieves unified rate cards by partner code
	GetByPartnerCode(ctx context.Context, partnerCode string) ([]*models.UnifiedRateCard, error)

	// GetActiveByPartnerCode retrieves active unified rate cards by partner code
	GetActiveByPartnerCode(ctx context.Context, partnerCode string) ([]*models.UnifiedRateCard, error)

	// GetDefaultByPartnerCode retrieves the default rate card for a partner
	GetDefaultByPartnerCode(ctx context.Context, partnerCode string) (*models.UnifiedRateCard, error)

	// GetByUnifiedRateCardID retrieves a rate card by its unified API ID
	GetByUnifiedRateCardID(ctx context.Context, unifiedRateCardID string) (*models.UnifiedRateCard, error)

	// GetAll retrieves all unified rate cards with optional filtering
	GetAll(ctx context.Context, filter *models.UnifiedRateCardFilter) ([]*models.UnifiedRateCard, error)

	// Update updates an existing unified rate card
	Update(ctx context.Context, rateCard *models.UnifiedRateCard) error

	// Delete deletes a unified rate card by ID
	Delete(ctx context.Context, id string) error

	// SetDefault sets a rate card as default and unsets others for the same partner
	SetDefault(ctx context.Context, id string, partnerCode string) error

	// GetPartnerConfiguration retrieves partner configuration for unified rate API
	GetPartnerConfiguration(ctx context.Context, partnerCode string) (*models.UnifiedRateCard, error)

	// GetConfigByPartnerCode retrieves unified API configuration for a partner
	GetConfigByPartnerCode(ctx context.Context, partnerCode string) (tenantID, apiKey, unifiedRateCardID string, err error)

	// ListPartnerCodes retrieves all unique partner codes that have rate cards
	ListPartnerCodes(ctx context.Context) ([]string, error)
}

// Database represents database operations interface
type Database interface {
	// Connect establishes database connection
	Connect() error

	// Close closes database connection
	Close() error

	// Migrate runs database migrations
	Migrate() error

	// GetDB returns the underlying database instance
	GetDB() interface{}

	// Health checks database health
	Health() error

	// Transaction executes function within a transaction
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}
