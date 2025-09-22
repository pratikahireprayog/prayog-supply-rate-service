package routes

import (
	"github.com/gofiber/fiber/v2"

	handlersv1 "github.com/prayog/prayog-rate-service/internal/infrastructure/api/http/v1/handlers"
	interfaces "github.com/prayog/prayog-rate-service/internal/shared/interfaces/v1"
)

// SetupRateRoutes sets up all rate-related routes
func SetupRateRoutes(
	router fiber.Router,
	rateService interfaces.RateService,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) {
	// Create rate handler
	rateHandler := handlersv1.NewRateHandler(rateService, logger, metrics)

	// Rate calculation routes
	rates := router.Group("/rates")
	{
		// POST /rate/v1/rates/calculate - Calculate rates from all implementations
		rates.Post("/calculate", rateHandler.CalculateRates)

		// POST /rate/v1/rates/quote - Get a quote (alias for calculate)
		rates.Post("/quote", rateHandler.CalculateRates)

		// POST /rate/v1/rates/compare - Compare rates from multiple implementations
		rates.Post("/compare", rateHandler.CompareRates)

		// POST /rate/v1/rates/best - Get the best rate from all implementations
		rates.Post("/best", rateHandler.GetBestRate)

		// POST /rate/v1/rates/implementation/:implementationId - Get rates from specific implementation
		rates.Post("/implementation/:implementationId", rateHandler.CalculateRatesByImplementation)
	}

	// Implementation health routes
	implementations := router.Group("/implementations")
	{
		// GET /rate/v1/implementations/health - Get health status of all implementations
		implementations.Get("/health", rateHandler.GetImplementationHealth)

		// POST /rate/v1/implementations/refresh - Refresh all implementation configurations
		implementations.Post("/refresh", rateHandler.RefreshImplementations)
	}

	// Add additional route groups for future endpoints
	setupPartnerRoutes(router, logger, metrics)
	setupAdminRoutes(router, logger, metrics)
}

// setupPartnerRoutes sets up partner management routes (for future implementation)
func setupPartnerRoutes(
	router fiber.Router,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) {
	partners := router.Group("/partners")
	{
		// GET /rate/v1/partners - List all partners
		partners.Get("/", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
				"success": false,
				"message": "Partner management endpoints will be implemented in Phase 3",
				"status":  "not_implemented",
			})
		})

		// GET /rate/v1/partners/:id - Get partner by ID
		partners.Get("/:id", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
				"success": false,
				"message": "Partner management endpoints will be implemented in Phase 3",
				"status":  "not_implemented",
			})
		})

		// POST /rate/v1/partners - Create new partner
		partners.Post("/", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
				"success": false,
				"message": "Partner management endpoints will be implemented in Phase 3",
				"status":  "not_implemented",
			})
		})

		// PUT /rate/v1/partners/:id - Update partner
		partners.Put("/:id", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
				"success": false,
				"message": "Partner management endpoints will be implemented in Phase 3",
				"status":  "not_implemented",
			})
		})

		// DELETE /rate/v1/partners/:id - Delete partner
		partners.Delete("/:id", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
				"success": false,
				"message": "Partner management endpoints will be implemented in Phase 3",
				"status":  "not_implemented",
			})
		})
	}
}

// setupAdminRoutes sets up administrative routes
func setupAdminRoutes(
	router fiber.Router,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) {
	admin := router.Group("/admin")
	{
		// GET /rate/v1/admin/stats - Get service statistics
		admin.Get("/stats", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"success": true,
				"data": fiber.Map{
					"service": "prayog-rate-service",
					"version": "1.0.0",
					"status":  "operational",
					"endpoints": fiber.Map{
						"rates_calculate":       "/rate/v1/rates/calculate",
						"rates_compare":         "/rate/v1/rates/compare",
						"rates_best":            "/rate/v1/rates/best",
						"implementation_health": "/rate/v1/implementations/health",
					},
				},
			})
		})

		// POST /rate/v1/admin/cache/clear - Clear cache
		admin.Post("/cache/clear", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
				"success": false,
				"message": "Cache management will be implemented in future phases",
				"status":  "not_implemented",
			})
		})

		// GET /rate/v1/admin/health/deep - Deep health check
		admin.Get("/health/deep", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
				"success": false,
				"message": "Deep health check will be implemented in future phases",
				"status":  "not_implemented",
			})
		})
	}
}
