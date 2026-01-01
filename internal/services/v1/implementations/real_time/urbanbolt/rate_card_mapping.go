package urbanbolt

// RateCardMapping holds the rate card ID mappings for urbanbolt service
var RateCardMapping = map[string]string{
	"urbanbolt": "27e926ed-03bb-4a23-ba98-ed8dde045222",
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

