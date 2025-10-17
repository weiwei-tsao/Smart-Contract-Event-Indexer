#!/bin/bash

# Verification script for Phase 1 setup
# Usage: ./scripts/verify_setup.sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   Phase 1 Setup Verification${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

PASSED=0
FAILED=0
TOTAL=0

# Helper function to check
check() {
    TOTAL=$((TOTAL + 1))
    echo -n "$1... "
    if eval "$2" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}"
        FAILED=$((FAILED + 1))
        if [ ! -z "$3" ]; then
            echo -e "  ${YELLOW}→ $3${NC}"
        fi
    fi
}

echo -e "${BLUE}1. Checking Project Structure${NC}"
check "  Directory structure" "test -d services && test -d shared && test -d infrastructure && test -d migrations && test -d docs"
check "  Service directories" "test -d services/indexer-service && test -d services/api-gateway && test -d services/query-service && test -d services/admin-service"
check "  Shared modules" "test -d shared/models && test -d shared/proto && test -d shared/config && test -d shared/utils && test -d shared/database"
echo ""

echo -e "${BLUE}2. Checking Configuration Files${NC}"
check "  Go workspace" "test -f go.work"
check "  Makefile" "test -f Makefile"
check "  Docker Compose" "test -f docker-compose.yml"
check "  .gitignore" "test -f .gitignore"
check "  .env.example" "test -f .env.example"
check "  .editorconfig" "test -f .editorconfig"
check "  .golangci.yml" "test -f .golangci.yml"
echo ""

echo -e "${BLUE}3. Checking Go Modules${NC}"
check "  Shared go.mod" "test -f shared/go.mod"
check "  Indexer go.mod" "test -f services/indexer-service/go.mod"
check "  API Gateway go.mod" "test -f services/api-gateway/go.mod"
check "  Query Service go.mod" "test -f services/query-service/go.mod"
check "  Admin Service go.mod" "test -f services/admin-service/go.mod"
echo ""

echo -e "${BLUE}4. Checking Shared Modules${NC}"
check "  Data models" "test -f shared/models/types.go && test -f shared/models/contract.go && test -f shared/models/event.go"
check "  Config loader" "test -f shared/config/config.go"
check "  Logger" "test -f shared/utils/logger.go"
check "  Error handling" "test -f shared/utils/errors.go"
check "  Database utilities" "test -f shared/database/db.go && test -f shared/database/redis.go"
check "  gRPC proto files" "test -f shared/proto/query_service.proto && test -f shared/proto/admin_service.proto"
echo ""

echo -e "${BLUE}5. Checking Database Migrations${NC}"
check "  Migration up" "test -f migrations/001_initial_schema.up.sql"
check "  Migration down" "test -f migrations/001_initial_schema.down.sql"
echo ""

echo -e "${BLUE}6. Checking Docker Infrastructure${NC}"
check "  Indexer Dockerfile" "test -f infrastructure/docker/Dockerfile.indexer"
check "  API Gateway Dockerfile" "test -f infrastructure/docker/Dockerfile.api-gateway"
echo ""

echo -e "${BLUE}7. Checking Scripts${NC}"
check "  Health check script" "test -f scripts/health_check.sh && test -x scripts/health_check.sh"
check "  Wait for services script" "test -f scripts/wait_for_services.sh && test -x scripts/wait_for_services.sh"
echo ""

echo -e "${BLUE}8. Checking Documentation${NC}"
check "  README.md" "test -f README.md"
check "  Feature log" "test -f docs/development/features/001-project-infrastructure.md"
check "  Architecture docs" "test -f docs/smart_contract_event_indexer_architecture.md"
check "  PRD" "test -f docs/smart_contract_event_indexer_prd.md"
check "  Plan" "test -f docs/smart_contract_event_indexer_plan.md"
echo ""

echo -e "${BLUE}9. Checking Docker Services (if running)${NC}"
if docker-compose ps > /dev/null 2>&1; then
    check "  PostgreSQL" "docker-compose ps postgres | grep -q Up" "Run 'make dev-up' to start"
    check "  Redis" "docker-compose ps redis | grep -q Up" "Run 'make dev-up' to start"
    check "  Ganache" "docker-compose ps ganache | grep -q Up" "Run 'make dev-up' to start"
else
    echo -e "  ${YELLOW}ℹ Docker services not running (run 'make dev-up')${NC}"
fi
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   Verification Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo -e "Total Checks: ${TOTAL}"
echo -e "${GREEN}Passed: ${PASSED}${NC}"
echo -e "${RED}Failed: ${FAILED}${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed! Phase 1 setup is complete.${NC}"
    echo ""
    echo -e "${BLUE}Next Steps:${NC}"
    echo "  1. Start development environment: ${YELLOW}make dev-up${NC}"
    echo "  2. Verify services are healthy: ${YELLOW}make health-check${NC}"
    echo "  3. Run database migrations: ${YELLOW}make migrate-up${NC}"
    echo "  4. Begin Phase 2 development: Indexer Service core"
    echo ""
    exit 0
else
    echo -e "${RED}✗ Some checks failed. Please review the output above.${NC}"
    echo ""
    exit 1
fi

