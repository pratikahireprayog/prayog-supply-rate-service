package baral_rate

import (
	"fmt"

	dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
)

// ValidateRateRequest validates the rate calculation request
func ValidateRateRequest(request *dtos.RateCalculationRequest) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Validate request ID
	if request.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}

	// Validate origin location
	if err := validateLocation(request.OriginCity, request.OriginCountry, "origin"); err != nil {
		return err
	}

	// Validate destination location
	if err := validateLocation(request.DestCity, request.DestCountry, "destination"); err != nil {
		return err
	}

	// Validate packages
	if len(request.Packages) == 0 {
		return fmt.Errorf("at least one package is required")
	}

	for i, pkg := range request.Packages {
		if err := validatePackage(&pkg, i); err != nil {
			return err
		}
	}

	// Validate currency if provided
	if request.Currency != "" {
		if !isValidCurrency(request.Currency) {
			return fmt.Errorf("invalid currency: %s (must be 3-letter ISO code)", request.Currency)
		}
	}

	return nil
}

// validateLocation validates location information
func validateLocation(city, country, locationType string) error {
	if city == "" {
		return fmt.Errorf("%s city/postal code is required", locationType)
	}

	if country == "" {
		return fmt.Errorf("%s country is required", locationType)
	}

	// Validate country code format (should be 2-letter ISO code)
	if len(country) != 2 {
		return fmt.Errorf("%s country code must be 2 letters (ISO 3166-1 alpha-2)", locationType)
	}

	return nil
}

// validatePackage validates package information
func validatePackage(pkg *dtos.PackageDetails, index int) error {
	if pkg == nil {
		return fmt.Errorf("package at index %d is nil", index)
	}

	// Validate weight
	if pkg.Weight <= 0 {
		return fmt.Errorf("package %d: weight must be positive", index)
	}

	// Validate weight unit
	if pkg.WeightUnit == "" {
		return fmt.Errorf("package %d: weight unit is required", index)
	}

	validWeightUnits := map[string]bool{
		"kg":  true,
		"KG":  true,
		"g":   true,
		"G":   true,
		"lb":  true,
		"LB":  true,
		"oz":  true,
		"OZ":  true,
	}

	if !validWeightUnits[pkg.WeightUnit] {
		return fmt.Errorf("package %d: invalid weight unit '%s' (supported: kg, g, lb, oz)", index, pkg.WeightUnit)
	}

	// Validate dimensions if provided
	if pkg.Length > 0 || pkg.Width > 0 || pkg.Height > 0 {
		if pkg.Length <= 0 || pkg.Width <= 0 || pkg.Height <= 0 {
			return fmt.Errorf("package %d: if dimensions are provided, all must be positive (length, width, height)", index)
		}

		// Validate dimension unit
		if pkg.DimUnit == "" {
			return fmt.Errorf("package %d: dimension unit is required when dimensions are provided", index)
		}

		validDimensionUnits := map[string]bool{
			"cm": true,
			"CM": true,
			"m":  true,
			"M":  true,
			"in": true,
			"IN": true,
			"ft": true,
			"FT": true,
			"mm": true,
			"MM": true,
		}

		if !validDimensionUnits[pkg.DimUnit] {
			return fmt.Errorf("package %d: invalid dimension unit '%s' (supported: cm, m, in, ft, mm)", index, pkg.DimUnit)
		}
	}

	return nil
}

// isValidCurrency checks if the currency is a valid 3-letter ISO code
func isValidCurrency(currency string) bool {
	if len(currency) != 3 {
		return false
	}

	// List of common currencies (can be extended)
	validCurrencies := map[string]bool{
		"INR": true,
		"USD": true,
		"EUR": true,
		"GBP": true,
		"CNY": true,
		"JPY": true,
		"AUD": true,
		"CAD": true,
		"CHF": true,
		"HKD": true,
		"SGD": true,
		"AED": true,
		"SAR": true,
		"KWD": true,
		"QAR": true,
		"OMR": true,
		"BHD": true,
	}

	return validCurrencies[currency]
}

// ValidateBaralRate validates a Baral rate structure
func ValidateBaralRate(rate *BaralRate) error {
	if rate == nil {
		return fmt.Errorf("rate cannot be nil")
	}

	// Check if rate has essential fields
	if rate.ID == "" {
		return fmt.Errorf("rate must have an ID")
	}

	// Validate rate card name
	if rate.RateCardName == "" {
		return fmt.Errorf("rate card name is required")
	}

	// Validate minimum freight if present
	if rate.MinimumFreight != "" {
		freight := rate.GetMinimumFreightINR()
		if freight <= 0 {
			return fmt.Errorf("minimum freight must be positive: %s", rate.MinimumFreight)
		}
	}

	return nil
}

// ValidateBaralAPIResponse validates the API response structure
func ValidateBaralAPIResponse(response *BaralAPIResponse) error {
	if response == nil {
		return fmt.Errorf("response cannot be nil")
	}

	// Check for error in response
	if response.Error != nil {
		return fmt.Errorf("API error: %s - %s", response.Error.Code, response.Error.Message)
	}

	// Check if response status is success
	if response.Status != 200 {
		if response.Message != "" {
			return fmt.Errorf("API returned non-success status %d: %s", response.Status, response.Message)
		}
		return fmt.Errorf("API returned non-success status: %d", response.Status)
	}

	// Check if response has data
	if response.Data == nil || len(response.Data.RateCard) == 0 {
		if response.Message != "" {
			return fmt.Errorf("API returned no data: %s", response.Message)
		}
		return fmt.Errorf("API returned no data")
	}

	return nil
}

