package handlers

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// RateHandler handles rate calculation HTTP requests
type RateHandler struct {
	rateService interfaces.RateService
	validator   *validator.Validate
	logger      interfaces.Logger
	metrics     interfaces.MetricsCollector
}

// NewRateHandler creates a new rate handler
func NewRateHandler(
	rateService interfaces.RateService,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *RateHandler {
	return &RateHandler{
		rateService: rateService,
		validator:   validator.New(),
		logger:      logger,
		metrics:     metrics,
	}
}

// GetQuotes handles POST /quotes
func (h *RateHandler) GetQuotes(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := c.Get(constants.HeaderRequestID)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Parse request body
	var req dtos.QuoteRequest
	if err := c.BodyParser(&req); err != nil {
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

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.logger.Warn("Request validation failed",
			"error", err,
			"request_id", requestID)

		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidationErrorResponse(err))
	}

	h.logger.Info("Processing quote request",
		"request_id", requestID,
		"source_postal_code", req.SourceLocation.PostalCode,
		"destination_postal_code", req.DestinationLocation.PostalCode,
		"partners_count", len(req.Partners))

	// Call service
	response, err := h.rateService.GetQuotes(c.Context(), &req, requestID)
	if err != nil {
		h.logger.Error("Get quotes failed",
			"error", err,
			"request_id", requestID)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	duration := time.Since(startTime)
	h.metrics.IncrementCounter("quotes_success", map[string]string{
		"endpoint":       "get_quotes",
		"total_partners": fmt.Sprintf("%d", response.Summary.TotalPartners),
	})
	h.metrics.RecordTimer("quotes_duration", duration, map[string]string{
		"endpoint": "get_quotes",
	})

	h.logger.Info("Quote request completed",
		"request_id", requestID,
		"total_partners", response.Summary.TotalPartners,
		"successful_partners", response.Summary.SuccessfulPartners,
		"total_rates", response.Summary.TotalRatesFound,
		"duration_ms", duration.Milliseconds())

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(response))
}

// GetImplementationHealth handles GET /implementations/health
func (h *RateHandler) GetImplementationHealth(c *fiber.Ctx) error {
	startTime := time.Now()

	h.logger.Info("Processing implementation health check request")

	// Call service
	healthResponse, err := h.rateService.GetImplementationHealth(c.Context())
	if err != nil {
		h.logger.Error("Provider health check failed", "error", err)

		return utils.HandleServiceError(c, err)
	}

	// Determine HTTP status based on health
	status := fiber.StatusOK
	switch healthResponse.Status {
	case "critical":
		status = fiber.StatusServiceUnavailable
	case "degraded":
		status = fiber.StatusPartialContent
	}

	// Record metrics
	duration := time.Since(startTime)
	h.metrics.IncrementCounter("implementation_health_check_success", map[string]string{
		"endpoint": "get_implementation_health",
		"status":   healthResponse.Status,
	})
	h.metrics.RecordTimer("implementation_health_check_duration", duration, map[string]string{
		"endpoint": "get_implementation_health",
	})

	return c.Status(status).JSON(utils.SuccessResponse(healthResponse))
}
