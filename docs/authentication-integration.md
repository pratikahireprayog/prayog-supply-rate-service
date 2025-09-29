# Authentication Integration for Unified Rate Service

## Overview
The Unified Rate service now includes a comprehensive authentication system that automatically handles login and bearer token management for accessing the Prayog Unified APIs. This replaces the previous API key authentication with a more secure token-based approach.

## Architecture

### Authentication Flow
```
Service Start → AuthService Init → Login API Call → Bearer Token Storage → API Calls with Token → Auto Token Refresh
```

### Key Components
1. **AuthService**: Handles authentication, token management, and refresh logic
2. **TokenInfo**: Stores token details, expiry, and user information  
3. **Service Integration**: Seamlessly integrates with the main Unified Rate service
4. **Auto-Refresh**: Automatically refreshes tokens before expiry

## Files Created/Modified

### 1. New Authentication Service
**File**: `internal/services/v1/implementations/pre_defined/unified_rate/auth.go`

**Key Features**:
- Login API integration
- Bearer token management
- Automatic token refresh (30 minutes before expiry by default)
- Thread-safe token storage
- Health check functionality
- Token validation and expiry checking

### 2. Updated Configuration
**File**: `internal/services/v1/implementations/pre_defined/unified_rate/config.go`

**Changes**:
- ❌ Removed `APIKey` field
- ✅ Added authentication configuration:
  - `LoginURL`: Authentication endpoint
  - `Username`: Login username
  - `Password`: Login password
  - `SigninType`: Authentication type (EMAIL)
  - `RefreshBufferMinutes`: Token refresh buffer time

### 3. Updated Service Integration
**File**: `internal/services/v1/implementations/pre_defined/unified_rate/service.go`

**Changes**:
- ✅ Added `AuthService` to service struct
- ✅ Authentication service initialization
- ✅ Bearer token authentication in API calls
- ✅ Health checks include authentication status
- ✅ Configuration includes auth information
- ✅ Token cleanup on service close

## Configuration

### Default Authentication Configuration
```json
{
  "login_url": "https://sandbox-apis.prayog.io/auth/login",
  "username": "avinash.singh@prayog.io", 
  "password": "Prayog@Avinash539",
  "signin_type": "EMAIL",
  "refresh_buffer_minutes": 30
}
```

### Custom Configuration Example
```go
config := map[string]interface{}{
    "base_url": "https://sandbox-apis.prayog.io/gateway/ure/api",
    "login_url": "https://sandbox-apis.prayog.io/auth/login",
    "username": "your.email@prayog.io",
    "password": "YourPassword",
    "signin_type": "EMAIL",
    "timeout_ms": 30000,
    "refresh_buffer_minutes": 30,
}
```

## Authentication Process

### 1. Login Request
```bash
curl 'https://sandbox-apis.prayog.io/auth/login' \
  -H 'Content-Type: application/json' \
  --data-raw '{
    "username": "avinash.singh@prayog.io",
    "password": "Prayog@Avinash539", 
    "signinType": "EMAIL"
  }'
```

### 2. Login Response
```json
{
  "id_token": "eyJraWQiOiJJZ0RTbEU1RzJkdFl4UVJEbUd6WFZuenR0Q0I0a0p5V3AyXC96VGh3dzQwaz0i...",
  "refresh_token": "eyJjdHkiOiJKV1QiLCJlbmMiOiJBMjU2R0NNIiwiYWxnIjoiUlNBLU9BRVAifQ...",
  "expires_in": 86400,
  "token_type": "Bearer",
  "user_id": "8113cdfa-b0d1-70e8-f113-2967182cf6d0",
  "user_email": "avinash.singh@prayog.io",
  "tenant_id": "68cd38e86423698971766a14"
}
```

### 3. API Call with Bearer Token
```bash
curl 'https://sandbox-apis.prayog.io/gateway/ure/api/external-rate-calculation/calculate' \
  -H 'Authorization: Bearer eyJraWQiOaWQiOiJJZ0RTbEU1RzJkdFl4UVJEbUd6WFZuenR0Q0I0a0p5V3AyXC96VGh3dzQwaz0i...' \
  -H 'Content-Type: application/json' \
  --data-raw '{ "sourceLocation": {...}, "destinationLocation": {...} }'
```

## Token Management Features

### 1. Automatic Token Refresh
- Tokens are refreshed automatically before expiry
- Default buffer: 30 minutes before expiration
- Configurable refresh buffer time
- Thread-safe refresh logic

### 2. Token Validation
- Checks token expiry before use
- Validates token format and content
- Handles expired tokens gracefully

### 3. Error Handling
- Network timeout handling
- Invalid credentials handling
- Token refresh failure handling
- Detailed error logging

### 4. Security Features
- Tokens are not exposed in configuration output
- Passwords are masked in serialization
- Thread-safe token storage
- Automatic token cleanup on service shutdown

## Testing the Integration

### 1. Basic Service Test
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

### 2. Health Check with Authentication
```bash
curl -X GET http://localhost:9046/supply-rate/health?deep=true \
  -H "Content-Type: application/json"
```

**Expected Response** (includes authentication info):
```json
{
  "providers": [
    {
      "name": "Unified Rate",
      "type": "pre_defined", 
      "healthy": true,
      "last_check": "2025-09-23T14:00:00Z",
      "configuration": {
        "is_authenticated": true,
        "auth_info": {
          "user_email": "avinash.singh@prayog.io",
          "user_id": "8113cdfa-b0d1-70e8-f113-2967182cf6d0",
          "tenant_id": "68cd38e86423698971766a14",
          "token_type": "Bearer",
          "expires_in": 86400,
          "obtained_at": "2025-09-23T13:30:00Z"
        }
      }
    }
  ]
}
```

### 3. Authentication Status Check
The service configuration now includes authentication status:

```json
{
  "provider_type": "pre_defined",
  "base_url": "https://sandbox-apis.prayog.io/gateway/ure/api",
  "is_authenticated": true,
  "login_url": "https://sandbox-apis.prayog.io/auth/login",
  "username": "avinash.singh@prayog.io",
  "signin_type": "EMAIL",
  "auth_info": {
    "user_email": "avinash.singh@prayog.io",
    "user_id": "8113cdfa-b0d1-70e8-f113-2967182cf6d0",
    "tenant_id": "68cd38e86423698971766a14",
    "token_type": "Bearer",
    "expires_in": 86400,
    "obtained_at": "2025-09-23T13:30:00Z"
  }
}
```

## Error Scenarios

### 1. Authentication Failure
```json
{
  "partner_rates": [
    {
      "partner": {"id": "", "code": "unified"},
      "success": false,
      "error": {
        "code": "RATE_FETCH_FAILED",
        "message": "Failed to fetch rates from partner",
        "details": "failed to get authentication token: authentication failed with status 401: invalid credentials"
      },
      "data_source": "pre_defined",
      "response_time_ms": 2000
    }
  ]
}
```

### 2. Token Refresh Failure
```json
{
  "error": {
    "code": "AUTH_TOKEN_REFRESH_FAILED",
    "message": "Failed to refresh authentication token",
    "details": "authentication request failed: connection timeout"
  }
}
```

### 3. Network Issues
```json
{
  "error": {
    "code": "AUTH_NETWORK_ERROR", 
    "message": "Authentication service unavailable",
    "details": "login API connection failed after 3 retries"
  }
}
```

## Performance Considerations

### Token Caching
- Tokens are cached in memory
- No unnecessary API calls for valid tokens
- Thread-safe access to token storage
- Automatic cleanup on service shutdown

### Network Optimization
- 15-second timeout for authentication requests
- Configurable retry logic
- Connection reuse for HTTP client
- Efficient token refresh scheduling

### Security
- Tokens are never logged in plain text
- Passwords are masked in configuration output
- Secure token storage in memory
- Automatic token expiration handling

## Monitoring and Logging

### Authentication Events
- Login success/failure events
- Token refresh events
- Authentication health check results
- Token expiration warnings

### Metrics Tracking
- Authentication success rate
- Token refresh frequency
- Authentication response times
- Authentication error rates

## Future Enhancements

### Planned Features
1. **Token Persistence**: Option to persist tokens across service restarts
2. **Multi-User Support**: Support for multiple user credentials
3. **Role-Based Access**: Integration with tenant hierarchy and roles
4. **Token Rotation**: Automatic refresh token rotation
5. **Authentication Middleware**: Reusable authentication middleware for other services

This authentication integration ensures secure, reliable access to the Prayog Unified APIs while maintaining high performance and providing comprehensive error handling and monitoring capabilities.
