package v1

import "errors"

// Domain-specific error constants
var (
	// Rate-related errors
	ErrRateNotFound        = errors.New("rate not found")
	ErrRateExpired         = errors.New("rate has expired")
	ErrRateInvalid         = errors.New("rate is invalid")
	ErrRateCalculationFail = errors.New("rate calculation failed")
	ErrNoRatesAvailable    = errors.New("no rates available for the given criteria")

	// Partner-related errors
	ErrPartnerNotFound     = errors.New("partner not found")
	ErrPartnerInactive     = errors.New("partner is inactive")
	ErrPartnerUnhealthy    = errors.New("partner is unhealthy")
	ErrPartnerTimeout      = errors.New("partner request timeout")
	ErrPartnerAPIError     = errors.New("partner API error")
	ErrPartnerNotSupported = errors.New("partner type not supported")

	// Implementation factory errors
	ErrImplementationNotFound      = errors.New("rate implementation not found")
	ErrImplementationNotRegistered = errors.New("rate implementation not registered")
	ErrImplementationInitFail      = errors.New("rate implementation initialization failed")
	ErrImplementationHealthCheck   = errors.New("rate implementation health check failed")

	// Validation errors
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidWeight      = errors.New("invalid weight specified")
	ErrInvalidDistance    = errors.New("invalid distance specified")
	ErrInvalidServiceType = errors.New("invalid service type")
	ErrInvalidDateRange   = errors.New("invalid date range")
	ErrInvalidCurrency    = errors.New("invalid currency")
	ErrMissingParameter   = errors.New("missing required parameter")

	// Database errors
	ErrDatabaseConnection  = errors.New("database connection error")
	ErrDatabaseQuery       = errors.New("database query error")
	ErrDatabaseTransaction = errors.New("database transaction error")
	ErrRecordNotFound      = errors.New("record not found")
	ErrRecordExists        = errors.New("record already exists")
	ErrRecordInvalid       = errors.New("invalid record data")

	// Cache errors
	ErrCacheConnection = errors.New("cache connection error")
	ErrCacheMiss       = errors.New("cache miss")
	ErrCacheInvalid    = errors.New("invalid cache data")

	// Authentication and authorization errors
	ErrUnauthorized     = errors.New("unauthorized access")
	ErrForbidden        = errors.New("forbidden operation")
	ErrInvalidToken     = errors.New("invalid authentication token")
	ErrTokenExpired     = errors.New("authentication token expired")
	ErrInsufficientRole = errors.New("insufficient role permissions")

	// Rate limiting and quota errors
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrQuotaExceeded     = errors.New("quota exceeded")
	ErrTooManyRequests   = errors.New("too many requests")

	// External service errors
	ErrExternalServiceTimeout     = errors.New("external service timeout")
	ErrExternalServiceUnavailable = errors.New("external service unavailable")
	ErrExternalServiceError       = errors.New("external service error")

	// Configuration errors
	ErrInvalidConfig  = errors.New("invalid configuration")
	ErrMissingConfig  = errors.New("missing configuration")
	ErrConfigLoadFail = errors.New("configuration load failed")

	// Generic errors
	ErrInternalServer     = errors.New("internal server error")
	ErrNotImplemented     = errors.New("feature not implemented")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrOperationTimeout   = errors.New("operation timeout")
)

// ErrorCode represents error codes for API responses
type ErrorCode string

const (
	// Rate-related error codes
	CodeRateNotFound        ErrorCode = "RATE_NOT_FOUND"
	CodeRateExpired         ErrorCode = "RATE_EXPIRED"
	CodeRateInvalid         ErrorCode = "RATE_INVALID"
	CodeRateCalculationFail ErrorCode = "RATE_CALCULATION_FAIL"
	CodeNoRatesAvailable    ErrorCode = "NO_RATES_AVAILABLE"

	// Partner-related error codes
	CodePartnerNotFound     ErrorCode = "PARTNER_NOT_FOUND"
	CodePartnerInactive     ErrorCode = "PARTNER_INACTIVE"
	CodePartnerUnhealthy    ErrorCode = "PARTNER_UNHEALTHY"
	CodePartnerTimeout      ErrorCode = "PARTNER_TIMEOUT"
	CodePartnerAPIError     ErrorCode = "PARTNER_API_ERROR"
	CodePartnerNotSupported ErrorCode = "PARTNER_NOT_SUPPORTED"

	// Implementation factory error codes
	CodeImplementationNotFound      ErrorCode = "IMPLEMENTATION_NOT_FOUND"
	CodeImplementationNotRegistered ErrorCode = "IMPLEMENTATION_NOT_REGISTERED"
	CodeImplementationInitFail      ErrorCode = "IMPLEMENTATION_INIT_FAIL"
	CodeImplementationHealthCheck   ErrorCode = "IMPLEMENTATION_HEALTH_CHECK_FAIL"

	// Validation error codes
	CodeInvalidRequest     ErrorCode = "INVALID_REQUEST"
	CodeInvalidWeight      ErrorCode = "INVALID_WEIGHT"
	CodeInvalidDistance    ErrorCode = "INVALID_DISTANCE"
	CodeInvalidServiceType ErrorCode = "INVALID_SERVICE_TYPE"
	CodeInvalidDateRange   ErrorCode = "INVALID_DATE_RANGE"
	CodeInvalidCurrency    ErrorCode = "INVALID_CURRENCY"
	CodeMissingParameter   ErrorCode = "MISSING_PARAMETER"

	// Database error codes
	CodeDatabaseConnection  ErrorCode = "DATABASE_CONNECTION_ERROR"
	CodeDatabaseQuery       ErrorCode = "DATABASE_QUERY_ERROR"
	CodeDatabaseTransaction ErrorCode = "DATABASE_TRANSACTION_ERROR"
	CodeRecordNotFound      ErrorCode = "RECORD_NOT_FOUND"
	CodeRecordExists        ErrorCode = "RECORD_EXISTS"
	CodeRecordInvalid       ErrorCode = "RECORD_INVALID"

	// Authentication error codes
	CodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	CodeForbidden        ErrorCode = "FORBIDDEN"
	CodeInvalidToken     ErrorCode = "INVALID_TOKEN"
	CodeTokenExpired     ErrorCode = "TOKEN_EXPIRED"
	CodeInsufficientRole ErrorCode = "INSUFFICIENT_ROLE"

	// Rate limiting error codes
	CodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	CodeQuotaExceeded     ErrorCode = "QUOTA_EXCEEDED"
	CodeTooManyRequests   ErrorCode = "TOO_MANY_REQUESTS"

	// Generic error codes
	CodeInternalServer     ErrorCode = "INTERNAL_SERVER_ERROR"
	CodeNotImplemented     ErrorCode = "NOT_IMPLEMENTED"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	CodeOperationTimeout   ErrorCode = "OPERATION_TIMEOUT"
)

// String returns the string representation of ErrorCode
func (ec ErrorCode) String() string {
	return string(ec)
}

// ErrorToCode maps errors to their corresponding error codes
func ErrorToCode(err error) ErrorCode {
	switch err {
	case ErrRateNotFound:
		return CodeRateNotFound
	case ErrRateExpired:
		return CodeRateExpired
	case ErrRateInvalid:
		return CodeRateInvalid
	case ErrRateCalculationFail:
		return CodeRateCalculationFail
	case ErrNoRatesAvailable:
		return CodeNoRatesAvailable
	case ErrPartnerNotFound:
		return CodePartnerNotFound
	case ErrPartnerInactive:
		return CodePartnerInactive
	case ErrPartnerUnhealthy:
		return CodePartnerUnhealthy
	case ErrPartnerTimeout:
		return CodePartnerTimeout
	case ErrPartnerAPIError:
		return CodePartnerAPIError
	case ErrPartnerNotSupported:
		return CodePartnerNotSupported
	case ErrImplementationNotFound:
		return CodeImplementationNotFound
	case ErrImplementationNotRegistered:
		return CodeImplementationNotRegistered
	case ErrImplementationInitFail:
		return CodeImplementationInitFail
	case ErrImplementationHealthCheck:
		return CodeImplementationHealthCheck
	case ErrInvalidRequest:
		return CodeInvalidRequest
	case ErrInvalidWeight:
		return CodeInvalidWeight
	case ErrInvalidDistance:
		return CodeInvalidDistance
	case ErrInvalidServiceType:
		return CodeInvalidServiceType
	case ErrInvalidDateRange:
		return CodeInvalidDateRange
	case ErrInvalidCurrency:
		return CodeInvalidCurrency
	case ErrMissingParameter:
		return CodeMissingParameter
	case ErrUnauthorized:
		return CodeUnauthorized
	case ErrForbidden:
		return CodeForbidden
	case ErrInvalidToken:
		return CodeInvalidToken
	case ErrTokenExpired:
		return CodeTokenExpired
	case ErrInsufficientRole:
		return CodeInsufficientRole
	case ErrRateLimitExceeded:
		return CodeRateLimitExceeded
	case ErrQuotaExceeded:
		return CodeQuotaExceeded
	case ErrTooManyRequests:
		return CodeTooManyRequests
	case ErrNotImplemented:
		return CodeNotImplemented
	case ErrServiceUnavailable:
		return CodeServiceUnavailable
	case ErrOperationTimeout:
		return CodeOperationTimeout
	default:
		return CodeInternalServer
	}
}
