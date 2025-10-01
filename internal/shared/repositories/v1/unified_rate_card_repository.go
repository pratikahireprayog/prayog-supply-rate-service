package v1

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
)

// UnifiedRateCardRepository implements database operations for unified rate cards
type UnifiedRateCardRepository struct {
	db *gorm.DB
}

// NewUnifiedRateCardRepository creates a new unified rate card repository instance
func NewUnifiedRateCardRepository(db *gorm.DB) *UnifiedRateCardRepository {
	return &UnifiedRateCardRepository{
		db: db,
	}
}

// Create creates a new unified rate card in the database
func (r *UnifiedRateCardRepository) Create(ctx context.Context, rateCard *models.UnifiedRateCard) error {
	if err := r.db.WithContext(ctx).Create(rateCard).Error; err != nil {
		return err
	}
	return nil
}

// GetByID retrieves a unified rate card by ID
func (r *UnifiedRateCardRepository) GetByID(ctx context.Context, id string) (*models.UnifiedRateCard, error) {
	var rateCard models.UnifiedRateCard

	rateCardID, err := uuid.Parse(id)
	if err != nil {
		return nil, constants.ErrInvalidRateCardID
	}

	err = r.db.WithContext(ctx).Where("id = ?", rateCardID).First(&rateCard).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrRateCardNotFound
		}
		return nil, err
	}

	return &rateCard, nil
}

// GetByPartnerCode retrieves unified rate cards by partner code
func (r *UnifiedRateCardRepository) GetByPartnerCode(ctx context.Context, partnerCode string) ([]*models.UnifiedRateCard, error) {
	var rateCards []*models.UnifiedRateCard

	err := r.db.WithContext(ctx).
		Where("partner_code = ?", partnerCode).
		Order("is_default DESC, effective_from DESC").
		Find(&rateCards).Error

	return rateCards, err
}

// GetActiveByPartnerCode retrieves active unified rate cards by partner code
func (r *UnifiedRateCardRepository) GetActiveByPartnerCode(ctx context.Context, partnerCode string) ([]*models.UnifiedRateCard, error) {
	var rateCards []*models.UnifiedRateCard
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("partner_code = ? AND is_active = ?", partnerCode, true).
		Where("effective_from <= ?", now).
		Where("effective_to IS NULL OR effective_to > ?", now).
		Order("is_default DESC, effective_from DESC").
		Find(&rateCards).Error

	return rateCards, err
}

// GetDefaultByPartnerCode retrieves the default rate card for a partner
func (r *UnifiedRateCardRepository) GetDefaultByPartnerCode(ctx context.Context, partnerCode string) (*models.UnifiedRateCard, error) {
	var rateCard models.UnifiedRateCard
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("partner_code = ? AND is_active = ? AND is_default = ?", partnerCode, true, true).
		Where("effective_from <= ?", now).
		Where("effective_to IS NULL OR effective_to > ?", now).
		Order("effective_from DESC").
		First(&rateCard).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrRateCardNotFound
		}
		return nil, err
	}

	return &rateCard, nil
}

// GetByUnifiedRateCardID retrieves a rate card by its unified API ID
func (r *UnifiedRateCardRepository) GetByUnifiedRateCardID(ctx context.Context, unifiedRateCardID string) (*models.UnifiedRateCard, error) {
	var rateCard models.UnifiedRateCard

	err := r.db.WithContext(ctx).Where("unified_rate_card_id = ?", unifiedRateCardID).First(&rateCard).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrRateCardNotFound
		}
		return nil, err
	}

	return &rateCard, nil
}

// GetAll retrieves all unified rate cards with optional filtering
func (r *UnifiedRateCardRepository) GetAll(ctx context.Context, filter *models.UnifiedRateCardFilter) ([]*models.UnifiedRateCard, error) {
	var rateCards []*models.UnifiedRateCard

	query := r.db.WithContext(ctx)

	// Apply filters
	if filter != nil {
		if filter.PartnerCode != "" {
			query = query.Where("partner_code = ?", filter.PartnerCode)
		}
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
		if filter.IsDefault != nil {
			query = query.Where("is_default = ?", *filter.IsDefault)
		}
		if filter.TenantID != "" {
			query = query.Where("tenant_id = ?", filter.TenantID)
		}
		if filter.ValidAt != nil {
			query = query.Where("effective_from <= ?", *filter.ValidAt)
			query = query.Where("effective_to IS NULL OR effective_to > ?", *filter.ValidAt)
		}
	}

	// Order by partner code, default flag, and effective date
	query = query.Order("partner_code ASC, is_default DESC, effective_from DESC")

	err := query.Find(&rateCards).Error
	return rateCards, err
}

// Update updates an existing unified rate card
func (r *UnifiedRateCardRepository) Update(ctx context.Context, rateCard *models.UnifiedRateCard) error {
	result := r.db.WithContext(ctx).Save(rateCard)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return constants.ErrRateCardNotFound
	}

	return nil
}

// Delete deletes a unified rate card by ID
func (r *UnifiedRateCardRepository) Delete(ctx context.Context, id string) error {
	rateCardID, err := uuid.Parse(id)
	if err != nil {
		return constants.ErrInvalidRateCardID
	}

	result := r.db.WithContext(ctx).Delete(&models.UnifiedRateCard{}, "id = ?", rateCardID)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return constants.ErrRateCardNotFound
	}

	return nil
}

// SetDefault sets a rate card as default and unsets others for the same partner
func (r *UnifiedRateCardRepository) SetDefault(ctx context.Context, id string, partnerCode string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, unset all defaults for this partner
		if err := tx.WithContext(ctx).Model(&models.UnifiedRateCard{}).
			Where("partner_code = ?", partnerCode).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Then set this one as default
		rateCardID, err := uuid.Parse(id)
		if err != nil {
			return constants.ErrInvalidRateCardID
		}

		result := tx.WithContext(ctx).Model(&models.UnifiedRateCard{}).
			Where("id = ?", rateCardID).
			Update("is_default", true)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return constants.ErrRateCardNotFound
		}

		return nil
	})
}

// GetPartnerConfiguration retrieves partner configuration for unified rate API
func (r *UnifiedRateCardRepository) GetPartnerConfiguration(ctx context.Context, partnerCode string) (*models.UnifiedRateCard, error) {
	// Try to get default rate card first
	defaultCard, err := r.GetDefaultByPartnerCode(ctx, partnerCode)
	if err == nil {
		return defaultCard, nil
	}

	// If no default, get the most recent active card
	activeCards, err := r.GetActiveByPartnerCode(ctx, partnerCode)
	if err != nil {
		return nil, err
	}

	if len(activeCards) == 0 {
		return nil, constants.ErrRateCardNotFound
	}

	return activeCards[0], nil
}

// GetConfigByPartnerCode retrieves unified API configuration for a partner
func (r *UnifiedRateCardRepository) GetConfigByPartnerCode(ctx context.Context, partnerCode string) (tenantID, apiKey, unifiedRateCardID string, err error) {
	config, err := r.GetPartnerConfiguration(ctx, partnerCode)
	if err != nil {
		return "", "", "", err
	}

	return config.TenantID, config.APIKey, config.UnifiedRateCardID, nil
}

// ListPartnerCodes retrieves all unique partner codes that have rate cards
func (r *UnifiedRateCardRepository) ListPartnerCodes(ctx context.Context) ([]string, error) {
	var codes []string

	err := r.db.WithContext(ctx).
		Model(&models.UnifiedRateCard{}).
		Distinct("partner_code").
		Where("is_active = ?", true).
		Pluck("partner_code", &codes).Error

	return codes, err
}
