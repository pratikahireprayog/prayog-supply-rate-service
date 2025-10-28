package indiapost

import (
	"fmt"
	"strings"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
)

// validateRequest validates the rate calculation request for India Post
func (s *Service) validateRequest(request *dtos.RateCalculationRequest) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Validate source location
	if err := s.validateLocation(request.OriginCity, request.OriginCountry, "source"); err != nil {
		return err
	}

	// Validate destination location
	if err := s.validateLocation(request.DestCity, request.DestCountry, "destination"); err != nil {
		return err
	}

	// India Post is for international shipping only
	if request.OriginCountry == request.DestCountry {
		return fmt.Errorf("India Post International only supports international shipments")
	}

	// India Post primarily operates from India
	if strings.ToUpper(request.OriginCountry) != "IN" {
		return fmt.Errorf("India Post International only supports shipments from India (IN)")
	}

	// Validate packages
	if len(request.Packages) == 0 {
		return fmt.Errorf("at least one package is required")
	}

	for i, pkg := range request.Packages {
		if err := s.validatePackage(pkg, i); err != nil {
			return fmt.Errorf("package %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateLocation validates a location
func (s *Service) validateLocation(postalCode, countryCode, locationType string) error {
	if postalCode == "" {
		return fmt.Errorf("%s postal code is required", locationType)
	}

	if countryCode == "" {
		return fmt.Errorf("%s country code is required", locationType)
	}

	if len(countryCode) != 2 {
		return fmt.Errorf("%s country code must be 2 characters (ISO 3166-1 alpha-2)", locationType)
	}

	return nil
}

// validatePackage validates a package
func (s *Service) validatePackage(pkg dtos.PackageDetails, index int) error {
	// Validate weight
	if pkg.Weight <= 0 {
		return fmt.Errorf("weight must be greater than 0")
	}

	// India Post weight limits (typically up to 30kg for parcels, 2kg for letters)
	// We'll use 30kg as the maximum
	if pkg.Weight > 30 {
		return fmt.Errorf("weight exceeds India Post maximum of 30kg")
	}

	// Validate dimensions
	if pkg.Length <= 0 || pkg.Width <= 0 || pkg.Height <= 0 {
		return fmt.Errorf("all dimensions must be greater than 0")
	}

	// India Post dimension limits (varies by product type)
	// For international parcels: typically max length 1.5m, max combined dimensions 3m
	maxLength := 150.0 // 150 cm
	if pkg.Length > maxLength || pkg.Width > maxLength || pkg.Height > maxLength {
		return fmt.Errorf("individual dimension exceeds India Post maximum of %.0f cm", maxLength)
	}

	combinedDimensions := pkg.Length + pkg.Width + pkg.Height
	maxCombined := 300.0 // 300 cm
	if combinedDimensions > maxCombined {
		return fmt.Errorf("combined dimensions (%.2f cm) exceed India Post maximum of %.0f cm", combinedDimensions, maxCombined)
	}

	return nil
}

// validateTariffRequest validates the tariff request before sending to API
func (s *Service) validateTariffRequest(req *TariffRequest) error {
	if req == nil {
		return fmt.Errorf("tariff request cannot be nil")
	}

	if req.ProductType == "" {
		return fmt.Errorf("product type is required")
	}

	if req.Weight <= 0 {
		return fmt.Errorf("weight must be greater than 0")
	}

	if req.CountryCode == "" {
		return fmt.Errorf("country code is required")
	}

	if len(req.CountryCode) != 2 {
		return fmt.Errorf("country code must be 2 characters")
	}

	if req.SourcePincode == "" {
		return fmt.Errorf("source pincode is required")
	}

	// Validate mode of transmission
	validModes := []string{"AMS", "SAL"}
	validMode := false
	for _, mode := range validModes {
		if req.ModeOfTransmission == mode {
			validMode = true
			break
		}
	}
	if !validMode {
		return fmt.Errorf("invalid mode of transmission, must be one of: %v", validModes)
	}

	// Validate insurance amount if insurance is enabled
	if req.Insurance && req.InsAmount <= 0 {
		return fmt.Errorf("insurance amount must be greater than 0 when insurance is enabled")
	}

	return nil
}

// isValidCountryCode checks if the country code is valid for India Post International
func isValidCountryCode(countryCode string) bool {
	// India Post supports most international destinations
	// Here we could add a whitelist or blacklist if needed
	// For now, accept any 2-character code except IN (already handled)
	return len(countryCode) == 2 && strings.ToUpper(countryCode) != "IN"
}

// estimateDeliveryDays estimates delivery days based on transmission mode and destination
func estimateDeliveryDays(mode ModeOfTransmission, countryCode string) int {
	// Default delivery estimates based on mode and region
	if mode == ModeAMS {
		// Air Mail Service - faster
		switch {
		case isAsiaRegion(countryCode):
			return 5 // 5-7 days for Asia
		case isEuropeRegion(countryCode):
			return 7 // 7-10 days for Europe
		case isAmericasRegion(countryCode):
			return 10 // 10-14 days for Americas
		default:
			return 12 // 12-15 days for other regions
		}
	} else {
		// Surface Air Lifted - slower but economical
		switch {
		case isAsiaRegion(countryCode):
			return 15 // 15-20 days for Asia
		case isEuropeRegion(countryCode):
			return 21 // 21-28 days for Europe
		case isAmericasRegion(countryCode):
			return 30 // 30-45 days for Americas
		default:
			return 35 // 35-50 days for other regions
		}
	}
}

// Helper functions for region detection
func isAsiaRegion(countryCode string) bool {
	asianCountries := []string{"CN", "JP", "SG", "MY", "TH", "ID", "PH", "VN", "KR", "BD", "PK", "LK"}
	code := strings.ToUpper(countryCode)
	for _, c := range asianCountries {
		if code == c {
			return true
		}
	}
	return false
}

func isEuropeRegion(countryCode string) bool {
	europeanCountries := []string{"GB", "DE", "FR", "IT", "ES", "NL", "BE", "CH", "AT", "SE", "NO", "DK", "FI"}
	code := strings.ToUpper(countryCode)
	for _, c := range europeanCountries {
		if code == c {
			return true
		}
	}
	return false
}

func isAmericasRegion(countryCode string) bool {
	americasCountries := []string{"US", "CA", "MX", "BR", "AR", "CL", "CO", "PE"}
	code := strings.ToUpper(countryCode)
	for _, c := range americasCountries {
		if code == c {
			return true
		}
	}
	return false
}

