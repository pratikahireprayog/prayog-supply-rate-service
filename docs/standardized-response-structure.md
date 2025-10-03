# Standardized Response Structure - Rate Service

## Overview
This document defines the standardized response structure for all rate quote endpoints in the Prayog Supply Rate Service. The structure has been updated to provide better organization, clearer metadata, and separation of successful vs failed partner responses.

## Standard Response Format

All quote responses follow this consistent structure:

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
    "successful_responses": [...],
    "failed_responses": [...]
  },
  "timestamp": "2025-09-30T02:20:00Z"
}
```

## Field Definitions

### Top Level Fields

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Always `true` for 200 responses (indicates API call succeeded) |
| `message` | string | Human-readable message describing the operation result |
| `metadata` | object | Request metadata and statistics |
| `data` | object | The actual response data with successful and failed responses |
| `timestamp` | string | ISO 8601 timestamp when the response was generated |

### Metadata Fields

| Field | Type | Description |
|-------|------|-------------|
| `request_id` | string | Unique identifier for the request |
| `timestamp` | string | ISO 8601 timestamp for the request |
| `response_time_ms` | number | Total response time in milliseconds |
| `partners_queried` | number | Total number of partners queried |
| `partners_succeeded` | number | Number of partners that returned rates successfully |
| `partners_failed` | number | Number of partners that failed |
| `total_rates_found` | number | Total number of rates returned across all successful partners |

### Data Structure

The `data` object contains two arrays:

#### Successful Responses

```json
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
      },
      "delivery_days": 7
    }
  ],
  "response_time_ms": 1250
}
```

#### Failed Responses

```json
{
  "partner": {
    "code": "fedex",
    "name": "FedEx"
  },
  "source": "real_time",
  "error": {
    "code": "PARTNER_TIMEOUT",
    "message": "The request to the partner API timed out after 3000ms.",
    "details": "Additional error details if available"
  }
}
```

## Partner Information

All partners are identified by:
- `code`: Normalized partner code (lowercase, underscores for spaces)
- `name`: Human-readable display name

### Partner Code to Name Mapping

| Code | Name |
|------|------|
| `dhl` | DHL Express |
| `fedex` | FedEx |
| `ups` | UPS |
| `blue_dart` | Blue Dart |
| `delhivery` | Delhivery |
| `porter` | Porter |
| `unified` | Unified Rate |
| `prayog` | Prayog Unified Rate |
| `dunzo` | Dunzo |
| `aramex` | Aramex |

## Source Types

Partners are categorized by their data source:

- `pre_defined`: Rates from pre-configured rate cards or unified services
- `real_time`: Rates from real-time API calls to partner systems
- `unknown`: Used when partner type cannot be determined (typically for errors)

## Rate Structure

Each rate follows this structure:

```json
{
  "rate_id": "partner-service-identifier-123",
  "service": "Human readable service name",
  "price": {
    "currency": "INR",
    "amount": 123.45,
    "type": "price_type_description",
    "criteria": {
      "key": "value"
    }
  },
  "delivery_days": 3
}
```

### Price Types by Partner

| Partner | Price Type | Description |
|---------|------------|-------------|
| DHL | `real_time_international` | Real-time international shipping rates |
| Delhivery | `weight_distance_based` | Rates based on weight and distance |
| Porter | `distance_based` | Local delivery rates based on distance |
| Unified | `weight_distance_based` | Pre-defined rates from rate cards |

## Error Codes

Standard error codes used in failed responses:

| Code | Description |
|------|-------------|
| `PARTNER_NOT_FOUND` | Partner implementation not found |
| `RATE_FETCH_FAILED` | Failed to fetch rates from partner API |
| `PARTNER_TIMEOUT` | Partner API request timed out |
| `AUTHENTICATION_FAILED` | Partner authentication failed |
| `INVALID_REQUEST` | Invalid request format or data |

## Implementation Status

✅ **Completed:**
- Updated DTOs in `internal/shared/dtos/v1/quotes.go`
- Updated service implementation in `internal/services/v1/rate_service.go`
- Updated documentation in `docs/quotes-endpoint-implementation.md`
- Updated API examples in `docs/api-curl-examples.md`

✅ **Consistent Across:**
- All quote endpoints use the same response structure
- Error handling follows the same pattern
- Partner identification is standardized
- Metadata is consistent across all responses

## Backward Compatibility

Legacy structures are preserved with deprecation notices:
- `QuoteSummary` (deprecated)
- `PartnerRateResult` (deprecated) 
- `LegacyPartnerInfo` (deprecated)

These will be removed in v2 of the API.

## Benefits

1. **Clear Separation**: Success and failure responses are clearly separated
2. **Rich Metadata**: Comprehensive request statistics at the top level  
3. **Consistent Partner Info**: All partners have both code and display name
4. **Flexible**: Structure accommodates different partner types and pricing models
5. **Extensible**: Easy to add new fields without breaking changes
6. **Debuggable**: Rich error information and timing data for troubleshooting

This standardized structure provides a solid foundation for the rate service that can scale as new partners and features are added.
