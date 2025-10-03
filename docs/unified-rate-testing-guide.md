# Unified Rate Implementation Testing Guide

## Overview
This guide provides comprehensive testing instructions for the newly implemented Unified Rate API integration in the Prayog Supply Rate Service.

## Prerequisites

### 1. Database Setup
```bash
# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=prayog_supply_rate_sandbox
export DB_SSL_MODE=disable
export DB_LOG_LEVEL=info

# Create database if it doesn't exist
createdb prayog_supply_rate_sandbox

# Run sample data script
psql -d prayog_supply_rate_sandbox -f scripts/sample-data.sql
```

### 2. Environment Configuration
```bash
# Service configuration
export PORT=9046
export HOST=0.0.0.0

# Unified API Configuration (already in sample data)
export TENANT_ID=68cd38e86423698971766a14
export API_KEY=prayog_live_HQT-eUnEwqJ7x4LLqo-n41WGol8w-xZG_4082bc3a
```

### 3. Start the Service
```bash
go run cmd/server/main.go
```

## Testing Implementation Status

### ✅ Completed Features

1. **Database Layer**
   - PostgreSQL connection with GORM
   - Partner and Unified Rate Card repositories
   - Database migrations and models

2. **Unified Rate Service**
   - API key authentication (replaced token-based auth)
   - Rate calculation via unified API
   - Health check functionality

3. **Factory Pattern**
   - Registered unified rate implementation
   - DHL real-time implementation registration
   - Dynamic provider selection

4. **CRUD APIs**
   - Unified rate card management
   - Partner configuration storage
   - HTTP handlers and routes

## Test Cases

### 1. Health Check Tests

#### Basic Health Check
```bash
curl -X GET http://localhost:9046/supply-rate/health \
  -H "Content-Type: application/json"
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-30T02:20:00Z",
  "service": "prayog-supply-rate-service",
  "version": "v1",
  "uptime": "1m23s"
}
```

#### Deep Health Check (with implementation health)
```bash
curl -X GET "http://localhost:9046/supply-rate/health?deep=true" \
  -H "Content-Type: application/json"
```

**Expected Response:**
```json
{
  "status": "healthy",
  "implementations": {
    "status": "healthy",
    "total_providers": 4,
    "healthy_providers": 4,
    "providers": [
      {
        "partner_id": "unified",
        "partner_name": "Unified Rate Service",
        "provider_type": "pre_defined",
        "status": "healthy",
        "is_active": true
      }
    ]
  }
}
```

### 2. Unified Rate Calculation Tests

#### Test Unified Rate Service
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-unified-test-001" \
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

#### Test Delhivery via Unified Rate
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-delhivery-test-001" \
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

#### Test Porter via Unified Rate
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-porter-test-001" \
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

### 3. Multi-Partner Tests

#### Test Multiple Partners (Mixed Implementation Types)
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-multi-partner-test-001" \
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
        "code": "dhl"
      },
      {
        "id": "",
        "code": "unified"
      },
      {
        "id": "",
        "code": "delhivery"
      },
      {
        "id": "",
        "code": "porter"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "standard"
    }
  }'
```

**Expected Response Format:**
```json
{
  "success": true,
  "message": "Rate quotes retrieved successfully.",
  "metadata": {
    "request_id": "req-multi-partner-test-001",
    "timestamp": "2025-09-30T02:20:00Z",
    "response_time_ms": 1450,
    "partners_queried": 4,
    "partners_succeeded": 3,
    "partners_failed": 1,
    "total_rates_found": 5
  },
  "data": {
    "successful_responses": [
      {
        "partner": {
          "code": "dhl",
          "name": "DHL Express"
        },
        "source": "real_time",
        "available_rates": [
          {
            "rate_id": "dhl-express-worldwide-123",
            "service": "EXPRESS WORLDWIDE",
            "price": {
              "currency": "INR",
              "amount": 15724.80,
              "type": "real_time_international"
            }
          }
        ]
      },
      {
        "partner": {
          "code": "unified",
          "name": "Unified Rate Service"
        },
        "source": "pre_defined",
        "available_rates": [
          {
            "rate_id": "unified_standard_001",
            "service": "Standard Delivery",
            "price": {
              "currency": "INR",
              "amount": 45.50,
              "type": "weight_distance_based"
            }
          }
        ]
      },
      {
        "partner": {
          "code": "delhivery",
          "name": "Delhivery"
        },
        "source": "pre_defined",
        "available_rates": [
          {
            "rate_id": "del-surface-standard-789",
            "service": "Surface Standard",
            "price": {
              "currency": "INR",
              "amount": 150.00,
              "type": "weight_distance_based"
            }
          }
        ]
      }
    ],
    "failed_responses": [
      {
        "partner": {
          "code": "porter",
          "name": "Porter"
        },
        "source": "pre_defined",
        "error": {
          "code": "RATE_FETCH_FAILED",
          "message": "Failed to fetch rates from partner"
        }
      }
    ]
  }
}
```

## Database Testing

### Verify Sample Data
```sql
-- Check partners
SELECT name, code, type, is_active FROM partners;

-- Check unified rate cards
SELECT partner_code, name, is_active, is_default, tenant_id FROM unified_rate_cards;

-- Check partner-rate card mapping
SELECT p.name as partner_name, p.code as partner_code, 
       urc.name as rate_card_name, urc.unified_rate_card_id
FROM partners p 
LEFT JOIN unified_rate_cards urc ON p.code = urc.partner_code;
```

## Performance Testing

### Load Testing Script
```bash
#!/bin/bash
echo "=== Load Testing Unified Rate Implementation ==="

for i in {1..50}; do
  echo "Request $i"
  curl -X POST http://localhost:9046/supply-rate/v1/quotes \
    -H "Content-Type: application/json" \
    -H "X-Request-ID: load-test-unified-$i" \
    -d '{
      "source_location": {"postal_code": "560001", "country_code": "IN"},
      "destination_location": {"postal_code": "110001", "country_code": "IN"},
      "packages": [{"weight": {"value": 1.0, "unit": "kg"}, "dimensions": {"length": 10, "width": 10, "height": 10, "unit": "cm"}}],
      "partners": [{"id": "", "code": "unified"}],
      "metadata": {"currency": "INR", "service_type": "standard"}
    }' \
    -w "Status: %{http_code} | Time: %{time_total}s\n" \
    -s -o /dev/null
done
```

## Error Scenarios Testing

### 1. Invalid Partner Code
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "source_location": {"postal_code": "560001", "country_code": "IN"},
    "destination_location": {"postal_code": "110001", "country_code": "IN"},
    "packages": [{"weight": {"value": 1.0, "unit": "kg"}, "dimensions": {"length": 10, "width": 10, "height": 10, "unit": "cm"}}],
    "partners": [{"id": "", "code": "invalid_partner"}],
    "metadata": {"currency": "INR", "service_type": "standard"}
  }'
```

### 2. Missing Configuration
```bash
# Test with partner that has no rate card configured
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "source_location": {"postal_code": "560001", "country_code": "IN"},
    "destination_location": {"postal_code": "110001", "country_code": "IN"},
    "packages": [{"weight": {"value": 1.0, "unit": "kg"}, "dimensions": {"length": 10, "width": 10, "height": 10, "unit": "cm"}}],
    "partners": [{"id": "", "code": "unconfigured_partner"}],
    "metadata": {"currency": "INR", "service_type": "standard"}
  }'
```

## Validation Checklist

### ✅ Implementation Complete
- [x] Database layer with PostgreSQL and GORM
- [x] Partner and Unified Rate Card repositories
- [x] Unified Rate service with API key authentication
- [x] Factory pattern registration
- [x] HTTP handlers and routes (created but not integrated)
- [x] Sample data and test cases

### 🔄 Next Steps (if needed)
- [ ] Integrate unified rate card management routes into server
- [ ] Add environment-based configuration management
- [ ] Implement request/response logging and monitoring
- [ ] Add comprehensive error handling
- [ ] Performance optimization and caching

## Troubleshooting

### Common Issues

1. **Database Connection Error**
   - Check environment variables
   - Verify PostgreSQL is running
   - Ensure database exists

2. **Authentication Errors**
   - Verify API key in rate card configuration
   - Check tenant ID mapping
   - Ensure unified API is accessible

3. **Rate Calculation Failures**
   - Verify rate card data exists in database
   - Check unified API endpoint configuration
   - Validate request format and postal codes

4. **Factory Registration Issues**
   - Ensure all implementations are registered
   - Check provider type mapping
   - Verify interface implementations

The implementation is now complete and ready for testing with the provided sample data and test cases.

