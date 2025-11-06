package shipcube

type Config struct {
	BaseURL  string
	TenantID string
	// APIKey   string
}

// ShipCubeRateRequest represents the request payload for ShipCube API
type ShipCubeRateRequest struct {
	ProductType           string                 `json:"productType"`
	ServiceType           string                 `json:"serviceType"`
	Location              string                 `json:"location"`
	Filters               map[string]interface{} `json:"filters"`
	Dimensions            ShipCubeDimensions     `json:"dimensions"`
	UserOptions           map[string]interface{} `json:"userOptions"`
	IncludeDefaultCharges bool                   `json:"includeDefaultCharges"`
}

type ShipCubeDimensions struct {
	Weight float64 `json:"weight"` // Weight in grams
}

// Response Models
type ShipCubeRateResponse struct {
	Status  string           `json:"status"`
	Message string           `json:"message"`
	Data    ShipCubeRateData `json:"data"`
}

type ShipCubeRateData struct {
	BaseRate    float64                `json:"baseRate"`
	TotalAmount float64                `json:"totalAmount"`
	Charges     []interface{}          `json:"charges"`
	Calculation ShipCubeCalculation    `json:"calculation"`
	Request     ShipCubeRequestDetails `json:"request"`
}

type ShipCubeCalculation struct {
	BaseAmount  float64 `json:"baseAmount"`
	Charges     float64 `json:"charges"`
	Discounts   float64 `json:"discounts"`
	Taxes       float64 `json:"taxes"`
	TotalAmount float64 `json:"totalAmount"`
}

type ShipCubeRequestDetails struct {
	ServiceType string                 `json:"serviceType"`
	Location    string                 `json:"location"`
	Dimensions  map[string]interface{} `json:"dimensions"`
	UserOptions map[string]interface{} `json:"userOptions"`
}
