#!/bin/bash

# Prayog Supply Rate Service Integration Test Script

set -e

echo "=== Prayog Supply Rate Service Integration Test ==="
echo

# Configuration
SERVICE_URL="http://localhost:9046"
DB_NAME="prayog_supply_rate_sandbox"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

run_test() {
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -e "\n${BLUE}Test $TESTS_RUN: $1${NC}"
}

# Check if PostgreSQL is running
check_database() {
    run_test "Database Connection"
    
    if pg_isready -d $DB_NAME >/dev/null 2>&1; then
        log_success "PostgreSQL is running and accessible"
    else
        log_error "PostgreSQL is not running or database does not exist"
        echo "Please ensure PostgreSQL is running and database '$DB_NAME' exists"
        exit 1
    fi
}

# Load sample data
load_sample_data() {
    run_test "Loading Sample Data"
    
    if [ -f "scripts/sample-data.sql" ]; then
        if psql -d $DB_NAME -f scripts/sample-data.sql >/dev/null 2>&1; then
            log_success "Sample data loaded successfully"
        else
            log_warning "Sample data loading may have failed or data already exists"
        fi
    else
        log_error "Sample data file not found: scripts/sample-data.sql"
    fi
}

# Test service health
test_health() {
    run_test "Service Health Check"
    
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVICE_URL/supply-rate/health" || echo "000")
    
    if [ "$HTTP_STATUS" = "200" ]; then
        log_success "Health check passed (HTTP $HTTP_STATUS)"
    else
        log_error "Health check failed (HTTP $HTTP_STATUS)"
        log_info "Make sure the service is running: go run cmd/server/main.go"
    fi
}

# Test deep health check
test_deep_health() {
    run_test "Deep Health Check"
    
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVICE_URL/supply-rate/health?deep=true" || echo "000")
    
    if [ "$HTTP_STATUS" = "200" ]; then
        log_success "Deep health check passed (HTTP $HTTP_STATUS)"
    else
        log_error "Deep health check failed (HTTP $HTTP_STATUS)"
    fi
}

# Test API info endpoint
test_api_info() {
    run_test "API Info Endpoint"
    
    RESPONSE=$(curl -s "$SERVICE_URL/supply-rate/" 2>/dev/null || echo "")
    
    if echo "$RESPONSE" | grep -q "unified_rate_cards"; then
        log_success "API info includes unified rate card endpoints"
    else
        log_error "API info missing unified rate card endpoints"
    fi
}

# Test unified rate calculation
test_unified_rate() {
    run_test "Unified Rate Calculation"
    
    RESPONSE=$(curl -s -X POST "$SERVICE_URL/supply-rate/v1/quotes" \
        -H "Content-Type: application/json" \
        -H "X-Request-ID: test-unified-$(date +%s)" \
        -d '{
            "source_location": {
                "postal_code": "560001",
                "country_code": "IN"
            },
            "destination_location": {
                "postal_code": "110001",
                "country_code": "IN"
            },
            "packages": [{
                "weight": {"value": 1.5, "unit": "kg"},
                "dimensions": {"length": 20.0, "width": 15.0, "height": 10.0, "unit": "cm"}
            }],
            "partners": [{"id": "", "code": "unified"}],
            "metadata": {"currency": "INR", "service_type": "standard"}
        }' 2>/dev/null || echo "")
    
    if echo "$RESPONSE" | grep -q '"success":true'; then
        log_success "Unified rate calculation successful"
    else
        log_error "Unified rate calculation failed"
        echo "Response: $RESPONSE"
    fi
}

# Test Delhivery via unified rate
test_delhivery_rate() {
    run_test "Delhivery Rate via Unified API"
    
    RESPONSE=$(curl -s -X POST "$SERVICE_URL/supply-rate/v1/quotes" \
        -H "Content-Type: application/json" \
        -H "X-Request-ID: test-delhivery-$(date +%s)" \
        -d '{
            "source_location": {
                "postal_code": "400001",
                "country_code": "IN"
            },
            "destination_location": {
                "postal_code": "500001",
                "country_code": "IN"
            },
            "packages": [{
                "weight": {"value": 2.0, "unit": "kg"},
                "dimensions": {"length": 25.0, "width": 20.0, "height": 15.0, "unit": "cm"}
            }],
            "partners": [{"id": "", "code": "delhivery"}],
            "metadata": {"currency": "INR", "service_type": "standard"}
        }' 2>/dev/null || echo "")
    
    if echo "$RESPONSE" | grep -q '"success":true'; then
        log_success "Delhivery rate calculation successful"
    else
        log_error "Delhivery rate calculation failed"
    fi
}

# Test multi-partner request
test_multi_partner() {
    run_test "Multi-Partner Rate Request"
    
    RESPONSE=$(curl -s -X POST "$SERVICE_URL/supply-rate/v1/quotes" \
        -H "Content-Type: application/json" \
        -H "X-Request-ID: test-multi-$(date +%s)" \
        -d '{
            "source_location": {
                "postal_code": "560001",
                "country_code": "IN"
            },
            "destination_location": {
                "postal_code": "110001",
                "country_code": "IN"
            },
            "packages": [{
                "weight": {"value": 1.5, "unit": "kg"},
                "dimensions": {"length": 20.0, "width": 15.0, "height": 10.0, "unit": "cm"}
            }],
            "partners": [
                {"id": "", "code": "dhl"},
                {"id": "", "code": "unified"},
                {"id": "", "code": "delhivery"}
            ],
            "metadata": {"currency": "INR", "service_type": "standard"}
        }' 2>/dev/null || echo "")
    
    if echo "$RESPONSE" | grep -q '"success":true' && echo "$RESPONSE" | grep -q '"partners_queried":3'; then
        log_success "Multi-partner request successful"
    else
        log_error "Multi-partner request failed"
    fi
}

# Test rate card listing
test_rate_card_list() {
    run_test "Unified Rate Card Listing"
    
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVICE_URL/supply-rate/v1/unified-rate-cards" || echo "000")
    
    if [ "$HTTP_STATUS" = "200" ]; then
        log_success "Rate card listing endpoint accessible (HTTP $HTTP_STATUS)"
    else
        log_error "Rate card listing failed (HTTP $HTTP_STATUS)"
    fi
}

# Test database content
test_database_content() {
    run_test "Database Content Verification"
    
    PARTNER_COUNT=$(psql -d $DB_NAME -t -c "SELECT COUNT(*) FROM partners;" 2>/dev/null | tr -d ' ' || echo "0")
    RATE_CARD_COUNT=$(psql -d $DB_NAME -t -c "SELECT COUNT(*) FROM unified_rate_cards;" 2>/dev/null | tr -d ' ' || echo "0")
    
    if [ "$PARTNER_COUNT" -gt "0" ] && [ "$RATE_CARD_COUNT" -gt "0" ]; then
        log_success "Database contains $PARTNER_COUNT partners and $RATE_CARD_COUNT rate cards"
    else
        log_error "Database content verification failed (Partners: $PARTNER_COUNT, Rate Cards: $RATE_CARD_COUNT)"
    fi
}

# Main execution
main() {
    echo "Starting integration tests..."
    echo
    
    # Database tests
    check_database
    load_sample_data
    test_database_content
    
    # Service tests
    test_health
    test_deep_health
    test_api_info
    
    # Rate calculation tests
    test_unified_rate
    test_delhivery_rate
    test_multi_partner
    
    # Management API tests
    test_rate_card_list
    
    # Summary
    echo
    echo "=== Test Summary ==="
    echo "Tests Run: $TESTS_RUN"
    echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
    
    if [ "$TESTS_FAILED" -gt "0" ]; then
        echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
        echo
        echo -e "${RED}Some tests failed. Please check the service configuration and ensure:${NC}"
        echo "1. PostgreSQL is running and accessible"
        echo "2. Database '$DB_NAME' exists and has proper permissions"
        echo "3. Service is running on $SERVICE_URL"
        echo "4. Network connectivity to unified API endpoints"
        exit 1
    else
        echo -e "Tests Failed: ${GREEN}0${NC}"
        echo
        echo -e "${GREEN}🎉 All tests passed! The Prayog Supply Rate Service is working correctly.${NC}"
        exit 0
    fi
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
