# Unified Rate Integration Guide

## Overview
This document describes the integration of **Unified Rate** as the **single unified service** for ALL predefined rate calculations in the rate service. The Unified Rate implementation is the ONLY predefined module that manages rate cards for all partners (Delhivery, Porter, etc.) through CRUD operations via the Prayog Unified API.

## Architecture

### Modular Structure
```
internal/services/v1/implementations/pre_defined/unified_rate/
├── service.go      # Main Unified Rate service implementation
├── config.go       # Configuration management
└── models.go       # Data models for API communication
```

### Integration Flow
```
Quote Request → Partner Code Recognition → Unified Rate Implementation → Prayog Unified API → Response Mapping → Quote Response
```

## Configuration

### API Settings
- **Base URL**: `https://sandbox-apis.prayog.io/gateway/ure/api`
- **Endpoint**: `/external-rate-calculation/calculate`
- **Authentication**: API Key header (`api_key`)
- **API Key**: `prayog_live_HQT-eUnEwqJ7x4LLqo-n41WGol8w-xZG_4082bc3a`

### Default Configuration
```json
{
  "base_url": "https://sandbox-apis.prayog.io/gateway/ure/api",
  "api_key": "prayog_live_HQT-eUnEwqJ7x4LLqo-n41WGol8w-xZG_4082bc3a",
  "calculate_rates_endpoint": "/external-rate-calculation/calculate",
  "timeout_ms": 30000,
  "retry_count": 2,
  "cache_ttl_minutes": 15,
  "default_currency": "INR",
  "supported_currencies": ["INR", "USD", "EUR", "GBP"],
  "default_service_types": ["standard", "express"],
  "enable_all_services": true
}
```

## Partner Code Recognition

The Unified Rate implementation handles ALL predefined partner codes:
- `unified` - Direct access to Unified Rate service
- `prayog` - Alternative code for Unified Rate services  
- `delhivery` - Redirected to Unified Rate service
- `porter` - Redirected to Unified Rate service
- Any other predefined partner - Managed through Unified Rate service

All predefined partner codes route to the same Unified Rate service instance, which manages rate cards for all partners through the Prayog Unified API.

## Unified Rate Card Management

### Single Service Architecture
The Unified Rate service acts as the single microservice for ALL predefined rate calculations:
- **Delhivery rates** → Managed via unified rate cards
- **Porter rates** → Managed via unified rate cards  
- **Any future predefined partner** → Managed via unified rate cards

### Rate Card CRUD Operations
The Unified Rate service supports full CRUD operations for rate cards:
- **Create**: Add new rate cards for partners
- **Read**: Fetch rates for calculations  
- **Update**: Modify existing rate cards
- **Delete**: Remove obsolete rate cards

### Partner-Agnostic Rate Calculation
When a request comes in for any predefined partner (e.g., "delhivery", "porter"), the Unified Rate service:
1. Uses the same unified API endpoint
2. Applies the appropriate rate card logic
3. Returns partner-specific rates through the unified response format

## Request/Response Flow

### 1. Input Request Format
```json
{
  "source_location": {
    "postal_code": "560001",
    "country_code": "IN"
  },
  "destination_location": {
    "postal_code": "110001", 
    "country_code": "IN"
  },
  "packages": [
    {
      "weight": {
        "value": 1.5,
        "unit": "kg"
      },
      "dimensions": {
        "length": 20.0,
        "width": 15.0,
        "height": 10.0,
        "unit": "cm"
      }
    }
  ],
  "partners": [
    {
      "id": "",
      "code": "unified"
    }
  ],
  "metadata": {
    "currency": "INR",
    "service_type": "standard"
  }
}
```

### 2. Unified API Request Format
```json
{
  "sourceLocation": {
    "postalCode": "560001",
    "countryCode": "IN"
  },
  "destinationLocation": {
    "postalCode": "110001",
    "countryCode": "IN"
  },
  "packages": [
    {
      "weight": {
        "value": 1500,
        "unit": "GRAMS"
      },
      "dimensions": {
        "length": 10,
        "width": 10,
        "height": 10,
        "unit": "cm"
      }
    }
  ],
  "serviceTypes": ["standard"],
  "currency": "INR"
}
```

### 3. Expected Response Format
```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": {
    "request_id": "req-12345",
    "partner_rates": [
      {
        "partner": {
          "id": "",
          "code": "unified"
        },
        "success": true,
        "available_rates": [
          {
            "rate_id": "unified_standard_0_0",
            "service": "Standard Delivery",
            "price": {
              "currency": "INR",
              "amount": 45.50,
              "type": "weight_distance_based",
              "criteria": {
                "weight": "1.5 kg",
                "service_type": "standard"
              }
            },
            "delivery_days": 2
          }
        ],
        "data_source": "pre_defined",
        "response_time_ms": 850
      }
    ],
    "summary": {
      "total_partners": 1,
      "successful_partners": 1,
      "total_rates_found": 1
    },
    "retrieved_at": "2025-09-23T13:30:00Z"
  }
}
```

## Testing

### 1. Direct API Test
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "source_location": {
      "postal_code": "560001",
      "country_code": "IN"
    },
    "destination_location": {
      "postal_code": "110001",
      "country_code": "IN"
    },
    "packages": [
      {
        "weight": {
          "value": 1.5,
          "unit": "kg"
        },
        "dimensions": {
          "length": 20.0,
          "width": 15.0,
          "height": 10.0,
          "unit": "cm"
        }
      }
    ],
    "partners": [
      {
        "id": "",
        "code": "unified"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "standard"
    }
  }'
```

### 2. Health Check Test
```bash
curl -X GET http://localhost:9046/supply-rate/v1/implementations/health \
  -H "Content-Type: application/json"
```

### 3. Testing Delhivery (Through Unified Rate Service)
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "source_location": {
      "postal_code": "400001",
      "country_code": "IN"
    },
    "destination_location": {
      "postal_code": "500001",
      "country_code": "IN"
    },
    "packages": [
      {
        "weight": {
          "value": 2.0,
          "unit": "kg"
        },
        "dimensions": {
          "length": 25.0,
          "width": 20.0,
          "height": 15.0,
          "unit": "cm"
        }
      }
    ],
    "partners": [
      {
        "id": "",
        "code": "delhivery"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "standard"
    }
  }'
```

### 4. Testing Porter (Through Unified Rate Service)  
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "source_location": {
      "postal_code": "560001",
      "country_code": "IN"
    },
    "destination_location": {
      "postal_code": "560100",
      "country_code": "IN"
    },
    "packages": [
      {
        "weight": {
          "value": 1.0,
          "unit": "kg"
        },
        "dimensions": {
          "length": 15.0,
          "width": 15.0,
          "height": 10.0,
          "unit": "cm"
        }
      }
    ],
    "partners": [
      {
        "id": "",
        "code": "porter"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "express"
    }
  }'
```

## Features

### 1. Weight Conversion
- Automatically converts weight units to grams for unified API
- Supports kg, g, lb input formats
- Handles decimal values accurately

### 2. Service Type Mapping
- Maps internal service types to unified API service types
- Supports: `standard`, `express`, `premium`
- Allows multiple service type requests

### 3. Price Breakdown
- Detailed charge breakdown from unified API
- Maps charges to standard categories:
  - Base Price
  - Weight Charges
  - Distance Charges
  - Fuel Surcharge
  - Handling Charges
  - Insurance Charges
  - Tax Amount

### 4. Error Handling
- Comprehensive error handling for API failures
- Network timeout management
- Retry logic for failed requests
- Detailed error messages and codes

### 5. Health Monitoring
- Built-in health check functionality
- API connectivity verification
- Performance metrics collection
- Response time tracking

## Supported Charge Types

The unified API supports various charge types from rate cards:
- **FIXED**: Fixed amount charges
- **PERCENTAGE**: Percentage-based charges
- **FORMULA**: Formula-based calculations
- **SLAB**: Weight/distance slab-based charges

## Rate Card Integration

The system can work with rate cards that include:
- Matrix-based pricing (weight, location, service type)
- Complex charge structures
- Tax calculations
- Dynamic pricing rules
- Multiple currency support

## Performance Considerations

### Timeout Configuration
- Default timeout: 30 seconds
- Configurable per implementation
- Retry count: 2 attempts

### Caching Strategy
- Response caching: 15 minutes default
- Configurable TTL
- Cache invalidation on errors

### Rate Limiting
- API key based rate limiting
- Automatic backoff on rate limit hits
- Usage tracking and monitoring

## Error Scenarios

### 1. API Connection Errors
```json
{
  "partner_rates": [
    {
      "partner": {"id": "", "code": "unified"},
      "success": false,
      "error": {
        "code": "RATE_FETCH_FAILED",
        "message": "Failed to fetch rates from partner",
        "details": "HTTP request failed: connection timeout"
      },
      "data_source": "pre_defined",
      "response_time_ms": 30000
    }
  ]
}
```

### 2. Authentication Errors
```json
{
  "error": {
    "code": "AUTH_FAILED",
    "message": "API authentication failed",
    "details": "Invalid API key provided"
  }
}
```

### 3. Validation Errors
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": "Invalid postal code format"
  }
}
```

## Monitoring and Metrics

The unified implementation tracks:
- Request success/failure rates
- Response times
- API call counts
- Error rates by type
- Cache hit/miss ratios

## Future Enhancements

### Planned Features
1. **Multi-currency Support**: Enhanced currency conversion
2. **Location Intelligence**: Better postal code parsing
3. **Bulk Operations**: Multiple package support
4. **Advanced Caching**: Intelligent cache strategies
5. **Rate Card Management**: CRUD operations for rate cards

### Partner Extensions
- Support for partner-specific configurations
- Custom charge calculations
- Location-based routing
- Service level customizations

This integration provides a robust, scalable foundation for accessing Prayog's unified rate calculation services while maintaining consistency with the existing rate service architecture.
