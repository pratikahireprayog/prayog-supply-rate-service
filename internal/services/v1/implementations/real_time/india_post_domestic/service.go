package india_post_domestic

import (
    "context"
    "encoding/json"
    "fmt"
    "net/url"
    "strings"
    "time"

    dtos "github.com/prayog/prayog-supply-rate-service/internal/shared/dtos/v1"
    interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
)

type Service struct {
    logger        interfaces.Logger
    metrics       interfaces.MetricsCollector
    httpClient    interfaces.HTTPClient
    config        *Config
    isInitialized bool

    accessToken     string
    refreshToken    string
    accessTokenExp  time.Time
    refreshTokenExp time.Time
}

func NewService(logger interfaces.Logger, metrics interfaces.MetricsCollector, httpClient interfaces.HTTPClient) *Service {
    return &Service{logger: logger, metrics: metrics, httpClient: httpClient, config: NewDefaultConfig(), isInitialized: true}
}

func (s *Service) GetImplementationType() dtos.ProviderType { return dtos.ProviderTypeRealTime }
func (s *Service) GetImplementationName() string            { return "India Post Domestic" }

func (s *Service) Initialize(cfg map[string]interface{}) error {
    if err := s.config.LoadFromMap(cfg); err != nil { return fmt.Errorf("india_post_domestic init failed: %w", err) }
    s.isInitialized = true
    return nil
}

func (s *Service) GetConfiguration() map[string]interface{} {
    return map[string]interface{}{ "base_url": s.config.BaseURL, "is_initialized": s.isInitialized, "provider_type": "real_time" }
}

func (s *Service) Close() error { s.isInitialized = false; return nil }

// ---- Public API ----
func (s *Service) GetRates(ctx context.Context, req *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
    if !s.isInitialized { return nil, fmt.Errorf("india_post_domestic service not initialized") }
    if strings.ToUpper(req.OriginCountry) != "IN" || strings.ToUpper(req.DestCountry) != "IN" {
        return nil, fmt.Errorf("india_post_domestic supports only IN→IN shipments")
    }

    if err := s.ensureAccessToken(ctx); err != nil {
        return nil, fmt.Errorf("auth failed: %w", err)
    }

    serviceType := strings.ToLower(getStringMeta(req, "service_type", "parcel"))

    var endpoint string
    var query url.Values

    switch serviceType {
    case "parcel":
        endpoint = s.config.BaseURL + "/parcel-tariff/calculate"
        query = s.buildParcelQuery(req)
    case "letter":
        endpoint = s.config.BaseURL + "/letter-tariff/calculate"
        query = s.buildLetterQuery(req)
    case "speed_post":
        endpoint = s.config.BaseURL + "/speed-post/tariffs"
        query = s.buildSpeedPostQuery(req)
    default:
        endpoint = s.config.BaseURL + "/parcel-tariff/calculate"
        query = s.buildParcelQuery(req)
    }

    headers := map[string]string{ "Authorization": "Bearer " + s.accessToken }

    fullURL := endpoint + "?" + query.Encode()
    start := time.Now()
    resp, err := s.httpClient.Get(ctx, fullURL, headers)
    duration := time.Since(start)
    s.metrics.RecordTimer("india_post_domestic_api_duration", duration, map[string]string{"endpoint": endpoint})
    if err != nil {
        s.metrics.IncrementCounter("india_post_domestic_api_error", map[string]string{"stage": "http_get"})
        return nil, fmt.Errorf("india post GET failed: %w", err)
    }

    if resp.StatusCode == 401 {
        // try refresh once
        if err := s.refreshAccessToken(ctx); err != nil {
            return nil, fmt.Errorf("token refresh failed: %w", err)
        }
        headers["Authorization"] = "Bearer " + s.accessToken
        resp, err = s.httpClient.Get(ctx, fullURL, headers)
        if err != nil { return nil, fmt.Errorf("india post GET retry failed: %w", err) }
    }

    var payload map[string]interface{}
    if err := json.Unmarshal(resp.Body, &payload); err != nil {
        return nil, fmt.Errorf("invalid response: %w", err)
    }

    // Build quote from payload
    quote := s.convertToQuote(payload, req, serviceType, duration)
    quotes := []dtos.RateQuote{}
    if quote.TotalPrice > 0 {
        quotes = append(quotes, quote)
    }

    return &dtos.RateCalculationResponse{
        RequestID:   req.RequestID,
        Status:      "success",
        Message:     "Rates retrieved from India Post Domestic",
        Quotes:      quotes,
        TotalQuotes: len(quotes),
        Timestamp:   time.Now(),
    }, nil
}

func (s *Service) IsHealthy(ctx context.Context) error {
    if !s.isInitialized { return fmt.Errorf("india_post_domestic service not initialized") }
    // Ensure we can get token
    if err := s.ensureAccessToken(ctx); err != nil { return err }
    return nil
}

// ---- Auth helpers ----
type loginResp struct {
    Success bool `json:"success"`
    Data struct {
        AccessToken string `json:"access_token"`
        ExpiresIn int `json:"expires_in"`
        RefreshToken string `json:"refresh_token"`
        RefreshExpiresIn int `json:"refresh_expires_in"`
    } `json:"data"`
}

func (s *Service) ensureAccessToken(ctx context.Context) error {
    if s.accessToken != "" && time.Now().Before(s.accessTokenExp.Add(-30*time.Second)) {
        return nil
    }
    if s.refreshToken != "" && time.Now().Before(s.refreshTokenExp.Add(-30*time.Second)) {
        return s.refreshAccessToken(ctx)
    }
    return s.fetchAccessToken(ctx)
}

func (s *Service) fetchAccessToken(ctx context.Context) error {
    body := map[string]interface{}{ "username": s.config.Username, "password": s.config.Password }
    headers := map[string]string{ "Content-Type": "application/json" }
    start := time.Now()
    resp, err := s.httpClient.Post(ctx, s.config.AccessTokenURL, body, headers)
    s.metrics.RecordTimer("india_post_domestic_auth_duration", time.Since(start), map[string]string{"action": "login"})
    if err != nil { return fmt.Errorf("login request failed: %w", err) }
    var lr loginResp
    if err := json.Unmarshal(resp.Body, &lr); err != nil { return fmt.Errorf("login parse failed: %w", err) }
    if !lr.Success || lr.Data.AccessToken == "" { return fmt.Errorf("login unsuccessful") }
    s.accessToken = lr.Data.AccessToken
    s.refreshToken = lr.Data.RefreshToken
    s.accessTokenExp = time.Now().Add(time.Duration(lr.Data.ExpiresIn) * time.Second)
    s.refreshTokenExp = time.Now().Add(time.Duration(lr.Data.RefreshExpiresIn) * time.Second)
    return nil
}

func (s *Service) refreshAccessToken(ctx context.Context) error {
    // India Post refresh uses refresh token in Authorization header (per sample curl)
    headers := map[string]string{ "Authorization": "Bearer " + s.refreshToken }
    start := time.Now()
    resp, err := s.httpClient.Post(ctx, s.config.RefreshTokenURL, nil, headers)
    s.metrics.RecordTimer("india_post_domestic_auth_duration", time.Since(start), map[string]string{"action": "refresh"})
    if err != nil { return fmt.Errorf("refresh request failed: %w", err) }
    var lr loginResp
    if err := json.Unmarshal(resp.Body, &lr); err != nil { return fmt.Errorf("refresh parse failed: %w", err) }
    if !lr.Success || lr.Data.AccessToken == "" { return fmt.Errorf("refresh unsuccessful") }
    s.accessToken = lr.Data.AccessToken
    s.refreshToken = lr.Data.RefreshToken
    s.accessTokenExp = time.Now().Add(time.Duration(lr.Data.ExpiresIn) * time.Second)
    s.refreshTokenExp = time.Now().Add(time.Duration(lr.Data.RefreshExpiresIn) * time.Second)
    return nil
}

// ---- Query builders ----
func (s *Service) buildParcelQuery(req *dtos.RateCalculationRequest) url.Values {
    p := url.Values{}
    p.Set("product-code", "PARCEL")
    weightG := toGrams(totalWeightKg(req)) // API example uses 1500 (grams?)
    p.Set("weight", fmt.Sprintf("%d", weightG))
    p.Set("source-pincode", strings.TrimSpace(req.OriginCity))
    p.Set("destination-pincode", strings.TrimSpace(req.DestCity))
    if len(req.Packages) > 0 {
        l,w,h := dimsCm(req)
        p.Set("length", fmt.Sprintf("%g", l))
        p.Set("width", fmt.Sprintf("%g", w))
        p.Set("height", fmt.Sprintf("%g", h))
    }
    if getBoolMeta(req, "cod", false) {
        p.Set("cod", "true")
        p.Set("cod-amount", fmt.Sprintf("%g", getFloatMeta(req, "cod_amount", 0)))
    }
    if getBoolMeta(req, "insurance", false) {
        p.Set("insurance", "true")
        p.Set("ins-amount", fmt.Sprintf("%g", getFloatMeta(req, "ins_amount", 0)))
    }
    return p
}

func (s *Service) buildLetterQuery(req *dtos.RateCalculationRequest) url.Values {
    p := url.Values{}
    p.Set("product-code", "LETTER")
    weightG := toGrams(totalWeightKg(req))
    p.Set("weight", fmt.Sprintf("%d", weightG))
    p.Set("source-pincode", strings.TrimSpace(req.OriginCity))
    p.Set("destination-pincode", strings.TrimSpace(req.DestCity))
    if getBoolMeta(req, "reg", false) { p.Set("reg", "true") }
    if getBoolMeta(req, "ack", false) { p.Set("ack", "true") }
    if getBoolMeta(req, "ins", false) {
        p.Set("ins", "true")
        p.Set("ins-amount", fmt.Sprintf("%g", getFloatMeta(req, "ins_amount", 0)))
    }
    return p
}

func (s *Service) buildSpeedPostQuery(req *dtos.RateCalculationRequest) url.Values {
    p := url.Values{}
    p.Set("product-code", "SP")
    weightG := toGrams(totalWeightKg(req))
    p.Set("weight", fmt.Sprintf("%d", weightG))
    p.Set("source-pincode", strings.TrimSpace(req.OriginCity))
    p.Set("destination-pincode", strings.TrimSpace(req.DestCity))
    if len(req.Packages) > 0 {
        l,w,h := dimsCm(req)
        p.Set("length", fmt.Sprintf("%g", l))
        p.Set("width", fmt.Sprintf("%g", w))
        p.Set("height", fmt.Sprintf("%g", h))
    }
    if ins := getFloatMeta(req, "INS", 0); ins > 0 { p.Set("INS", fmt.Sprintf("%g", ins)) }
    if getBoolMeta(req, "POD", false) { p.Set("POD", "YES") }
    return p
}

// ---- Response mapping ----
func (s *Service) convertToQuote(payload map[string]interface{}, req *dtos.RateCalculationRequest, serviceType string, duration time.Duration) dtos.RateQuote {
    getNum := func(m map[string]interface{}, k string) float64 {
        if v, ok := m[k]; ok {
            switch t := v.(type) {
            case float64: return t
            case int: return float64(t)
            case int64: return float64(t)
            }
        }
        return 0
    }
    // Base tariff may be named differently per endpoint
    base := getNum(payload, "base_tariff")
    if base == 0 { base = getNum(payload, "basic_charge") }
    // Taxes can be split or summarized
    cgst := getNum(payload, "cgst")
    sgst := getNum(payload, "sgst")
    igst := getNum(payload, "igst")
    totalTax := getNum(payload, "total_tax")
    gst := cgst + sgst + igst
    if gst == 0 && totalTax > 0 { gst = totalTax }
    // Total amount may be named differently
    total := getNum(payload, "total_with_tax")
    if total == 0 { total = getNum(payload, "total_amount") }
    if total == 0 { total = getNum(payload, "final_amount") }
    if total == 0 {
        before := getNum(payload, "total_before_tax")
        if before > 0 && gst >= 0 { total = before + gst }
    }

    // Additional charges
    insuranceCharge := getNum(payload, "insurance_charge")
    codCharge := getNum(payload, "cod_charge")
    regCharge := getNum(payload, "registration_charge")
    ackCharge := getNum(payload, "ack_charge")
    if ackCharge == 0 { ackCharge = getNum(payload, "acknowledgment_charge") }
    podCharge := getNum(payload, "pod_charge")
    vasCharges := getNum(payload, "vas_charges")

    handling := codCharge + regCharge + ackCharge + podCharge + vasCharges

    // Build quote (EstimatedDays unknown -> 0)
    quote := dtos.RateQuote{
        QuoteID:      fmt.Sprintf("india_post_domestic_%d", time.Now().UnixNano()),
        PartnerID:    "india_post_domestic",
        PartnerName:  "India Post Domestic",
        ProviderType: dtos.ProviderTypeRealTime,
        BasePrice:    base,
        TotalPrice:   total,
        Currency:     getStringMeta(req, "currency", "INR"),
        PriceBreakdown: dtos.PriceBreakdown{
            BasePrice:       base,
            HandlingCharge:  handling,
            InsuranceCharge: insuranceCharge,
            TaxAmount:       gst,
            TotalPrice:      total,
        },
        ServiceType:   serviceType,
        ServiceLevel:  "standard",
        EstimatedDays: 0,
        ValidUntil:    time.Now().Add(24 * time.Hour),
        Confidence:    0.85,
        Source:        "real_time",
        ResponseTimeMs: duration.Milliseconds(),
        Metadata: map[string]interface{}{
            "distance": getNum(payload, "distance"),
            "chargeable_weight": getNum(payload, "chargeable_weight"),
            "volumetric_weight": getNum(payload, "volumetric_weight"),
        },
    }
    return quote
}

// ---- Helpers ----
func getStringMeta(req *dtos.RateCalculationRequest, key, def string) string {
    if req.Metadata == nil { return def }
    if v, ok := req.Metadata[key]; ok {
        if s, ok := v.(string); ok && s != "" { return s }
    }
    return def
}

func getBoolMeta(req *dtos.RateCalculationRequest, key string, def bool) bool {
    if req.Metadata == nil { return def }
    if v, ok := req.Metadata[key]; ok {
        if b, ok := v.(bool); ok { return b }
    }
    return def
}

func getFloatMeta(req *dtos.RateCalculationRequest, key string, def float64) float64 {
    if req.Metadata == nil { return def }
    if v, ok := req.Metadata[key]; ok {
        switch t := v.(type) {
        case float64: return t
        case int: return float64(t)
        case int64: return float64(t)
        }
    }
    return def
}

func totalWeightKg(req *dtos.RateCalculationRequest) float64 {
    total := 0.0
    for _, p := range req.Packages {
        wkg := p.Weight
        switch strings.ToLower(p.WeightUnit) {
        case "g": wkg = p.Weight / 1000
        case "lb", "lbs": wkg = p.Weight * 0.453592
        }
        total += wkg
    }
    if total == 0 { total = req.Weight }
    return total
}

func dimsCm(req *dtos.RateCalculationRequest) (float64, float64, float64) {
    p := req.Packages[0]
    unit := strings.ToLower(p.DimUnit)
    conv := func(v float64) float64 {
        switch unit {
        case "m": return v * 100
        case "mm": return v / 10
        case "in", "inch": return v * 2.54
        case "ft": return v * 30.48
        default: return v
        }
    }
    return conv(p.Length), conv(p.Width), conv(p.Height)
}

func toGrams(kg float64) int { return int(kg * 1000) }


