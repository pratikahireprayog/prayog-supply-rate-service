package india_post_international

import "fmt"

// TariffRequest represents the simplified tariff calculation request
// Format: { "weight": 800, "countryCode": "DE", "sourcePincode": "110001" }
type TariffRequest struct {
	Weight        int    `json:"weight"`        // Weight in grams
	CountryCode   string `json:"countryCode"`    // Destination country code (ISO 2-letter)
	SourcePincode string `json:"sourcePincode"` // Source pincode
}

// TariffResponse represents the tariff calculation response
// API returns: { "statusCode": 200, "data": { "success": true, "totalAmount": 1256.7, ... } }
type TariffResponse struct {
	Status       string  `json:"status"`        // "success" or "failed"
	Success      bool    `json:"success"`       // Boolean success flag
	TariffAmount float64 `json:"tariff_amount"` // Total tariff amount
	Currency     string  `json:"currency"`      // Currency code (default: INR)
	DeliveryTime string  `json:"delivery_time"` // Estimated delivery time
	Message      string  `json:"message"`       // Response message
}

// LoginRequest represents the login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
// API returns: { "success": true, "data": { "access_token": "...", "expires_in": 900, ... } }
type LoginResponse struct {
	Success bool      `json:"success"`
	Data    LoginData `json:"data"`
}

// LoginData contains the authentication data
type LoginData struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
}

// RefreshTokenResponse represents the refresh token response
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// ErrorResponse represents an error response from India Post API
type ErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

// IndiaPostAPIError represents an API error
type IndiaPostAPIError struct {
	StatusCode int
	Message    string
	RawBody    string
}

func (e *IndiaPostAPIError) Error() string {
	return fmt.Sprintf("India Post API error (status %d): %s", e.StatusCode, e.Message)
}

