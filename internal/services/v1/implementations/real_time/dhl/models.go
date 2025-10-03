package dhl

// DHLProduct represents a DHL product/service
type DHLProduct struct {
	ProductName             string                  `json:"productName"`
	ProductCode             string                  `json:"productCode"`
	LocalProductCode        string                  `json:"localProductCode"`
	LocalProductCountryCode string                  `json:"localProductCountryCode"`
	NetworkTypeCode         string                  `json:"networkTypeCode"`
	IsCustomerAgreement     bool                    `json:"isCustomerAgreement"`
	Weight                  DHLWeight               `json:"weight"`
	TotalPrice              []DHLPrice              `json:"totalPrice"`
	TotalPriceBreakdown     []DHLPriceBreakdown     `json:"totalPriceBreakdown"`
	DetailedPriceBreakdown  []DHLDetailedBreakdown  `json:"detailedPriceBreakdown"`
	PickupCapabilities      DHLPickupCapabilities   `json:"pickupCapabilities"`
	DeliveryCapabilities    DHLDeliveryCapabilities `json:"deliveryCapabilities"`
	PricingDate             string                  `json:"pricingDate"`
}

// DHLWeight represents weight information from DHL
type DHLWeight struct {
	Volumetric        float64 `json:"volumetric"`
	Provided          float64 `json:"provided"`
	UnitOfMeasurement string  `json:"unitOfMeasurement"`
}

// DHLPrice represents pricing information
type DHLPrice struct {
	CurrencyType  string  `json:"currencyType"`
	PriceCurrency string  `json:"priceCurrency"`
	Price         float64 `json:"price"`
}

// DHLPriceBreakdown represents price breakdown
type DHLPriceBreakdown struct {
	CurrencyType   string           `json:"currencyType"`
	PriceCurrency  string           `json:"priceCurrency"`
	PriceBreakdown []DHLPriceDetail `json:"priceBreakdown"`
}

// DHLPriceDetail represents individual price components
type DHLPriceDetail struct {
	TypeCode  string  `json:"typeCode"`
	Price     float64 `json:"price"`
	Rate      float64 `json:"rate,omitempty"`
	BasePrice float64 `json:"basePrice,omitempty"`
}

// DHLDetailedBreakdown represents detailed price breakdown
type DHLDetailedBreakdown struct {
	CurrencyType  string                `json:"currencyType"`
	PriceCurrency string                `json:"priceCurrency"`
	Breakdown     []DHLDetailedLineItem `json:"breakdown"`
}

// DHLDetailedLineItem represents a line item in detailed breakdown
type DHLDetailedLineItem struct {
	Name                string           `json:"name"`
	ServiceCode         string           `json:"serviceCode,omitempty"`
	LocalServiceCode    string           `json:"localServiceCode,omitempty"`
	ServiceTypeCode     string           `json:"serviceTypeCode,omitempty"`
	Price               float64          `json:"price"`
	IsCustomerAgreement bool             `json:"isCustomerAgreement,omitempty"`
	IsMarketedService   bool             `json:"isMarketedService,omitempty"`
	PriceBreakdown      []DHLPriceDetail `json:"priceBreakdown,omitempty"`
}

// DHLPickupCapabilities represents pickup capabilities
type DHLPickupCapabilities struct {
	NextBusinessDay                       bool   `json:"nextBusinessDay"`
	LocalCutoffDateAndTime                string `json:"localCutoffDateAndTime"`
	PickupEarliest                        string `json:"pickupEarliest"`
	PickupLatest                          string `json:"pickupLatest"`
	PickupCutoffSameDayOutboundProcessing string `json:"pickupCutoffSameDayOutboundProcessing"`
	OriginServiceAreaCode                 string `json:"originServiceAreaCode"`
	OriginFacilityAreaCode                string `json:"originFacilityAreaCode"`
	PickupAdditionalDays                  int    `json:"pickupAdditionalDays"`
	PickupDayOfWeek                       int    `json:"pickupDayOfWeek"`
}

// DHLDeliveryCapabilities represents delivery capabilities
type DHLDeliveryCapabilities struct {
	DeliveryTypeCode             string `json:"deliveryTypeCode"`
	EstimatedDeliveryDateAndTime string `json:"estimatedDeliveryDateAndTime"`
	DestinationServiceAreaCode   string `json:"destinationServiceAreaCode"`
	DestinationFacilityAreaCode  string `json:"destinationFacilityAreaCode"`
	DeliveryAdditionalDays       int    `json:"deliveryAdditionalDays"`
	DeliveryDayOfWeek            int    `json:"deliveryDayOfWeek"`
	TotalTransitDays             int    `json:"totalTransitDays"`
}

// DHLResponse represents the complete response from DHL API
type DHLResponse struct {
	Products      []DHLProduct      `json:"products"`
	ExchangeRates []DHLExchangeRate `json:"exchangeRates"`
}

// DHLExchangeRate represents exchange rate information
type DHLExchangeRate struct {
	CurrentExchangeRate float64 `json:"currentExchangeRate"`
	Currency            string  `json:"currency"`
	BaseCurrency        string  `json:"baseCurrency"`
}

// DHLRequest represents the request format for DHL API
type DHLRequest struct {
	CustomerDetails         DHLCustomerDetails       `json:"customerDetails"`
	Accounts                []DHLAccount             `json:"accounts"`
	ProductsAndServices     []DHLProductService      `json:"productsAndServices"`
	PayerCountryCode        string                   `json:"payerCountryCode"`
	PlannedShippingDateTime string                   `json:"plannedShippingDateAndTime"`
	UnitOfMeasurement       string                   `json:"unitOfMeasurement"`
	IsCustomsDeclarable     bool                     `json:"isCustomsDeclarable"`
	EstimatedDeliveryDate   DHLEstimatedDeliveryDate `json:"estimatedDeliveryDate"`
	ReturnStandardProducts  bool                     `json:"returnStandardProductsOnly"`
	Packages                []DHLPackage             `json:"packages"`
}

// DHLCustomerDetails represents customer details in DHL request
type DHLCustomerDetails struct {
	ShipperDetails  DHLLocationDetails `json:"shipperDetails"`
	ReceiverDetails DHLLocationDetails `json:"receiverDetails"`
}

// DHLLocationDetails represents location details
type DHLLocationDetails struct {
	PostalCode  string `json:"postalCode"`
	CityName    string `json:"cityName"`
	CountryCode string `json:"countryCode"`
}

// DHLAccount represents account information
type DHLAccount struct {
	TypeCode string `json:"typeCode"`
	Number   string `json:"number"`
}

// DHLProductService represents product/service selection
type DHLProductService struct {
	ProductCode      string `json:"productCode"`
	LocalProductCode string `json:"localProductCode"`
}

// DHLEstimatedDeliveryDate represents delivery date configuration
type DHLEstimatedDeliveryDate struct {
	IsRequested bool   `json:"isRequested"`
	TypeCode    string `json:"typeCode"`
}

// DHLPackage represents package information
type DHLPackage struct {
	Weight     float64       `json:"weight"`
	Dimensions DHLDimensions `json:"dimensions"`
}

// DHLDimensions represents package dimensions
type DHLDimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
