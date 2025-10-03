package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"

	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

const (
	// HeaderAPIKey is the header name for API key authentication
	HeaderAPIKey = "X-API-Key"

	// HeaderAuthorization is the standard authorization header
	HeaderAuthorization = "Authorization"
)

// AuthConfig holds authentication middleware configuration
type AuthConfig struct {
	AdminAPIKey     string
	RateCardAPIKey  string
	RequireAuth     bool
	SkipPaths       []string
	AllowedAPIKeys  []string
	UseEnvVariables bool
}

// NewDefaultAuthConfig creates default auth configuration
func NewDefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		AdminAPIKey:     os.Getenv("ADMIN_API_KEY"),
		RateCardAPIKey:  os.Getenv("RATE_CARD_API_KEY"),
		RequireAuth:     true,
		UseEnvVariables: true,
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/api/v1/health",
		},
		AllowedAPIKeys: []string{},
	}
}

// APIKeyAuth creates an API key authentication middleware
func APIKeyAuth(config *AuthConfig) fiber.Handler {
	// If config is nil, use default
	if config == nil {
		config = NewDefaultAuthConfig()
	}

	// Load API keys from environment if enabled
	if config.UseEnvVariables {
		if adminKey := os.Getenv("ADMIN_API_KEY"); adminKey != "" {
			config.AdminAPIKey = adminKey
		}
		if rateCardKey := os.Getenv("RATE_CARD_API_KEY"); rateCardKey != "" {
			config.RateCardAPIKey = rateCardKey
		}
	}

	// Build allowed API keys map for faster lookups
	allowedKeys := make(map[string]bool)
	if config.AdminAPIKey != "" {
		allowedKeys[config.AdminAPIKey] = true
	}
	if config.RateCardAPIKey != "" {
		allowedKeys[config.RateCardAPIKey] = true
	}
	for _, key := range config.AllowedAPIKeys {
		if key != "" {
			allowedKeys[key] = true
		}
	}

	return func(c *fiber.Ctx) error {
		// Skip authentication for certain paths
		path := c.Path()
		for _, skipPath := range config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// If auth is not required, continue
		if !config.RequireAuth {
			return c.Next()
		}

		// Extract API key from headers
		apiKey := extractAPIKey(c)

		// Check if API key is valid
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse(
				"Missing API key. Please provide a valid API key in X-API-Key header or Authorization header.",
				constants.CodeUnauthorized,
				"Missing authentication credentials",
			))
		}

		// Validate API key
		if !allowedKeys[apiKey] {
			return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse(
				"Invalid API key. Access denied.",
				constants.CodeForbidden,
				"Invalid authentication credentials",
			))
		}

		// Store API key in locals for later use
		c.Locals("api_key", apiKey)

		// Determine role based on API key
		if apiKey == config.AdminAPIKey {
			c.Locals("role", "admin")
		} else if apiKey == config.RateCardAPIKey {
			c.Locals("role", "rate_card_manager")
		} else {
			c.Locals("role", "user")
		}

		return c.Next()
	}
}

// RateCardManagerAuth creates authentication middleware specifically for rate card management endpoints
func RateCardManagerAuth() fiber.Handler {
	config := NewDefaultAuthConfig()
	return APIKeyAuth(config)
}

// AdminAuth creates authentication middleware for admin-only endpoints
func AdminAuth() fiber.Handler {
	config := NewDefaultAuthConfig()

	return func(c *fiber.Ctx) error {
		// First check API key
		apiKey := extractAPIKey(c)

		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse(
				"Missing API key. Admin access required.",
				constants.CodeUnauthorized,
				"Admin authentication required",
			))
		}

		if apiKey != config.AdminAPIKey {
			return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse(
				"Insufficient permissions. Admin access required.",
				constants.CodeInsufficientRole,
				"Admin role required for this operation",
			))
		}

		c.Locals("api_key", apiKey)
		c.Locals("role", "admin")

		return c.Next()
	}
}

// OptionalAuth creates middleware that allows both authenticated and unauthenticated requests
func OptionalAuth() fiber.Handler {
	config := NewDefaultAuthConfig()
	config.RequireAuth = false

	return APIKeyAuth(config)
}

// extractAPIKey extracts the API key from request headers
func extractAPIKey(c *fiber.Ctx) string {
	// Try X-API-Key header first
	apiKey := c.Get(HeaderAPIKey)
	if apiKey != "" {
		return apiKey
	}

	// Try Authorization header
	auth := c.Get(HeaderAuthorization)
	if auth != "" {
		// Support both "Bearer <token>" and "<token>" formats
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	}

	return ""
}

// GetAPIKey retrieves the API key from context locals
func GetAPIKey(c *fiber.Ctx) string {
	if apiKey, ok := c.Locals("api_key").(string); ok {
		return apiKey
	}
	return ""
}

// GetRole retrieves the role from context locals
func GetRole(c *fiber.Ctx) string {
	if role, ok := c.Locals("role").(string); ok {
		return role
	}
	return ""
}

// IsAdmin checks if the current user has admin role
func IsAdmin(c *fiber.Ctx) bool {
	return GetRole(c) == "admin"
}

// IsRateCardManager checks if the current user has rate card manager role or higher
func IsRateCardManager(c *fiber.Ctx) bool {
	role := GetRole(c)
	return role == "admin" || role == "rate_card_manager"
}

