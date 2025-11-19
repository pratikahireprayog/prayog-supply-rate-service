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

// CalculatePrice calculates the total price for given weight and distance
// This implements weight-based pricing similar to Shipcube
func (r *BaralRate) CalculatePrice(weightKg float64, distanceKm float64) float64 {
	minimumFreight := r.GetMinimumFreightINR()
	minimumWeight := r.GetMinimumWeightKg()
	
	// Use minimum weight if actual weight is less
	actualWeight := weightKg
	if weightKg < minimumWeight && minimumWeight > 0 {
		actualWeight = minimumWeight
	}
	
	// Base price calculation: minimum freight + weight-based charge
	// For weight-based: assume per kg rate based on minimum freight
	// This is a simplified calculation - in production, you'd have weight slabs
	basePrice := minimumFreight
	
	// If weight exceeds minimum, add additional weight charges
	// Formula: basePrice + (excess_weight * per_kg_rate)
	// Per kg rate is estimated as minimum_freight / minimum_weight
	if actualWeight > minimumWeight && minimumWeight > 0 {
		perKgRate := minimumFreight / minimumWeight
		excessWeight := actualWeight - minimumWeight
		weightCharge := excessWeight * perKgRate
		basePrice = minimumFreight + weightCharge
	}
	
	// Apply distance-based ODA charges if applicable
	odaCharge := r.CalculateODACharge(weightKg, distanceKm)
	
	// Add fuel surcharge (percentage of base price)
	fuelSurchargePercent := r.GetFuelSurchargePercent()
	fuelSurcharge := basePrice * (fuelSurchargePercent / 100)
	
	// Add docket charges
	docketCharges := r.GetDocketCharges()
	
	// Add green tax if applicable
	greenTax := r.GetGreenTax()
	
	// Add platform fee if applicable
	platformFee := r.GetPlatformFee()
	
	// Calculate total
	totalPrice := basePrice + fuelSurcharge + docketCharges + odaCharge + greenTax + platformFee
	
	return totalPrice
}

// CalculateODACharge calculates Out of Delivery Area charges based on distance
func (r *BaralRate) CalculateODACharge(weightKg float64, distanceKm float64) float64 {
	// ODA charges apply for distances beyond certain thresholds
	// ODA1: typically for distances 50-100 km
	// ODA2: typically for distances > 100 km
	
	if distanceKm <= 50 {
		// No ODA charge for local deliveries
		return 0
	}
	
	if distanceKm > 50 && distanceKm <= 100 {
		// Apply ODA1 charges
		if r.ODA.ODA1.PerKgAmount > 0 {
			return r.ODA.ODA1.PerKgAmount * weightKg
		}
		return r.ODA.ODA1.TotalAmount
	}
	
	if distanceKm > 100 {
		// Apply ODA2 charges (higher)
		if r.ODA.ODA2.PerKgAmount > 0 {
			return r.ODA.ODA2.PerKgAmount * weightKg
		}
		return r.ODA.ODA2.TotalAmount
	}
	
	return 0
}

// GetGreenTax returns green tax amount
func (r *BaralRate) GetGreenTax() float64 {
	if r.AdditionalCharges.GreenTax == "" {
		return 0
	}
	var tax float64
	fmt.Sscanf(r.AdditionalCharges.GreenTax, "%f", &tax)
	return tax
}

// GetPlatformFee returns platform fee amount
func (r *BaralRate) GetPlatformFee() float64 {
	if r.AdditionalCharges.PlatformFee == "" {
		return 0
	}
	var fee float64
	fmt.Sscanf(r.AdditionalCharges.PlatformFee, "%f", &fee)
	return fee
}

