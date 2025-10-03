# 🟡 Medium Priority Improvements

**Target Timeline:** 1-2 weeks  
**Last Updated:** October 1, 2025

---

## Performance & Optimization

### 1. ⏳ Implement Rate Card Caching
**Status:** Not Started  
**Priority:** Medium - Performance  
**Estimated Time:** 2 hours  
**Impact:** Reduced database load, faster response times

**Issue:**
- Database queried on every rate calculation
- No caching of partner configurations
- Increased latency

**Fix Required:**
Code provided in `docs/integration-review-and-improvements.md` section 4.2

**Implementation Steps:**
1. Add `CacheManager` to `RateCardService`
2. Cache partner configurations with 5 minute TTL
3. Implement cache invalidation on updates

**Cache Keys:**
- `rate_card_config:{partner_code}` - Partner configuration
- `rate_card_default:{partner_code}` - Default rate card

**Testing:**
- [ ] First request hits database
- [ ] Second request hits cache
- [ ] Cache expires after TTL
- [ ] Cache invalidated on update
- [ ] Measure performance improvement

---

### 2. ⏳ Expand Postal Code Service
**Status:** Not Started  
**Priority:** Medium - Accuracy  
**Estimated Time:** 3 hours  
**Impact:** Better location resolution

**Issue:**
- Using hardcoded postal code to city mapping
- Limited postal code support (only 7 codes mapped)

**Fix Required:**
1. Integrate with postal code lookup service or database
2. Support international postal codes
3. Cache postal code lookups

**Options:**
- Use Google Maps Geocoding API
- Use postal code database (MaxMind, GeoNames)
- Create internal postal code service

**Implementation:**
```go
type PostalCodeService interface {
    GetLocationInfo(postalCode, countryCode string) (*LocationInfo, error)
    GetCityName(postalCode, countryCode string) (string, error)
}
```

**Testing:**
- [ ] Test Indian postal codes
- [ ] Test international postal codes
- [ ] Test invalid postal codes
- [ ] Measure lookup performance
- [ ] Test cache effectiveness

---

### 3. ⏳ Add Effective Date Overlap Validation
**Status:** Not Started  
**Priority:** Medium - Data Integrity  
**Estimated Time:** 1.5 hours  
**Impact:** Prevent ambiguous rate card selection

**Issue:**
- Could allow overlapping effective dates for same partner
- Unclear which rate card to use

**Fix Required:**
Add validation in `SetDefault` and `Create`/`Update` methods:

```go
func (r *UnifiedRateCardRepository) ValidateEffectiveDates(
    ctx context.Context, 
    partnerCode string, 
    effectiveFrom time.Time, 
    effectiveTo *time.Time,
    excludeID *uuid.UUID,
) error {
    // Check for overlapping date ranges
    query := r.db.WithContext(ctx).
        Where("partner_code = ?", partnerCode).
        Where("is_active = ?", true)
    
    if excludeID != nil {
        query = query.Where("id != ?", *excludeID)
    }
    
    // Complex overlap logic here
    // ...
}
```

**Testing:**
- [ ] Create two rate cards with overlapping dates (should fail)
- [ ] Create consecutive rate cards (should succeed)
- [ ] Update to create overlap (should fail)
- [ ] Test with null effective_to dates

---

## Error Handling & Reliability

### 4. ⏳ Implement Error Recovery for Unified API
**Status:** Not Started  
**Priority:** Medium - Data Consistency  
**Estimated Time:** 2 hours  
**Impact:** Prevent inconsistent state

**Issue:**
- If unified API call fails, local DB not rolled back
- Inconsistent state between local and remote

**Fix Required:**
Implement compensation logic:

```go
func (s *RateCardService) CreateRateCard(...) error {
    // 1. Create in unified API first
    unifiedID, err := s.createInUnifiedAPI(...)
    if err != nil {
        return err
    }
    
    // 2. Create in local DB
    err = s.rateCardRepo.Create(ctx, rateCard)
    if err != nil {
        // Compensate: delete from unified API
        _ = s.deleteFromUnifiedAPI(ctx, unifiedID)
        return err
    }
    
    return nil
}
```

**Testing:**
- [ ] Test unified API failure (should not create locally)
- [ ] Test database failure (should delete from unified API)
- [ ] Test network timeout scenarios
- [ ] Verify no orphaned records

---

### 5. ⏳ Add Audit Logging
**Status:** Not Started  
**Priority:** Medium - Compliance  
**Estimated Time:** 2 hours  
**Impact:** Track who made changes

**Issue:**
- `CreatedBy` and `UpdatedBy` fields not populated
- Cannot track changes

**Fix Required:**
1. Extract user from authentication context
2. Populate audit fields on create/update
3. Store user information

**Implementation:**
```go
// In middleware
func ExtractUser() fiber.Handler {
    return func(c *fiber.Ctx) error {
        apiKey := c.Get("X-API-Key")
        user := getUserFromAPIKey(apiKey) // Lookup user
        c.Locals("user_id", user.ID)
        c.Locals("user_email", user.Email)
        return c.Next()
    }
}

// In handler
userID := c.Locals("user_id").(string)
req.CreatedBy = &userID
```

**Testing:**
- [ ] Create rate card (verify created_by set)
- [ ] Update rate card (verify updated_by set)
- [ ] Different users make changes (verify different IDs)
- [ ] Query audit trail

---

## Code Quality

### 6. ⏳ Expand DHL Service Type Mappings
**Status:** Not Started  
**Priority:** Medium - Feature Completeness  
**Estimated Time:** 1 hour  
**Impact:** Support all DHL services

**Issue:**
- Only 3 service types mapped (express, standard, economy)
- DHL has many more service options

**Fix Required:**
Expand mapping in `dhl/service.go`:

```go
func (s *Service) mapServiceToProductCode(serviceType string) string {
    mappings := map[string]string{
        "express":           "P",  // EXPRESS WORLDWIDE
        "express_9":         "9",  // DOMESTIC EXPRESS 9:00
        "express_12":        "12", // DOMESTIC EXPRESS 12:00
        "standard":          "N",  // DOMESTIC EXPRESS
        "economy":           "U",  // EXPRESS WORLDWIDE NONDOC
        "economy_select":    "W",  // ECONOMY SELECT
        "express_envelope":  "X",  // EXPRESS ENVELOPE
        "express_pak":       "D",  // EXPRESS PAK
        "jumbo_box":         "J",  // JUMBO BOX
        "freight":           "E",  // EXPRESS FREIGHT
    }
    
    if code, exists := mappings[strings.ToLower(serviceType)]; exists {
        return code
    }
    return "P" // Default
}
```

**Testing:**
- [ ] Test each service type
- [ ] Verify correct DHL product returned
- [ ] Test unknown service type (defaults to express)

---

### 7. ⏳ Implement Request Deduplication
**Status:** Not Started  
**Priority:** Medium - Cost Optimization  
**Estimated Time:** 2.5 hours  
**Impact:** Avoid duplicate API calls

**Issue:**
- Same request could be sent multiple times
- Waste of API calls and cost

**Fix Required:**
1. Generate request hash/fingerprint
2. Cache responses with short TTL (1-2 minutes)
3. Return cached response for duplicate requests

**Implementation:**
```go
func generateRequestFingerprint(req *dtos.RateCalculationRequest) string {
    key := fmt.Sprintf("%s:%s:%f:%s:%s",
        req.OriginCity,
        req.DestCity,
        req.Weight,
        req.ServiceType,
        req.PickupDate.Format("2006-01-02"),
    )
    hash := sha256.Sum256([]byte(key))
    return hex.EncodeToString(hash[:])
}

func (s *Service) GetRates(ctx context.Context, request *dtos.RateCalculationRequest) (*dtos.RateCalculationResponse, error) {
    // Check cache first
    fingerprint := generateRequestFingerprint(request)
    if cached, err := s.cache.Get(ctx, fingerprint); err == nil {
        return cached, nil
    }
    
    // Make API call
    response, err := s.callDHLAPI(ctx, request)
    
    // Cache for 2 minutes
    _ = s.cache.Set(ctx, fingerprint, response, 2*time.Minute)
    
    return response, err
}
```

**Testing:**
- [ ] Send same request twice (second should hit cache)
- [ ] Send different requests (should not hit cache)
- [ ] Wait for cache expiry (should call API again)
- [ ] Measure cost savings

---

### 8. ⏳ Add Comprehensive Error Messages
**Status:** Not Started  
**Priority:** Medium - Developer Experience  
**Estimated Time:** 1.5 hours  
**Impact:** Better debugging and user experience

**Issue:**
- Some error messages lack context
- Not all edge cases handled

**Fix Required:**
Improve error messages throughout:

```go
// Bad
return fmt.Errorf("validation failed")

// Good  
return fmt.Errorf("validation failed for field '%s': %s (expected: %s, got: %s)", 
    field, reason, expected, actual)
```

**Areas to Improve:**
- DHL API error responses
- Unified API error responses
- Database errors
- Validation errors

**Testing:**
- [ ] Trigger each error scenario
- [ ] Verify error message is helpful
- [ ] Check error includes context
- [ ] Ensure errors are logged properly

---

## Summary

| Item | Component | Estimated Time | Status |
|------|-----------|---------------|---------|
| 1. Rate card caching | Unified | 2 hours | ⏳ Not Started |
| 2. Postal code service | DHL | 3 hours | ⏳ Not Started |
| 3. Date overlap validation | Unified | 1.5 hours | ⏳ Not Started |
| 4. Error recovery | Unified | 2 hours | ⏳ Not Started |
| 5. Audit logging | Unified | 2 hours | ⏳ Not Started |
| 6. Service type mappings | DHL | 1 hour | ⏳ Not Started |
| 7. Request deduplication | DHL | 2.5 hours | ⏳ Not Started |
| 8. Error messages | All | 1.5 hours | ⏳ Not Started |

**Total Estimated Time:** 15.5 hours  
**Target Completion:** 1-2 weeks

---

## Recommended Order

1. **Week 1:**
   - Rate card caching (Item #1)
   - Date overlap validation (Item #3)
   - Audit logging (Item #5)
   - Service type mappings (Item #6)

2. **Week 2:**
   - Postal code service (Item #2)
   - Error recovery (Item #4)
   - Request deduplication (Item #7)
   - Error messages (Item #8)

