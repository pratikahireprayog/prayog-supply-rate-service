package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	httpserver "github.com/prayog/prayog-rate-service/internal/infrastructure/api/http"
	constants "github.com/prayog/prayog-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-rate-service/internal/shared/interfaces/v1"
)

const (
	// Service information
	ServiceName    = "prayog-rate-service"
	ServiceVersion = "1.0.0"

	// Default configuration values
	DefaultPort = "8080"
	DefaultHost = "0.0.0.0"
)

func main() {
	// Print startup banner
	printBanner()

	// Load configuration (for now, use defaults)
	config := loadConfiguration()

	// Initialize dependencies (Phase 1: Mock implementations)
	dependencies := initializeDependencies()

	// Create and configure HTTP server
	server := httpserver.NewServer(
		config,
		dependencies.RateService,
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

Rate Card Service - Phase 1: Project Structure Complete
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

// Dependencies holds all service dependencies
type Dependencies struct {
	RateService interfaces.RateService
	Logger      interfaces.Logger
	Metrics     interfaces.MetricsCollector
}

func initializeDependencies() *Dependencies {
	log.Println("Initializing dependencies (Phase 1: Mock implementations)")

	// For Phase 1, we create mock implementations to demonstrate the structure
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	logger.Info("Dependencies initialized successfully")

	return &Dependencies{
		RateService: NewMockRateService(logger, metrics),
		Logger:      logger,
		Metrics:     metrics,
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
	logger  MockLogger
	metrics MockMetrics
}

func NewMockRateService(logger MockLogger, metrics MockMetrics) *MockRateService {
	return &MockRateService{
		logger:  logger,
		metrics: metrics,
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
			"next_phase": "Dynamic rate providers implementation",
		},
	}, nil
}

func (m *MockRateService) CalculateRatesByProvider(ctx context.Context, request *dtos.RateCalculationRequest, providerID string) (*dtos.RateCalculationResponse, error) {
	m.logger.Info("Mock: CalculateRatesByProvider called", "provider_id", providerID)
	return &dtos.RateCalculationResponse{
		RequestID:   request.RequestID,
		Status:      "not_implemented",
		Message:     "This endpoint will be implemented in Phase 2",
		Quotes:      []dtos.RateQuote{},
		TotalQuotes: 0,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"provider_id": providerID,
		},
	}, fmt.Errorf("%w: provider-specific calculation", constants.ErrNotImplemented)
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

func (m *MockRateService) GetProviderHealth(ctx context.Context) (*dtos.ProviderHealthResponse, error) {
	m.logger.Info("Mock: GetProviderHealth called")
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

func (m *MockRateService) RefreshProviders(ctx context.Context) error {
	m.logger.Info("Mock: RefreshProviders called")
	m.metrics.IncrementCounter("mock_provider_refresh", map[string]string{"status": "success"})
	return nil
}
