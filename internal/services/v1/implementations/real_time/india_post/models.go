package indiapost

import "time"

// LoginRequest represents the login request to India Post API
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response from India Post API
type LoginResponse struct {
	StatusCode int             `json:"statusCode"`
	Data       LoginData       `json:"data"`
	Timestamp  time.Time       `json:"timestamp"`
}

// LoginData contains the authentication data
type LoginData struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// TariffRequest represents the tariff calculation request
type TariffRequest struct {
	ProductType         string  `json:"productType"`
	Weight              float64 `json:"weight"`
	CountryCode         string  `json:"countryCode"`
	Registration        bool    `json:"registration"`
	Insurance           bool    `json:"insurance"`
	InsAmount           float64 `json:"insAmount"`
	AdviceOfDelivery    bool    `json:"adviceOfDelivery"`
	ModeOfTransmission  string  `json:"modeOfTransmission"`
	SourcePincode       string  `json:"sourcePincode"`
}

// TariffResponse represents the tariff calculation response
type TariffResponse struct {
	StatusCode int         `json:"statusCode"`
	Data       TariffData  `json:"data"`
	Timestamp  time.Time   `json:"timestamp"`
}

// TariffData contains the tariff calculation details
type TariffData struct {
	Success           bool           `json:"success"`
	ProductCode       string         `json:"productCode"`
	ProductName       string         `json:"productName"`
	CountryCode       string         `json:"countryCode"`
	Weight            float64        `json:"weight"`
	TariffGroup       int            `json:"tariffGroup"`
	BasicTariff       BasicTariff    `json:"basicTariff"`
	VASCharges        VASCharges     `json:"vasCharges"`
	TaxCalculation    TaxCalculation `json:"taxCalculation"`
	TotalBeforeTax    float64        `json:"totalBeforeTax"`
	TotalAmount       float64        `json:"totalAmount"`
	ModeOfTransmission string        `json:"modeOfTransmission"`
	CalculatedAt      time.Time      `json:"calculatedAt"`
}

// BasicTariff represents basic tariff charges
type BasicTariff struct {
	BasePrice       float64 `json:"basePrice"`
	AmsCharge       float64 `json:"amsCharge"`
	SalCharge       float64 `json:"salCharge"`
	TotalBasicPrice float64 `json:"totalBasicPrice"`
}

// VASCharges represents value-added service charges
type VASCharges struct {
	Registration            float64 `json:"registration"`
	Insurance               float64 `json:"insurance"`
	AdviceOfDelivery        float64 `json:"adviceOfDelivery"`
	DoorDelivery            float64 `json:"doorDelivery"`
	CompulsoryRegistration  bool    `json:"compulsoryRegistration"`
	Total                   float64 `json:"total"`
}

// TaxCalculation represents tax calculation details
type TaxCalculation struct {
	CGST            float64         `json:"cgst"`
	SGST            float64         `json:"sgst"`
	UTGST           float64         `json:"utgst"`
	IGST            float64         `json:"igst"`
	TotalGST        float64         `json:"totalGst"`
	GSTType         string          `json:"gstType"`
	ApplicableRate  string          `json:"applicableRate"`
	Breakdown       TaxBreakdown    `json:"breakdown"`
}

// TaxBreakdown provides detailed tax breakdown
type TaxBreakdown struct {
	TaxableAmount float64 `json:"taxableAmount"`
	CGSTRate      *string `json:"cgstRate"`
	SGSTRate      *string `json:"sgstRate"`
	UTGSTRate     *string `json:"utgstRate"`
	IGSTRate      string  `json:"igstRate"`
}

// ErrorResponse represents an error response from India Post API
type ErrorResponse struct {
	StatusCode int       `json:"statusCode"`
	Message    string    `json:"message"`
	Error      string    `json:"error"`
	Timestamp  time.Time `json:"timestamp"`
}

// ProductType represents the different product types available
type ProductType string

const (
	ProductTypeFGNLetter     ProductType = "FGN_LETTER"
	ProductTypeFGNParcel     ProductType = "FGN_PARCEL"
	ProductTypeEMS           ProductType = "EMS"
	ProductTypeSmallPacket   ProductType = "SMALL_PACKET"
)

// ModeOfTransmission represents the transmission mode
type ModeOfTransmission string

const (
	ModeAMS ModeOfTransmission = "AMS" // Air Mail Service
	ModeSAL ModeOfTransmission = "SAL" // Surface Air Lifted
)

// MapServiceTypeToProductType maps our service types to India Post product types
func MapServiceTypeToProductType(serviceType string) ProductType {
	switch serviceType {
	case "express":
		return ProductTypeEMS
	case "standard":
		return ProductTypeFGNLetter
	case "economy":
		return ProductTypeSmallPacket
	default:
		return ProductTypeFGNLetter
	}
}

// MapServiceTypeToTransmissionMode maps service types to transmission modes
func MapServiceTypeToTransmissionMode(serviceType string) ModeOfTransmission {
	switch serviceType {
	case "express":
		return ModeAMS
	case "economy":
		return ModeSAL
	default:
		return ModeAMS
	}
}

