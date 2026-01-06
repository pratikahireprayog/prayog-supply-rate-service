package delhivery

// KinkoRateResponse represents the response from the Delhivery Kinko API
type KinkoRateResponse []KinkoRateItem

// KinkoRateItem represents a single rate item in the response
type KinkoRateItem struct {
	TotalAmount   float64    `json:"total_amount"`
	GrossAmount   float64    `json:"gross_amount"`
	ChargedWeight float64    `json:"charged_weight"`
	Status        string     `json:"status"`
	Zone          string     `json:"zone"`
	TaxData       TaxData    `json:"tax_data"`
	ChargeDL      float64    `json:"charge_DL"`  // Base charge?
	ChargeDPH     float64    `json:"charge_DPH"` // Fuel/other?
	// Add other charges if needed for breakdown
}

// TaxData represents tax details in the Kinko API
type TaxData struct {
	CGST float64 `json:"CGST"`
	SGST float64 `json:"SGST"`
	IGST float64 `json:"IGST"`
}

// Keeping basic ErrorResponse for error handling
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
