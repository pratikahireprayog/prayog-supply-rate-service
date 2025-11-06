package v1

import (
	"regexp"
	"strings"
)

// NormalizePartnerCode converts partner codes to snake_case format
func NormalizePartnerCode(code string) string {
	if code == "" {
		return ""
	}

	// Convert to lowercase
	normalized := strings.ToLower(code)

	// Replace spaces, hyphens, and special characters with underscores
	re := regexp.MustCompile(`[^a-z0-9]+`)
	normalized = re.ReplaceAllString(normalized, "_")

	// Remove leading/trailing underscores
	normalized = strings.Trim(normalized, "_")

	// Replace multiple consecutive underscores with single underscore
	re = regexp.MustCompile(`_+`)
	normalized = re.ReplaceAllString(normalized, "_")

	return normalized
}

// GetPartnerDisplayName returns a properly formatted display name for a partner code
func GetPartnerDisplayName(normalizedCode string) string {
	switch normalizedCode {
	case "dhl":
		return "DHL Express"
	case "fedex":
		return "FedEx"
    case "india_post_domestic":
        return "India Post Domestic"
	case "ups":
		return "UPS"
	case "blue_dart":
		return "Blue Dart Express"
	case "aramex":
		return "Aramex"
	case "delhivery":
		return "Delhivery"
	default:
		// Convert snake_case back to Title Case
		parts := strings.Split(normalizedCode, "_")
		for i, part := range parts {
			if part != "" {
				parts[i] = strings.Title(part)
			}
		}
		return strings.Join(parts, " ")
	}
}

// IsRealTimePartner checks if a partner code corresponds to a real-time implementation
func IsRealTimePartner(normalizedCode string) bool {
	realTimePartners := map[string]bool{
		"dhl":    true,
		"fedex":  true,
		"ups":    true,
		"aramex": true,
        "india_post_domestic": true,
	}

	return realTimePartners[normalizedCode]
}

// IsPreDefinedPartner checks if a partner code corresponds to a pre-defined implementation
func IsPreDefinedPartner(normalizedCode string) bool {
	preDefinedPartners := map[string]bool{
		"delhivery": true,
		"blue_dart": true,
		"porter":    true,
		"dunzo":     true,
	}

	return preDefinedPartners[normalizedCode]
}

// Helper functions for unit conversion
func ConvertWeightToKg(weight float64, unit string) float64 {
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

func ConvertWeightToGrams(weight float64, unit string) float64 {
	switch strings.ToUpper(unit) {
	case "KG":
		return weight * 1000
	case "LB", "LBS":
		return weight * 453.592
	case "OZ":
		return weight * 28.3495
	default: // G
		return weight
	}
}

func ConvertDimensionToCm(dimension float64, unit string) float64 {
    switch strings.ToUpper(unit) {
    case "M":
        return dimension * 100
    case "MM":
        return dimension / 10
    case "IN", "INCH":
        return dimension * 2.54
    case "FT":
        return dimension * 30.48
    default: 
        return dimension
    }
}
