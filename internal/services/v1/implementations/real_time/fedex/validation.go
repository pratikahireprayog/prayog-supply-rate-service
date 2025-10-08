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

