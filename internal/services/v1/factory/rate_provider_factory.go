package factory

import (
	"context"
	"fmt"
	"sync"
	"time"

	constants "github.com/prayog/prayog-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-rate-service/internal/shared/interfaces/v1"
)

// RateProviderFactory implements the factory pattern for creating rate providers
type RateProviderFactory struct {
	creators  map[dtos.ProviderType]interfaces.ProviderCreator
	instances map[string]interfaces.RateProvider
	mu        sync.RWMutex

	// Dependencies
	partnerRepo interfaces.PartnerRepository
	logger      interfaces.Logger
	metrics     interfaces.MetricsCollector
}

// NewRateProviderFactory creates a new rate provider factory
func NewRateProviderFactory(
	partnerRepo interfaces.PartnerRepository,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) interfaces.RateProviderFactory {
	return &RateProviderFactory{
		creators:    make(map[dtos.ProviderType]interfaces.ProviderCreator),
		instances:   make(map[string]interfaces.RateProvider),
		partnerRepo: partnerRepo,
		logger:      logger,
		metrics:     metrics,
	}
}

// CreateProvider creates a provider instance by type and partner ID
func (f *RateProviderFactory) CreateProvider(providerType dtos.ProviderType, partnerID string) (interfaces.RateProvider, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if instance already exists
	if provider, exists := f.instances[partnerID]; exists {
		f.logger.Debug("Returning existing provider instance", "partner_id", partnerID, "type", providerType)
		return provider, nil
	}

	// Get provider creator
	creator, exists := f.creators[providerType]
	if !exists {
		f.metrics.IncrementCounter("provider_creation_failed", map[string]string{
			"reason": "creator_not_found",
			"type":   string(providerType),
		})
		return nil, fmt.Errorf("%w: %s", constants.ErrProviderNotRegistered, providerType)
	}

	// Get partner details
	partner, err := f.partnerRepo.GetByID(context.Background(), partnerID)
	if err != nil {
		f.metrics.IncrementCounter("provider_creation_failed", map[string]string{
			"reason": "partner_not_found",
			"type":   string(providerType),
		})
		return nil, fmt.Errorf("failed to get partner %s: %w", partnerID, err)
	}

	// Validate partner type matches requested type
	if dtos.ProviderType(partner.Type) != providerType {
		f.metrics.IncrementCounter("provider_creation_failed", map[string]string{
			"reason": "type_mismatch",
			"type":   string(providerType),
		})
		return nil, fmt.Errorf("partner type mismatch: expected %s, got %s", providerType, partner.Type)
	}

	// Create provider instance
	startTime := time.Now()
	provider, err := creator(partner)
	if err != nil {
		f.metrics.IncrementCounter("provider_creation_failed", map[string]string{
			"reason": "creation_error",
			"type":   string(providerType),
		})
		return nil, fmt.Errorf("failed to create provider for partner %s: %w", partnerID, err)
	}

	// Initialize provider
	if err := provider.Initialize(partner.Config); err != nil {
		f.metrics.IncrementCounter("provider_creation_failed", map[string]string{
			"reason": "initialization_error",
			"type":   string(providerType),
		})
		return nil, fmt.Errorf("failed to initialize provider for partner %s: %w", partnerID, err)
	}

	// Store instance
	f.instances[partnerID] = provider

	// Record metrics
	f.metrics.IncrementCounter("provider_created", map[string]string{
		"type":       string(providerType),
		"partner_id": partnerID,
	})
	f.metrics.RecordTimer("provider_creation_time", time.Since(startTime), map[string]string{
		"type": string(providerType),
	})

	f.logger.Info("Created provider instance",
		"partner_id", partnerID,
		"type", providerType,
		"duration_ms", time.Since(startTime).Milliseconds())

	return provider, nil
}

// RegisterProvider registers a new provider implementation
func (f *RateProviderFactory) RegisterProvider(providerType dtos.ProviderType, creator interfaces.ProviderCreator) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if creator == nil {
		return fmt.Errorf("creator cannot be nil for provider type %s", providerType)
	}

	f.creators[providerType] = creator

	f.logger.Info("Registered provider creator", "type", providerType)
	f.metrics.IncrementCounter("provider_registered", map[string]string{
		"type": string(providerType),
	})

	return nil
}

// GetRegisteredProviders returns all registered provider types
func (f *RateProviderFactory) GetRegisteredProviders() []dtos.ProviderType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]dtos.ProviderType, 0, len(f.creators))
	for providerType := range f.creators {
		types = append(types, providerType)
	}

	return types
}

// IsProviderRegistered checks if a provider type is registered
func (f *RateProviderFactory) IsProviderRegistered(providerType dtos.ProviderType) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, exists := f.creators[providerType]
	return exists
}

// GetProviderInstance gets an existing provider instance
func (f *RateProviderFactory) GetProviderInstance(partnerID string) (interfaces.RateProvider, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	provider, exists := f.instances[partnerID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", constants.ErrProviderNotFound, partnerID)
	}

	return provider, nil
}

// RemoveProviderInstance removes a provider instance
func (f *RateProviderFactory) RemoveProviderInstance(partnerID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	provider, exists := f.instances[partnerID]
	if !exists {
		return fmt.Errorf("%w: %s", constants.ErrProviderNotFound, partnerID)
	}

	// Close provider gracefully
	if err := provider.Close(); err != nil {
		f.logger.Warn("Failed to close provider gracefully", "partner_id", partnerID, "error", err)
	}

	delete(f.instances, partnerID)

	f.logger.Info("Removed provider instance", "partner_id", partnerID)
	f.metrics.IncrementCounter("provider_removed", map[string]string{
		"partner_id": partnerID,
	})

	return nil
}

// GetAllInstances returns all active provider instances
func (f *RateProviderFactory) GetAllInstances() map[string]interfaces.RateProvider {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Return a copy to prevent external modification
	instances := make(map[string]interfaces.RateProvider, len(f.instances))
	for partnerID, provider := range f.instances {
		instances[partnerID] = provider
	}

	return instances
}

// HealthCheckAll performs health check on all provider instances
func (f *RateProviderFactory) HealthCheckAll(ctx context.Context) map[string]error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	results := make(map[string]error)

	// Create a context with timeout for health checks
	healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Check each provider concurrently
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	for partnerID, provider := range f.instances {
		wg.Add(1)
		go func(id string, p interfaces.RateProvider) {
			defer wg.Done()

			start := time.Now()
			err := p.IsHealthy(healthCtx)
			duration := time.Since(start)

			resultsMu.Lock()
			results[id] = err
			resultsMu.Unlock()

			// Record metrics
			status := "healthy"
			if err != nil {
				status = "unhealthy"
			}

			f.metrics.IncrementCounter("provider_health_check", map[string]string{
				"partner_id": id,
				"status":     status,
			})
			f.metrics.RecordTimer("provider_health_check_time", duration, map[string]string{
				"partner_id": id,
			})

			if err != nil {
				f.logger.Warn("Provider health check failed", "partner_id", id, "error", err, "duration_ms", duration.Milliseconds())
			} else {
				f.logger.Debug("Provider health check passed", "partner_id", id, "duration_ms", duration.Milliseconds())
			}
		}(partnerID, provider)
	}

	wg.Wait()

	return results
}

// Shutdown gracefully shuts down all provider instances
func (f *RateProviderFactory) Shutdown(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errors []error

	for partnerID, provider := range f.instances {
		if err := provider.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close provider %s: %w", partnerID, err))
		}
	}

	// Clear all instances
	f.instances = make(map[string]interfaces.RateProvider)

	f.logger.Info("Factory shutdown complete", "closed_providers", len(f.instances))

	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	return nil
}

// GetInstanceCount returns the number of active provider instances
func (f *RateProviderFactory) GetInstanceCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.instances)
}

// GetCreatorCount returns the number of registered provider creators
func (f *RateProviderFactory) GetCreatorCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.creators)
}
