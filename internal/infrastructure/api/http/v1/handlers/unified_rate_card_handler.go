package handlers

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	unified_rate "github.com/prayog/prayog-supply-rate-service/internal/services/v1/implementations/pre_defined/unified_rate"
	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// UnifiedRateCardHandler handles unified rate card HTTP requests
type UnifiedRateCardHandler struct {
	rateCardService *unified_rate.RateCardService
	validator       *validator.Validate
	logger          interfaces.Logger
	metrics         interfaces.MetricsCollector
}

// NewUnifiedRateCardHandler creates a new unified rate card handler
func NewUnifiedRateCardHandler(
	rateCardService *unified_rate.RateCardService,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *UnifiedRateCardHandler {
	return &UnifiedRateCardHandler{
		rateCardService: rateCardService,
		validator:       validator.New(),
		logger:          logger,
		metrics:         metrics,
	}
}

// CreateRateCard handles POST /unified-rate-cards
func (h *UnifiedRateCardHandler) CreateRateCard(c *fiber.Ctx) error {
	requestID := c.Get(constants.HeaderRequestID)

	// Parse into a combined struct or map first
	var combinedRequest struct {
		models.UnifiedRateCardRequest
		RateCardData unified_rate.RateCardRequest `json:"rate_card_data"`
	}

	if err := c.BodyParser(&combinedRequest); err != nil {
		h.logger.Warn("Invalid request body",
			"error", err,
			"request_id", requestID,
			"path", c.Path())

		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			constants.MessageInvalidRequest,
			constants.CodeInvalidRequest,
			err.Error(),
		))
	}

	// Now you have both structs populated
	req := combinedRequest.UnifiedRateCardRequest
	rateCardData := combinedRequest.RateCardData
	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.logger.Warn("Request validation failed",
			"error", err,
			"request_id", requestID)

		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidationErrorResponse(err))
	}

	// Extract user information from auth context for audit logging
	userID := extractUserID(c)
	req.CreatedBy = userID
	req.UpdatedBy = userID

	h.logger.Info("Creating unified rate card",
		"request_id", requestID,
		"partner_code", req.PartnerCode,
		"name", req.Name,
		"created_by", userID)

	// Call service
	response, err := h.rateCardService.CreateRateCard(c.Context(), &req, &rateCardData)
	if err != nil {
		h.logger.Error("Create rate card failed",
			"error", err,
			"request_id", requestID)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	h.metrics.IncrementCounter("rate_cards_create_success", map[string]string{
		"endpoint":     "create_rate_card",
		"partner_code": req.PartnerCode,
	})

	h.logger.Info("Rate card created successfully",
		"request_id", requestID,
		"rate_card_id", response.ID,
		"partner_code", req.PartnerCode)

	return c.Status(fiber.StatusCreated).JSON(utils.SuccessResponse(response))
}

// GetRateCard handles GET /unified-rate-cards/:id
func (h *UnifiedRateCardHandler) GetRateCard(c *fiber.Ctx) error {
	requestID := c.Get(constants.HeaderRequestID)
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			"Rate card ID is required",
			constants.CodeMissingParameter,
			"id parameter is missing",
		))
	}

	h.logger.Info("Getting unified rate card",
		"request_id", requestID,
		"rate_card_id", id)

	// Call service
	response, err := h.rateCardService.GetRateCard(c.Context(), id)
	if err != nil {
		h.logger.Error("Get rate card failed",
			"error", err,
			"request_id", requestID,
			"rate_card_id", id)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	h.metrics.IncrementCounter("rate_cards_get_success", map[string]string{
		"endpoint": "get_rate_card",
	})

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(response))
}

// UpdateRateCard handles PUT /unified-rate-cards/:id
func (h *UnifiedRateCardHandler) UpdateRateCard(c *fiber.Ctx) error {
	requestID := c.Get(constants.HeaderRequestID)
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			"Rate card ID is required",
			constants.CodeMissingParameter,
			"id parameter is missing",
		))
	}

	// Parse request body
	var req models.UnifiedRateCardRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warn("Invalid request body",
			"error", err,
			"request_id", requestID)

		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			constants.MessageInvalidRequest,
			constants.CodeInvalidRequest,
			err.Error(),
		))
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.logger.Warn("Request validation failed",
			"error", err,
			"request_id", requestID)

		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidationErrorResponse(err))
	}

	// Parse rate card data from request body
	var rateCardData unified_rate.RateCardRequest
	if err := c.BodyParser(&rateCardData); err != nil {
		h.logger.Warn("Invalid rate card data",
			"error", err,
			"request_id", requestID)

		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			"Invalid rate card data",
			constants.CodeInvalidRequest,
			err.Error(),
		))
	}

	// Extract user information from auth context for audit logging
	userID := extractUserID(c)
	req.UpdatedBy = userID

	h.logger.Info("Updating unified rate card",
		"request_id", requestID,
		"rate_card_id", id,
		"partner_code", req.PartnerCode,
		"updated_by", userID)

	// Call service
	response, err := h.rateCardService.UpdateRateCard(c.Context(), id, &req, &rateCardData)
	if err != nil {
		h.logger.Error("Update rate card failed",
			"error", err,
			"request_id", requestID,
			"rate_card_id", id)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	h.metrics.IncrementCounter("rate_cards_update_success", map[string]string{
		"endpoint":     "update_rate_card",
		"partner_code": req.PartnerCode,
	})

	h.logger.Info("Rate card updated successfully",
		"request_id", requestID,
		"rate_card_id", id)

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(response))
}

// DeleteRateCard handles DELETE /unified-rate-cards/:id
func (h *UnifiedRateCardHandler) DeleteRateCard(c *fiber.Ctx) error {
	requestID := c.Get(constants.HeaderRequestID)
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			"Rate card ID is required",
			constants.CodeMissingParameter,
			"id parameter is missing",
		))
	}

	h.logger.Info("Deleting unified rate card",
		"request_id", requestID,
		"rate_card_id", id)

	// Call service
	if err := h.rateCardService.DeleteRateCard(c.Context(), id); err != nil {
		h.logger.Error("Delete rate card failed",
			"error", err,
			"request_id", requestID,
			"rate_card_id", id)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	h.metrics.IncrementCounter("rate_cards_delete_success", map[string]string{
		"endpoint": "delete_rate_card",
	})

	h.logger.Info("Rate card deleted successfully",
		"request_id", requestID,
		"rate_card_id", id)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ListRateCards handles GET /unified-rate-cards
func (h *UnifiedRateCardHandler) ListRateCards(c *fiber.Ctx) error {
	requestID := c.Get(constants.HeaderRequestID)

	// Parse query parameters for filtering
	filter := &models.UnifiedRateCardFilter{}

	if partnerCode := c.Query("partner_code"); partnerCode != "" {
		filter.PartnerCode = partnerCode
	}

	if tenantID := c.Query("tenant_id"); tenantID != "" {
		filter.TenantID = tenantID
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filter.IsActive = &isActive
		}
	}

	if isDefaultStr := c.Query("is_default"); isDefaultStr != "" {
		if isDefault, err := strconv.ParseBool(isDefaultStr); err == nil {
			filter.IsDefault = &isDefault
		}
	}

	h.logger.Info("Listing unified rate cards",
		"request_id", requestID,
		"partner_code", filter.PartnerCode,
		"tenant_id", filter.TenantID)

	// Call service
	response, err := h.rateCardService.ListRateCards(c.Context(), filter)
	if err != nil {
		h.logger.Error("List rate cards failed",
			"error", err,
			"request_id", requestID)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	h.metrics.IncrementCounter("rate_cards_list_success", map[string]string{
		"endpoint":    "list_rate_cards",
		"count":       strconv.Itoa(len(response)),
		"has_filters": strconv.FormatBool(filter.PartnerCode != "" || filter.TenantID != ""),
	})

	h.logger.Info("Rate cards listed successfully",
		"request_id", requestID,
		"count", len(response))

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(fiber.Map{
		"rate_cards": response,
		"total":      len(response),
	}))
}

// GetRateCardsByPartner handles GET /unified-rate-cards/partner/:partner_code
func (h *UnifiedRateCardHandler) GetRateCardsByPartner(c *fiber.Ctx) error {
	requestID := c.Get(constants.HeaderRequestID)
	partnerCode := c.Params("partner_code")

	if partnerCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			"Partner code is required",
			constants.CodeMissingParameter,
			"partner_code parameter is missing",
		))
	}

	h.logger.Info("Getting rate cards by partner",
		"request_id", requestID,
		"partner_code", partnerCode)

	// Call service
	response, err := h.rateCardService.GetRateCardsByPartner(c.Context(), partnerCode)
	if err != nil {
		h.logger.Error("Get rate cards by partner failed",
			"error", err,
			"request_id", requestID,
			"partner_code", partnerCode)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	h.metrics.IncrementCounter("rate_cards_get_by_partner_success", map[string]string{
		"endpoint":     "get_rate_cards_by_partner",
		"partner_code": partnerCode,
		"count":        strconv.Itoa(len(response)),
	})

	h.logger.Info("Rate cards by partner retrieved successfully",
		"request_id", requestID,
		"partner_code", partnerCode,
		"count", len(response))

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(fiber.Map{
		"rate_cards":   response,
		"partner_code": partnerCode,
		"total":        len(response),
	}))
}

// SetDefaultRateCard handles PUT /unified-rate-cards/:id/set-default
func (h *UnifiedRateCardHandler) SetDefaultRateCard(c *fiber.Ctx) error {
	requestID := c.Get(constants.HeaderRequestID)
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			"Rate card ID is required",
			constants.CodeMissingParameter,
			"id parameter is missing",
		))
	}

	// Parse partner code from request body
	var req struct {
		PartnerCode string `json:"partner_code" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			"Invalid request body",
			constants.CodeInvalidRequest,
			err.Error(),
		))
	}

	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidationErrorResponse(err))
	}

	h.logger.Info("Setting default rate card",
		"request_id", requestID,
		"rate_card_id", id,
		"partner_code", req.PartnerCode)

	// Call service
	if err := h.rateCardService.SetDefaultRateCard(c.Context(), id, req.PartnerCode); err != nil {
		h.logger.Error("Set default rate card failed",
			"error", err,
			"request_id", requestID,
			"rate_card_id", id)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	h.metrics.IncrementCounter("rate_cards_set_default_success", map[string]string{
		"endpoint":     "set_default_rate_card",
		"partner_code": req.PartnerCode,
	})

	h.logger.Info("Default rate card set successfully",
		"request_id", requestID,
		"rate_card_id", id,
		"partner_code", req.PartnerCode)

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(fiber.Map{
		"message": "Rate card set as default successfully",
	}))
}

// extractUserID extracts user ID from authentication context
// Returns the API key or a default value if not authenticated
func extractUserID(c *fiber.Ctx) string {
	// Try to get API key from context (set by auth middleware)
	if apiKey := c.Locals("api_key"); apiKey != nil {
		if keyStr, ok := apiKey.(string); ok {
			return keyStr
		}
	}

	// Try to get from header directly
	if apiKey := c.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Fallback to system
	return "system"
}
