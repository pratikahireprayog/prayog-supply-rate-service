package fedex

import (
	"fmt"
	"strings"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
)

// validateRequest validates the rate calculation request
func (s *Service) validateRequest(request *dtos.RateCalculationRequest) error {
    if request == nil {
        return fmt.Errorf("request cannot be nil")
    }

    if strings.TrimSpace(request.OriginCity) == "" {
        return fmt.Errorf("origin city is required")
    }

    if strings.TrimSpace(request.OriginCountry) == "" {
        return fmt.Errorf("origin country is required")
    }

    if strings.TrimSpace(request.DestCity) == "" {
        return fmt.Errorf("destination city is required")
    }

    if strings.TrimSpace(request.DestCountry) == "" {
        return fmt.Errorf("destination country is required")
    }

    if len(request.Packages) == 0 {
        return fmt.Errorf("at least one package is required")
    }

    for i, pkg := range request.Packages {
        if pkg.Weight <= 0 {
            return fmt.Errorf("package %d weight must be greater than 0", i+1)
        }

        if pkg.Length <= 0 || pkg.Width <= 0 || pkg.Height <= 0 {
            return fmt.Errorf("package %d dimensions must be greater than 0", i+1)
        }
    }

    return nil
}

// Helper functions for unit conversion
func convertWeightToKg(weight float64, unit string) float64 {
    switch strings.ToUpper(unit) {
    case "G":
        return weight / 1000
    case "LB", "LBS":
        return weight * 0.453592
    case "OZ":
        return weight * 0.0283495
    default: // KG
        return weight
    }
}

func convertDimensionToCm(dimension float64, unit string) float64 {
    switch strings.ToUpper(unit) {
    case "M":
        return dimension * 100
    case "MM":
        return dimension / 10
    case "IN", "INCH":
        return dimension * 2.54
    case "FT":
        return dimension * 30.48
    default: // CM
        return dimension
    }
}