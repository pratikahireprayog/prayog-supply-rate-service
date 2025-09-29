package v1

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/prayog/prayog-supply-rate-service/internal/services/v1/implementations/pre_defined/unified_rate"
	"github.com/prayog/prayog-supply-rate-service/internal/services/v1/implementations/real_time/dhl"
	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
	utils "github.com/prayog/prayog-supply-rate-service/internal/shared/utils/v1"
)

// RateService implements the main rate calculation service
type RateService struct {
	factory      interfaces.RateFactory
	partnerRepo  interfaces.PartnerRepository
	cacheManager interfaces.CacheManager
	logger       interfaces.Logger
	metrics      interfaces.MetricsCollector
	httpClient   interfaces.HTTPClient

	// Configuration
	maxConcurrency   int
	defaultTimeoutMs int
	cacheEnabled     bool
	cacheTTL         time.Duration
}

// NewRateService creates a new rate service instance
func NewRateService(
	factory interfaces.RateFactory,
	partnerRepo interfaces.PartnerRepository,
	cacheManager interfaces.CacheManager,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	httpClient interfaces.HTTPClient,
) interfaces.RateService {
	return &RateService{
		factory:          factory,
		partnerRepo:      partnerRepo,
		cacheManager:     cacheManager,
		logger:           logger,
		metrics:          metrics,
		httpClient:       httpClient,
		maxConcurrency:   10,    // Default max concurrent requests
		defaultTimeoutMs: 10000, // 10 seconds default timeout
		cacheEnabled:     true,
		cacheTTL:         15 * time.Minute, // 15 minutes cache TTL
	}
}

// CalculateRates calculates rates from all available implementations
func (s *RateService) CalculateRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	startTime := time.Now()

	// Validate request
	if err := request.Validate(); err != nil {
		s.metrics.IncrementCounter("rate_calculation_failed", map[string]string{
			"reason": "validation_error",
		})
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	s.logger.Info("Starting rate calculation",
		"request_id", request.RequestID,
		"origin", request.OriginCity,
		"destination", request.DestCity,
		"weight", request.Weight,
		"service_type", request.ServiceType)

	// Check cache first
	var cacheHit bool
	if s.cacheEnabled {
		if cachedResponse, err := s.getCachedRates(ctx, request); err == nil && cachedResponse != nil {
			cacheHit = true
			cachedResponse.CacheHit = true
			cachedResponse.ResponseTime = time.Since(startTime).Milliseconds()

			s.metrics.IncrementCounter("rate_calculation_cache_hit", map[string]string{
				"request_id": request.RequestID,
			})
			s.logger.Debug("Returning cached rates", "request_id", request.RequestID)
			return cachedResponse, nil
		}
	}

	// Get active partners
	partners, err := s.getActivePartners(ctx, request)
	if err != nil {
		s.metrics.IncrementCounter("rate_calculation_failed", map[string]string{
			"reason": "partners_fetch_error",
		})
		return nil, fmt.Errorf("failed to get active partners: %w", err)
	}

	if len(partners) == 0 {
		s.metrics.IncrementCounter("rate_calculation_failed", map[string]string{
			"reason": "no_partners_available",
		})
		return nil, constants.ErrNoRatesAvailable
	}

	// Calculate rates concurrently
	quotes, implementationErrors := s.calculateRatesConcurrently(ctx, request, partners)

	// Build response
	response := &dtos.RateCalculationResponse{
		RequestID:    request.RequestID,
		Status:       "success",
		Message:      constants.MessageRateCalculated,
		Quotes:       quotes,
		TotalQuotes:  len(quotes),
		ResponseTime: time.Since(startTime).Milliseconds(),
		CacheHit:     cacheHit,
		Timestamp:    time.Now(),
		Errors:       implementationErrors,
	}

	// Set best quote
	if len(quotes) > 0 {
		bestQuote := s.selectBestQuote(quotes)
		response.BestQuote = &bestQuote

		// Calculate price and delivery ranges
		response.PriceRange = s.calculatePriceRange(quotes)
		response.DeliveryRange = s.calculateDeliveryRange(quotes)
	} else {
		response.Status = "partial_failure"
		response.Message = constants.MessageRateNotFound
	}

	// Cache the response
	if s.cacheEnabled && len(quotes) > 0 {
		go s.cacheRates(context.Background(), request, response)
	}

	// Record metrics
	s.metrics.IncrementCounter("rate_calculation_completed", map[string]string{
		"status":       response.Status,
		"total_quotes": fmt.Sprintf("%d", response.TotalQuotes),
		"cache_hit":    fmt.Sprintf("%t", cacheHit),
	})
	s.metrics.RecordTimer("rate_calculation_time", time.Since(startTime), map[string]string{
		"status": response.Status,
	})

	s.logger.Info("Rate calculation completed",
		"request_id", request.RequestID,
		"total_quotes", response.TotalQuotes,
		"errors", len(implementationErrors),
		"duration_ms", response.ResponseTime)

	return response, nil
}

// CalculateRatesByProvider calculates rates from a specific implementation
func (s *RateService) CalculateRatesByImplementation(ctx context.Context, request *dtos.RateCalculationRequest, implementationID string) (*dtos.RateCalculationResponse, error) {
	startTime := time.Now()

	// Validate request
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// Get implementation instance
	implementation, err := s.factory.GetImplementationInstance(implementationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get implementation %s: %w", implementationID, err)
	}

	// Calculate rates
	implementationResponse, err := implementation.GetRates(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("implementation %s failed to calculate rates: %w", implementationID, err)
	}

	implementationResponse.ResponseTime = time.Since(startTime).Milliseconds()
	return implementationResponse, nil
}

// GetBestRate returns the best rate from all available implementations
func (s *RateService) GetBestRate(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateQuote, error) {
	response, err := s.CalculateRates(ctx, request)
	if err != nil {
		return nil, err
	}

	if response.BestQuote == nil {
		return nil, constants.ErrNoRatesAvailable
	}

	return response.BestQuote, nil
}

// CompareRates compares rates from multiple implementations
func (s *RateService) CompareRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateComparisonResponse, error) {
	// Convert to comparison request
	comparisonRequest := &dtos.RateComparisonRequest{
		RateCalculationRequest: *request,
		SortBy:                 "price",
		SortOrder:              "asc",
		Limit:                  50,
	}

	return s.compareRatesWithOptions(ctx, comparisonRequest)
}

// GetProviderHealth returns health status of all implementations
func (s *RateService) GetProviderHealth(ctx context.Context) (*dtos.ProviderHealthResponse, error) {
	startTime := time.Now()

	// Get all implementation instances
	instances := s.factory.GetAllInstances()

	// Perform health checks
	healthResults := s.factory.HealthCheckAll(ctx)

	// Build response
	implementationStatuses := make([]dtos.ProviderHealthStatus, 0, len(instances))
	healthyCount := 0

	for partnerID, implementation := range instances {
		err := healthResults[partnerID]
		isHealthy := err == nil
		if isHealthy {
			healthyCount++
		}

		status := dtos.ProviderHealthStatus{
			PartnerID:    partnerID,
			PartnerName:  implementation.GetImplementationName(),
			ProviderType: implementation.GetImplementationType(),
			Status:       s.getHealthStatusString(isHealthy),
			IsActive:     true,
			LastChecked:  time.Now(),
			ResponseTime: 0, // TODO: Implement response time tracking
		}

		if err != nil {
			status.ErrorMessage = err.Error()
		}

		implementationStatuses = append(implementationStatuses, status)
	}

	overallStatus := "healthy"
	if healthyCount == 0 {
		overallStatus = "critical"
	} else if healthyCount < len(instances) {
		overallStatus = "degraded"
	}

	response := &dtos.ProviderHealthResponse{
		Status:             overallStatus,
		TotalProviders:     len(instances),
		HealthyProviders:   healthyCount,
		UnhealthyProviders: len(instances) - healthyCount,
		Providers:          implementationStatuses,
		CheckedAt:          time.Now(),
		ResponseTime:       time.Since(startTime).Milliseconds(),
	}

	return response, nil
}

// RefreshProviders refreshes all implementation configurations
func (s *RateService) RefreshProviders(ctx context.Context) error {
	s.logger.Info("Starting implementation refresh")

	// Get all active partners
	partners, err := s.partnerRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active partners: %w", err)
	}

	var refreshErrors []error

	for _, partner := range partners {
		// Try to get existing implementation instance
		implementation, err := s.factory.GetImplementationInstance(partner.ID.String())
		if err != nil {
			// Implementation doesn't exist, create new one
			_, err := s.factory.CreateImplementation(dtos.ProviderType(partner.Type), partner.ID.String())
			if err != nil {
				refreshErrors = append(refreshErrors, fmt.Errorf("failed to create implementation for partner %s: %w", partner.ID, err))
				continue
			}
		} else {
			// Reinitialize existing implementation
			if err := implementation.Initialize(partner.Config); err != nil {
				refreshErrors = append(refreshErrors, fmt.Errorf("failed to reinitialize implementation for partner %s: %w", partner.ID, err))
			}
		}
	}

	s.logger.Info("Provider refresh completed",
		"total_partners", len(partners),
		"errors", len(refreshErrors))

	if len(refreshErrors) > 0 {
		return fmt.Errorf("implementation refresh completed with errors: %v", refreshErrors)
	}

	return nil
}

// Private helper methods

func (s *RateService) getActivePartners(ctx context.Context, request *dtos.RateCalculationRequest) ([]*models.Partner, error) {
	// If specific partner IDs are requested, get only those
	if len(request.PartnerIDs) > 0 {
		var partners []*models.Partner
		for _, partnerID := range request.PartnerIDs {
			partner, err := s.partnerRepo.GetByID(ctx, partnerID)
			if err != nil {
				s.logger.Warn("Failed to get partner", "partner_id", partnerID, "error", err)
				continue
			}
			if partner.IsHealthy() {
				partners = append(partners, partner)
			}
		}
		return partners, nil
	}

	// Get all active partners
	allPartners, err := s.partnerRepo.GetActive(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by implementation types if specified
	if len(request.ProviderTypes) > 0 {
		var filteredPartners []*models.Partner
		typeMap := make(map[string]bool)
		for _, t := range request.ProviderTypes {
			typeMap[t] = true
		}

		for _, partner := range allPartners {
			if typeMap[string(partner.Type)] && partner.IsHealthy() {
				filteredPartners = append(filteredPartners, partner)
			}
		}
		return filteredPartners, nil
	}

	// Return all healthy partners
	var healthyPartners []*models.Partner
	for _, partner := range allPartners {
		if partner.IsHealthy() {
			healthyPartners = append(healthyPartners, partner)
		}
	}

	return healthyPartners, nil
}

func (s *RateService) calculateRatesConcurrently(ctx context.Context, request *dtos.RateCalculationRequest, partners []*models.Partner) ([]dtos.RateQuote, []dtos.ProviderError) {
	// Create channels for results
	quoteChan := make(chan dtos.RateQuote, len(partners))
	errorChan := make(chan dtos.ProviderError, len(partners))

	// Create semaphore for concurrency control
	sem := make(chan struct{}, s.maxConcurrency)

	var wg sync.WaitGroup

	for _, partner := range partners {
		wg.Add(1)
		go func(p *models.Partner) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			s.calculateRateForPartner(ctx, request, p, quoteChan, errorChan)
		}(partner)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(quoteChan)
		close(errorChan)
	}()

	// Collect results
	var quotes []dtos.RateQuote
	var errors []dtos.ProviderError

	// Collect quotes
	for quote := range quoteChan {
		quotes = append(quotes, quote)
	}

	// Collect errors
	for err := range errorChan {
		errors = append(errors, err)
	}

	return quotes, errors
}

func (s *RateService) calculateRateForPartner(ctx context.Context, request *dtos.RateCalculationRequest, partner *models.Partner, quoteChan chan<- dtos.RateQuote, errorChan chan<- dtos.ProviderError) {
	startTime := time.Now()

	// Create timeout context
	timeoutMs := partner.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = s.defaultTimeoutMs
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Get or create implementation
	implementation, err := s.factory.CreateImplementation(dtos.ProviderType(partner.Type), partner.ID.String())
	if err != nil {
		errorChan <- dtos.ProviderError{
			PartnerID:    partner.ID.String(),
			PartnerName:  partner.Name,
			ErrorCode:    string(constants.CodeImplementationInitFail),
			ErrorMessage: err.Error(),
			Timestamp:    time.Now(),
		}
		return
	}

	// Calculate rates
	response, err := implementation.GetRates(timeoutCtx, request)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Warn("Provider rate calculation failed",
			"partner_id", partner.ID,
			"partner_name", partner.Name,
			"error", err,
			"duration_ms", duration.Milliseconds())

		errorChan <- dtos.ProviderError{
			PartnerID:    partner.ID.String(),
			PartnerName:  partner.Name,
			ErrorCode:    string(constants.CodeRateCalculationFail),
			ErrorMessage: err.Error(),
			Timestamp:    time.Now(),
		}

		s.metrics.IncrementCounter("implementation_rate_calculation_failed", map[string]string{
			"partner_id": partner.ID.String(),
		})
		return
	}

	// Send all quotes from the response
	for _, quote := range response.Quotes {
		quote.ResponseTimeMs = duration.Milliseconds()
		quoteChan <- quote
	}

	s.metrics.IncrementCounter("implementation_rate_calculation_success", map[string]string{
		"partner_id": partner.ID.String(),
	})
	s.metrics.RecordTimer("implementation_rate_calculation_time", duration, map[string]string{
		"partner_id": partner.ID.String(),
	})
}

func (s *RateService) selectBestQuote(quotes []dtos.RateQuote) dtos.RateQuote {
	if len(quotes) == 0 {
		return dtos.RateQuote{}
	}

	// Sort quotes by a combination of price, delivery time, and confidence
	sort.Slice(quotes, func(i, j int) bool {
		// Calculate score for each quote (lower is better)
		scoreI := s.calculateQuoteScore(quotes[i])
		scoreJ := s.calculateQuoteScore(quotes[j])
		return scoreI < scoreJ
	})

	bestQuote := quotes[0]
	bestQuote.IsRecommended = true
	bestQuote.RecommendReason = "Best overall value based on price, delivery time, and reliability"

	return bestQuote
}

func (s *RateService) calculateQuoteScore(quote dtos.RateQuote) float64 {
	// Normalize factors (weights can be adjusted based on business logic)
	priceWeight := 0.5
	timeWeight := 0.3
	confidenceWeight := 0.2

	// Normalize values (assuming reasonable ranges)
	normalizedPrice := quote.TotalPrice / 10000.0         // Assuming max price around 10000
	normalizedTime := float64(quote.EstimatedDays) / 30.0 // Assuming max 30 days
	normalizedConfidence := 1.0 - quote.Confidence        // Invert confidence (lower is better for score)

	return (priceWeight * normalizedPrice) + (timeWeight * normalizedTime) + (confidenceWeight * normalizedConfidence)
}

func (s *RateService) calculatePriceRange(quotes []dtos.RateQuote) *dtos.PriceRange {
	if len(quotes) == 0 {
		return nil
	}

	minPrice := quotes[0].TotalPrice
	maxPrice := quotes[0].TotalPrice
	totalPrice := 0.0
	currency := quotes[0].Currency

	for _, quote := range quotes {
		if quote.TotalPrice < minPrice {
			minPrice = quote.TotalPrice
		}
		if quote.TotalPrice > maxPrice {
			maxPrice = quote.TotalPrice
		}
		totalPrice += quote.TotalPrice
	}

	return &dtos.PriceRange{
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		AvgPrice: totalPrice / float64(len(quotes)),
		Currency: currency,
	}
}

func (s *RateService) calculateDeliveryRange(quotes []dtos.RateQuote) *dtos.TimeRange {
	if len(quotes) == 0 {
		return nil
	}

	minDays := quotes[0].EstimatedDays
	maxDays := quotes[0].EstimatedDays
	totalDays := 0

	for _, quote := range quotes {
		if quote.EstimatedDays < minDays {
			minDays = quote.EstimatedDays
		}
		if quote.EstimatedDays > maxDays {
			maxDays = quote.EstimatedDays
		}
		totalDays += quote.EstimatedDays
	}

	return &dtos.TimeRange{
		MinDays: minDays,
		MaxDays: maxDays,
		AvgDays: totalDays / len(quotes),
	}
}

func (s *RateService) getCachedRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
	if s.cacheManager == nil {
		return nil, constants.ErrCacheConnection
	}

	cacheKey := request.GetCacheKey()
	var cachedResponse dtos.RateCalculationResponse

	err := s.cacheManager.Get(ctx, cacheKey, &cachedResponse)
	if err != nil {
		return nil, err
	}

	return &cachedResponse, nil
}

func (s *RateService) cacheRates(ctx context.Context, request *dtos.RateCalculationRequest, response *dtos.RateCalculationResponse) {
	if s.cacheManager == nil {
		return
	}

	cacheKey := request.GetCacheKey()
	if err := s.cacheManager.Set(ctx, cacheKey, response, s.cacheTTL); err != nil {
		s.logger.Warn("Failed to cache rates", "cache_key", cacheKey, "error", err)
	}
}

func (s *RateService) compareRatesWithOptions(ctx context.Context, request *dtos.RateComparisonRequest) (*dtos.RateComparisonResponse, error) {
	// Calculate rates first
	rateResponse, err := s.CalculateRates(ctx, &request.RateCalculationRequest)
	if err != nil {
		return nil, err
	}

	quotes := rateResponse.Quotes

	// Apply filters
	if len(request.Filters) > 0 {
		quotes = s.applyFilters(quotes, request.Filters)
	}

	// Sort quotes
	s.sortQuotes(quotes, request.SortBy, request.SortOrder)

	// Apply limit
	if request.Limit > 0 && len(quotes) > request.Limit {
		quotes = quotes[:request.Limit]
	}

	// Build comparison
	comparison := s.buildComparison(quotes)
	summary := s.buildComparisonSummary(quotes)

	return &dtos.RateComparisonResponse{
		RequestID:    request.RequestID,
		Comparison:   comparison,
		Quotes:       quotes,
		Summary:      summary,
		ResponseTime: rateResponse.ResponseTime,
		Timestamp:    time.Now(),
	}, nil
}

func (s *RateService) applyFilters(quotes []dtos.RateQuote, filters []dtos.Filter) []dtos.RateQuote {
	var filteredQuotes []dtos.RateQuote

	for _, quote := range quotes {
		if s.matchesFilters(quote, filters) {
			filteredQuotes = append(filteredQuotes, quote)
		}
	}

	return filteredQuotes
}

func (s *RateService) matchesFilters(quote dtos.RateQuote, filters []dtos.Filter) bool {
	for _, filter := range filters {
		if !s.matchesFilter(quote, filter) {
			return false
		}
	}
	return true
}

func (s *RateService) matchesFilter(quote dtos.RateQuote, filter dtos.Filter) bool {
	// Simplified filter implementation
	// In a real implementation, you would use reflection or a more sophisticated approach
	switch filter.Field {
	case "total_price":
		return s.compareValues(quote.TotalPrice, filter.Operator, filter.Value)
	case "estimated_days":
		return s.compareValues(float64(quote.EstimatedDays), filter.Operator, filter.Value)
	case "service_type":
		return s.compareValues(quote.ServiceType, filter.Operator, filter.Value)
	case "partner_name":
		return s.compareValues(quote.PartnerName, filter.Operator, filter.Value)
	default:
		return true // Unknown field, pass through
	}
}

func (s *RateService) compareValues(actual interface{}, operator string, expected interface{}) bool {
	// Simplified comparison - in production, use a proper comparison library
	switch operator {
	case "eq":
		return actual == expected
	case "neq":
		return actual != expected
	// Add more operators as needed
	default:
		return true
	}
}

func (s *RateService) sortQuotes(quotes []dtos.RateQuote, sortBy, sortOrder string) {
	sort.Slice(quotes, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "price":
			less = quotes[i].TotalPrice < quotes[j].TotalPrice
		case "delivery_time":
			less = quotes[i].EstimatedDays < quotes[j].EstimatedDays
		case "confidence":
			less = quotes[i].Confidence > quotes[j].Confidence // Higher confidence is better
		default:
			less = quotes[i].TotalPrice < quotes[j].TotalPrice
		}

		if sortOrder == "desc" {
			less = !less
		}

		return less
	})
}

func (s *RateService) buildComparison(quotes []dtos.RateQuote) dtos.RateComparison {
	if len(quotes) == 0 {
		return dtos.RateComparison{}
	}

	// Find best in each category
	bestPrice := quotes[0]
	fastestDelivery := quotes[0]
	mostReliable := quotes[0]

	for _, quote := range quotes {
		if quote.TotalPrice < bestPrice.TotalPrice {
			bestPrice = quote
		}
		if quote.EstimatedDays < fastestDelivery.EstimatedDays {
			fastestDelivery = quote
		}
		if quote.Confidence > mostReliable.Confidence {
			mostReliable = quote
		}
	}

	// Select overall recommended (could be same as best quote from main service)
	recommended := s.selectBestQuote(quotes)

	return dtos.RateComparison{
		BestPrice:       &bestPrice,
		FastestDelivery: &fastestDelivery,
		MostReliable:    &mostReliable,
		Recommended:     &recommended,
	}
}

func (s *RateService) buildComparisonSummary(quotes []dtos.RateQuote) dtos.ComparisonSummary {
	if len(quotes) == 0 {
		return dtos.ComparisonSummary{}
	}

	// Calculate aggregated statistics
	priceRange := s.calculatePriceRange(quotes)
	deliveryRange := s.calculateDeliveryRange(quotes)

	// Get unique implementation types and service types
	implementationTypesMap := make(map[string]bool)
	serviceTypesMap := make(map[string]bool)
	totalConfidence := 0.0

	for _, quote := range quotes {
		implementationTypesMap[string(quote.ProviderType)] = true
		serviceTypesMap[quote.ServiceType] = true
		totalConfidence += quote.Confidence
	}

	var implementationTypes []string
	for pt := range implementationTypesMap {
		implementationTypes = append(implementationTypes, pt)
	}

	var serviceTypes []string
	for st := range serviceTypesMap {
		serviceTypes = append(serviceTypes, st)
	}

	return dtos.ComparisonSummary{
		TotalQuotes:       len(quotes),
		PriceRange:        *priceRange,
		DeliveryRange:     *deliveryRange,
		ProviderTypes:     implementationTypes,
		ServiceTypes:      serviceTypes,
		AverageConfidence: totalConfidence / float64(len(quotes)),
	}
}

func (s *RateService) getHealthStatusString(isHealthy bool) string {
	if isHealthy {
		return "healthy"
	}
	return "unhealthy"
}

// GetQuotes retrieves quotes from multiple partners with standardized response structure
func (s *RateService) GetQuotes(ctx context.Context, request *dtos.QuoteRequest, requestID string) (*dtos.QuoteResponse, error) {
	startTime := time.Now()

	var successfulResponses []dtos.SuccessfulPartnerResponse
	var failedResponses []dtos.FailedPartnerResponse

	// Process each partner
	for _, partner := range request.Partners {
		partnerStartTime := time.Now()

		// Normalize partner code
		normalizedCode := utils.NormalizePartnerCode(partner.Code)
		partnerName := s.getPartnerName(normalizedCode)

		partnerInfo := dtos.PartnerInfo{
			Code: normalizedCode,
			Name: partnerName,
		}

		// Use partner.ID if provided, otherwise use normalized code
		partnerIdentifier := partner.ID
		if partnerIdentifier == "" {
			partnerIdentifier = normalizedCode
		}

		// Get implementation for this partner
		implementation, err := s.factory.GetImplementationInstance(partnerIdentifier)
		if err != nil {
			// Try to create implementation based on partner code
			implementation, err = s.createImplementationByCode(normalizedCode)
			if err != nil {
				failedResponses = append(failedResponses, dtos.FailedPartnerResponse{
					Partner: partnerInfo,
					Source:  "unknown",
					Error: dtos.RateError{
						Code:    "PARTNER_NOT_FOUND",
						Message: "Partner implementation not found",
						Details: fmt.Sprintf("No implementation found for partner code: %s", normalizedCode),
					},
				})
				continue
			}
		}

		// Determine source type
		source := "real_time"
		if implementation.GetImplementationType() == dtos.ProviderTypePreDefined {
			source = "pre_defined"
		}

		// Convert request to internal format
		internalRequest := s.convertQuoteRequestToRateRequest(request)

		// Get rates from implementation
		rateResponse, err := implementation.GetRates(ctx, internalRequest)
		responseTimeMs := time.Since(partnerStartTime).Milliseconds()

		if err != nil {
			failedResponses = append(failedResponses, dtos.FailedPartnerResponse{
				Partner: partnerInfo,
				Source:  source,
				Error: dtos.RateError{
					Code:    "RATE_FETCH_FAILED",
					Message: "Failed to fetch rates from partner",
					Details: err.Error(),
				},
			})
		} else {
			// Convert response to simplified format
			availableRates := s.convertRateResponseToRates(rateResponse)
			successfulResponses = append(successfulResponses, dtos.SuccessfulPartnerResponse{
				Partner:        partnerInfo,
				Source:         source,
				AvailableRates: availableRates,
				ResponseTimeMs: &responseTimeMs,
			})
		}
	}

	// Calculate metadata
	totalResponseTime := time.Since(startTime).Milliseconds()
	totalRatesFound := 0
	for _, successful := range successfulResponses {
		totalRatesFound += len(successful.AvailableRates)
	}

	// Build standardized response
	response := &dtos.QuoteResponse{
		Success: true,
		Message: "Rate quotes retrieved successfully.",
		Metadata: dtos.QuoteResponseMeta{
			RequestID:         requestID,
			ResponseTimeMs:    totalResponseTime,
			PartnersQueried:   len(request.Partners),
			PartnersSucceeded: len(successfulResponses),
			PartnersFailed:    len(failedResponses),
			TotalRatesFound:   totalRatesFound,
		},
		Data: dtos.QuoteResponseData{
			SuccessfulResponses: successfulResponses,
			FailedResponses:     failedResponses,
		},
		Timestamp: time.Now(),
	}

	return response, nil
}

// Helper method to convert quote request to internal rate request
func (s *RateService) convertQuoteRequestToRateRequest(req *dtos.QuoteRequest) *dtos.RateCalculationRequest {
	// Extract metadata values with defaults
	currency := "INR"
	serviceType := "standard"

	if req.Metadata != nil {
		if curr, ok := req.Metadata["currency"].(string); ok {
			currency = curr
		}
		if svc, ok := req.Metadata["service_type"].(string); ok {
			serviceType = svc
		}
	}

	// Calculate total weight (simplified - sum all packages)
	totalWeight := 0.0
	for _, pkg := range req.Packages {
		totalWeight += pkg.Weight.Value
	}

	return &dtos.RateCalculationRequest{
		RequestID:    uuid.New().String(),
		CustomerID:   "default", // Default customer ID
		OriginCity:   req.SourceLocation.PostalCode,
		DestCity:     req.DestinationLocation.PostalCode,
		Weight:       totalWeight,
		Distance:     0, // Will be calculated if needed
		ServiceType:  serviceType,
		Currency:     currency,
		PickupDate:   time.Now(),
		DeliveryDate: time.Now().AddDate(0, 0, 1), // Default next day
		Priority:     "normal",
		Source:       "api",
	}
}

// Helper method to convert internal rate response to simplified rates
func (s *RateService) convertRateResponseToRates(resp *dtos.RateCalculationResponse) []dtos.Rate {
	rates := make([]dtos.Rate, 0, len(resp.Quotes))

	for _, quote := range resp.Quotes {
		rate := dtos.Rate{
			RateID:  quote.QuoteID,
			Service: quote.PartnerName,
			Price: dtos.Price{
				Currency: quote.Currency,
				Amount:   quote.TotalPrice,
				Type:     "standard",
			},
		}

		if quote.EstimatedDays > 0 {
			rate.DeliveryDays = &quote.EstimatedDays
		}

		rates = append(rates, rate)
	}

	return rates
}

// Helper method to calculate quote summary
func (s *RateService) calculateQuoteSummary(partnerRates []dtos.PartnerRateResult) dtos.QuoteSummary {
	summary := dtos.QuoteSummary{
		TotalPartners: len(partnerRates),
	}

	for _, partnerRate := range partnerRates {
		if partnerRate.Success {
			summary.SuccessfulPartners++
			summary.TotalRatesFound += len(partnerRate.AvailableRates)
		}
	}

	return summary
}

// getPartnerName returns the display name for a partner code
func (s *RateService) getPartnerName(partnerCode string) string {
	partnerNames := map[string]string{
		"dhl":       "DHL Express",
		"fedex":     "FedEx",
		"ups":       "UPS",
		"blue_dart": "Blue Dart",
		"delhivery": "Delhivery",
		"porter":    "Porter",
		"unified":   "Unified Rate",
		"prayog":    "Prayog Unified Rate",
		"dunzo":     "Dunzo",
		"aramex":    "Aramex",
	}

	if name, exists := partnerNames[partnerCode]; exists {
		return name
	}

	// Return a capitalized version of the code if not found
	return strings.Title(strings.Replace(partnerCode, "_", " ", -1))
}

// createImplementationByCode creates implementation based on partner code
func (s *RateService) createImplementationByCode(normalizedCode string) (interfaces.RateImplementation, error) {
	switch normalizedCode {
	case "dhl":
		s.logger.Info("Creating DHL service", "partner_code", normalizedCode)
		return dhl.NewService(s.logger, s.metrics, s.httpClient), nil
	case "fedex":
		return nil, fmt.Errorf("FedEx implementation not yet available")
	case "ups":
		return nil, fmt.Errorf("UPS implementation not yet available")
	case "blue_dart":
		return nil, fmt.Errorf("Blue Dart implementation not yet available")
	case "delhivery":
		s.logger.Info("Redirecting Delhivery to Unified Rate service", "partner_code", normalizedCode)
		return unified_rate.NewService(s.logger, s.metrics, s.httpClient), nil
	case "porter":
		s.logger.Info("Redirecting Porter to Unified Rate service", "partner_code", normalizedCode)
		return unified_rate.NewService(s.logger, s.metrics, s.httpClient), nil
	case "unified":
		s.logger.Info("Creating Unified Rate service", "partner_code", normalizedCode)
		return unified_rate.NewService(s.logger, s.metrics, s.httpClient), nil
	case "prayog":
		s.logger.Info("Creating Unified Rate service", "partner_code", normalizedCode)
		return unified_rate.NewService(s.logger, s.metrics, s.httpClient), nil
	case "dunzo":
		return nil, fmt.Errorf("Dunzo implementation not yet available")
	case "aramex":
		return nil, fmt.Errorf("Aramex implementation not yet available")
	default:
		return nil, fmt.Errorf("unknown partner code: %s", normalizedCode)
	}
}

// GetImplementationHealth returns health status of all implementations
func (s *RateService) GetImplementationHealth(ctx context.Context) (*dtos.ProviderHealthResponse, error) {
	startTime := time.Now()

	// Get all implementation instances
	instances := s.factory.GetAllInstances()

	// Perform health checks
	healthResults := s.factory.HealthCheckAll(ctx)

	// Build response
	implementationStatuses := make([]dtos.ProviderHealthStatus, 0, len(instances))
	healthyCount := 0

	for partnerID, implementation := range instances {
		err := healthResults[partnerID]
		isHealthy := err == nil
		if isHealthy {
			healthyCount++
		}

		status := dtos.ProviderHealthStatus{
			PartnerID:    partnerID,
			PartnerName:  implementation.GetImplementationName(),
			ProviderType: implementation.GetImplementationType(),
			Status:       s.getHealthStatusString(isHealthy),
			IsActive:     true,
			LastChecked:  time.Now(),
			ResponseTime: 0, // TODO: Implement response time tracking
		}

		if err != nil {
			status.ErrorMessage = err.Error()
		}

		implementationStatuses = append(implementationStatuses, status)
	}

	overallStatus := "healthy"
	if healthyCount == 0 {
		overallStatus = "critical"
	} else if healthyCount < len(instances) {
		overallStatus = "degraded"
	}

	response := &dtos.ProviderHealthResponse{
		Status:             overallStatus,
		TotalProviders:     len(instances),
		HealthyProviders:   healthyCount,
		UnhealthyProviders: len(instances) - healthyCount,
		Providers:          implementationStatuses,
		CheckedAt:          time.Now(),
		ResponseTime:       time.Since(startTime).Milliseconds(),
	}

	return response, nil
}

// RefreshImplementations refreshes all implementation configurations
func (s *RateService) RefreshImplementations(ctx context.Context) error {
	s.logger.Info("Starting implementation refresh")

	// Get all active partners
	partners, err := s.partnerRepo.GetActive(ctx)
	if err != nil {
		s.logger.Error("Failed to get active partners", "error", err)
		return fmt.Errorf("failed to get active partners: %w", err)
	}

	refreshErrors := make([]string, 0)
	successCount := 0

	for _, partner := range partners {
		s.logger.Info("Refreshing implementation", "partner_id", partner.ID, "partner_name", partner.Name)

		// Remove existing implementation instance
		err := s.factory.RemoveImplementationInstance(partner.ID.String())
		if err != nil {
			s.logger.Warn("Failed to remove existing implementation", "partner_id", partner.ID, "error", err)
		}

		// Create new implementation instance
		_, err = s.factory.CreateImplementation(partner.Type, partner.ID.String())
		if err != nil {
			errMsg := fmt.Sprintf("failed to create implementation for partner %s: %v", partner.ID, err)
			refreshErrors = append(refreshErrors, errMsg)
			s.logger.Error("Failed to create implementation", "partner_id", partner.ID, "error", err)
			continue
		}

		successCount++
		s.logger.Info("Implementation refreshed successfully", "partner_id", partner.ID)
	}

	if len(refreshErrors) > 0 {
		s.logger.Warn("Implementation refresh completed with errors",
			"success_count", successCount,
			"error_count", len(refreshErrors),
			"errors", refreshErrors)
		return fmt.Errorf("refresh completed with %d errors: %v", len(refreshErrors), refreshErrors)
	}

	s.logger.Info("Implementation refresh completed successfully",
		"success_count", successCount,
		"total_partners", len(partners))

	return nil
}
