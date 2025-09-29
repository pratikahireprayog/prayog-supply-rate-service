package http

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/prayog/prayog-supply-rate-service/internal/infrastructure/api/http/middleware"
	routesv1 "github.com/prayog/prayog-supply-rate-service/internal/infrastructure/api/http/v1/routes"
	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// ServerConfig holds the server configuration
type ServerConfig struct {
	Port              string
	Host              string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	GracefulShutdown  time.Duration
	RequestBodyLimit  int
	EnableCORS        bool
	EnableHealthCheck bool
	EnableMetrics     bool
	EnableProfiling   bool
	TrustedProxies    []string
}

// DefaultServerConfig returns a default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:              "8080",
		Host:              "0.0.0.0",
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		GracefulShutdown:  10 * time.Second,
		RequestBodyLimit:  10 * 1024 * 1024, // 10MB
		EnableCORS:        true,
		EnableHealthCheck: true,
		EnableMetrics:     true,
		EnableProfiling:   false,
	}
}

// Server wraps the Fiber application with additional functionality
type Server struct {
	app    *fiber.App
	config *ServerConfig

	// Dependencies
	rateService interfaces.RateService
	logger      interfaces.Logger
	metrics     interfaces.MetricsCollector
}

// NewServer creates a new HTTP server instance
func NewServer(
	config *ServerConfig,
	rateService interfaces.RateService,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	server := &Server{
		config:      config,
		rateService: rateService,
		logger:      logger,
		metrics:     metrics,
	}

	server.app = server.createFiberApp()
	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	address := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)

	s.logger.Info("Starting HTTP server",
		"address", address,
		"read_timeout", s.config.ReadTimeout,
		"write_timeout", s.config.WriteTimeout)

	// Start server in a goroutine
	go func() {
		if err := s.app.Listen(address); err != nil {
			s.logger.Fatal("Failed to start server", "error", err, "address", address)
		}
	}()

	s.logger.Info("HTTP server started successfully", "address", address)

	// Wait for interrupt signal to gracefully shutdown
	return s.waitForShutdown()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server...")

	if err := s.app.ShutdownWithContext(ctx); err != nil {
		s.logger.Error("Error during server shutdown", "error", err)
		return err
	}

	s.logger.Info("HTTP server stopped successfully")
	return nil
}

// GetApp returns the underlying Fiber app for testing
func (s *Server) GetApp() *fiber.App {
	return s.app
}

// Private methods

func (s *Server) createFiberApp() *fiber.App {
	return fiber.New(fiber.Config{
		AppName:                 "Rate Card Service",
		EnableTrustedProxyCheck: len(s.config.TrustedProxies) > 0,
		TrustedProxies:          s.config.TrustedProxies,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		ServerHeader:            "Rate-Service/1.0",

		// Body limits
		BodyLimit: s.config.RequestBodyLimit,

		// Timeouts
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,

		// Error handling
		ErrorHandler: s.errorHandler,

		// Disable startup banner
		DisableStartupMessage: true,

		// Case sensitivity
		CaseSensitive: false,

		// Strict routing
		StrictRouting: false,
	})
}

func (s *Server) setupMiddleware() {
	// Recovery middleware (should be first)
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Request ID middleware
	s.app.Use(requestid.New(requestid.Config{
		Header: constants.HeaderRequestID,
		Generator: func() string {
			return fmt.Sprintf("%d", time.Now().UnixNano())
		},
	}))

	// Logger middleware
	s.app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path} ${error}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "UTC",
	}))

	// CORS middleware
	if s.config.EnableCORS {
		s.app.Use(cors.New(cors.Config{
			AllowOrigins:     "*",
			AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
			AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With,X-Request-ID,X-API-Version",
			ExposeHeaders:    "X-Request-ID,X-Response-Time,X-Rate-Limit-Limit,X-Rate-Limit-Remaining",
			AllowCredentials: false,
			MaxAge:           86400, // 24 hours
		}))
	}

	// Health check middleware
	if s.config.EnableHealthCheck {
		s.app.Use(healthcheck.New(healthcheck.Config{
			LivenessProbe: func(c *fiber.Ctx) bool {
				return true
			},
			LivenessEndpoint: "/health/live",
			ReadinessProbe: func(c *fiber.Ctx) bool {
				// Check if services are ready
				return s.isServiceReady()
			},
			ReadinessEndpoint: "/health/ready",
		}))
	}

	// Custom middleware
	s.app.Use(middleware.ResponseTime())
	s.app.Use(middleware.RateLimiter(s.logger, s.metrics))

	// Metrics middleware
	if s.config.EnableMetrics {
		s.app.Use(middleware.Metrics(s.metrics))
	}

	// API version middleware
	s.app.Use(middleware.APIVersion())
}

func (s *Server) setupRoutes() {
	// Supply Rate API with global prefix
	supplyRate := s.app.Group("/supply-rate")

	// Health endpoints under supply-rate prefix
	supplyRate.Get("/health", s.healthHandler)

	// Metrics endpoint under supply-rate prefix
	if s.config.EnableMetrics {
		supplyRate.Get("/metrics", s.metricsHandler)
	}

	// API documentation at supply-rate root
	supplyRate.Get("/", s.apiInfoHandler)

	// Version 1 routes directly under supply-rate
	v1 := supplyRate.Group("/v1")
	routesv1.SetupRateRoutes(v1, s.rateService, s.logger, s.metrics)
}

func (s *Server) errorHandler(c *fiber.Ctx, err error) error {
	// Default error code
	code := fiber.StatusInternalServerError
	message := constants.MessageInternalError

	// Check for Fiber errors
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log the error
	s.logger.Error("HTTP error occurred",
		"error", err.Error(),
		"path", c.Path(),
		"method", c.Method(),
		"status_code", code,
		"request_id", c.Get(constants.HeaderRequestID))

	// Record metrics
	s.metrics.IncrementCounter("http_errors", map[string]string{
		"status_code": fmt.Sprintf("%d", code),
		"method":      c.Method(),
		"path":        c.Path(),
	})

	// Return error response
	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"message": message,
		"error": fiber.Map{
			"code":    fmt.Sprintf("HTTP_%d", code),
			"message": message,
		},
		"timestamp":  time.Now(),
		"request_id": c.Get(constants.HeaderRequestID),
	})
}

func (s *Server) waitForShutdown() error {
	// Create channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal received
	sig := <-quit
	s.logger.Info("Shutdown signal received", "signal", sig.String())

	// Create context for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.GracefulShutdown)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.Stop(ctx); err != nil {
		s.logger.Error("Forced shutdown due to error", "error", err)
		return err
	}

	return nil
}

func (s *Server) isServiceReady() bool {
	// Check if rate service is available
	if s.rateService == nil {
		return false
	}

	// Additional readiness checks can be added here
	// e.g., database connectivity, cache availability, etc.

	return true
}

// HTTP Handlers

func (s *Server) healthHandler(c *fiber.Ctx) error {
	deep := c.QueryBool("deep", false)

	healthStatus := fiber.Map{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "prayog-supply-rate-service",
		"version":   constants.APIVersionV1,
		"uptime":    time.Since(time.Now()).String(), // This should be actual uptime
	}

	if deep {
		// Perform deep health checks
		if s.rateService != nil {
			if healthResponse, err := s.rateService.GetImplementationHealth(c.Context()); err == nil {
				healthStatus["implementations"] = healthResponse
			} else {
				healthStatus["implementations"] = fiber.Map{
					"status": "unhealthy",
					"error":  err.Error(),
				}
				healthStatus["status"] = "degraded"
			}
		}
	}

	// Determine overall status
	status := fiber.StatusOK
	if healthStatus["status"] == "unhealthy" {
		status = fiber.StatusServiceUnavailable
	} else if healthStatus["status"] == "degraded" {
		status = fiber.StatusPartialContent
	}

	return c.Status(status).JSON(healthStatus)
}

func (s *Server) metricsHandler(c *fiber.Ctx) error {
	// Return basic metrics information
	// In production, this might integrate with Prometheus or similar
	metrics := fiber.Map{
		"service":   "prayog-supply-rate-service",
		"timestamp": time.Now(),
		"metrics": fiber.Map{
			"http_requests_total":     "counter",
			"http_request_duration":   "histogram",
			"rate_calculations_total": "counter",
			"provider_health_status":  "gauge",
		},
	}

	return c.JSON(metrics)
}

func (s *Server) apiInfoHandler(c *fiber.Ctx) error {
	info := fiber.Map{
		"service":     "Prayog Supply Rate Service",
		"description": "A microservice for calculating shipping rates from multiple logistics partners",
		"version":     constants.APIVersionV1,
		"endpoints": fiber.Map{
			"health":    "/supply-rate/health",
			"metrics":   "/supply-rate/metrics",
			"api_info":  "/supply-rate/",
			"quotes_v1": "/supply-rate/v1/quotes",
		},
		"documentation": "/docs",
		"timestamp":     time.Now(),
	}

	return c.JSON(info)
}
