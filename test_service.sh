#!/bin/bash

echo "🚀 Prayog Rate Service - Comprehensive Test Script"
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test endpoint
test_endpoint() {
    local name="$1"
    local cmd="$2"
    local expected="$3"
    
    echo -e "\n${YELLOW}Testing: $name${NC}"
    echo "Command: $cmd"
    
    result=$(eval $cmd 2>/dev/null)
    
    if [ "$result" = "$expected" ]; then
        echo -e "${GREEN}✅ PASS${NC}: $result"
    else
        echo -e "${RED}❌ FAIL${NC}: Expected '$expected', got '$result'"
    fi
}

# Stop any running service
echo "Stopping any running service..."
pkill -f rate-service 2>/dev/null || true
sleep 2

# Build the service
echo "Building service..."
go build -o bin/rate-service cmd/server/main.go

# Start the service
echo "Starting service..."
./bin/rate-service &
SERVICE_PID=$!
sleep 5

echo -e "\n🧪 Running Tests..."

# Test 1: Basic Health
test_endpoint "Basic Health" \
    "curl -s http://localhost:8080/health | jq -r '.status'" \
    "healthy"

# Test 2: API Info
test_endpoint "API Info" \
    "curl -s http://localhost:8080/rate/ | jq -r '.service'" \
    "Prayog Rate Card Service"

# Test 3: Rate Calculation
test_endpoint "Rate Calculation" \
    "curl -s -X POST http://localhost:8080/rate/v1/rates/calculate -H 'Content-Type: application/json' -d '{\"request_id\": \"test-123\",\"customer_id\": \"cust-456\",\"origin_city\": \"Mumbai\",\"dest_city\": \"Delhi\",\"weight\": 5.0,\"distance\": 1400.0,\"service_type\": \"standard\",\"pickup_date\": \"2025-01-15T10:00:00Z\",\"delivery_date\": \"2025-01-17T18:00:00Z\",\"priority\": \"normal\",\"currency\": \"INR\",\"source\": \"api\"}' | jq -r '.success'" \
    "true"

# Test 4: Rate Calculation with Provider Types (should work now)
echo -e "\n${YELLOW}Testing: Rate Calculation with Provider Types${NC}"
response=$(curl -s -X POST http://localhost:8080/rate/v1/rates/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "request_id": "test-types",
    "customer_id": "cust-456",
    "origin_city": "Mumbai",
    "dest_city": "Delhi",
    "weight": 5.0,
    "distance": 1400.0,
    "service_type": "standard",
    "pickup_date": "2025-01-15T10:00:00Z",
    "delivery_date": "2025-01-17T18:00:00Z",
    "priority": "normal",
    "currency": "INR",
    "source": "api",
    "provider_types": ["pre_defined", "real_time"]
  }')

success=$(echo $response | jq -r '.success')
if [ "$success" = "true" ]; then
    echo -e "${GREEN}✅ PASS${NC}: Provider types validation working"
else
    echo -e "${RED}❌ FAIL${NC}: Provider types validation failed"
    echo "Error: $(echo $response | jq '.error')"
fi

# Test 5: Metrics Endpoint
test_endpoint "Metrics Endpoint" \
    "curl -s http://localhost:8080/metrics | jq -r '.service'" \
    "prayog-rate-service"

echo -e "\n📊 Service Status Summary:"
echo "=========================="
echo "🔗 Service URL: http://localhost:8080"
echo "🏥 Health: http://localhost:8080/health"
echo "📋 API Info: http://localhost:8080/rate/"
echo "📈 Metrics: http://localhost:8080/metrics"
echo "🧮 Rate Calc: http://localhost:8080/rate/v1/rates/calculate"

echo -e "\n🛑 To stop the service:"
echo "pkill -f rate-service"

echo -e "\n✅ Test completed! Service PID: $SERVICE_PID"
