package mover

// OrderEstimateRequest represents the request to Mover API
type OrderEstimateRequest struct {
	Type          string    `json:"type"`           // "parcel"
	ShipMode      string    `json:"ship_mode"`      // "on_demand"
	Desc          string    `json:"desc"`           // Description
	WeightKg      float64   `json:"weight_kg"`      // Weight in kg
	GoodsWorth    float64   `json:"goods_worth"`    // Value of goods
	OptimiseRoute bool      `json:"optimise_route"` // Route optimization
	BoxRequired   bool      `json:"box_required"`  // Box required
	PaymentMethod int       `json:"payment_method"` // 0 = prepaid, 1 = COD
	RouteInfo     RouteInfo `json:"route_info"`    // Route information
	Pickup        Location  `json:"pickup"`        // Pickup location
	Drop          Location  `json:"drop"`          // Drop location
	Waypoints     []string  `json:"waypoints"`      // Waypoints (empty array)
}

// RouteInfo represents route information
type RouteInfo struct {
	Distance float64 `json:"distance"` // Distance in meters
	Duration int     `json:"duration"` // Duration in seconds
}

// Location represents pickup/drop location
type Location struct {
	Lat          float64 `json:"lat"`           // Latitude
	Lon          float64 `json:"lon"`           // Longitude
	Address      string  `json:"address"`      // Address
	DeliveryNote string  `json:"delivery_note"` // Delivery note
	ContactMobile string `json:"contact_mobile"` // Contact mobile
	ContactName   string  `json:"contact_name"`   // Contact name
	Udf1          string  `json:"udf1"`          // User defined field 1
}

// OrderEstimateResponse represents the response from Mover API
type OrderEstimateResponse struct {
	List              []VehicleOption `json:"list"`
	PolyData          interface{}    `json:"PolyData,omitempty"`
	CouponApplyStatus int            `json:"couponApplyStatus"`
	AvailableCoupons  interface{}     `json:"availableCoupons,omitempty"`
	WalletBalance     float64        `json:"WalletBalance"`
	EstimateID        string          `json:"estimateId"`
	ValidTill         string          `json:"validTill"`
}

// VehicleOption represents a vehicle option in the estimate response
type VehicleOption struct {
	ID                   int     `json:"id"`
	Name                 string  `json:"name"`
	Distance             float64 `json:"distance"`             // Distance in meters
	Duration             int     `json:"duration"`              // Duration in seconds
	Amount               float64 `json:"amount"`               // Base amount
	AmountWithHelper     float64 `json:"amountWithHelper"`      // Amount with helper
	HelperAmount         float64 `json:"helperAmount"`          // Helper amount
	Discount             float64 `json:"discount"`              // Discount amount
	DiscountedAmount     float64 `json:"discountedAmount"`      // Final discounted amount
	Status               int     `json:"status"`               // Status (1 = available)
	Image                string  `json:"image,omitempty"`       // Vehicle image URL
	IsEcoFriendly        int     `json:"isEcoFriendly"`        // Eco-friendly flag
	HelperApplicable     int     `json:"helperApplicable"`       // Helper applicable flag
	AvailableCoupons     []interface{} `json:"availableCoupons,omitempty"`
	Drivers              interface{}   `json:"drivers,omitempty"`
	Tolls                []interface{} `json:"tolls,omitempty"`
	CapacityInKg         float64 `json:"CapacityInKg"`         // Capacity in kg
	LengthInFt           float64 `json:"LengthInFt"`            // Length in feet
	WidthInFt            float64 `json:"WidthInFt"`             // Width in feet
	HeightInFt           float64 `json:"HeightInFt"`            // Height in feet
	UnchargedLoadingTime int     `json:"UnchargedLoadingTime"` // Uncharged loading time
	Coupons              interface{} `json:"coupons,omitempty"`
	CostPerMin           float64 `json:"CostPerMin"`           // Cost per minute
}

