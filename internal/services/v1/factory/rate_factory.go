package factory

import (
	"context"
	"fmt"
	"sync"
	"time"

	constants "github.com/prayog/prayog-supply-rate-service/internal/shared/constants/v1"
	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

// RateFactory implements the factory pattern for creating rate implementations
type RateFactory struct {
	creators  map[dtos.ProviderType]interfaces.ImplementationCreator
	instances map[string]interfaces.RateImplementation
	mu        sync.RWMutex

	// Dependencies
	partnerRepo interfaces.PartnerRepository
	logger      interfaces.Logger
	metrics     interfaces.MetricsCollector
}

// NewRateFactory creates a new rate implementation factory
func NewRateFactory(
	partnerRepo interfaces.PartnerRepository,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) interfaces.RateFactory {
	return &RateFactory{
		creators:    make(map[dtos.ProviderType]interfaces.ImplementationCreator),
		instances:   make(map[string]interfaces.RateImplementation),
		partnerRepo: partnerRepo,
		logger:      logger,
		metrics:     metrics,
	}
}

// CreateImplementation creates an implementation instance by type and partner ID
func (f *RateFactory) CreateImplementation(implementationType dtos.ProviderType, partnerID string) (interfaces.RateImplementation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if instance already exists
	if implementation, exists := f.instances[partnerID]; exists {
		f.logger.Debug("Returning existing implementation instance", "partner_id", partnerID, "type", implementationType)
		return implementation, nil
	}

	// Get implementation creator
	creator, exists := f.creators[implementationType]
	if !exists {
		f.metrics.IncrementCounter("implementation_creation_failed", map[string]string{
			"reason": "creator_not_found",
			"type":   string(implementationType),
		})
		return nil, fmt.Errorf("%w: %s", constants.ErrImplementationNotRegistered, implementationType)
	}

	// Get partner details
	partner, err := f.partnerRepo.GetByID(context.Background(), partnerID)
	if err != nil {
		f.metrics.IncrementCounter("implementation_creation_failed", map[string]string{
			"reason": "partner_not_found",
			"type":   string(implementationType),
		})
		return nil, fmt.Errorf("failed to get partner %s: %w", partnerID, err)
	}

	// Validate partner type matches requested type
	if dtos.ProviderType(partner.Type) != implementationType {
		f.metrics.IncrementCounter("implementation_creation_failed", map[string]string{
			"reason": "type_mismatch",
			"type":   string(implementationType),
		})
		return nil, fmt.Errorf("partner type mismatch: expected %s, got %s", implementationType, partner.Type)
	}

	// Create implementation instance
	startTime := time.Now()
	implementation, err := creator(partner)
	if err != nil {
		f.metrics.IncrementCounter("implementation_creation_failed", map[string]string{
			"reason": "creation_error",
			"type":   string(implementationType),
		})
		return nil, fmt.Errorf("failed to create implementation for partner %s: %w", partnerID, err)
	}

	// Initialize implementation
	if err := implementation.Initialize(partner.Config); err != nil {
		f.metrics.IncrementCounter("implementation_creation_failed", map[string]string{
			"reason": "initialization_error",
			"type":   string(implementationType),
		})
		return nil, fmt.Errorf("failed to initialize implementation for partner %s: %w", partnerID, err)
	}

	// Store instance
	f.instances[partnerID] = implementation

	// Record metrics
	f.metrics.IncrementCounter("implementation_created", map[string]string{
		"type":       string(implementationType),
		"partner_id": partnerID,
	})
	f.metrics.RecordTimer("implementation_creation_time", time.Since(startTime), map[string]string{
		"type": string(implementationType),
	})

	f.logger.Info("Created implementation instance",
		"partner_id", partnerID,
		"type", implementationType,
		"duration_ms", time.Since(startTime).Milliseconds())

	return implementation, nil
}

// RegisterImplementation registers a new implementation implementation
func (f *RateFactory) RegisterImplementation(implementationType dtos.ProviderType, creator interfaces.ImplementationCreator) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if creator == nil {
		return fmt.Errorf("creator cannot be nil for implementation type %s", implementationType)
	}

	f.creators[implementationType] = creator

	f.logger.Info("Registered implementation creator", "type", implementationType)
	f.metrics.IncrementCounter("implementation_registered", map[string]string{
		"type": string(implementationType),
	})

	return nil
}

// GetRegisteredImplementations returns all registered implementation types
func (f *RateFactory) GetRegisteredImplementations() []dtos.ProviderType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]dtos.ProviderType, 0, len(f.creators))
	for implementationType := range f.creators {
		types = append(types, implementationType)
	}

	return types
}

// IsImplementationRegistered checks if a implementation type is registered
func (f *RateFactory) IsImplementationRegistered(implementationType dtos.ProviderType) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, exists := f.creators[implementationType]
	return exists
}

// GetImplementationInstance gets an existing implementation instance
func (f *RateFactory) GetImplementationInstance(partnerID string) (interfaces.RateImplementation, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	implementation, exists := f.instances[partnerID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", constants.ErrImplementationNotFound, partnerID)
	}

	return implementation, nil
}

// RemoveImplementationInstance removes a implementation instance
func (f *RateFactory) RemoveImplementationInstance(partnerID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	implementation, exists := f.instances[partnerID]
	if !exists {
		return fmt.Errorf("%w: %s", constants.ErrImplementationNotFound, partnerID)
	}

	// Close implementation gracefully
	if err := implementation.Close(); err != nil {
		f.logger.Warn("Failed to close implementation gracefully", "partner_id", partnerID, "error", err)
	}

	delete(f.instances, partnerID)

	f.logger.Info("Removed implementation instance", "partner_id", partnerID)
	f.metrics.IncrementCounter("implementation_removed", map[string]string{
		"partner_id": partnerID,
	})

	return nil
}

// GetAllInstances returns all active implementation instances
func (f *RateFactory) GetAllInstances() map[string]interfaces.RateImplementation {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Return a copy to prevent external modification
	instances := make(map[string]interfaces.RateImplementation, len(f.instances))
	for partnerID, implementation := range f.instances {
		instances[partnerID] = implementation
	}

	return instances
}

// HealthCheckAll performs health check on all implementation instances
func (f *RateFactory) HealthCheckAll(ctx context.Context) map[string]error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	results := make(map[string]error)

	// Create a context with timeout for health checks
	healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Check each implementation concurrently
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	for partnerID, implementation := range f.instances {
		wg.Add(1)
		go func(id string, p interfaces.RateImplementation) {
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

			f.metrics.IncrementCounter("implementation_health_check", map[string]string{
				"partner_id": id,
				"status":     status,
			})
			f.metrics.RecordTimer("implementation_health_check_time", duration, map[string]string{
				"partner_id": id,
			})

			if err != nil {
				f.logger.Warn("Provider health check failed", "partner_id", id, "error", err, "duration_ms", duration.Milliseconds())
			} else {
				f.logger.Debug("Provider health check passed", "partner_id", id, "duration_ms", duration.Milliseconds())
			}
		}(partnerID, implementation)
	}

	wg.Wait()

	return results
}

// Shutdown gracefully shuts down all implementation instances
func (f *RateFactory) Shutdown(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errors []error

	for partnerID, implementation := range f.instances {
		if err := implementation.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close implementation %s: %w", partnerID, err))
		}
	}

	// Clear all instances
	f.instances = make(map[string]interfaces.RateImplementation)

	f.logger.Info("Factory shutdown complete", "closed_implementations", len(f.instances))

	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	return nil
}

// GetInstanceCount returns the number of active implementation instances
func (f *RateFactory) GetInstanceCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.instances)
}

// GetCreatorCount returns the number of registered implementation creators
func (f *RateFactory) GetCreatorCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.creators)
}
