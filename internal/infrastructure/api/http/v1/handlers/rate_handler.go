package handlers

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	constants "github.com/prayog/prayog-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-rate-service/internal/shared/interfaces/v1"
	utils "github.com/prayog/prayog-rate-service/internal/shared/utils/v1"
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

// CalculateRates handles POST /rates/calculate
func (h *RateHandler) CalculateRates(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := c.Get(constants.HeaderRequestID)

	// Parse request body
	var req dtos.RateCalculationRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warn("Invalid request body",
			"error", err,
			"request_id", requestID,
			"path", c.Path())

		h.metrics.IncrementCounter("request_validation_failed", map[string]string{
			"endpoint": "calculate_rates",
			"reason":   "invalid_body",
		})

		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			constants.MessageInvalidRequest,
			constants.CodeInvalidRequest,
			err.Error(),
		))
	}

	// Set request ID if not provided
	if req.RequestID == "" {
		req.RequestID = requestID
		if req.RequestID == "" {
			req.RequestID = uuid.New().String()
		}
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.logger.Warn("Request validation failed",
			"error", err,
			"request_id", req.RequestID,
			"path", c.Path())

		h.metrics.IncrementCounter("request_validation_failed", map[string]string{
			"endpoint": "calculate_rates",
			"reason":   "validation_error",
		})

		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidationErrorResponse(err))
	}

	// Additional business validation
	if err := req.Validate(); err != nil {
		h.logger.Warn("Business validation failed",
			"error", err,
			"request_id", req.RequestID)

		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			"Business validation failed",
			constants.ErrorToCode(err),
			err.Error(),
		))
	}

	h.logger.Info("Processing rate calculation request",
		"request_id", req.RequestID,
		"customer_id", req.CustomerID,
		"origin", req.OriginCity,
		"destination", req.DestCity,
		"weight", req.Weight,
		"distance", req.Distance,
		"service_type", req.ServiceType)

	// Call service
	response, err := h.rateService.CalculateRates(c.Context(), &req)
	if err != nil {
		h.logger.Error("Rate calculation failed",
			"error", err,
			"request_id", req.RequestID)

		h.metrics.IncrementCounter("rate_calculation_failed", map[string]string{
			"endpoint": "calculate_rates",
			"reason":   "service_error",
		})

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	duration := time.Since(startTime)
	h.metrics.IncrementCounter("rate_calculation_success", map[string]string{
		"endpoint":     "calculate_rates",
		"total_quotes": fmt.Sprintf("%d", response.TotalQuotes),
		"cache_hit":    fmt.Sprintf("%t", response.CacheHit),
	})
	h.metrics.RecordTimer("rate_calculation_duration", duration, map[string]string{
		"endpoint": "calculate_rates",
	})

	h.logger.Info("Rate calculation completed",
		"request_id", req.RequestID,
		"total_quotes", response.TotalQuotes,
		"cache_hit", response.CacheHit,
		"duration_ms", duration.Milliseconds())

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(response))
}

// CompareRates handles POST /rates/compare
func (h *RateHandler) CompareRates(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := c.Get(constants.HeaderRequestID)

	// Parse request body
	var req dtos.RateComparisonRequest
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

	// Set request ID if not provided
	if req.RequestID == "" {
		req.RequestID = requestID
		if req.RequestID == "" {
			req.RequestID = uuid.New().String()
		}
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidationErrorResponse(err))
	}

	// Set defaults for comparison parameters
	if req.SortBy == "" {
		req.SortBy = "price"
	}
	if req.SortOrder == "" {
		req.SortOrder = "asc"
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	h.logger.Info("Processing rate comparison request",
		"request_id", req.RequestID,
		"sort_by", req.SortBy,
		"sort_order", req.SortOrder,
		"limit", req.Limit)

	// Call service
	response, err := h.rateService.CompareRates(c.Context(), &req.RateCalculationRequest)
	if err != nil {
		h.logger.Error("Rate comparison failed",
			"error", err,
			"request_id", req.RequestID)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	duration := time.Since(startTime)
	h.metrics.IncrementCounter("rate_comparison_success", map[string]string{
		"endpoint": "compare_rates",
	})
	h.metrics.RecordTimer("rate_comparison_duration", duration, map[string]string{
		"endpoint": "compare_rates",
	})

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(response))
}

// GetBestRate handles POST /rates/best
func (h *RateHandler) GetBestRate(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := c.Get(constants.HeaderRequestID)

	// Parse request body
	var req dtos.RateCalculationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			constants.MessageInvalidRequest,
			constants.CodeInvalidRequest,
			err.Error(),
		))
	}

	// Set request ID if not provided
	if req.RequestID == "" {
		req.RequestID = requestID
		if req.RequestID == "" {
			req.RequestID = uuid.New().String()
		}
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidationErrorResponse(err))
	}

	h.logger.Info("Processing best rate request", "request_id", req.RequestID)

	// Call service
	bestRate, err := h.rateService.GetBestRate(c.Context(), &req)
	if err != nil {
		h.logger.Error("Get best rate failed",
			"error", err,
			"request_id", req.RequestID)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	duration := time.Since(startTime)
	h.metrics.IncrementCounter("best_rate_success", map[string]string{
		"endpoint": "get_best_rate",
	})
	h.metrics.RecordTimer("best_rate_duration", duration, map[string]string{
		"endpoint": "get_best_rate",
	})

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(bestRate))
}

// CalculateRatesByImplementation handles POST /rates/implementation/:implementationId
func (h *RateHandler) CalculateRatesByImplementation(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := c.Get(constants.HeaderRequestID)
	implementationID := c.Params("implementationId")

	if implementationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			"Implementation ID is required",
			constants.CodeMissingParameter,
			"implementationId parameter is missing",
		))
	}

	// Parse request body
	var req dtos.RateCalculationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse(
			constants.MessageInvalidRequest,
			constants.CodeInvalidRequest,
			err.Error(),
		))
	}

	// Set request ID if not provided
	if req.RequestID == "" {
		req.RequestID = requestID
		if req.RequestID == "" {
			req.RequestID = uuid.New().String()
		}
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidationErrorResponse(err))
	}

	h.logger.Info("Processing implementation-specific rate request",
		"request_id", req.RequestID,
		"implementation_id", implementationID)

	// Call service
	response, err := h.rateService.CalculateRatesByImplementation(c.Context(), &req, implementationID)
	if err != nil {
		h.logger.Error("Provider rate calculation failed",
			"error", err,
			"request_id", req.RequestID,
			"implementation_id", implementationID)

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	duration := time.Since(startTime)
	h.metrics.IncrementCounter("implementation_rate_calculation_success", map[string]string{
		"endpoint":    "calculate_rates_by_implementation",
		"implementation_id": implementationID,
	})
	h.metrics.RecordTimer("implementation_rate_calculation_duration", duration, map[string]string{
		"endpoint":    "calculate_rates_by_implementation",
		"implementation_id": implementationID,
	})

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

// RefreshImplementations handles POST /implementations/refresh
func (h *RateHandler) RefreshImplementations(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := c.Get(constants.HeaderRequestID)

	h.logger.Info("Processing implementation refresh request", "request_id", requestID)

	// Call service
	err := h.rateService.RefreshImplementations(c.Context())
	if err != nil {
		h.logger.Error("Provider refresh failed",
			"error", err,
			"request_id", requestID)

		h.metrics.IncrementCounter("implementation_refresh_failed", map[string]string{
			"endpoint": "refresh_implementations",
		})

		return utils.HandleServiceError(c, err)
	}

	// Record metrics
	duration := time.Since(startTime)
	h.metrics.IncrementCounter("implementation_refresh_success", map[string]string{
		"endpoint": "refresh_implementations",
	})
	h.metrics.RecordTimer("implementation_refresh_duration", duration, map[string]string{
		"endpoint": "refresh_implementations",
	})

	h.logger.Info("Provider refresh completed",
		"request_id", requestID,
		"duration_ms", duration.Milliseconds())

	return c.Status(fiber.StatusOK).JSON(utils.SuccessResponse(fiber.Map{
		"status":     "success",
		"message":    "Providers refreshed successfully",
		"timestamp":  time.Now(),
		"request_id": requestID,
	}))
}
