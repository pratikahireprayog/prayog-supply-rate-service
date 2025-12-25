package smile

// RateCardMapping holds the rate card ID mappings for smile service
var RateCardMapping = map[string]string{
	"smile": "b6ab836d-0f0c-4ad6-a01a-a626e48f2efa",
	// Add more mappings here as needed
	// "partner_code": "rate_card_id",
}

// GetRateCardID returns the rate card ID for a given partner code
func GetRateCardID(partnerCode string) string {
	if rateCardID, exists := RateCardMapping[partnerCode]; exists {
		return rateCardID
	}
	// Return default rate card ID for smile
	return RateCardMapping["smile"]
}

// SetRateCardID sets the rate card ID for a partner code
func SetRateCardID(partnerCode string, rateCardID string) {
	RateCardMapping[partnerCode] = rateCardID
}

