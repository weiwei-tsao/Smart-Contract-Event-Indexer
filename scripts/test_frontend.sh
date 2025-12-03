#!/bin/bash
# Frontend Integration Test Script
# Tests landing page, GraphQL playground, and API endpoints

set -e

API_URL="${API_URL:-http://localhost:8000}"
PASS=0
FAIL=0

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

log_pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASS++))
}

log_fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAIL++))
}

log_info() {
    echo -e "${YELLOW}→${NC} $1"
}

echo "========================================"
echo "Frontend Integration Tests"
echo "========================================"
echo "Target: $API_URL"
echo ""

# Test 1: Health Check
log_info "Testing health endpoint..."
if curl -s -f "$API_URL/api/v1/health" > /dev/null 2>&1; then
    log_pass "Health endpoint responding"
else
    log_fail "Health endpoint not responding"
    echo "  Make sure services are running: docker-compose up -d"
    exit 1
fi

# Test 2: Landing Page
log_info "Testing landing page (GET /)..."
LANDING=$(curl -s -w "%{http_code}" -o /tmp/landing.html "$API_URL/")
if [ "$LANDING" = "200" ]; then
    if grep -q "Smart Contract Event Indexer" /tmp/landing.html; then
        log_pass "Landing page served with correct content"
    else
        log_fail "Landing page missing expected content"
    fi
else
    log_fail "Landing page returned HTTP $LANDING"
fi

# Test 3: Config.js
log_info "Testing config.js..."
CONFIG=$(curl -s -w "%{http_code}" -o /tmp/config.js "$API_URL/config.js")
if [ "$CONFIG" = "200" ]; then
    if grep -q "window.CONFIG" /tmp/config.js; then
        log_pass "Config.js served correctly"
    else
        log_fail "Config.js missing CONFIG object"
    fi
else
    log_fail "Config.js returned HTTP $CONFIG"
fi

# Test 4: GraphQL Playground
log_info "Testing GraphQL Playground..."
PLAYGROUND=$(curl -s -w "%{http_code}" -o /tmp/playground.html "$API_URL/playground")
if [ "$PLAYGROUND" = "200" ]; then
    if grep -q "GraphQL" /tmp/playground.html; then
        log_pass "GraphQL Playground served"
    else
        log_fail "GraphQL Playground missing content"
    fi
else
    log_fail "GraphQL Playground returned HTTP $PLAYGROUND"
fi

# Test 5: GraphQL API - systemStatus query
log_info "Testing GraphQL API (systemStatus)..."
GRAPHQL_RESPONSE=$(curl -s -X POST "$API_URL/graphql" \
    -H "Content-Type: application/json" \
    -d '{"query":"query { systemStatus { isHealthy totalContracts totalEvents } }"}')

if echo "$GRAPHQL_RESPONSE" | grep -q '"isHealthy"'; then
    log_pass "GraphQL systemStatus query works"
else
    log_fail "GraphQL systemStatus query failed"
    echo "  Response: $GRAPHQL_RESPONSE"
fi

# Test 6: GraphQL API - contracts query
log_info "Testing GraphQL API (contracts)..."
CONTRACTS_RESPONSE=$(curl -s -X POST "$API_URL/graphql" \
    -H "Content-Type: application/json" \
    -d '{"query":"query { contracts { id address } }"}')

if echo "$CONTRACTS_RESPONSE" | grep -q '"contracts"'; then
    log_pass "GraphQL contracts query works"
else
    log_fail "GraphQL contracts query failed"
    echo "  Response: $CONTRACTS_RESPONSE"
fi

# Test 7: REST API - GET /api/v1/contracts
log_info "Testing REST API (GET /api/v1/contracts)..."
REST_CONTRACTS=$(curl -s -w "%{http_code}" -o /tmp/rest_contracts.json "$API_URL/api/v1/contracts")
if [ "$REST_CONTRACTS" = "200" ]; then
    log_pass "REST contracts endpoint works"
else
    log_fail "REST contracts endpoint returned HTTP $REST_CONTRACTS"
fi

# Test 8: REST API - GET /api/v1/events
log_info "Testing REST API (GET /api/v1/events)..."
REST_EVENTS=$(curl -s -w "%{http_code}" -o /tmp/rest_events.json "$API_URL/api/v1/events")
if [ "$REST_EVENTS" = "200" ]; then
    log_pass "REST events endpoint works"
else
    log_fail "REST events endpoint returned HTTP $REST_EVENTS"
fi

# Test 9: CORS Headers
log_info "Testing CORS headers..."
CORS_HEADERS=$(curl -s -I -X OPTIONS "$API_URL/graphql" \
    -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: POST" 2>&1)

if echo "$CORS_HEADERS" | grep -qi "access-control-allow"; then
    log_pass "CORS headers present"
else
    log_fail "CORS headers missing"
fi

# Summary
echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"
echo -e "Passed: ${GREEN}$PASS${NC}"
echo -e "Failed: ${RED}$FAIL${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
fi
