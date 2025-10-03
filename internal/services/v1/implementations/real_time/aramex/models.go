package aramex

import "encoding/json"

// AramexRateRequest represents the request structure for Aramex API
type AramexRateRequest struct {
    ClientInfo           ClientInfo           `json:"ClientInfo"`
    OriginAddress        Address              `json:"OriginAddress"`
    DestinationAddress   Address              `json:"DestinationAddress"`
    ShipmentDetails      ShipmentDetails      `json:"ShipmentDetails"`
}

type ClientInfo struct {
    UserName           string `json:"UserName"`
    Password           string `json:"Password"`
    Version            string `json:"Version"`
    AccountNumber      string `json:"AccountNumber"`
    AccountPin         string `json:"AccountPin"`
    AccountEntity      string `json:"AccountEntity"`
    AccountCountryCode string `json:"AccountCountryCode"`
    Source             int    `json:"Source"`
}

type Address struct {
    Line1              string `json:"Line1"`
    Line2              string `json:"Line2"`
    Line3              string `json:"Line3"`
    City               string `json:"City"`
    StateOrProvinceCode string `json:"StateOrProvinceCode"`
    PostCode           string `json:"PostCode"`
    CountryCode        string `json:"CountryCode"`
}

type ShipmentDetails struct {
    Dimensions         Dimensions     `json:"Dimensions"`
    ActualWeight       Weight         `json:"ActualWeight"`
    ChargeableWeight   *Weight        `json:"ChargeableWeight,omitempty"`
    DescriptionOfGoods string         `json:"DescriptionOfGoods"`
    GoodsOriginCountry string         `json:"GoodsOriginCountry"`
    NumberOfPieces     int            `json:"NumberOfPieces"`
    ProductGroup       string         `json:"ProductGroup"`
    ProductType        string         `json:"ProductType"`
    PaymentType        string         `json:"PaymentType"`
    PaymentOptions     string         `json:"PaymentOptions"`
    Services           string         `json:"Services"`
}

type Dimensions struct {
    Length float64 `json:"Length"`
    Width  float64 `json:"Width"`
    Height float64 `json:"Height"`
    Unit   string  `json:"Unit"`
}

type Weight struct {
    Unit  string  `json:"Unit"`
    Value float64 `json:"Value"`
}

// AramexRateResponse represents the response structure from Aramex API
type AramexRateResponse struct {
    Transaction  interface{} `json:"Transaction"`
    Notifications interface{} `json:"Notifications"`
    HasErrors    bool        `json:"HasErrors"`
    TotalAmount  Amount      `json:"TotalAmount"`
    RateDetails  RateDetails `json:"RateDetails"`
}

type Amount struct {
    CurrencyCode string  `json:"CurrencyCode"`
    Value        float64 `json:"Value"`
}

type RateDetails struct {
    Amount              float64 `json:"Amount"`
    OtherAmount1        float64 `json:"OtherAmount1"`
    OtherAmount2        float64 `json:"OtherAmount2"`
    OtherAmount3        float64 `json:"OtherAmount3"`
    OtherAmount4        float64 `json:"OtherAmount4"`
    OtherAmount5        float64 `json:"OtherAmount5"`
    TotalAmountBeforeTax float64 `json:"TotalAmountBeforeTax"`
    TaxAmount           float64 `json:"TaxAmount"`
}

// In models.go, add custom marshaling for ShipmentDetails
func (sd ShipmentDetails) MarshalJSON() ([]byte, error) {
    type Alias ShipmentDetails
    return json.Marshal(&struct {
        ChargeableWeight *Weight `json:"ChargeableWeight"`
        *Alias
    }{
        ChargeableWeight: sd.ChargeableWeight,
        Alias:           (*Alias)(&sd),
    })
}