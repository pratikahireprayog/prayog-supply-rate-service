# 🔴 Critical Priority Fixes

**Target Timeline:** 1-2 days  
**Last Updated:** October 1, 2025

---

## DHL Integration

### 1. ⏳ Remove Hardcoded API Credentials
**Status:** Not Started  
**Priority:** Critical - Security Risk  
**Estimated Time:** 30 minutes  
**Impact:** Credentials exposed in repository

**Issue:**
- Hardcoded credentials in `internal/services/v1/implementations/real_time/dhl/config.go:21`
- Base64 encoded credentials visible in source code

**Fix Required:**
```go
// Remove from config.go:21
Credentials: "c2hyZWVtYXJ1dDhJTjpJITBwTV40c1IjNG5KJDF1",

// Replace with environment variable loading
Credentials: os.Getenv("DHL_CREDENTIALS"),
```

**Environment Setup:**
```bash
export DHL_CREDENTIALS="base64_encoded_username_password"
export DHL_ACCOUNT_NUMBER="533748932"
export DHL_ENVIRONMENT="test"
```

**Testing:**
- [ ] Verify credentials load from environment
- [ ] Test DHL API calls with env credentials
- [ ] Confirm no credentials in source code

---

### 2. ⏳ Add Country Code Support
**Status:** Not Started  
**Priority:** Critical - Incorrect Pricing  
**Estimated Time:** 1 hour  
**Impact:** Cannot handle international shipments properly

**Issue:**
- Hardcoded country codes "IN" and "CN" in `dhl/service.go:133, 138`
- Cannot ship to other countries

**Fix Required:**
1. Add country code fields to `RateCalculationRequest` DTO:
```go
type RateCalculationRequest struct {
    // ... existing fields ...
    OriginCountry string `json:"origin_country"`
    DestCountry   string `json:"dest_country"`
}
```

2. Update DHL request conversion:
```go
"countryCode": req.OriginCountry, // Instead of "IN"
```

3. Update quotes DTO to include country codes

**Testing:**
- [ ] Test domestic shipment (IN to IN)
- [ ] Test international shipment (IN to CN)
- [ ] Test international shipment (IN to US)
- [ ] Verify pricing accuracy

---

### 3. ⏳ Support Package Dimensions
**Status:** Not Started  
**Priority:** Critical - Pricing Inaccuracy  
**Estimated Time:** 1 hour  
**Impact:** Inaccurate volumetric weight calculations

**Issue:**
- Hardcoded dimensions (30x20x15) in `dhl/service.go:166-168`
- Volumetric weight affects pricing significantly

**Fix Required:**
1. Update request to include actual package dimensions from request
2. Support multiple packages with individual dimensions
3. Calculate volumetric weight correctly

**Testing:**
- [ ] Test with small package (actual dimensions)
- [ ] Test with large package (volumetric weight higher)
- [ ] Verify DHL API accepts dimensions
- [ ] Compare prices with/without correct dimensions

---

## Unified Rate Integration

### 4. ⏳ Connect Real Database
**Status:** Not Started  
**Priority:** Critical - Data Loss  
**Estimated Time:** 30 minutes  
**Impact:** All data lost on service restart

**Issue:**
- Using mock repository in `cmd/server/main.go:170-172`
- No persistence of rate cards

**Fix Required:**
```go
// In cmd/server/main.go
// Uncomment database initialization
dbConfig := loadDatabaseConfig()
db := database.NewPostgresConnection(dbConfig)

// Replace mock repository
rateCardRepo := repositoriesv1.NewUnifiedRateCardRepository(db.DB)
partnerRepo := repositoriesv1.NewPartnerRepository(db.DB)
```

**Prerequisites:**
- [ ] PostgreSQL running
- [ ] Database created
- [ ] Migration applied (see next item)

**Testing:**
- [ ] Create rate card via API
- [ ] Restart service
- [ ] Verify rate card still exists
- [ ] Test rate calculations work after restart

---

### 5. ⏳ Create Database Migration
**Status:** Not Started  
**Priority:** Critical - Database Schema  
**Estimated Time:** 30 minutes  
**Impact:** No database schema for unified rate cards

**File:** `scripts/migrations/001_create_unified_rate_cards.sql`

**Migration Script:** Already provided in `docs/integration-review-and-improvements.md` section 3

**Steps:**
1. Create migrations directory: `mkdir -p scripts/migrations`
2. Create migration file with provided SQL
3. Run migration:
```bash
psql -d prayog_rate_service -f scripts/migrations/001_create_unified_rate_cards.sql
```

**Testing:**
- [ ] Run migration successfully
- [ ] Verify table created
- [ ] Verify indexes created
- [ ] Verify triggers work (updated_at)
- [ ] Test constraint (effective_to > effective_from)

---

### 6. ⏳ Add Authentication Middleware
**Status:** Not Started  
**Priority:** Critical - Security  
**Estimated Time:** 1 hour  
**Impact:** Anyone can create/modify/delete rate cards

**Issue:**
- Rate card management endpoints unprotected
- No authentication on CRUD operations

**Fix Required:**
1. Create authentication middleware (code provided in review doc)
2. Add API key validation
3. Protect all rate card endpoints

**File:** `internal/infrastructure/api/http/middleware/auth.go`

**Update Routes:**
```go
// In unified_rate_card_routes.go
rateCards := router.Group("/unified-rate-cards")
rateCards.Use(middleware.RequireAPIKey(validAPIKeys))

// Then add routes
rateCards.Post("/", handler.CreateRateCard)
// ... etc
```

**Testing:**
- [ ] Request without API key (should fail with 401)
- [ ] Request with invalid API key (should fail with 401)
- [ ] Request with valid API key (should succeed)
- [ ] Verify all CRUD endpoints protected

---

### 7. ⏳ Add Request Validation for DHL
**Status:** Not Started  
**Priority:** Critical - Prevent Invalid API Calls  
**Estimated Time:** 45 minutes  
**Impact:** Unnecessary API calls with invalid data

**File:** `internal/services/v1/implementations/real_time/dhl/validation.go`

**Validation Code:** Provided in review document section 4.1

**What to Validate:**
- Origin and destination cities/postal codes required
- Country codes required and valid (2 char ISO)
- Weight must be positive
- At least one package required
- Package dimensions must be positive

**Implementation:**
1. Create validation file
2. Add `validateRequest()` method
3. Call validation before API call in `GetRates()`

**Testing:**
- [ ] Test with missing origin city
- [ ] Test with invalid country code
- [ ] Test with negative weight
- [ ] Test with no packages
- [ ] Test with invalid dimensions
- [ ] Verify error messages are clear

---

## Summary

| Item | Component | Estimated Time | Status |
|------|-----------|---------------|---------|
| 1. Remove hardcoded credentials | DHL | 30 min | ⏳ Not Started |
| 2. Add country code support | DHL | 1 hour | ⏳ Not Started |
| 3. Support package dimensions | DHL | 1 hour | ⏳ Not Started |
| 4. Connect real database | Unified | 30 min | ⏳ Not Started |
| 5. Create database migration | Unified | 30 min | ⏳ Not Started |
| 6. Add auth middleware | Unified | 1 hour | ⏳ Not Started |
| 7. Add request validation | DHL | 45 min | ⏳ Not Started |

**Total Estimated Time:** 5 hours 45 minutes  
**Target Completion:** Within 1-2 days

---

## Quick Win Items

These can be completed in 30 minutes each:

✅ **Quick Wins:**
1. Remove hardcoded credentials (Item #1)
2. Connect real database (Item #4)
3. Create database migration (Item #5)

Do these three first for immediate security and stability improvements!

