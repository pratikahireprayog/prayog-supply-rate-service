package smile

// RateCardMapping holds the rate card ID mappings for smile service
var RateCardMapping = map[string]string{
	"smile":       getEnv("SMILE_RATE_CARD_ID", "b6ab836d-0f0c-4ad6-a01a-a626e48f2efa"),
	"smile_ecomm": getEnv("SMILE_ECOMM_RATE_CARD_ID", "b6ab836d-0f0c-4ad6-a01a-a626e48f2efa"),
	// Add more mappings here as needed
	// "partner_code": "rate_card_id",
}


// GetRateCardID returns the rate card ID for a given partner code
func GetRateCardID(partnerCode string) string {
	if rateCardID, exists := RateCardMapping[partnerCode]; exists {
		return rateCardID
	}
	// Return default rate card ID for smile_ecomm (or smile for backward compatibility)
	if rateCardID, exists := RateCardMapping["smile_ecomm"]; exists {
		return rateCardID
	}
	return RateCardMapping["smile"]
}

// SetRateCardID sets the rate card ID for a partner code
func SetRateCardID(partnerCode string, rateCardID string) {
	RateCardMapping[partnerCode] = rateCardID
}

