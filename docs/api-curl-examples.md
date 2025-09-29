# Supply Rate Service API - CURL Documentation

## Overview
This document provides comprehensive CURL examples for all APIs available in the Prayog Supply Rate Service. The service provides rate calculations from multiple shipping partners including real-time API integrations (DHL, FedEx, UPS) and pre-defined rate cards (Delhivery, Porter) through the unified rate service.

**Base URL**: `http://localhost:9046` (development)

---

## 📋 Table of Contents
1. [Health Check APIs](#health-check-apis)
2. [Quote APIs](#quote-apis) 
   - [Standardized Response Structure](#5-multi-partner-request-with-success-and-failure-examples)
3. [Metrics & Monitoring](#metrics--monitoring)
4. [API Information](#api-information)
5. [Error Scenarios](#error-scenarios)
6. [Authentication Examples](#authentication-examples)

---

## 🏥 Health Check APIs

### 1. Liveness Probe
**Endpoint**: `GET /health/live`  
**Purpose**: Kubernetes liveness probe - checks if service is running

```bash
curl -X GET http://localhost:9046/health/live \
  -H "Content-Type: application/json" \
  -w "\nStatus: %{http_code}\nTime: %{time_total}s\n"
```

**Expected Response:**
```
Status: 200
Time: 0.001s
```

### 2. Readiness Probe  
**Endpoint**: `GET /health/ready`  
**Purpose**: Kubernetes readiness probe - checks if service is ready to receive traffic

```bash
curl -X GET http://localhost:9046/health/ready \
  -H "Content-Type: application/json" \
  -w "\nStatus: %{http_code}\nTime: %{time_total}s\n"
```

**Expected Response:**
```
Status: 200  
Time: 0.002s
```

### 3. Application Health Check
**Endpoint**: `GET /supply-rate/health`  
**Purpose**: Main application health check with optional deep health checking

#### Basic Health Check
```bash
curl -X GET http://localhost:9046/supply-rate/health \
  -H "Content-Type: application/json"
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-29T10:30:00Z",
  "service": "prayog-supply-rate-service",
  "version": "v1",
  "uptime": "2h34m12s"
}
```

#### Deep Health Check (includes implementation health)
```bash
curl -X GET "http://localhost:9046/supply-rate/health?deep=true" \
  -H "Content-Type: application/json"
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-29T10:30:00Z",
  "service": "prayog-supply-rate-service", 
  "version": "v1",
  "uptime": "2h34m12s",
  "implementations": {
    "status": "healthy",
    "total_providers": 4,
    "healthy_providers": 4,
    "unhealthy_providers": 0,
    "providers": [
      {
        "partner_id": "dhl",
        "partner_name": "DHL Express",
        "provider_type": "real_time",
        "status": "healthy",
        "is_active": true,
        "last_checked": "2025-09-29T10:29:45Z",
        "response_time": 0
      }
    ],
    "checked_at": "2025-09-29T10:30:00Z",
    "response_time_ms": 1250
  }
}
```

---

## 📦 Quote APIs

> **Note**: All quote endpoints use the standardized response structure shown in [section 5](#5-multi-partner-request-with-success-and-failure-examples) below. This structure consistently separates successful responses from failed responses and provides comprehensive metadata about each request.

### 1. Get Quotes - Unified Rate Service (Pre-defined)
**Endpoint**: `POST /supply-rate/v1/quotes`  
**Purpose**: Get rates from pre-defined rate cards through unified service

#### Basic Request
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-$(uuidgen)" \
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

#### Multiple Partners Request
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-$(uuidgen)" \
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

**Expected Response:**
```json
{
  "success": true,
  "message": "Rate quotes retrieved successfully.",
  "metadata": {
    "request_id": "req-12345-67890",
    "timestamp": "2025-09-29T10:30:00Z",
    "response_time_ms": 950,
    "partners_queried": 1,
    "partners_succeeded": 1,
    "partners_failed": 0,
    "total_rates_found": 1
  },
  "data": {
    "successful_responses": [
      {
        "partner": {
          "code": "unified",
          "name": "Unified Rate"
        },
        "source": "pre_defined",
        "available_rates": [
          {
            "rate_id": "unified_standard_001",
            "service": "Standard Delivery",
            "price": {
              "currency": "INR",
              "amount": 45.50,
              "type": "weight_distance_based",
              "criteria": {
                "weight": "1.5 kg",
                "service_type": "standard",
                "route": "domestic"
              }
            },
            "delivery_days": 2
          }
        ],
        "response_time_ms": 850
      }
    ],
    "failed_responses": []
  },
  "timestamp": "2025-09-29T10:30:00Z"
}
```

### 2. Get Quotes - DHL Express (Real-time)
**Endpoint**: `POST /supply-rate/v1/quotes`  
**Purpose**: Get real-time rates from DHL Express API

```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-$(uuidgen)" \
  -d '{
    "source_location": {
      "postal_code": "560086",
      "country_code": "IN"
    },
    "destination_location": {
      "postal_code": "266001",
      "country_code": "CN"
    },
    "packages": [
      {
        "weight": {
          "value": 0.5,
          "unit": "kg"
        },
        "dimensions": {
          "length": 121.0,
          "width": 20.0,
          "height": 30.0,
          "unit": "cm"
        }
      }
    ],
    "partners": [
      {
        "id": "",
        "code": "DHL"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "express"
    }
  }'
```

**Expected Response:**
```json
{
  "success": true,
  "message": "Rate quotes retrieved successfully.",
  "metadata": {
    "request_id": "req-dhl-12345",
    "timestamp": "2025-09-29T10:30:00Z",
    "response_time_ms": 1350,
    "partners_queried": 1,
    "partners_succeeded": 1,
    "partners_failed": 0,
    "total_rates_found": 1
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
            "rate_id": "dhl_p_0",
            "service": "EXPRESS WORLDWIDE",
            "price": {
              "currency": "INR",
              "amount": 15724.8,
              "type": "international_express",
              "criteria": {
                "product_code": "P",
                "weight_charged": 14.52,
                "weight_provided": 15,
                "route": "international"
              }
            },
            "delivery_days": 7
          }
        ],
        "response_time_ms": 1250
      }
    ],
    "failed_responses": []
  },
  "timestamp": "2025-09-29T10:30:00Z"
}
```

### 3. Get Quotes - Delhivery (Through Unified Rate)
**Endpoint**: `POST /supply-rate/v1/quotes`  
**Purpose**: Get Delhivery rates through unified rate service

```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-$(uuidgen)" \
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

### 4. Get Quotes - Porter (Through Unified Rate)
**Endpoint**: `POST /supply-rate/v1/quotes`  
**Purpose**: Get Porter rates through unified rate service (local city delivery)

```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-$(uuidgen)" \
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

### 5. Multi-Partner Request with Success and Failure Examples
**Endpoint**: `POST /supply-rate/v1/quotes`  
**Purpose**: Get rates from multiple partners showing both successful and failed responses

```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-$(uuidgen)" \
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
        "code": "delhivery"
      },
      {
        "id": "",
        "code": "fedex"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "standard"
    }
  }'
```

**Generic Standardized Response Structure:**
```json
{
  "success": true,
  "message": "Rate quotes retrieved successfully.",
  "metadata": {
    "request_id": "req-a1b2c3d4-e5f6-7890-1234-567890abcdef",
    "timestamp": "2025-09-30T02:20:00Z",
    "response_time_ms": 1450,
    "partners_queried": 3,
    "partners_succeeded": 2,
    "partners_failed": 1,
    "total_rates_found": 3
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
          },
          {
            "rate_id": "dhl-economy-select-456",
            "service": "ECONOMY SELECT",
            "price": {
              "currency": "INR",
              "amount": 12500.00,
              "type": "real_time_international"
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
          "code": "fedex",
          "name": "FedEx"
        },
        "source": "real_time",
        "error": {
          "code": "PARTNER_TIMEOUT",
          "message": "The request to the partner API timed out after 3000ms."
        }
      }
    ]
  }
}
```

### 6. Bulk Quote Request (Multiple Packages)
**Endpoint**: `POST /supply-rate/v1/quotes`  
**Purpose**: Get rates for multiple packages in one request

```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-$(uuidgen)" \
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
          "value": 1.0,
          "unit": "kg"
        },
        "dimensions": {
          "length": 20.0,
          "width": 15.0,
          "height": 10.0,
          "unit": "cm"
        }
      },
      {
        "weight": {
          "value": 2.5,
          "unit": "kg"
        },
        "dimensions": {
          "length": 30.0,
          "width": 25.0,
          "height": 15.0,
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

---

## 📊 Metrics & Monitoring

### 1. Service Metrics
**Endpoint**: `GET /supply-rate/metrics`  
**Purpose**: Get service metrics and performance data

```bash
curl -X GET http://localhost:9046/supply-rate/metrics \
  -H "Content-Type: application/json"
```

**Expected Response:**
```json
{
  "service": "prayog-supply-rate-service",
  "timestamp": "2025-09-29T10:30:00Z",
  "metrics": {
    "http_requests_total": "counter",
    "http_request_duration": "histogram", 
    "rate_calculations_total": "counter",
    "provider_health_status": "gauge"
  }
}
```

---

## 📋 API Information

### 1. API Documentation Root
**Endpoint**: `GET /supply-rate/`  
**Purpose**: Get API information and available endpoints

```bash
curl -X GET http://localhost:9046/supply-rate/ \
  -H "Accept: application/json"
```

**Expected Response:**
```json
{
  "service": "prayog-supply-rate-service",
  "version": "v1",
  "description": "Microservice for supply rate calculations",
  "endpoints": {
    "quotes": "/supply-rate/v1/quotes",
    "health": "/supply-rate/health",
    "metrics": "/supply-rate/metrics"
  },
  "supported_partners": [
    {
      "code": "unified",
      "name": "Unified Rate Service",
      "type": "pre_defined"
    },
    {
      "code": "dhl",
      "name": "DHL Express", 
      "type": "real_time"
    }
  ]
}
```

---

## ❌ Error Scenarios

### 1. Invalid Request Format
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "invalid": "request"
  }'
```

**Expected Error Response:**
```json
{
  "success": false,
  "message": "Request validation failed",
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": "source_location is required"
  },
  "meta": {
    "request_id": "req-error-123",
    "response_time_ms": 10,
    "version": "v1"
  },
  "timestamp": "2025-09-29T10:30:00Z"
}
```

### 2. Unknown Partner Code
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
        "weight": {"value": 1.0, "unit": "kg"},
        "dimensions": {"length": 10, "width": 10, "height": 10, "unit": "cm"}
      }
    ],
    "partners": [
      {
        "id": "",
        "code": "unknown_partner"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "standard"
    }
  }'
```

**Expected Error Response:**
```json
{
  "success": true,
  "message": "Rate quotes retrieved successfully.",
  "metadata": {
    "request_id": "req-error-456",
    "timestamp": "2025-09-29T10:30:00Z",
    "response_time_ms": 15,
    "partners_queried": 1,
    "partners_succeeded": 0,
    "partners_failed": 1,
    "total_rates_found": 0
  },
  "data": {
    "successful_responses": [],
    "failed_responses": [
      {
        "partner": {
          "code": "unknown_partner",
          "name": "Unknown Partner"
        },
        "source": "unknown",
        "error": {
          "code": "PARTNER_NOT_FOUND",
          "message": "Partner implementation not found",
          "details": "No implementation found for partner code: unknown_partner"
        }
      }
    ]
  },
  "timestamp": "2025-09-29T10:30:00Z"
}
```

### 3. Partner API Failure
```bash
curl -X POST http://localhost:9046/supply-rate/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "source_location": {
      "postal_code": "invalid",
      "country_code": "IN"
    },
    "destination_location": {
      "postal_code": "110001",
      "country_code": "IN"
    },
    "packages": [
      {
        "weight": {"value": 1.0, "unit": "kg"},
        "dimensions": {"length": 10, "width": 10, "height": 10, "unit": "cm"}
      }
    ],
    "partners": [
      {
        "id": "",
        "code": "dhl"
      }
    ],
    "metadata": {
      "currency": "INR",
      "service_type": "standard"
    }
  }'
```

**Expected Error Response:**
```json
{
  "success": true,
  "message": "Rate quotes retrieved successfully.",
  "metadata": {
    "request_id": "req-error-789",
    "timestamp": "2025-09-29T10:30:00Z",
    "response_time_ms": 520,
    "partners_queried": 1,
    "partners_succeeded": 0,
    "partners_failed": 1,
    "total_rates_found": 0
  },
  "data": {
    "successful_responses": [],
    "failed_responses": [
      {
        "partner": {
          "code": "dhl",
          "name": "DHL Express"
        },
        "source": "real_time",
        "error": {
          "code": "RATE_FETCH_FAILED",
          "message": "Failed to fetch rates from partner",
          "details": "DHL API returned status 400: Invalid postal code"
        }
      }
    ]
  },
  "timestamp": "2025-09-29T10:30:00Z"
}
```

---

## 🔐 Authentication Examples

### 1. Service Authentication Status
Authentication status can be checked through the deep health check:

```bash
curl -X GET "http://localhost:9046/supply-rate/health?deep=true" \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: health-$(date +%s)"
```

This will show authentication status for the Unified Rate service in the implementations section of the health response.

---

## 🔧 Testing Scripts

### Quick Health Check Script
```bash
#!/bin/bash
echo "=== Supply Rate Service Health Check ==="
echo "Liveness: $(curl -s -o /dev/null -w "%{http_code}" http://localhost:9046/health/live)"
echo "Readiness: $(curl -s -o /dev/null -w "%{http_code}" http://localhost:9046/health/ready)"  
echo "App Health: $(curl -s -o /dev/null -w "%{http_code}" http://localhost:9046/supply-rate/health)"
echo "Deep Health: $(curl -s -o /dev/null -w "%{http_code}" "http://localhost:9046/supply-rate/health?deep=true")"
```

### Load Testing Script
```bash
#!/bin/bash
for i in {1..10}; do
  echo "Request $i"
  curl -X POST http://localhost:9046/supply-rate/v1/quotes \
    -H "Content-Type: application/json" \
    -H "X-Request-ID: load-test-$i" \
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

---

## 🏷️ Supported Partner Codes

### Pre-defined Partners (via Unified Rate Service)
- `unified` - Direct unified rate service access
- `prayog` - Alternative unified rate service access
- `delhivery` - Delhivery rates via unified service
- `porter` - Porter rates via unified service

### Real-time Partners  
- `dhl` - DHL Express international shipping
- `fedex` - FedEx (placeholder - not implemented)
- `ups` - UPS (placeholder - not implemented)

### Partner Code Normalization
Partner codes are automatically normalized:
- `DHL` → `dhl`
- `FedEx Express` → `fedex_express`  
- `Blue Dart` → `blue_dart`
- `DHL-Express` → `dhl_express`

---

## 💡 Tips & Best Practices

### 1. Request ID Tracking
Always include a request ID for better tracing:
```bash
-H "X-Request-ID: req-$(uuidgen)"
```

### 2. Timeout Settings
Set appropriate timeouts for long-running requests:
```bash
--max-time 30 --connect-timeout 5
```

### 3. Error Handling
Check HTTP status codes and parse error responses:
```bash
-w "Status: %{http_code}\nTime: %{time_total}s\n"
```

### 4. JSON Pretty Printing
Use jq for better response formatting:
```bash
curl ... | jq '.'
```

### 5. Performance Monitoring
Monitor response times and success rates:
```bash
curl -w "@curl-format.txt" ...
```

Create `curl-format.txt`:
```
     time_namelookup:  %{time_namelookup}\n
        time_connect:  %{time_connect}\n
     time_appconnect:  %{time_appconnect}\n
    time_pretransfer:  %{time_pretransfer}\n
       time_redirect:  %{time_redirect}\n
  time_starttransfer:  %{time_starttransfer}\n
                     ----------\n
          time_total:  %{time_total}\n
```

This documentation provides comprehensive examples for testing and integrating with the Prayog Supply Rate Service APIs.
