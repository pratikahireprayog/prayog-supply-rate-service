package middleware

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// ResponseTime middleware adds response time header
func ResponseTime() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start)
		c.Set(constants.HeaderResponseTime, fmt.Sprintf("%.2fms", float64(duration.Nanoseconds())/1e6))

		return err
	}
}

// APIVersion middleware adds API version header
func APIVersion() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set(constants.HeaderAPIVersion, constants.APIVersionV1)
		return c.Next()
	}
}

// RateLimiter middleware implements rate limiting
func RateLimiter(logger interfaces.Logger, metrics interfaces.MetricsCollector) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        constants.DefaultRateLimit,
		Expiration: time.Duration(constants.DefaultRateLimitWindow) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use IP address as the key, but could be enhanced with user ID
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			logger.Warn("Rate limit exceeded",
				"ip", c.IP(),
				"path", c.Path(),
				"method", c.Method(),
				"user_agent", c.Get("User-Agent"))

			metrics.IncrementCounter("rate_limit_exceeded", map[string]string{
				"ip":     c.IP(),
				"path":   c.Path(),
				"method": c.Method(),
			})

			// Set rate limit headers
			c.Set(constants.HeaderRateLimit, strconv.Itoa(constants.DefaultRateLimit))
			c.Set(constants.HeaderRateRemaining, "0")
			c.Set(constants.HeaderRateReset, strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": constants.MessageRateLimitExceeded,
				"error": fiber.Map{
					"code":    string(constants.CodeRateLimitExceeded),
					"message": constants.MessageRateLimitExceeded,
				},
				"timestamp":  time.Now(),
				"request_id": c.Get(constants.HeaderRequestID),
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
		Storage:                nil, // Use in-memory storage by default
	})
}

// Metrics middleware collects HTTP metrics
func Metrics(metrics interfaces.MetricsCollector) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		// Record metrics
		metrics.IncrementCounter("http_requests_total", map[string]string{
			"method":      c.Method(),
			"path":        c.Path(),
			"status_code": strconv.Itoa(statusCode),
		})

		metrics.RecordHistogram("http_request_duration_seconds", duration.Seconds(), map[string]string{
			"method": c.Method(),
			"path":   c.Path(),
		})

		// Record status code distribution
		statusClass := fmt.Sprintf("%dxx", statusCode/100)
		metrics.IncrementCounter("http_responses_by_status", map[string]string{
			"status_class": statusClass,
			"method":       c.Method(),
		})

		return err
	}
}

// RequestValidation middleware validates common request parameters
func RequestValidation() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Validate request size
		if len(c.Body()) > constants.DefaultRequestTimeout {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"success": false,
				"message": "Request body too large",
				"error": fiber.Map{
					"code":    "REQUEST_TOO_LARGE",
					"message": "Request body exceeds maximum allowed size",
				},
				"timestamp":  time.Now(),
				"request_id": c.Get(constants.HeaderRequestID),
			})
		}

		// Validate Content-Type for POST/PUT requests
		if c.Method() == "POST" || c.Method() == "PUT" {
			contentType := c.Get("Content-Type")
			if contentType != "" && contentType != constants.ContentTypeJSON {
				return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
					"success": false,
					"message": "Unsupported media type",
					"error": fiber.Map{
						"code":    "UNSUPPORTED_MEDIA_TYPE",
						"message": fmt.Sprintf("Content-Type %s is not supported. Use %s", contentType, constants.ContentTypeJSON),
					},
					"timestamp":  time.Now(),
					"request_id": c.Get(constants.HeaderRequestID),
				})
			}
		}

		return c.Next()
	}
}

// SecurityHeaders middleware adds security headers
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Security headers
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Content-Security-Policy", "default-src 'self'")

		return c.Next()
	}
}

// RequestLogging middleware logs detailed request information
func RequestLogging(logger interfaces.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Log request
		logger.Info("HTTP request started",
			"method", c.Method(),
			"path", c.Path(),
			"remote_addr", c.IP(),
			"user_agent", c.Get("User-Agent"),
			"request_id", c.Get(constants.HeaderRequestID),
			"content_length", len(c.Body()))

		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		// Log response
		logLevel := "info"
		if statusCode >= 400 {
			logLevel = "warn"
		}
		if statusCode >= 500 {
			logLevel = "error"
		}

		logger.Info("HTTP request completed",
			"method", c.Method(),
			"path", c.Path(),
			"status_code", statusCode,
			"duration_ms", duration.Milliseconds(),
			"response_size", len(c.Response().Body()),
			"request_id", c.Get(constants.HeaderRequestID),
			"level", logLevel)

		return err
	}
}
