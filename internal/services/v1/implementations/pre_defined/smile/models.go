package smile

// SmileRateRequest represents the request format for Smile Rate API
type SmileRateRequest struct {
	RateCardID          string               `json:"rateCardId"`
	FromPincode         int                  `json:"fromPincode"`
	ToPincode           int                  `json:"toPincode"`
	ServiceType         string               `json:"serviceType"`
	ProductType         string               `json:"productType"`
	Weight              float64              `json:"weight"`
	Length              float64              `json:"length"`
	Height              float64              `json:"height"`
	Width               float64              `json:"width"`
	IncludeDefaultCharges bool               `json:"includeDefaultCharges"`
	UserOptions         *UserOptionsRequest  `json:"userOptions,omitempty"`
}

// UserOptionsRequest represents user options in the request
type UserOptionsRequest struct {
	Insurance *InsuranceOption `json:"insurance,omitempty"`
	COD       bool             `json:"cod,omitempty"`
}

// InsuranceOption represents insurance configuration
type InsuranceOption struct {
	Enabled bool    `json:"enabled"`
	Amount  float64 `json:"amount"`
}

// SmileRateResponse represents the response format from Smile Rate API
type SmileRateResponse struct {
	Status  string                `json:"status"`
	Message string                `json:"message,omitempty"`
	Data    *SmileRateResponseData `json:"data,omitempty"`
	Error   *SmileAPIError        `json:"error,omitempty"`
}

// Success returns true if the response status is "success"
func (r *SmileRateResponse) Success() bool {
	return r.Status == "success"
}

// SmileRateResponseData represents the data section of response (flat structure)
type SmileRateResponseData struct {
	BaseRate          float64                     `json:"baseRate"`
	TotalAmount       float64                     `json:"totalAmount"`
	Charges           []ChargeDetail              `json:"charges,omitempty"`
	Calculation       *CalculationDetails         `json:"calculation,omitempty"`
	Request           *RequestDetails             `json:"request,omitempty"`
	WeightCalculation *WeightCalculationDetails   `json:"weightCalculation,omitempty"`
	PincodeDetails    *PincodeDetails             `json:"pincodeDetails,omitempty"`
	ZoneResolution    *ZoneResolutionDetails      `json:"zoneResolution,omitempty"`
}

// CalculationDetails represents the calculation breakdown
type CalculationDetails struct {
	BaseAmount  float64 `json:"baseAmount"`
	Charges     float64 `json:"charges"`
	Discounts   float64 `json:"discounts"`
	Taxes       float64 `json:"taxes"`
	TotalAmount float64 `json:"totalAmount"`
}

// RequestDetails represents the request details in the response
type RequestDetails struct {
	ServiceType string              `json:"serviceType"`
	Location    string              `json:"location,omitempty"`
	Dimensions  *PackageDimensions  `json:"dimensions,omitempty"`
	UserOptions *UserOptionsRequest `json:"userOptions,omitempty"`
}

// PackageDimensions represents package dimensions
type PackageDimensions struct {
	Weight float64 `json:"weight"`
}

// WeightCalculationDetails represents weight calculation details
type WeightCalculationDetails struct {
	ActualWeight              float64 `json:"actualWeight"`
	CalculatedVolumetricWeight float64 `json:"calculatedVolumetricWeight"`
	FinalWeight               float64 `json:"finalWeight"`
	WeightUsed                string  `json:"weightUsed"`
}

// PincodeDetails represents pincode details
type PincodeDetails struct {
	From *LocationDetails `json:"from,omitempty"`
	To   *LocationDetails `json:"to,omitempty"`
}

// LocationDetails represents location information
type LocationDetails struct {
	Pincode int    `json:"pincode"`
	City    string `json:"city"`
	State   string `json:"state"`
	Country string `json:"country"`
}

// ZoneResolutionDetails represents zone resolution information
type ZoneResolutionDetails struct {
	Zone          string                 `json:"zone"`
	ZoneType      string                 `json:"zoneType"`
	AdditionalInfo map[string]interface{} `json:"additionalInfo,omitempty"`
}

// ChargeDetail represents individual charge details
type ChargeDetail struct {
	ChargeName  string      `json:"chargeName"`
	Amount      float64     `json:"amount"`
	Description string      `json:"description,omitempty"`
	Tax         *TaxDetails `json:"tax,omitempty"`
}

// TaxDetails represents tax information
type TaxDetails struct {
	IsTaxable       bool    `json:"isTaxable"`
	TaxAmount       float64 `json:"taxAmount"`
	TaxRate         float64 `json:"taxRate"`
	TaxDescription  string  `json:"taxDescription,omitempty"`
}

// ChargeCode returns a charge code based on charge name (for compatibility with price breakdown conversion)
func (c *ChargeDetail) ChargeCode() string {
	switch c.ChargeName {
	case "COD Charge", "COD":
		return "COD_CHARGE"
	case "Fuel Surcharge":
		return "FUEL_SURCHARGE"
	case "GST":
		return "GST"
	case "Insurance":
		return "INSURANCE_CHARGES"
	case "DCC Based delivery":
		return "DCC_DELIVERY"
	case "KYC Based delivery":
		return "KYC_DELIVERY"
	default:
		return "OTHER"
	}
}

// SmileAPIError represents error information from API
type SmileAPIError struct {
	Code     string                 `json:"code"`
	Message  string                 `json:"message"`
	Details  string                 `json:"details,omitempty"`
	Field    string                 `json:"field,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

