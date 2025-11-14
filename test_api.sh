#!/bin/bash

# Smart Contract Event Indexer - API Integration Test
# This script tests the API endpoints to verify they work correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="http://localhost:8000/api/v1"
HEALTH_URL="$API_BASE_URL/health"

echo -e "${YELLOW}🚀 Starting Smart Contract Event Indexer API Integration Test${NC}"
echo "=================================================="

# Function to check if service is running
check_service() {
    local service_name=$1
    local url=$2
    
    echo -e "\n${YELLOW}Checking $service_name...${NC}"
    
    if curl -s -f "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ $service_name is running${NC}"
        return 0
    else
        echo -e "${RED}❌ $service_name is not running or not responding${NC}"
        return 1
    fi
}

# Function to make API request and check response
test_api() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local description=$5
    
    echo -e "\n${YELLOW}Testing: $description${NC}"
    echo "Endpoint: $method $endpoint"
    
    local response
    local status_code
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$API_BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            "$API_BASE_URL$endpoint")
    fi
    
    # Extract status code (last line)
    status_code=$(echo "$response" | tail -n1)
    
    # Extract response body (all but last line)
    response_body=$(echo "$response" | head -n -1)
    
    if [ "$status_code" -eq "$expected_status" ]; then
        echo -e "${GREEN}✅ Status: $status_code (Expected: $expected_status)${NC}"
        echo "Response: $response_body"
    else
        echo -e "${RED}❌ Status: $status_code (Expected: $expected_status)${NC}"
        echo "Response: $response_body"
        return 1
    fi
}

# Main test execution
main() {
    echo -e "\n${YELLOW}Step 1: Health Check${NC}"
    if ! check_service "API Gateway" "$HEALTH_URL"; then
        echo -e "${RED}❌ API Gateway is not running. Please start the services first:${NC}"
        echo "   docker-compose up -d"
        exit 1
    fi
    
    echo -e "\n${YELLOW}Step 2: Testing API Endpoints${NC}"
    
    # Test health endpoint
    test_api "GET" "/health" "" 200 "Health Check"
    
    # Test getting contracts (should be empty initially)
    test_api "GET" "/contracts" "" 200 "Get Contracts (Empty)"
    
    # Test adding a contract
    local contract_data='{
        "address": "0x1234567890123456789012345678901234567890",
        "name": "Test Contract",
        "abi": "[{\"type\":\"function\",\"name\":\"test\",\"inputs\":[],\"outputs\":[]}]",
        "start_block": 1000000
    }'
    test_api "POST" "/contracts" "$contract_data" 201 "Add Contract"
    
    # Test getting contracts (should now have one)
    test_api "GET" "/contracts" "" 200 "Get Contracts (With Data)"
    
    # Test getting specific contract
    test_api "GET" "/contracts/0x1234567890123456789012345678901234567890" "" 200 "Get Specific Contract"
    
    # Test getting contract stats
    test_api "GET" "/contracts/0x1234567890123456789012345678901234567890/stats" "" 200 "Get Contract Stats"
    
    # Test getting events (should be empty)
    test_api "GET" "/events" "" 200 "Get Events (Empty)"
    
    # Test getting events with filters
    test_api "GET" "/events?contract=0x1234567890123456789012345678901234567890" "" 200 "Get Events with Contract Filter"
    
    # Test getting events by transaction (should be empty)
    test_api "GET" "/events/tx/0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890" "" 200 "Get Events by Transaction"
    
    # Test getting events by address (should be empty)
    test_api "GET" "/events/address/0x1234567890123456789012345678901234567890" "" 200 "Get Events by Address"
    
    # Test removing contract
    test_api "DELETE" "/contracts/0x1234567890123456789012345678901234567890" "" 200 "Remove Contract"
    
    # Test getting contracts after removal
    test_api "GET" "/contracts" "" 200 "Get Contracts After Removal"
    
    echo -e "\n${GREEN}🎉 All API tests passed!${NC}"
    echo "=================================================="
    echo -e "${GREEN}✅ API Gateway is working correctly${NC}"
    echo -e "${GREEN}✅ All endpoints are responding as expected${NC}"
    echo -e "${GREEN}✅ Contract management is working${NC}"
    echo -e "${GREEN}✅ Event queries are working${NC}"
}

# Run the tests
main "$@"
