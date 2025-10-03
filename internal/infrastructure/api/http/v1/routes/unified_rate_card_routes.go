package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/prayog/prayog-supply-rate-service/internal/infrastructure/api/http/middleware"
	handlersv1 "github.com/prayog/prayog-supply-rate-service/internal/infrastructure/api/http/v1/handlers"
	unified_rate "github.com/prayog/prayog-supply-rate-service/internal/services/v1/implementations/pre_defined/unified_rate"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// SetupUnifiedRateCardRoutes sets up unified rate card management routes
func SetupUnifiedRateCardRoutes(
	router fiber.Router,
	rateCardService *unified_rate.RateCardService,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) {
	// Create unified rate card handler
	handler := handlersv1.NewUnifiedRateCardHandler(rateCardService, logger, metrics)

	// Rate card management endpoints with authentication
	// All rate card operations require rate card manager or admin permissions
	rateCards := router.Group("/unified-rate-cards", middleware.RateCardManagerAuth())

	// CRUD operations (protected by authentication)
	rateCards.Post("/", handler.CreateRateCard)      // POST /unified-rate-cards
	rateCards.Get("/", handler.ListRateCards)        // GET /unified-rate-cards
	rateCards.Get("/:id", handler.GetRateCard)       // GET /unified-rate-cards/:id
	rateCards.Put("/:id", handler.UpdateRateCard)    // PUT /unified-rate-cards/:id
	rateCards.Delete("/:id", handler.DeleteRateCard) // DELETE /unified-rate-cards/:id

	// Partner-specific operations (protected by authentication)
	rateCards.Get("/partner/:partner_code", handler.GetRateCardsByPartner) // GET /unified-rate-cards/partner/:partner_code
	rateCards.Put("/:id/set-default", handler.SetDefaultRateCard)          // PUT /unified-rate-cards/:id/set-default
}

