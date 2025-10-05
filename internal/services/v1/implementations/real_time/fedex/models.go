package fedex

// FedexRateRequest represents the request structure for FedEx API
type FedexRateRequest struct {
    AccountNumber struct {
        Value string `json:"value"`
    } `json:"accountNumber"`
    RequestedShipment struct {
        Shipper struct {
            Address struct {
                PostalCode  string `json:"postalCode"`
                CountryCode string `json:"countryCode"`
            } `json:"address"`
        } `json:"shipper"`
        Recipient struct {
            Address struct {
                PostalCode  string `json:"postalCode"`
                CountryCode string `json:"countryCode"`
            } `json:"address"`
        } `json:"recipient"`
        PickupType          string   `json:"pickupType"`
        RateRequestType     []string `json:"rateRequestType"`
        PreferredCurrency   string   `json:"preferredCurrency"`
        PackageCount        int      `json:"packageCount"`
        RequestedPackageLineItems []RequestedPackageLineItem `json:"requestedPackageLineItems"`
    } `json:"requestedShipment"`
}

type RequestedPackageLineItem struct {
    GroupPackageCount int `json:"groupPackageCount"`
    Weight struct {
        Units string  `json:"units"`
        Value float64 `json:"value"`
    } `json:"weight"`
    Dimensions struct {
        Length int    `json:"length"`
        Width  int    `json:"width"`
        Height int    `json:"height"`
        Units  string `json:"units"`
    } `json:"dimensions"`
}

// FedexRateResponse represents the response structure from FedEx API
type FedexRateResponse struct {
    TransactionID         string `json:"transactionId"`
    CustomerTransactionID string `json:"customerTransactionId"`
    Output                struct {
        RateReplyDetails []RateReplyDetail `json:"rateReplyDetails"`
    } `json:"output"`
}

type RateReplyDetail struct {
    ServiceType     string `json:"serviceType"`
    ServiceName     string `json:"serviceName"`
    PackagingType   string `json:"packagingType"`
    RatedShipmentDetails []struct {
        RateType                    string  `json:"rateType"`
        RatedWeightMethod           string  `json:"ratedWeightMethod"`
        TotalDiscounts              float64 `json:"totalDiscounts"`
        TotalBaseCharge             float64 `json:"totalBaseCharge"`
        TotalNetCharge              float64 `json:"totalNetCharge"`
        TotalVatCharge              float64 `json:"totalVatCharge"`
        TotalNetFedExCharge         float64 `json:"totalNetFedExCharge"`
        TotalDutiesAndTaxes         float64 `json:"totalDutiesAndTaxes"`
        TotalNetChargeWithDutiesAndTaxes float64 `json:"totalNetChargeWithDutiesAndTaxes"`
        TotalDutiesTaxesAndFees     float64 `json:"totalDutiesTaxesAndFees"`
        TotalAncillaryFeesAndTaxes  float64 `json:"totalAncillaryFeesAndTaxes"`
        ShipmentRateDetail struct {
            RateZone                string  `json:"rateZone"`
            DimDivisor              int     `json:"dimDivisor"`
            FuelSurchargePercent    float64 `json:"fuelSurchargePercent"`
            TotalSurcharges         float64 `json:"totalSurcharges"`
            TotalFreightDiscount    float64 `json:"totalFreightDiscount"`
            Currency                string  `json:"currency"`
            RateScale               string  `json:"rateScale"`
            TotalRateScaleWeight    struct {
                Units  string  `json:"units"`
                Value  float64 `json:"value"`
            } `json:"totalRateScaleWeight"`
        } `json:"shipmentRateDetail"`
        RatedPackages []struct {
            GroupNumber int `json:"groupNumber"`
            PackageRateDetail struct {
                RateType            string  `json:"rateType"`
                RatedWeightMethod   string  `json:"ratedWeightMethod"`
                BaseCharge          float64 `json:"baseCharge"`
                NetFreight          float64 `json:"netFreight"`
                TotalSurcharges     float64 `json:"totalSurcharges"`
                NetFedExCharge      float64 `json:"netFedExCharge"`
                TotalTaxes          float64 `json:"totalTaxes"`
                NetCharge           float64 `json:"netCharge"`
                Currency            string  `json:"currency"`
            } `json:"packageRateDetail"`
        } `json:"ratedPackages"`
        Currency string `json:"currency"`
    } `json:"ratedShipmentDetails"`
    Commitment struct {
        DateDetail struct {
            DayOfWeek string `json:"dayOfWeek"`
            Time      string `json:"time"`
        } `json:"dateDetail"`
    } `json:"commitment"`
}

// FedexTokenResponse represents the OAuth token response
type FedexTokenResponse struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    ExpiresIn   int    `json:"expires_in"`
    Scope       string `json:"scope"`
}