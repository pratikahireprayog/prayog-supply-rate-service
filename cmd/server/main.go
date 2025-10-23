package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	httpserver "github.com/prayog/prayog-supply-rate-service/internal/infrastructure/api/http"
	database "github.com/prayog/prayog-supply-rate-service/internal/infrastructure/database"
	rateservice "github.com/prayog/prayog-supply-rate-service/internal/services/v1"
	"github.com/prayog/prayog-supply-rate-service/internal/services/v1/factory"
	"github.com/prayog/prayog-supply-rate-service/internal/services/v1/implementations/pre_defined/unified_rate"
	"github.com/prayog/prayog-supply-rate-service/internal/services/v1/implementations/real_time/dhl"
	indiapost "github.com/prayog/prayog-supply-rate-service/internal/services/v1/implementations/real_time/india_post"
	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
	repositoriesv1 "github.com/prayog/prayog-supply-rate-service/internal/shared/repositories/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

const (
	// Service information
	ServiceName    = "prayog-supply-rate-service"
	ServiceVersion = "1.0.0"

	// Default configuration values
	DefaultPort = "9046"
	DefaultHost = "0.0.0.0"
)

func main() {
	// Load .env file (silently ignore if not found)
	if err := godotenv.Load(); err != nil {
		log.Printf("Note: .env file not found or could not be loaded (this is fine if using environment variables)")
	} else {
		log.Printf("Loaded configuration from .env file")
	}

	// Print startup banner
	printBanner()

	// Load configuration
	config := loadConfiguration()

	// Load database configuration
	dbConfig := loadDatabaseConfig()

	// Initialize dependencies (with real database)
	dependencies, cleanup := initializeDependencies(dbConfig)
	defer cleanup()

	// Create and configure HTTP server
	server := httpserver.NewServer(
		config,
		dependencies.RateService,
		dependencies.RateCardService,
		dependencies.Logger,
		dependencies.Metrics,
	)

	// Start server
	log.Printf("Starting %s v%s", ServiceName, ServiceVersion)
	log.Printf("Server will listen on %s:%s", config.Host, config.Port)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func printBanner() {
	banner := `
    ____                              ____        __         ____                  _          
   / __ \________  ____  ____  ____ / __ \____ _/ /____    / __/___  ______  __(_)_______  
  / /_/ / ___/ _ \/ __ \/ __ \/ __ // /_/ / __ '/ __/ _ \  _\ \/ _ \/ ___/ / / / / ___/ _ \ 
 / ____/ /  /  __/ /_/ / /_/ / /_/ / _, _/ /_/ / /_/  __/ /___/  __/ /  / /_/ / / /__/  __/ 
/_/   /_/   \___/ .___/\____/\__, /_/ |_|\__,_/\__/\___/ /____/\___/_/   \__,_/_/\___/\___/  
               /_/         /____/                                                           

Rate Service - Production Ready Implementation
========================================================
`
	fmt.Println(banner)
	fmt.Printf("Version: %s\n", ServiceVersion)
	fmt.Printf("Go Version: %s\n", "1.25.1")
	fmt.Printf("Framework: Fiber v2\n")
	fmt.Printf("Architecture: Clean Architecture with Factory Pattern\n")
	fmt.Println("========================================================")
}

func loadConfiguration() *httpserver.ServerConfig {
	// For Phase 1, use default configuration
	// In later phases, this will load from environment variables, config files, etc.

	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = DefaultHost
	}

	config := &httpserver.ServerConfig{
		Port:              port,
		Host:              host,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		GracefulShutdown:  10 * time.Second,
		RequestBodyLimit:  10 * 1024 * 1024, // 10MB
		EnableCORS:        true,
		EnableHealthCheck: true,
		EnableMetrics:     true,
		EnableProfiling:   false,
		TrustedProxies:    []string{},
	}

	log.Printf("Configuration loaded - Host: %s, Port: %s", config.Host, config.Port)
	return config
}

// loadDatabaseConfig loads database configuration from environment variables
func loadDatabaseConfig() *database.PostgresConfig {
	config := database.DefaultPostgresConfig()

	// Load from environment variables if available
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		config.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		if port, err := strconv.Atoi(dbPort); err == nil {
			config.Port = port
		}
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		config.Username = dbUser
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		config.Password = dbPassword
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		config.Database = dbName
	}
	if sslMode := os.Getenv("DB_SSL_MODE"); sslMode != "" {
		config.SSLMode = sslMode
	}
	if logLevel := os.Getenv("DB_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	log.Printf("Database configuration loaded - Host: %s, Port: %d, Database: %s",
		config.Host, config.Port, config.Database)
	return config
}

// Dependencies holds all service dependencies
type Dependencies struct {
	RateService     interfaces.RateService
	RateCardService *unified_rate.RateCardService
	Logger          interfaces.Logger
	Metrics         interfaces.MetricsCollector
}

func initializeDependencies(dbConfig *database.PostgresConfig) (*Dependencies, func()) {
	log.Println("Initializing dependencies with real database and implementations")

	// Initialize real dependencies
	logger := NewMockLogger()                            // Using mock logger for now, can be replaced with real logger later
	metrics := NewMockMetrics()                          // Using mock metrics for now, can be replaced with real metrics later
	httpClient := utils.NewHTTPClient(30*time.Second, 2) // 30s timeout, 2 retries

	// Initialize database
	db := database.NewPostgresDB(dbConfig, logger)
	if err := db.Connect(); err != nil {
		// Fall back to mock repositories if database connection fails
		log.Printf("WARNING: Failed to connect to database: %v", err)
		log.Println("WARNING: Falling back to mock repositories for development")
		return initializeMockDependencies(logger, metrics, httpClient)
	}

	// Run database migrations
	// NOTE: Database migration is commented out. Run migrations manually if needed.
	// if err := db.Migrate(); err != nil {
	// 	log.Printf("WARNING: Database migration failed: %v", err)
	// 	db.Close()
	// 	return initializeMockDependencies(logger, metrics, httpClient)
	// }

	// Get GORM DB instance
	gormDB := db.GetDB().(*gorm.DB)

	// Initialize repositories with real database
	partnerRepo := repositoriesv1.NewPartnerRepository(gormDB)
	rateCardRepo := repositoriesv1.NewUnifiedRateCardRepository(gormDB)
	cacheManager := NewMockCacheManager() // Still using mock cache for now

	cleanup := func() {
		log.Println("Cleaning up database connection...")
		db.Close()
	}

	// Create rate factory
	rateFactory := factory.NewRateFactory(partnerRepo, logger, metrics)

	// Register unified real-time implementation creator that dispatches based on partner code
	realTimeCreator := func(partner *models.Partner) (interfaces.RateImplementation, error) {
		switch strings.ToLower(partner.Code) {
		case "dhl", "dhl_express":
			return dhl.NewService(logger, metrics, httpClient), nil
		case "india_post", "india_post_international":
			return indiapost.NewService(logger, metrics, httpClient), nil
		default:
			return nil, fmt.Errorf("unsupported real-time partner: %s", partner.Code)
		}
	}
	if err := rateFactory.RegisterImplementation(dtos.ProviderTypeRealTime, realTimeCreator); err != nil {
		log.Fatalf("Failed to register real-time implementation: %v", err)
	}

	// Register Unified Rate pre-defined implementation
	unifiedCreator := func(partner *models.Partner) (interfaces.RateImplementation, error) {
		return unified_rate.NewService(logger, metrics, httpClient, rateCardRepo), nil
	}
	if err := rateFactory.RegisterImplementation(dtos.ProviderTypePreDefined, unifiedCreator); err != nil {
		log.Fatalf("Failed to register Unified Rate implementation: %v", err)
	}

	// Create real rate service
	rateService := rateservice.NewRateService(
		rateFactory,
		partnerRepo,
		cacheManager,
		logger,
		metrics,
		httpClient,
	)

	// Create unified rate card service
	unifiedConfig := unified_rate.NewDefaultConfig()
	rateCardService := unified_rate.NewRateCardService(
		rateCardRepo,
		httpClient,
		logger,
		metrics,
		unifiedConfig,
	)

	logger.Info("All dependencies initialized successfully with real database")

	return &Dependencies{
		RateService:     rateService,
		RateCardService: rateCardService,
		Logger:          logger,
		Metrics:         metrics,
	}, cleanup
}

// initializeMockDependencies creates dependencies with mock repositories (fallback)
func initializeMockDependencies(logger MockLogger, metrics MockMetrics, httpClient interfaces.HTTPClient) (*Dependencies, func()) {
	log.Println("Initializing with mock repositories (fallback mode)")

	// Initialize mock repositories
	partnerRepo := NewMockPartnerRepository()
	rateCardRepo := NewMockUnifiedRateCardRepository()
	cacheManager := NewMockCacheManager()

	// Create rate factory
	rateFactory := factory.NewRateFactory(partnerRepo, logger, metrics)

	// Register unified real-time implementation creator that dispatches based on partner code
	realTimeCreator := func(partner *models.Partner) (interfaces.RateImplementation, error) {
		switch strings.ToLower(partner.Code) {
		case "dhl", "dhl_express":
			return dhl.NewService(logger, metrics, httpClient), nil
		case "india_post", "india_post_international":
			return indiapost.NewService(logger, metrics, httpClient), nil
		default:
			return nil, fmt.Errorf("unsupported real-time partner: %s", partner.Code)
		}
	}
	if err := rateFactory.RegisterImplementation(dtos.ProviderTypeRealTime, realTimeCreator); err != nil {
		log.Fatalf("Failed to register real-time implementation: %v", err)
	}

	// Register Unified Rate pre-defined implementation
	unifiedCreator := func(partner *models.Partner) (interfaces.RateImplementation, error) {
		return unified_rate.NewService(logger, metrics, httpClient, rateCardRepo), nil
	}
	if err := rateFactory.RegisterImplementation(dtos.ProviderTypePreDefined, unifiedCreator); err != nil {
		log.Fatalf("Failed to register Unified Rate implementation: %v", err)
	}

	// Create real rate service
	rateService := rateservice.NewRateService(
		rateFactory,
		partnerRepo,
		cacheManager,
		logger,
		metrics,
		httpClient,
	)

	// Create unified rate card service
	unifiedConfig := unified_rate.NewDefaultConfig()
	rateCardService := unified_rate.NewRateCardService(
		rateCardRepo,
		httpClient,
		logger,
		metrics,
		unifiedConfig,
	)

	logger.Info("Mock dependencies initialized successfully")

	return &Dependencies{
			RateService:     rateService,
			RateCardService: rateCardService,
			Logger:          logger,
			Metrics:         metrics,
		}, func() {
			log.Println("No cleanup needed for mock dependencies")
		}
}

// Mock implementations for Phase 1

// MockLogger provides a basic logger implementation for Phase 1
type MockLogger struct{}

func NewMockLogger() MockLogger {
	return MockLogger{}
}

func (l MockLogger) Debug(msg string, fields ...interface{}) {
	log.Printf("[DEBUG] %s %v", msg, fields)
}

func (l MockLogger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

func (l MockLogger) Warn(msg string, fields ...interface{}) {
	log.Printf("[WARN] %s %v", msg, fields)
}

func (l MockLogger) Error(msg string, fields ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, fields)
}

func (l MockLogger) Fatal(msg string, fields ...interface{}) {
	log.Fatalf("[FATAL] %s %v", msg, fields)
}

func (l MockLogger) With(fields ...interface{}) interfaces.Logger {
	return l
}

// MockMetrics provides a basic metrics implementation for Phase 1
type MockMetrics struct{}

func NewMockMetrics() MockMetrics {
	return MockMetrics{}
}

func (m MockMetrics) IncrementCounter(name string, tags map[string]string) {
	log.Printf("[METRICS] Counter incremented: %s, tags: %v", name, tags)
}

func (m MockMetrics) RecordTimer(name string, duration time.Duration, tags map[string]string) {
	log.Printf("[METRICS] Timer recorded: %s, duration: %v, tags: %v", name, duration, tags)
}

func (m MockMetrics) RecordGauge(name string, value float64, tags map[string]string) {
	log.Printf("[METRICS] Gauge recorded: %s, value: %f, tags: %v", name, value, tags)
}

func (m MockMetrics) RecordHistogram(name string, value float64, tags map[string]string) {
	log.Printf("[METRICS] Histogram recorded: %s, value: %f, tags: %v", name, value, tags)
}

// MockRateService provides a mock rate service for Phase 1
type MockRateService struct {
	logger     MockLogger
	metrics    MockMetrics
	httpClient interfaces.HTTPClient
}

func NewMockRateService(logger MockLogger, metrics MockMetrics, httpClient interfaces.HTTPClient) *MockRateService {
	return &MockRateService{
		logger:     logger,
		metrics:    metrics,
		httpClient: httpClient,
	}
}

func (m *MockRateService) CalculateRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	m.logger.Info("Mock: CalculateRates called")
	m.metrics.IncrementCounter("mock_rate_calculation", map[string]string{"status": "success"})

	// Return a mock response indicating this is Phase 1
	return &dtos.RateCalculationResponse{
		RequestID:   request.RequestID,
		Status:      "success",
		Message:     "Phase 1: Project structure is ready. Rate calculation will be implemented in Phase 2.",
		Quotes:      []dtos.RateQuote{},
		TotalQuotes: 0,
		CacheHit:    false,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"phase":      1,
			"next_phase": "Real-time rate providers implementation",
		},
	}, nil
}

func (m *MockRateService) CalculateRatesByImplementation(ctx context.Context, request *dtos.RateCalculationRequest, implementationID string) (*dtos.RateCalculationResponse, error) {
	m.logger.Info("Mock: CalculateRatesByImplementation called", "implementation_id", implementationID)
	return &dtos.RateCalculationResponse{
		RequestID:   request.RequestID,
		Status:      "not_implemented",
		Message:     "This endpoint will be implemented in Phase 2",
		Quotes:      []dtos.RateQuote{},
		TotalQuotes: 0,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"implementation_id": implementationID,
		},
	}, fmt.Errorf("%w: implementation-specific calculation", constants.ErrNotImplemented)
}

func (m *MockRateService) GetBestRate(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateQuote, error) {
	m.logger.Info("Mock: GetBestRate called")
	return nil, fmt.Errorf("%w: best rate selection", constants.ErrNotImplemented)
}

func (m *MockRateService) CompareRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateComparisonResponse, error) {
	m.logger.Info("Mock: CompareRates called")
	return &dtos.RateComparisonResponse{
		RequestID:    request.RequestID,
		Quotes:       []dtos.RateQuote{},
		ResponseTime: 0,
		Timestamp:    time.Now(),
		Metadata: map[string]interface{}{
			"status":  "not_implemented",
			"message": "Rate comparison will be implemented in Phase 2",
		},
	}, fmt.Errorf("%w: rate comparison", constants.ErrNotImplemented)
}

func (m *MockRateService) GetImplementationHealth(ctx context.Context) (*dtos.ProviderHealthResponse, error) {
	m.logger.Info("Mock: GetImplementationHealth called")
	return &dtos.ProviderHealthResponse{
		Status:             "healthy",
		TotalProviders:     0,
		HealthyProviders:   0,
		UnhealthyProviders: 0,
		Providers:          []dtos.ProviderHealthStatus{},
		CheckedAt:          time.Now(),
		ResponseTime:       0,
	}, nil
}

func (m *MockRateService) RefreshImplementations(ctx context.Context) error {
	m.logger.Info("Mock: RefreshImplementations called")
	m.metrics.IncrementCounter("mock_implementation_refresh", map[string]string{"status": "success"})
	return nil
}

func (m *MockRateService) GetQuotes(ctx context.Context, request *dtos.QuoteRequest, requestID string) (*dtos.QuoteResponse, error) {
	m.logger.Info("Mock: GetQuotes called")
	m.metrics.IncrementCounter("mock_get_quotes", map[string]string{"status": "success"})

	// Return a mock response
	return &dtos.QuoteResponse{
		Success: true,
		Message: "Mock rate quotes retrieved successfully.",
		Metadata: dtos.QuoteResponseMeta{
			RequestID:         requestID,
			ResponseTimeMs:    50,
			PartnersQueried:   1,
			PartnersSucceeded: 1,
			PartnersFailed:    0,
			TotalRatesFound:   1,
		},
		Data: dtos.QuoteResponseData{
			SuccessfulResponses: []dtos.SuccessfulPartnerResponse{
				{
					Partner: dtos.PartnerInfo{
						Code: "MOCK_PARTNER",
						Name: "Mock Partner",
					},
					Source: "pre_defined",
					AvailableRates: []dtos.Rate{
						{
							RateID:  "mock-rate-1",
							Service: "Mock Service",
							Price: dtos.Price{
								Currency: "INR",
								Amount:   100.00,
								Type:     "standard",
								Criteria: map[string]interface{}{
									"mock": "true",
								},
							},
							DeliveryDays: func() *int { d := 3; return &d }(),
						},
					},
					ResponseTimeMs: func() *int64 { t := int64(50); return &t }(),
				},
			},
			FailedResponses: []dtos.FailedPartnerResponse{},
		},
		Timestamp: time.Now(),
	}, nil
}

// MockPartnerRepository provides a basic partner repository for testing
type MockPartnerRepository struct{}

func NewMockPartnerRepository() *MockPartnerRepository {
	return &MockPartnerRepository{}
}

func (r *MockPartnerRepository) GetByID(ctx context.Context, id string) (*models.Partner, error) {
	// Return a mock DHL partner
	return &models.Partner{
		ID:   uuid.New(),
		Code: "dhl",
		Name: "DHL Express",
		Type: dtos.ProviderTypeRealTime,
		Config: map[string]interface{}{
			"base_url":       "https://express.api.dhl.com/mydhlapi/test/rates",
			"credentials":    "c2hyZWVtYXJ1dDhJTjpJITBwTV40c1IjNG5KJDF1",
			"account_number": "533748932",
			"timeout_ms":     30000,
			"environment":    "test",
		},
		IsActive:  true,
		Priority:  1,
		TimeoutMs: 30000,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (r *MockPartnerRepository) GetByCode(ctx context.Context, code string) (*models.Partner, error) {
	// Return appropriate partner based on code
	switch strings.ToLower(code) {
	case "dhl", "dhl_express":
		return &models.Partner{
			ID:   uuid.New(),
			Code: "dhl",
			Name: "DHL Express",
			Type: dtos.ProviderTypeRealTime,
			Config: map[string]interface{}{
				"base_url":       "https://express.api.dhl.com/mydhlapi/test/rates",
				"credentials":    "c2hyZWVtYXJ1dDhJTjpJITBwTV40c1IjNG5KJDF1",
				"account_number": "533748932",
				"timeout_ms":     30000,
				"environment":    "test",
			},
			IsActive:  true,
			Priority:  1,
			TimeoutMs: 30000,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	case "india_post", "india_post_international":
		return &models.Partner{
			ID:   uuid.New(),
			Code: "india_post_international",
			Name: "India Post International",
			Type: dtos.ProviderTypeRealTime,
			Config: map[string]interface{}{
				"base_url":        "https://test.cept.gov.in/beextcustomer/v1",
				"login_endpoint":  "/access/login",
				"tariff_endpoint": "/international-tariff/calculate",
				"username":        "9999999999",
				"password":        "Dop@1234",
				"timeout_ms":      30000,
				"environment":     "test",
				"token_expiry_sec": 900,
			},
			IsActive:  true,
			Priority:  2,
			TimeoutMs: 30000,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	default:
		return nil, fmt.Errorf("partner not found: %s", code)
	}
}

func (r *MockPartnerRepository) GetAll(ctx context.Context, filters map[string]interface{}) ([]*models.Partner, error) {
	return r.GetActivePartners(ctx)
}

func (r *MockPartnerRepository) GetPartnersByType(ctx context.Context, partnerType models.PartnerType) ([]*models.Partner, error) {
	return []*models.Partner{}, nil
}

func (r *MockPartnerRepository) GetActivePartners(ctx context.Context) ([]*models.Partner, error) {
	// Return mock active partners including DHL and India Post
	return []*models.Partner{
		{
			ID:   uuid.New(),
			Code: "dhl",
			Name: "DHL Express",
			Type: dtos.ProviderTypeRealTime,
			Config: map[string]interface{}{
				"base_url":       "https://express.api.dhl.com/mydhlapi/test/rates",
				"credentials":    "c2hyZWVtYXJ1dDhJTjpJITBwTV40c1IjNG5KJDF1",
				"account_number": "533748932",
				"timeout_ms":     30000,
				"environment":    "test",
			},
			IsActive:  true,
			Priority:  1,
			TimeoutMs: 30000,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:   uuid.New(),
			Code: "india_post_international",
			Name: "India Post International",
			Type: dtos.ProviderTypeRealTime,
			Config: map[string]interface{}{
				"base_url":        "https://test.cept.gov.in/beextcustomer/v1",
				"login_endpoint":  "/access/login",
				"tariff_endpoint": "/international-tariff/calculate",
				"username":        "9999999999",
				"password":        "Dop@1234",
				"timeout_ms":      30000,
				"environment":     "test",
				"token_expiry_sec": 900,
			},
			IsActive:  true,
			Priority:  2,
			TimeoutMs: 30000,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}, nil
}

func (r *MockPartnerRepository) Create(ctx context.Context, partner *models.Partner) error {
	return nil
}

func (r *MockPartnerRepository) Update(ctx context.Context, partner *models.Partner) error {
	return nil
}

func (r *MockPartnerRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (r *MockPartnerRepository) UpdateHealthStatus(ctx context.Context, id string, status string) error {
	return nil
}

// MockCacheManager provides a basic cache manager for testing
type MockCacheManager struct{}

func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{}
}

func (c *MockCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (c *MockCacheManager) Get(ctx context.Context, key string, dest interface{}) error {
	return errors.New("cache not found") // Using generic error since constants.ErrCacheNotFound doesn't exist
}

func (c *MockCacheManager) Delete(ctx context.Context, key string) error {
	return nil
}

func (c *MockCacheManager) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (c *MockCacheManager) Clear(ctx context.Context) error {
	return nil
}

func (c *MockCacheManager) GetStats(ctx context.Context) (*dtos.CacheStats, error) {
	return &dtos.CacheStats{
		Hits:     0,
		Misses:   0,
		HitRate:  0.0,
		MissRate: 0.0,
	}, nil
}

// MockUnifiedRateCardRepository provides a basic unified rate card repository for testing
type MockUnifiedRateCardRepository struct{}

func NewMockUnifiedRateCardRepository() *MockUnifiedRateCardRepository {
	return &MockUnifiedRateCardRepository{}
}

func (r *MockUnifiedRateCardRepository) Create(ctx context.Context, rateCard *models.UnifiedRateCard) error {
	return nil
}

func (r *MockUnifiedRateCardRepository) GetByID(ctx context.Context, id string) (*models.UnifiedRateCard, error) {
	return &models.UnifiedRateCard{
		ID:                uuid.New(),
		PartnerCode:       "unified",
		TenantID:          "test-tenant",
		APIKey:            "test-api-key",
		UnifiedRateCardID: "test-rate-card-id",
		IsActive:          true,
		IsDefault:         true,
		Name:              "Test Rate Card",
		ProductType:       "LOGISTICS",
		EffectiveFrom:     time.Now().Add(-24 * time.Hour),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil
}

func (r *MockUnifiedRateCardRepository) GetByPartnerCode(ctx context.Context, partnerCode string) ([]*models.UnifiedRateCard, error) {
	return []*models.UnifiedRateCard{}, nil
}

func (r *MockUnifiedRateCardRepository) GetActiveByPartnerCode(ctx context.Context, partnerCode string) ([]*models.UnifiedRateCard, error) {
	return []*models.UnifiedRateCard{}, nil
}

func (r *MockUnifiedRateCardRepository) GetDefaultByPartnerCode(ctx context.Context, partnerCode string) (*models.UnifiedRateCard, error) {
	return r.GetByID(ctx, partnerCode)
}

func (r *MockUnifiedRateCardRepository) GetByUnifiedRateCardID(ctx context.Context, unifiedRateCardID string) (*models.UnifiedRateCard, error) {
	return r.GetByID(ctx, unifiedRateCardID)
}

func (r *MockUnifiedRateCardRepository) GetAll(ctx context.Context, filter *models.UnifiedRateCardFilter) ([]*models.UnifiedRateCard, error) {
	return []*models.UnifiedRateCard{}, nil
}

func (r *MockUnifiedRateCardRepository) Update(ctx context.Context, rateCard *models.UnifiedRateCard) error {
	return nil
}

func (r *MockUnifiedRateCardRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (r *MockUnifiedRateCardRepository) SetDefault(ctx context.Context, id string, partnerCode string) error {
	return nil
}

func (r *MockUnifiedRateCardRepository) GetPartnerConfiguration(ctx context.Context, partnerCode string) (*models.UnifiedRateCard, error) {
	return r.GetByID(ctx, partnerCode)
}

func (r *MockUnifiedRateCardRepository) GetConfigByPartnerCode(ctx context.Context, partnerCode string) (tenantID, apiKey, unifiedRateCardID string, err error) {
	return "test-tenant", "test-api-key", "test-rate-card-id", nil
}

func (r *MockUnifiedRateCardRepository) ListPartnerCodes(ctx context.Context) ([]string, error) {
	return []string{"unified", "delhivery", "porter"}, nil
}
