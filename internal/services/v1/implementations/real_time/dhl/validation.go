package dhl

import (
	"fmt"
	"strings"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
)

// validateRequest validates the rate calculation request for DHL
func (s *Service) validateRequest(req *dtos.RateCalculationRequest) error {
	// Validate origin
	if req.OriginCity == "" {
		return fmt.Errorf("origin_city is required")
	}
	if req.OriginCountry == "" {
		return fmt.Errorf("origin_country is required")
	}
	if len(req.OriginCountry) != 2 {
		return fmt.Errorf("origin_country must be ISO 3166-1 alpha-2 code (2 characters), got: %s", req.OriginCountry)
	}

	// Validate destination
	if req.DestCity == "" {
		return fmt.Errorf("dest_city is required")
	}
	if req.DestCountry == "" {
		return fmt.Errorf("dest_country is required")
	}
	if len(req.DestCountry) != 2 {
		return fmt.Errorf("dest_country must be ISO 3166-1 alpha-2 code (2 characters), got: %s", req.DestCountry)
	}

	// Validate weight
	if req.Weight <= 0 {
		return fmt.Errorf("weight must be positive, got: %.2f", req.Weight)
	}
	if req.Weight > 10000 {
		return fmt.Errorf("weight exceeds maximum allowed (10000 kg), got: %.2f", req.Weight)
	}

	// Validate packages
	if len(req.Packages) == 0 {
		return fmt.Errorf("at least one package is required")
	}

	for i, pkg := range req.Packages {
		if pkg.Weight <= 0 {
			return fmt.Errorf("package %d: weight must be positive, got: %.2f", i+1, pkg.Weight)
		}
		if pkg.Length <= 0 || pkg.Width <= 0 || pkg.Height <= 0 {
			return fmt.Errorf("package %d: dimensions must be positive (length: %.2f, width: %.2f, height: %.2f)",
				i+1, pkg.Length, pkg.Width, pkg.Height)
		}

		// Validate dimension units
		dimUnit := strings.ToLower(pkg.DimUnit)
		if dimUnit != "cm" && dimUnit != "in" && dimUnit != "mm" {
			return fmt.Errorf("package %d: invalid dimension unit '%s', must be 'cm', 'in', or 'mm'", i+1, pkg.DimUnit)
		}

		// Validate weight units
		weightUnit := strings.ToLower(pkg.WeightUnit)
		if weightUnit != "kg" && weightUnit != "g" && weightUnit != "lb" {
			return fmt.Errorf("package %d: invalid weight unit '%s', must be 'kg', 'g', or 'lb'", i+1, pkg.WeightUnit)
		}
	}

	// Validate service type
	if req.ServiceType == "" {
		return fmt.Errorf("service_type is required")
	}

	// Validate dates
	if req.PickupDate.IsZero() {
		return fmt.Errorf("pickup_date is required")
	}

	return nil
}



