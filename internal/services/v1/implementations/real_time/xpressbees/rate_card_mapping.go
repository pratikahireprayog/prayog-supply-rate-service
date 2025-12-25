package xpressbees

// RateCardMapping holds the rate card ID mappings for xpressbees service
var RateCardMapping = map[string]string{
	"xpressbees": "24e617bd-b72f-4534-b721-63e2e70eafcc",
	// Add more mappings here as needed
	// "partner_code": "rate_card_id",
}

// GetRateCardID returns the rate card ID for a given partner code
func GetRateCardID(partnerCode string) string {
	if rateCardID, exists := RateCardMapping[partnerCode]; exists {
		return rateCardID
	}
	// Return default rate card ID for xpressbees
	return RateCardMapping["xpressbees"]
}

// SetRateCardID sets the rate card ID for a partner code
func SetRateCardID(partnerCode string, rateCardID string) {
	RateCardMapping[partnerCode] = rateCardID
}

