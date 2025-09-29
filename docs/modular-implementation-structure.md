# Modular Implementation Structure

This document explains the modular approach used for organizing partner implementations within the rate service.

## Directory Structure Overview

```
internal/services/v1/implementations/
├── pre_defined/                    # Pre-defined rate implementations (unified API-based)
│   ├── base.go                     # Base implementation for pre-defined rates
│   └── unified_rate/               # Single Unified Rate module for ALL predefined partners
│       ├── service.go              # Main Unified Rate service implementation
│       ├── config.go               # Configuration management
│       └── models.go               # Unified Rate API data models
└── real_time/                      # Real-time API implementations
    ├── base.go                     # Base implementation for real-time rates
    ├── dhl/                        # DHL Express module (FULLY IMPLEMENTED ✅)
    │   ├── service.go              # Main service implementation
    │   ├── config.go               # DHL-specific configuration
    │   └── models.go               # DHL API data models
    ├── fedex/                      # FedEx module (placeholder)
    │   ├── service.go              # Service skeleton
    │   └── config.go               # Configuration skeleton
    └── ups/                        # UPS module (placeholder)
        ├── service.go              # Service skeleton
        └── config.go               # Configuration skeleton
```

## Modular Design Principles

### 1. **Partner Isolation**
Each partner has its own dedicated directory containing all related files:
- Service implementation
- Configuration management
- Partner-specific data models
- Rate matrices (for pre-defined implementations)
- Utility functions (if needed)

### 2. **No "Implementation" Suffix**
Following clean naming conventions:
- ✅ `dhl/service.go` - Clean, modular
- ❌ `dhl_implementation.go` - Old monolithic approach

### 3. **Partner-Specific Organization**
Each partner module can contain:

```
partner_name/
├── service.go          # Main implementation (required)
├── config.go          # Configuration management (required)
├── models.go          # API/data models (optional)
├── rate_matrix.go     # Rate data structures (for pre-defined)
├── utils.go           # Partner-specific utilities (optional)
├── client.go          # API client wrapper (optional)
└── constants.go       # Partner constants (optional)
```

### 4. **Consistent Interface**
All partner services implement the same interface (`interfaces.RateImplementation`):
- `GetRates()` - Fetch rates
- `IsHealthy()` - Health check
- `Initialize()` - Configuration setup
- `Close()` - Graceful shutdown
- `GetImplementationType()` - Provider type
- `GetImplementationName()` - Display name

## Implementation Status

### ✅ **Fully Implemented**
- **DHL Express** (`real_time/dhl/`)
  - Complete API integration
  - Configuration management
  - Data models for request/response
  - Health checks
  - Error handling with metrics
  - Tested and working

- **Unified Rate** (`pre_defined/unified_rate/`)
  - Single service for ALL predefined rate cards
  - Manages Delhivery, Porter, and other predefined partners
  - CRUD operations for rate card management
  - Complete API integration with authentication
  - Configuration management and health checks
  - Tested and working

### 🚧 **Placeholder Structure**
- **FedEx** (`real_time/fedex/`)
- **UPS** (`real_time/ups/`)

## Partner Integration Examples

### Real-Time Implementation (DHL)

```go
// Create DHL service
dhlService := dhl.NewService(logger, metrics, httpClient)

// Initialize with custom config
config := map[string]interface{}{
    "environment": "production",
    "timeout_ms": 25000,
}
err := dhlService.Initialize(config)

// Get rates
response, err := dhlService.GetRates(ctx, request)
```

### Pre-Defined Implementation (Unified Rate for ALL Partners)

```go
// Create Unified Rate service for ALL predefined partners (Porter, Delhivery, etc.)
unifiedRateService := unified_rate.NewService(logger, metrics, httpClient)

// Initialize with Unified Rate API configuration
config := map[string]interface{}{
    "base_url": "https://sandbox-apis.prayog.io/gateway/ure/api",
    "api_key": "prayog_live_HQT-eUnEwqJ7x4LLqo-n41WGol8w-xZG_4082bc3a",
    "timeout_ms": 30000,
}
err := unifiedRateService.Initialize(config)

// Get rates from Unified Rate API (supports Porter, Delhivery, etc.)
response, err := unifiedRateService.GetRates(ctx, request)
```

## Service Registration

Partners are dynamically created based on partner codes in `rate_service.go`:

```go
func (s *RateService) createImplementationByCode(normalizedCode string) (interfaces.RateImplementation, error) {
    switch normalizedCode {
    case "dhl":
        return dhl.NewService(s.logger, s.metrics, s.httpClient), nil
    case "fedex":
        return fedex.NewService(s.logger, s.metrics, s.httpClient), nil
    case "delhivery":
        // Redirect to Unified Rate service
        return unified_rate.NewService(s.logger, s.metrics, s.httpClient), nil
    case "porter":
        // Redirect to Unified Rate service  
        return unified_rate.NewService(s.logger, s.metrics, s.httpClient), nil
    case "unified", "prayog":
        // Direct access to Unified Rate service
        return unified_rate.NewService(s.logger, s.metrics, s.httpClient), nil
    // All predefined partners route through Unified Rate service
    }
}
```

## Benefits of Modular Approach

### 🎯 **Maintainability**
- Each partner's code is isolated and self-contained
- Easy to modify one partner without affecting others
- Clear separation of concerns

### 🔄 **Scalability**
- Easy to add new partners by creating new modules
- No risk of naming conflicts
- Parallel development by different teams

### 🧪 **Testability**
- Each module can be independently tested
- Mock partner-specific dependencies easily
- Isolated integration tests

### 📦 **Deployability**
- Ability to enable/disable partners independently
- Partner-specific configuration management
- Easier debugging and monitoring

## Adding New Partners

### Step 1: Create Module Directory
```bash
mkdir -p internal/services/v1/implementations/real_time/new_partner
```

### Step 2: Implement Required Files
```go
// service.go
package new_partner

type Service struct {
    // Partner-specific fields
}

func NewService(...) *Service {
    // Implementation
}

// Implement all interface methods...
```

### Step 3: Register Partner
Add to `createImplementationByCode()` method in `rate_service.go`

### Step 4: Add Partner Recognition
Update partner utilities in `utils/partner_utils.go`

## Configuration Management

Each partner manages its own configuration:

```go
// DHL Configuration
type Config struct {
    BaseURL       string `json:"base_url"`
    Credentials   string `json:"credentials"`
    AccountNumber string `json:"account_number"`
    Environment   string `json:"environment"`
}

// Porter Configuration  
type Config struct {
    ExcelFilePath   string `json:"excel_file_path"`
    RefreshInterval int    `json:"refresh_interval_minutes"`
    DefaultCurrency string `json:"default_currency"`
}
```

## Best Practices

### ✅ **Do:**
- Keep each partner module self-contained
- Use consistent naming conventions
- Implement proper error handling and logging
- Add comprehensive tests for each module
- Document partner-specific behavior

### ❌ **Don't:**
- Mix partner logic in shared files
- Use generic names like "implementation" in filenames
- Create dependencies between partner modules
- Hardcode partner-specific values in shared code

---

This modular approach ensures the rate service can scale efficiently while maintaining clean, maintainable code for each shipping partner integration.
