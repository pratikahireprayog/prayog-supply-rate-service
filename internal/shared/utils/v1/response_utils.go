package v1

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
)

// SuccessResponse creates a standardized success response
func SuccessResponse(data interface{}) *dtos.APIResponse {
	return &dtos.APIResponse{
		Success:   true,
		Message:   constants.MessageSuccess,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// ErrorResponse creates a standardized error response
func ErrorResponse(message string, code constants.ErrorCode, details string) *dtos.APIResponse {
	return &dtos.APIResponse{
		Success: false,
		Message: message,
		Error: &dtos.ErrorDetail{
			Code:    string(code),
			Message: message,
			Details: map[string]interface{}{
				"detail": details,
			},
		},
		Timestamp: time.Now(),
	}
}

// ValidationErrorResponse creates a standardized validation error response
func ValidationErrorResponse(err error) *dtos.APIResponse {
	details := make(map[string]interface{})

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		fields := make(map[string]string)
		for _, fieldError := range validationErrors {
			fields[fieldError.Field()] = getValidationErrorMessage(fieldError)
		}
		details["validation_errors"] = fields
	} else {
		details["error"] = err.Error()
	}

	return &dtos.APIResponse{
		Success: false,
		Message: constants.MessageValidationFailed,
		Error: &dtos.ErrorDetail{
			Code:    string(constants.CodeInvalidRequest),
			Message: constants.MessageValidationFailed,
			Details: details,
		},
		Timestamp: time.Now(),
	}
}

// HandleServiceError handles service errors and converts them to appropriate HTTP responses
func HandleServiceError(c *fiber.Ctx, err error) error {
	// Map service errors to HTTP status codes and error responses
	statusCode, errorCode, message := mapServiceError(err)

	response := &dtos.APIResponse{
		Success: false,
		Message: message,
		Error: &dtos.ErrorDetail{
			Code:    string(errorCode),
			Message: message,
			Details: map[string]interface{}{
				"original_error": err.Error(),
			},
		},
		Timestamp: time.Now(),
		Meta: &dtos.ResponseMetadata{
			RequestID: c.Get(constants.HeaderRequestID),
		},
	}

	return c.Status(statusCode).JSON(response)
}

// PaginatedResponse creates a standardized paginated response
func PaginatedResponse(data interface{}, pagination *dtos.Pagination) *dtos.APIResponse {
	return &dtos.APIResponse{
		Success:   true,
		Message:   constants.MessageSuccess,
		Data:      data,
		Timestamp: time.Now(),
		Meta: &dtos.ResponseMetadata{
			Pagination: pagination,
		},
	}
}

// NoContentResponse creates a standardized no content response
func NoContentResponse() *dtos.APIResponse {
	return &dtos.APIResponse{
		Success:   true,
		Message:   "No content",
		Timestamp: time.Now(),
	}
}

// CreatedResponse creates a standardized created response
func CreatedResponse(data interface{}, resourceID string) *dtos.APIResponse {
	return &dtos.APIResponse{
		Success: true,
		Message: constants.MessageCreated,
		Data:    data,
		Meta: &dtos.ResponseMetadata{
			RequestID: resourceID,
		},
		Timestamp: time.Now(),
	}
}

// UpdatedResponse creates a standardized updated response
func UpdatedResponse(data interface{}) *dtos.APIResponse {
	return &dtos.APIResponse{
		Success:   true,
		Message:   constants.MessageUpdated,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// DeletedResponse creates a standardized deleted response
func DeletedResponse() *dtos.APIResponse {
	return &dtos.APIResponse{
		Success:   true,
		Message:   constants.MessageDeleted,
		Timestamp: time.Now(),
	}
}

// Private helper functions

func getValidationErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("Value must be at least %s", err.Param())
	case "max":
		return fmt.Sprintf("Value must not exceed %s", err.Param())
	case "len":
		return fmt.Sprintf("Value must be exactly %s characters long", err.Param())
	case "oneof":
		return fmt.Sprintf("Value must be one of: %s", err.Param())
	case "gte":
		return fmt.Sprintf("Value must be greater than or equal to %s", err.Param())
	case "lte":
		return fmt.Sprintf("Value must be less than or equal to %s", err.Param())
	case "gt":
		return fmt.Sprintf("Value must be greater than %s", err.Param())
	case "lt":
		return fmt.Sprintf("Value must be less than %s", err.Param())
	case "gtfield":
		return fmt.Sprintf("Value must be greater than %s", err.Param())
	case "ltfield":
		return fmt.Sprintf("Value must be less than %s", err.Param())
	case "alphanum":
		return "Value must contain only letters and numbers"
	case "alpha":
		return "Value must contain only letters"
	case "numeric":
		return "Value must be numeric"
	case "url":
		return "Invalid URL format"
	case "uuid":
		return "Invalid UUID format"
	default:
		return fmt.Sprintf("Validation failed for tag: %s", err.Tag())
	}
}

func mapServiceError(err error) (int, constants.ErrorCode, string) {
	// Map common service errors to HTTP status codes
	switch err {
	case constants.ErrRateNotFound, constants.ErrNoRatesAvailable:
		return fiber.StatusNotFound, constants.CodeRateNotFound, constants.MessageRateNotFound

	case constants.ErrPartnerNotFound:
		return fiber.StatusNotFound, constants.CodePartnerNotFound, "Partner not found"

	case constants.ErrImplementationNotFound, constants.ErrImplementationNotRegistered:
		return fiber.StatusNotFound, constants.CodeImplementationNotFound, "Implementation not found"

	case constants.ErrInvalidRequest, constants.ErrInvalidWeight, constants.ErrInvalidDistance, constants.ErrInvalidServiceType, constants.ErrInvalidDateRange, constants.ErrInvalidCurrency, constants.ErrMissingParameter:
		return fiber.StatusBadRequest, constants.CodeInvalidRequest, constants.MessageInvalidRequest

	case constants.ErrUnauthorized, constants.ErrInvalidToken:
		return fiber.StatusUnauthorized, constants.CodeUnauthorized, constants.MessageUnauthorized

	case constants.ErrForbidden, constants.ErrInsufficientRole:
		return fiber.StatusForbidden, constants.CodeForbidden, constants.MessageForbidden

	case constants.ErrRateLimitExceeded, constants.ErrTooManyRequests:
		return fiber.StatusTooManyRequests, constants.CodeRateLimitExceeded, constants.MessageRateLimitExceeded

	case constants.ErrPartnerTimeout, constants.ErrOperationTimeout:
		return fiber.StatusRequestTimeout, constants.CodePartnerTimeout, "Request timeout"

	case constants.ErrPartnerUnhealthy, constants.ErrServiceUnavailable:
		return fiber.StatusServiceUnavailable, constants.CodeServiceUnavailable, constants.MessageServiceUnavailable

	case constants.ErrNotImplemented:
		return fiber.StatusNotImplemented, constants.CodeNotImplemented, "Feature not implemented"

	default:
		// For unknown errors, return internal server error
		return fiber.StatusInternalServerError, constants.CodeInternalServer, constants.MessageInternalError
	}
}

// ResponseWrapper wraps responses with consistent metadata
func ResponseWrapper(c *fiber.Ctx, statusCode int, response *dtos.APIResponse) error {
	// Add request ID to metadata if not present
	if response.Meta == nil {
		response.Meta = &dtos.ResponseMetadata{}
	}

	if response.Meta.RequestID == "" {
		response.Meta.RequestID = c.Get(constants.HeaderRequestID)
	}

	// Add response time
	if response.Meta.ResponseTime == 0 {
		// This would ideally be calculated from request start time
		response.Meta.ResponseTime = 0
	}

	// Add version
	response.Meta.Version = constants.APIVersionV1

	return c.Status(statusCode).JSON(response)
}

// HealthResponse creates a health check response
func HealthResponse(status string, checks map[string]interface{}) *dtos.HealthCheckResponse {
	return &dtos.HealthCheckResponse{
		Status:    status,
		Message:   fmt.Sprintf("Service is %s", status),
		Services:  convertToServiceHealth(checks),
		Timestamp: time.Now(),
		Version:   constants.APIVersionV1,
	}
}

func convertToServiceHealth(checks map[string]interface{}) map[string]dtos.ServiceHealth {
	services := make(map[string]dtos.ServiceHealth)

	for name, check := range checks {
		if healthData, ok := check.(map[string]interface{}); ok {
			status := "unknown"
			if s, exists := healthData["status"]; exists {
				if statusStr, ok := s.(string); ok {
					status = statusStr
				}
			}

			services[name] = dtos.ServiceHealth{
				Status:      status,
				LastChecked: time.Now(),
			}
		}
	}

	return services
}
