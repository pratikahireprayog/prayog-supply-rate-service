package delhivery

// LoginRequest represents the login request to Delhivery API
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response from Delhivery API
type LoginResponse struct {
	Success   bool      `json:"success"`
	RequestID string    `json:"request_id,omitempty"`
	Data      LoginData `json:"data"`
}

// LoginData contains the authentication data
type LoginData struct {
	JWT string `json:"jwt"`
}

// FreightEstimateRequest represents the freight estimate request
type FreightEstimateRequest struct {
	Dimensions []Dimension `json:"dimensions"`
	WeightG    int         `json:"weight_g"`
	ChequePayment bool     `json:"cheque_payment"`
	SourcePin    string    `json:"source_pin"`
	ConsigneePin string    `json:"consignee_pin"`
	PaymentMode  string    `json:"payment_mode"` // "prepaid" or "cod"
	InvAmount    float64   `json:"inv_amount"`
	FreightMode  string    `json:"freight_mode"` // "fod" (Freight on Delivery) or "prepaid"
	ROVInsurance bool      `json:"rov_insurance"`
}

// Dimension represents package dimensions
type Dimension struct {
	LengthCM float64 `json:"length_cm"`
	WidthCM  float64 `json:"width_cm"`
	HeightCM float64 `json:"height_cm"`
	BoxCount int     `json:"box_count"`
}

// FreightEstimateResponse represents the freight estimate response
type FreightEstimateResponse struct {
	Success   bool              `json:"success"`
	RequestID string            `json:"request_id,omitempty"`
	Data      FreightEstimateData `json:"data,omitempty"`
	Error     *ErrorResponse    `json:"error,omitempty"`
	Message   string            `json:"message,omitempty"`
}

// FreightEstimateData contains the freight estimate details
type FreightEstimateData struct {
	TotalFreight    float64            `json:"total_freight"`
	BaseFreight     float64            `json:"base_freight"`
	FuelSurcharge   float64            `json:"fuel_surcharge"`
	ODACharge       float64            `json:"oda_charge,omitempty"`
	InsuranceCharge float64            `json:"insurance_charge,omitempty"`
	CODCharge       float64            `json:"cod_charge,omitempty"`
	OtherCharges    float64            `json:"other_charges,omitempty"`
	Taxes           TaxDetails         `json:"taxes,omitempty"`
	Breakdown       []ChargeBreakdown  `json:"breakdown,omitempty"`
	EstimatedDays   int                `json:"estimated_days,omitempty"`
	ServiceType     string             `json:"service_type,omitempty"`
}

// TaxDetails represents tax information
type TaxDetails struct {
	CGST  float64 `json:"cgst,omitempty"`
	SGST  float64 `json:"sgst,omitempty"`
	IGST  float64 `json:"igst,omitempty"`
	Total float64 `json:"total,omitempty"`
}

// ChargeBreakdown provides detailed charge breakdown
type ChargeBreakdown struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

