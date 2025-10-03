package v1

// HTTP status messages
const (
	StatusOK                  = "OK"
	StatusCreated             = "Created"
	StatusAccepted            = "Accepted"
	StatusNoContent           = "No Content"
	StatusBadRequest          = "Bad Request"
	StatusUnauthorized        = "Unauthorized"
	StatusForbidden           = "Forbidden"
	StatusNotFound            = "Not Found"
	StatusMethodNotAllowed    = "Method Not Allowed"
	StatusConflict            = "Conflict"
	StatusUnprocessableEntity = "Unprocessable Entity"
	StatusTooManyRequests     = "Too Many Requests"
	StatusInternalServerError = "Internal Server Error"
	StatusBadGateway          = "Bad Gateway"
	StatusServiceUnavailable  = "Service Unavailable"
	StatusGatewayTimeout      = "Gateway Timeout"
)

// API response messages
const (
	MessageSuccess            = "Operation completed successfully"
	MessageCreated            = "Resource created successfully"
	MessageUpdated            = "Resource updated successfully"
	MessageDeleted            = "Resource deleted successfully"
	MessageNotFound           = "Resource not found"
	MessageValidationFailed   = "Validation failed"
	MessageUnauthorized       = "Authentication required"
	MessageForbidden          = "Access denied"
	MessageInternalError      = "Internal server error occurred"
	MessageServiceUnavailable = "Service temporarily unavailable"
	MessageRateLimitExceeded  = "Rate limit exceeded. Please try again later"
	MessageInvalidRequest     = "Invalid request parameters"
	MessageDuplicateResource  = "Resource already exists"
)

// Rate-specific response messages
const (
	MessageRateCalculated     = "Rate calculated successfully"
	MessageRateNotFound       = "No rates found for the specified criteria"
	MessageRateExpired        = "Rate quote has expired"
	MessagePartnerUnavailable = "Rate partner is currently unavailable"
	MessageCalculationFailed  = "Rate calculation failed"
	MessageInvalidCriteria    = "Invalid rate calculation criteria"
)

// Content types
const (
	ContentTypeJSON = "application/json"
	ContentTypeXML  = "application/xml"
	ContentTypeForm = "application/x-www-form-urlencoded"
	ContentTypeText = "text/plain"
)

// Headers
const (
	HeaderContentType   = "Content-Type"
	HeaderAccept        = "Accept"
	HeaderAuthorization = "Authorization"
	HeaderUserAgent     = "User-Agent"
	HeaderRequestID     = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
	HeaderRateLimit     = "X-Rate-Limit-Limit"
	HeaderRateRemaining = "X-Rate-Limit-Remaining"
	HeaderRateReset     = "X-Rate-Limit-Reset"
	HeaderAPIVersion    = "X-API-Version"
	HeaderResponseTime  = "X-Response-Time"
)

// API versions
const (
	APIVersionV1 = "v1"
	APIVersionV2 = "v2"
)

// Default timeouts (in milliseconds)
const (
	DefaultRequestTimeout  = 30000 // 30 seconds
	DefaultConnectTimeout  = 5000  // 5 seconds
	DefaultReadTimeout     = 15000 // 15 seconds
	DefaultWriteTimeout    = 15000 // 15 seconds
	DefaultPartnerTimeout  = 10000 // 10 seconds for partner API calls
	DefaultDatabaseTimeout = 5000  // 5 seconds for database operations
	DefaultCacheTimeout    = 1000  // 1 second for cache operations
)

// Rate limiting defaults
const (
	DefaultRateLimit       = 1000 // 1000 requests per window
	DefaultRateLimitWindow = 3600 // 1 hour in seconds
	DefaultBurstLimit      = 100  // 100 requests burst
)

// Pagination defaults
const (
	DefaultPageSize    = 20
	DefaultMaxPageSize = 100
	DefaultPage        = 1
)

// Cache durations (in seconds)
const (
	CacheDurationShort  = 300   // 5 minutes
	CacheDurationMedium = 1800  // 30 minutes
	CacheDurationLong   = 3600  // 1 hour
	CacheDurationDay    = 86400 // 24 hours
)

// URL patterns
const (
	URLPatternRateCalculate = "/api/v1/rates/calculate"
	URLPatternRateQuote     = "/api/v1/rates/quote"
	URLPatternPartners      = "/api/v1/partners"
	URLPatternHealth        = "/health"
	URLPatternMetrics       = "/metrics"
	URLPatternDocs          = "/docs"
)
