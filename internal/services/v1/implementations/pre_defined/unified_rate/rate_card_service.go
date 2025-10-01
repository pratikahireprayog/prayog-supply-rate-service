package unified_rate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
)

// RateCardService handles unified rate card management operations
type RateCardService struct {
	rateCardRepo interfaces.UnifiedRateCardRepository
	httpClient   interfaces.HTTPClient
	logger       interfaces.Logger
	metrics      interfaces.MetricsCollector
	config       *Config
}

// NewRateCardService creates a new rate card service
func NewRateCardService(
	rateCardRepo interfaces.UnifiedRateCardRepository,
	httpClient interfaces.HTTPClient,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	config *Config,
) *RateCardService {
	return &RateCardService{
		rateCardRepo: rateCardRepo,
		httpClient:   httpClient,
		logger:       logger,
		metrics:      metrics,
		config:       config,
	}
}

// CreateRateCard creates a new rate card in both local DB and unified API
func (s *RateCardService) CreateRateCard(ctx context.Context, req *models.UnifiedRateCardRequest, rateCardData *RateCardRequest) (*models.UnifiedRateCardResponse, error) {
	s.logger.Info("Creating unified rate card", "partner_code", req.PartnerCode, "name", req.Name)

	// Prepare headers with API key
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "*/*",
		"api-key":      req.APIKey,
	}

	// Make request to unified API to create rate card
	url := fmt.Sprintf("%s/rate-cards/unified", s.config.BaseURL)

	httpResponse, err := s.httpClient.Post(ctx, url, rateCardData, headers)
	if err != nil {
		s.metrics.IncrementCounter("rate_card_creation_failed", map[string]string{
			"error": "api_call",
		})
		return nil, fmt.Errorf("failed to create rate card in unified API: %w", err)
	}

	if httpResponse.StatusCode != 200 && httpResponse.StatusCode != 201 {
		s.metrics.IncrementCounter("rate_card_creation_failed", map[string]string{
			"error":       "api_error",
			"status_code": fmt.Sprintf("%d", httpResponse.StatusCode),
		})
		return nil, fmt.Errorf("unified API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Parse response to get the unified rate card ID
	var unifiedResponse struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(httpResponse.Body, &unifiedResponse); err != nil {
		return nil, fmt.Errorf("failed to parse unified API response: %w", err)
	}

	if !unifiedResponse.Success {
		return nil, fmt.Errorf("unified API call failed: %s", unifiedResponse.Message)
	}

	// Create local record with the unified rate card ID
	rateCard := &models.UnifiedRateCard{
		ID:                uuid.New(),
		UnifiedRateCardID: unifiedResponse.Data.ID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	rateCard.FromRequest(req)

	if err := s.rateCardRepo.Create(ctx, rateCard); err != nil {
		s.metrics.IncrementCounter("rate_card_creation_failed", map[string]string{
			"error": "database_error",
		})
		return nil, fmt.Errorf("failed to save rate card to database: %w", err)
	}

	s.metrics.IncrementCounter("rate_card_created", map[string]string{
		"partner_code": req.PartnerCode,
	})

	s.logger.Info("Rate card created successfully",
		"id", rateCard.ID,
		"unified_rate_card_id", rateCard.UnifiedRateCardID,
		"partner_code", req.PartnerCode)

	return rateCard.ToResponse(), nil
}

// GetRateCard retrieves a rate card by ID
func (s *RateCardService) GetRateCard(ctx context.Context, id string) (*models.UnifiedRateCardResponse, error) {
	rateCard, err := s.rateCardRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return rateCard.ToResponse(), nil
}

// GetRateCardsByPartner retrieves all rate cards for a partner
func (s *RateCardService) GetRateCardsByPartner(ctx context.Context, partnerCode string) ([]*models.UnifiedRateCardResponse, error) {
	rateCards, err := s.rateCardRepo.GetByPartnerCode(ctx, partnerCode)
	if err != nil {
		return nil, err
	}

	responses := make([]*models.UnifiedRateCardResponse, len(rateCards))
	for i, card := range rateCards {
		responses[i] = card.ToResponse()
	}

	return responses, nil
}

// UpdateRateCard updates an existing rate card
func (s *RateCardService) UpdateRateCard(ctx context.Context, id string, req *models.UnifiedRateCardRequest, rateCardData *RateCardRequest) (*models.UnifiedRateCardResponse, error) {
	// Get existing rate card
	rateCard, err := s.rateCardRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Updating unified rate card", "id", id, "unified_rate_card_id", rateCard.UnifiedRateCardID)

	// Prepare headers with API key
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "*/*",
		"api-key":      req.APIKey,
	}

	// Update in unified API
	url := fmt.Sprintf("%s/rate-cards/unified/%s", s.config.BaseURL, rateCard.UnifiedRateCardID)

	httpResponse, err := s.httpClient.Put(ctx, url, rateCardData, headers)
	if err != nil {
		s.metrics.IncrementCounter("rate_card_update_failed", map[string]string{
			"error": "api_call",
		})
		return nil, fmt.Errorf("failed to update rate card in unified API: %w", err)
	}

	if httpResponse.StatusCode != 200 {
		s.metrics.IncrementCounter("rate_card_update_failed", map[string]string{
			"error":       "api_error",
			"status_code": fmt.Sprintf("%d", httpResponse.StatusCode),
		})
		return nil, fmt.Errorf("unified API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Update local record
	rateCard.FromRequest(req)
	rateCard.UpdatedAt = time.Now()

	if err := s.rateCardRepo.Update(ctx, rateCard); err != nil {
		s.metrics.IncrementCounter("rate_card_update_failed", map[string]string{
			"error": "database_error",
		})
		return nil, fmt.Errorf("failed to update rate card in database: %w", err)
	}

	s.metrics.IncrementCounter("rate_card_updated", map[string]string{
		"partner_code": rateCard.PartnerCode,
	})

	s.logger.Info("Rate card updated successfully", "id", id)

	return rateCard.ToResponse(), nil
}

// DeleteRateCard deletes a rate card
func (s *RateCardService) DeleteRateCard(ctx context.Context, id string) error {
	// Get existing rate card
	rateCard, err := s.rateCardRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	s.logger.Info("Deleting unified rate card", "id", id, "unified_rate_card_id", rateCard.UnifiedRateCardID)

	// Prepare headers with API key
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "*/*",
		"api-key":      rateCard.APIKey,
	}

	// Delete from unified API
	url := fmt.Sprintf("%s/rate-cards/unified/%s", s.config.BaseURL, rateCard.UnifiedRateCardID)

	httpResponse, err := s.httpClient.Delete(ctx, url, headers)
	if err != nil {
		s.metrics.IncrementCounter("rate_card_deletion_failed", map[string]string{
			"error": "api_call",
		})
		return fmt.Errorf("failed to delete rate card from unified API: %w", err)
	}

	if httpResponse.StatusCode != 200 && httpResponse.StatusCode != 204 {
		s.metrics.IncrementCounter("rate_card_deletion_failed", map[string]string{
			"error":       "api_error",
			"status_code": fmt.Sprintf("%d", httpResponse.StatusCode),
		})
		return fmt.Errorf("unified API returned status %d: %s", httpResponse.StatusCode, string(httpResponse.Body))
	}

	// Delete from local database
	if err := s.rateCardRepo.Delete(ctx, id); err != nil {
		s.metrics.IncrementCounter("rate_card_deletion_failed", map[string]string{
			"error": "database_error",
		})
		return fmt.Errorf("failed to delete rate card from database: %w", err)
	}

	s.metrics.IncrementCounter("rate_card_deleted", map[string]string{
		"partner_code": rateCard.PartnerCode,
	})

	s.logger.Info("Rate card deleted successfully", "id", id)

	return nil
}

// ListRateCards retrieves all rate cards with optional filtering
func (s *RateCardService) ListRateCards(ctx context.Context, filter *models.UnifiedRateCardFilter) ([]*models.UnifiedRateCardResponse, error) {
	rateCards, err := s.rateCardRepo.GetAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	responses := make([]*models.UnifiedRateCardResponse, len(rateCards))
	for i, card := range rateCards {
		responses[i] = card.ToResponse()
	}

	return responses, nil
}

// SetDefaultRateCard sets a rate card as default for a partner
func (s *RateCardService) SetDefaultRateCard(ctx context.Context, id string, partnerCode string) error {
	return s.rateCardRepo.SetDefault(ctx, id, partnerCode)
}

// GetPartnerConfiguration retrieves the configuration for making unified API calls for a partner
func (s *RateCardService) GetPartnerConfiguration(ctx context.Context, partnerCode string) (tenantID, apiKey, unifiedRateCardID string, err error) {
	return s.rateCardRepo.GetConfigByPartnerCode(ctx, partnerCode)
}
