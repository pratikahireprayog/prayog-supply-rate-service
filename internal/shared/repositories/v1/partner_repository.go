package v1

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
)

// PartnerRepository implements database operations for partners
type PartnerRepository struct {
	db *gorm.DB
}

// NewPartnerRepository creates a new partner repository instance
func NewPartnerRepository(db *gorm.DB) *PartnerRepository {
	return &PartnerRepository{
		db: db,
	}
}

// Create creates a new partner in the database
func (r *PartnerRepository) Create(ctx context.Context, partner *models.Partner) error {
	if err := r.db.WithContext(ctx).Create(partner).Error; err != nil {
		if isDuplicateKeyError(err) {
			return constants.ErrPartnerCodeAlreadyExists
		}
		return err
	}
	return nil
}

// GetByID retrieves a partner by ID
func (r *PartnerRepository) GetByID(ctx context.Context, id string) (*models.Partner, error) {
	var partner models.Partner

	partnerID, err := uuid.Parse(id)
	if err != nil {
		return nil, constants.ErrInvalidPartnerID
	}

	err = r.db.WithContext(ctx).Where("id = ?", partnerID).First(&partner).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrPartnerNotFound
		}
		return nil, err
	}

	return &partner, nil
}

// GetByCode retrieves a partner by code
func (r *PartnerRepository) GetByCode(ctx context.Context, code string) (*models.Partner, error) {
	var partner models.Partner

	err := r.db.WithContext(ctx).Where("code = ?", code).First(&partner).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrPartnerNotFound
		}
		return nil, err
	}

	return &partner, nil
}

// GetAll retrieves all partners with optional filtering
func (r *PartnerRepository) GetAll(ctx context.Context, filters map[string]interface{}) ([]*models.Partner, error) {
	var partners []*models.Partner

	query := r.db.WithContext(ctx)

	// Apply filters
	if partnerType, ok := filters["type"]; ok {
		query = query.Where("type = ?", partnerType)
	}
	if isActive, ok := filters["is_active"]; ok {
		query = query.Where("is_active = ?", isActive)
	}

	// Order by priority and name
	query = query.Order("priority ASC, name ASC")

	err := query.Find(&partners).Error
	return partners, err
}

// Update updates an existing partner
func (r *PartnerRepository) Update(ctx context.Context, partner *models.Partner) error {
	result := r.db.WithContext(ctx).Save(partner)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return constants.ErrPartnerCodeAlreadyExists
		}
		return result.Error
	}

	if result.RowsAffected == 0 {
		return constants.ErrPartnerNotFound
	}

	return nil
}

// Delete deletes a partner by ID
func (r *PartnerRepository) Delete(ctx context.Context, id string) error {
	partnerID, err := uuid.Parse(id)
	if err != nil {
		return constants.ErrInvalidPartnerID
	}

	result := r.db.WithContext(ctx).Delete(&models.Partner{}, "id = ?", partnerID)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return constants.ErrPartnerNotFound
	}

	return nil
}

// UpdateHealthStatus updates partner health status
func (r *PartnerRepository) UpdateHealthStatus(ctx context.Context, id string, status string) error {
	partnerID, err := uuid.Parse(id)
	if err != nil {
		return constants.ErrInvalidPartnerID
	}

	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Partner{}).
		Where("id = ?", partnerID).
		Updates(map[string]interface{}{
			"health_status": status,
			"last_checked":  &now,
			"updated_at":    now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return constants.ErrPartnerNotFound
	}

	return nil
}

// GetActivePartners retrieves all active partners
func (r *PartnerRepository) GetActivePartners(ctx context.Context) ([]*models.Partner, error) {
	return r.GetAll(ctx, map[string]interface{}{
		"is_active": true,
	})
}

// GetPartnersByType retrieves partners by type
func (r *PartnerRepository) GetPartnersByType(ctx context.Context, partnerType models.PartnerType) ([]*models.Partner, error) {
	return r.GetAll(ctx, map[string]interface{}{
		"type": partnerType,
	})
}

// Helper function to check for duplicate key errors
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL duplicate key error
	return err.Error() == "ERROR: duplicate key value violates unique constraint \"partners_code_key\" (SQLSTATE 23505)" ||
		err.Error() == "ERROR: duplicate key value violates unique constraint \"partners_name_key\" (SQLSTATE 23505)"
}

