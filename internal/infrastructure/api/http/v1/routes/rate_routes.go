package routes

import (
	"github.com/gofiber/fiber/v2"

	handlersv1 "github.com/prayog/prayog-supply-rate-service/internal/infrastructure/api/http/v1/handlers"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// SetupRateRoutes sets up the simplified quote route
func SetupRateRoutes(
	router fiber.Router,
	rateService interfaces.RateService,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) {
	// Create rate handler
	rateHandler := handlersv1.NewRateHandler(rateService, logger, metrics)

	// Single quotes endpoint
	router.Post("/quotes", rateHandler.GetQuotes)
}
