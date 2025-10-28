package baral_rate

import (
	"fmt"
	"time"
)

// BaralAPIResponse represents the response from Baral API
type BaralAPIResponse struct {
	Status    int            `json:"status"`
	Message   string         `json:"message,omitempty"`
	Data      *BaralData     `json:"data,omitempty"`
	Error     *BaralAPIError `json:"error,omitempty"`
	Timestamp time.Time      `json:"timestamp,omitempty"`
}

// BaralData represents the data object in Baral API response
type BaralData struct {
	Count             int         `json:"count"`
	RateCard          []BaralRate `json:"rateCard"`
	ActivateRateCard  interface{} `json:"activateRateCard"`
}

// BaralAPIError represents an error from Baral API
type BaralAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// BaralRate represents a single rate card from Baral
type BaralRate struct {
	ID              string                 `json:"_id,omitempty"`
	RateCardName    string                 `json:"rateCardName,omitempty"`
	PartnerCode     string                 `json:"partnerCode,omitempty"`
	RateCardType    string                 `json:"rateCardType,omitempty"` // surface, air, etc.
	IsPrimary       bool                   `json:"isPrimary,omitempty"`
	IsEnable        bool                   `json:"isEnable,omitempty"`
	
	// Weight Information (as strings in API)
	MinimumWeight   string `json:"minimumWeight,omitempty"`
	MinimumFreight  string `json:"minimumFreight,omitempty"`
	
	// Volumetric Weight
	VolumetricWeight struct {
		VolumetricWeightValue string `json:"volumetricWeightValue,omitempty"`
		VolumetricWeightName  string `json:"volumetricWeightName,omitempty"`
		ID                    string `json:"_id,omitempty"`
	} `json:"volumetricWeight,omitempty"`
	
	// GST Information
	GST struct {
		CGST string `json:"cGST,omitempty"`
		IGST string `json:"iGST,omitempty"`
	} `json:"GST,omitempty"`
	
	// Additional Charges (as strings in API)
	AdditionalCharges struct {
		FuelSurcharge         string `json:"fuelSurcharge,omitempty"`
		DocketCharges         string `json:"docketCharges,omitempty"`
		MaxLiabilityPerDocket string `json:"maxLiabilityPerDocket,omitempty"`
		GreenTax              string `json:"greenTax,omitempty"`
		PlatformFee           string `json:"platformFee,omitempty"`
	} `json:"additionalCharges,omitempty"`
	
	// Appointment Delivery
	AppointmentDelivery struct {
		DeliveryCharge string `json:"deliveryCharge,omitempty"`
	} `json:"appointmentDelivery,omitempty"`
	
	// ODA (Out of Delivery Area) charges
	ODA struct {
		ODA1 struct {
			PerKgAmount  float64 `json:"perKgAmount,omitempty"`
			TotalAmount  float64 `json:"totalAmount,omitempty"`
			ID           string  `json:"_id,omitempty"`
		} `json:"ODA1,omitempty"`
		ODA2 struct {
			PerKgAmount  float64 `json:"perKgAmount,omitempty"`
			TotalAmount  float64 `json:"totalAmount,omitempty"`
			ID           string  `json:"_id,omitempty"`
		} `json:"ODA2,omitempty"`
		ID string `json:"_id,omitempty"`
	} `json:"ODA,omitempty"`
	
	// Insurance
	Insurance struct {
		CarrierInsurance struct {
			InsurancePercentage float64 `json:"insurancePercentage,omitempty"`
			InsuranceCommission float64 `json:"insuranceCommission,omitempty"`
			ID                  string  `json:"_id,omitempty"`
		} `json:"carrierInsurance,omitempty"`
		SelfInsurance struct {
			InsurancePercentage float64 `json:"insurancePercentage,omitempty"`
			InsuranceCommission float64 `json:"insuranceCommission,omitempty"`
			ID                  string  `json:"_id,omitempty"`
		} `json:"selfInsurance,omitempty"`
		ThirdPartyInsurance interface{} `json:"thirdPartyInsurance,omitempty"`
		ID                  string      `json:"_id,omitempty"`
	} `json:"insurance,omitempty"`
	
	// Partner Details
	PartnerDetails struct {
		Email       string `json:"email,omitempty"`
		Name        string `json:"name,omitempty"`
		CompanyName string `json:"companyName,omitempty"`
		Mobile      string `json:"mobile,omitempty"`
		ID          string `json:"_id,omitempty"`
	} `json:"partnerDetails,omitempty"`
	
	// Timestamps
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	Version   int       `json:"__v,omitempty"`
}

// IsValid checks if the rate is currently valid and enabled
func (r *BaralRate) IsValid() bool {
	return r.IsEnable
}

// GetMinimumWeightKg returns the minimum weight in kg (converted from string)
func (r *BaralRate) GetMinimumWeightKg() float64 {
	if r.MinimumWeight == "" {
		return 0
	}
	var weight float64
	fmt.Sscanf(r.MinimumWeight, "%f", &weight)
	return weight
}

// GetMinimumFreightINR returns the minimum freight charge in INR
func (r *BaralRate) GetMinimumFreightINR() float64 {
	if r.MinimumFreight == "" {
		return 0
	}
	var freight float64
	fmt.Sscanf(r.MinimumFreight, "%f", &freight)
	return freight
}

// GetFuelSurchargePercent returns fuel surcharge as percentage
func (r *BaralRate) GetFuelSurchargePercent() float64 {
	if r.AdditionalCharges.FuelSurcharge == "" {
		return 0
	}
	var surcharge float64
	fmt.Sscanf(r.AdditionalCharges.FuelSurcharge, "%f", &surcharge)
	return surcharge
}

// GetDocketCharges returns docket charges
func (r *BaralRate) GetDocketCharges() float64 {
	if r.AdditionalCharges.DocketCharges == "" {
		return 0
	}
	var charges float64
	fmt.Sscanf(r.AdditionalCharges.DocketCharges, "%f", &charges)
	return charges
}

// CalculatePrice calculates the total price for given weight
func (r *BaralRate) CalculatePrice(weightKg float64) float64 {
	minimumFreight := r.GetMinimumFreightINR()
	
	// For now, return minimum freight as base price
	// In production, you'd implement proper rate calculation based on weight slabs
	basePrice := minimumFreight
	
	// Add fuel surcharge
	fuelSurchargePercent := r.GetFuelSurchargePercent()
	fuelSurcharge := basePrice * (fuelSurchargePercent / 100)
	
	// Add docket charges
	docketCharges := r.GetDocketCharges()
	
	// Calculate total
	totalPrice := basePrice + fuelSurcharge + docketCharges
	
	return totalPrice
}

