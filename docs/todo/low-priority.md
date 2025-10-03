# 🟢 Low Priority Enhancements

**Target Timeline:** 2-3 weeks (or as time permits)  
**Last Updated:** October 1, 2025

---

## API Enhancements

### 1. ⏳ Add Pagination Support
**Status:** Not Started  
**Priority:** Low - Scalability  
**Estimated Time:** 2 hours  
**Impact:** Better performance for large result sets

**Issue:**
- List endpoints return all records
- Could be slow with many rate cards

**Fix Required:**
Add pagination to list endpoints:

```go
type PaginationParams struct {
    Page     int `query:"page" validate:"min=1"`
    PageSize int `query:"page_size" validate:"min=1,max=100"`
}

type PaginatedResponse struct {
    Data       interface{} `json:"data"`
    TotalCount int         `json:"total_count"`
    Page       int         `json:"page"`
    PageSize   int         `json:"page_size"`
    TotalPages int         `json:"total_pages"`
}
```

**Endpoints to Update:**
- `GET /unified-rate-cards`
- Any future list endpoints

**Testing:**
- [ ] Test first page
- [ ] Test middle page
- [ ] Test last page
- [ ] Test page beyond total
- [ ] Test different page sizes

---

### 2. ⏳ Add Search Functionality
**Status:** Not Started  
**Priority:** Low - User Experience  
**Estimated Time:** 2.5 hours  
**Impact:** Easier to find specific rate cards

**Issue:**
- Can only filter by exact match
- No search by name or partial match

**Fix Required:**
Add search query parameters:

```go
// In handler
searchTerm := c.Query("search")
if searchTerm != "" {
    query = query.Where("name ILIKE ? OR partner_code ILIKE ?", 
        "%"+searchTerm+"%", "%"+searchTerm+"%")
}
```

**Search Fields:**
- Rate card name
- Partner code
- Partner name
- Tenant ID

**Testing:**
- [ ] Search by name (partial match)
- [ ] Search by partner code
- [ ] Search with no results
- [ ] Search with special characters
- [ ] Case-insensitive search

---

### 3. ⏳ Add Rate Limiting
**Status:** Not Started  
**Priority:** Low - Protection  
**Estimated Time:** 2 hours  
**Impact:** Prevent API abuse

**Issue:**
- No rate limiting on endpoints
- Could be abused

**Fix Required:**
Implement rate limiting middleware:

```go
func RateLimiter(requestsPerMinute int) fiber.Handler {
    limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), requestsPerMinute)
    
    return func(c *fiber.Ctx) error {
        if !limiter.Allow() {
            return c.Status(fiber.StatusTooManyRequests).JSON(utils.ErrorResponse(
                "Rate limit exceeded",
                "RATE_LIMIT_EXCEEDED",
                fmt.Sprintf("Maximum %d requests per minute", requestsPerMinute),
            ))
        }
        return c.Next()
    }
}
```

**Apply To:**
- Quote endpoints: 60 requests/minute
- Rate card management: 30 requests/minute
- Health check: unlimited

**Testing:**
- [ ] Make requests under limit (should succeed)
- [ ] Exceed limit (should get 429)
- [ ] Wait for reset (should work again)
- [ ] Different endpoints different limits

---

## Testing & Quality

### 4. ⏳ Write Integration Tests
**Status:** Not Started  
**Priority:** Low - Quality Assurance  
**Estimated Time:** 8 hours  
**Impact:** Confidence in code changes

**Areas to Cover:**

**DHL Integration:**
- [ ] Successful rate fetching
- [ ] Invalid credentials handling
- [ ] Network timeout handling
- [ ] Retry logic
- [ ] Request validation
- [ ] Health check

**Unified Rate:**
- [ ] Rate card CRUD operations
- [ ] Rate calculation
- [ ] Effective date filtering
- [ ] Default rate card selection
- [ ] API error handling
- [ ] Database operations

**Framework:**
```go
// test/integration/dhl_test.go
func TestDHLRateFetching(t *testing.T) {
    // Setup
    service := setupDHLService()
    
    // Test cases
    t.Run("successful rate fetch", func(t *testing.T) {
        // ...
    })
    
    t.Run("invalid credentials", func(t *testing.T) {
        // ...
    })
}
```

---

### 5. ⏳ Add Load Testing
**Status:** Not Started  
**Priority:** Low - Performance  
**Estimated Time:** 3 hours  
**Impact:** Understand system limits

**Tools:**
- Apache JMeter
- K6
- Gatling
- Custom Go benchmarks

**Test Scenarios:**
1. DHL API with concurrent requests (10, 50, 100 concurrent)
2. Database rate card queries (1000, 5000, 10000 requests)
3. Cache effectiveness under load
4. Memory usage over time
5. Response time degradation

**Metrics to Collect:**
- Requests per second
- Average response time
- 95th percentile response time
- Error rate
- Memory usage
- CPU usage

**Testing:**
- [ ] Run baseline test (1 user)
- [ ] Run load test (50 concurrent users)
- [ ] Run stress test (100+ concurrent users)
- [ ] Run soak test (sustained load over 1 hour)
- [ ] Identify bottlenecks
- [ ] Document performance characteristics

---

## Documentation

### 6. ⏳ Add OpenAPI/Swagger Documentation
**Status:** Not Started  
**Priority:** Low - Developer Experience  
**Estimated Time:** 4 hours  
**Impact:** Better API documentation

**File:** `api_docs/openapi/v1/rate-service.yaml`

**Include:**
- All endpoints with examples
- Request/response schemas
- Authentication requirements
- Error responses
- Rate limiting info

**Tools:**
- Swagger UI for testing
- ReDoc for documentation
- Swagger Codegen for client libraries

**Testing:**
- [ ] Generate from annotations
- [ ] Validate YAML file
- [ ] Test in Swagger UI
- [ ] Verify all endpoints documented
- [ ] Check examples work

---

### 7. ⏳ Create API Client Libraries
**Status:** Not Started  
**Priority:** Low - Developer Experience  
**Estimated Time:** 6 hours  
**Impact:** Easier integration for consumers

**Languages:**
- Go (primary)
- Python
- Node.js/TypeScript
- Java

**Generate Using:**
- OpenAPI Generator
- Custom templates

**Each Client Should Have:**
- Type-safe methods
- Error handling
- Retry logic
- Authentication
- Examples

**Testing:**
- [ ] Generate clients
- [ ] Test basic operations
- [ ] Verify type safety
- [ ] Test error handling
- [ ] Write usage examples

---

## Monitoring & Observability

### 8. ⏳ Add Distributed Tracing
**Status:** Not Started  
**Priority:** Low - Observability  
**Estimated Time:** 4 hours  
**Impact:** Better debugging across services

**Tools:**
- Jaeger
- Zipkin
- OpenTelemetry

**Implementation:**
```go
import "go.opentelemetry.io/otel"

func (s *Service) GetRates(ctx context.Context, req *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
    ctx, span := otel.Tracer("rate-service").Start(ctx, "DHL.GetRates")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("origin", req.OriginCity),
        attribute.String("destination", req.DestCity),
        attribute.Float64("weight", req.Weight),
    )
    
    // ... existing code ...
}
```

**Trace:**
- Quote request flow
- Partner API calls
- Database queries
- Cache operations
- External API calls

**Testing:**
- [ ] Setup Jaeger locally
- [ ] Add tracing to key operations
- [ ] View traces in UI
- [ ] Identify slow operations
- [ ] Add relevant span attributes

---

### 9. ⏳ Add Metrics Dashboard
**Status:** Not Started  
**Priority:** Low - Monitoring  
**Estimated Time:** 3 hours  
**Impact:** Better visibility into system health

**Stack:**
- Prometheus for metrics
- Grafana for visualization

**Metrics to Track:**
- Request rate by endpoint
- Response time by endpoint
- Error rate by type
- API call success/failure rate
- Database query duration
- Cache hit/miss ratio
- Active connections

**Dashboards:**
1. **Service Overview**
   - Request rate
   - Error rate
   - Response times (p50, p95, p99)

2. **Partner Performance**
   - Success rate by partner
   - Response time by partner
   - Error breakdown

3. **Database Performance**
   - Query duration
   - Connection pool usage
   - Slow queries

**Testing:**
- [ ] Export metrics to Prometheus
- [ ] Create Grafana dashboards
- [ ] Set up alerts for errors
- [ ] Test alert triggers
- [ ] Document dashboard usage

---

### 10. ⏳ Add Request/Response Logging
**Status:** Not Started  
**Priority:** Low - Debugging  
**Estimated Time:** 2 hours  
**Impact:** Better troubleshooting

**Implementation:**
```go
func RequestResponseLogger() fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        
        // Log request
        logger.Info("Incoming request",
            "method", c.Method(),
            "path", c.Path(),
            "ip", c.IP(),
            "request_id", c.Get("X-Request-ID"),
        )
        
        // Continue
        err := c.Next()
        
        // Log response
        logger.Info("Request completed",
            "method", c.Method(),
            "path", c.Path(),
            "status", c.Response().StatusCode(),
            "duration_ms", time.Since(start).Milliseconds(),
            "request_id", c.Get("X-Request-ID"),
        )
        
        return err
    }
}
```

**What to Log:**
- Request method, path, headers
- Response status, duration
- Request/response body (sanitized)
- Error details
- User information

**Testing:**
- [ ] Verify all requests logged
- [ ] Check sensitive data masked
- [ ] Test log rotation
- [ ] Verify log format parseable

---

## Summary

| Item | Component | Estimated Time | Status |
|------|-----------|---------------|---------|
| 1. Pagination | API | 2 hours | ⏳ Not Started |
| 2. Search | API | 2.5 hours | ⏳ Not Started |
| 3. Rate limiting | API | 2 hours | ⏳ Not Started |
| 4. Integration tests | Testing | 8 hours | ⏳ Not Started |
| 5. Load testing | Testing | 3 hours | ⏳ Not Started |
| 6. OpenAPI docs | Documentation | 4 hours | ⏳ Not Started |
| 7. Client libraries | Documentation | 6 hours | ⏳ Not Started |
| 8. Distributed tracing | Monitoring | 4 hours | ⏳ Not Started |
| 9. Metrics dashboard | Monitoring | 3 hours | ⏳ Not Started |
| 10. Request logging | Monitoring | 2 hours | ⏳ Not Started |

**Total Estimated Time:** 36.5 hours  
**Target Completion:** 2-3 weeks (or as time permits)

---

## Optional Nice-to-Haves

These are truly optional and can be done when all other items are complete:

- [ ] GraphQL API endpoint
- [ ] WebSocket support for real-time updates
- [ ] Rate card versioning system
- [ ] Multi-language support
- [ ] Dark mode for any UI
- [ ] Mobile app SDK
- [ ] Admin dashboard UI
- [ ] Automated deployment pipeline
- [ ] Blue-green deployment support
- [ ] Canary release capability

