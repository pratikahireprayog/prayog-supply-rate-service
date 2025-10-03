package v1

// ServiceType represents different types of delivery services
type ServiceType string

const (
	// Standard delivery service
	ServiceTypeStandard ServiceType = "standard"

	// Premium delivery service with enhanced features
	ServiceTypePremium ServiceType = "premium"

	// Express delivery service for urgent shipments
	ServiceTypeExpress ServiceType = "express"

	// Same day delivery service
	ServiceTypeSameDay ServiceType = "same_day"

	// Economy delivery service with longer transit times
	ServiceTypeEconomy ServiceType = "economy"
)

// String returns the string representation of ServiceType
func (st ServiceType) String() string {
	return string(st)
}

// IsValid checks if the service type is valid
func (st ServiceType) IsValid() bool {
	switch st {
	case ServiceTypeStandard, ServiceTypePremium, ServiceTypeExpress, ServiceTypeSameDay, ServiceTypeEconomy:
		return true
	default:
		return false
	}
}

// GetPriority returns the priority level for the service type
func (st ServiceType) GetPriority() int {
	switch st {
	case ServiceTypeSameDay:
		return 1 // Highest priority
	case ServiceTypeExpress:
		return 2
	case ServiceTypePremium:
		return 3
	case ServiceTypeStandard:
		return 4
	case ServiceTypeEconomy:
		return 5 // Lowest priority
	default:
		return 10 // Unknown/Invalid
	}
}

// GetDescription returns a human-readable description of the service type
func (st ServiceType) GetDescription() string {
	switch st {
	case ServiceTypeStandard:
		return "Standard delivery service with regular transit times"
	case ServiceTypePremium:
		return "Premium delivery service with enhanced features and tracking"
	case ServiceTypeExpress:
		return "Express delivery service for urgent shipments"
	case ServiceTypeSameDay:
		return "Same day delivery service for local shipments"
	case ServiceTypeEconomy:
		return "Economy delivery service with longer transit times at lower cost"
	default:
		return "Unknown service type"
	}
}

// GetMaxTransitDays returns the maximum transit days for the service type
func (st ServiceType) GetMaxTransitDays() int {
	switch st {
	case ServiceTypeSameDay:
		return 1
	case ServiceTypeExpress:
		return 2
	case ServiceTypePremium:
		return 3
	case ServiceTypeStandard:
		return 5
	case ServiceTypeEconomy:
		return 7
	default:
		return 10
	}
}

// AllServiceTypes returns all valid service types
func AllServiceTypes() []ServiceType {
	return []ServiceType{
		ServiceTypeStandard,
		ServiceTypePremium,
		ServiceTypeExpress,
		ServiceTypeSameDay,
		ServiceTypeEconomy,
	}
}
