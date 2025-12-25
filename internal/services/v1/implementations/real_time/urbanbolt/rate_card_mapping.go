package urbanbolt

// RateCardMapping holds the rate card ID mappings for urbanbolt service
var RateCardMapping = map[string]string{
	"urbanbolt": "6f67e45e-cd11-4c8a-820a-247324c20da2",
	// Add more mappings here as needed
	// "partner_code": "rate_card_id",
}

// GetRateCardID returns the rate card ID for a given partner code
func GetRateCardID(partnerCode string) string {
	if rateCardID, exists := RateCardMapping[partnerCode]; exists {
		return rateCardID
	}
	// Return default rate card ID for urbanbolt
	return RateCardMapping["urbanbolt"]
}

// SetRateCardID sets the rate card ID for a partner code
func SetRateCardID(partnerCode string, rateCardID string) {
	RateCardMapping[partnerCode] = rateCardID
}

