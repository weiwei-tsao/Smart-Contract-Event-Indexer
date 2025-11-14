#!/bin/bash

# Phase 3 API Layer Testing Script
# This script helps test Phase 3 features locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Phase 3 API Layer Testing Script${NC}"
echo "=================================="

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
    fi
}

# Function to test service build
test_service_build() {
    local service_name=$1
    local service_path=$2
    local binary_name=$3
    
    echo -e "\n${YELLOW}Testing $service_name build...${NC}"
    cd "$service_path"
    
    if go build -o "../../bin/$binary_name" ./cmd/main.go 2>/dev/null; then
        print_status 0 "$service_name builds successfully"
        return 0
    else
        print_status 1 "$service_name build failed"
        echo -e "${RED}Build errors:${NC}"
        go build -o "../../bin/$binary_name" ./cmd/main.go 2>&1 | head -10
        return 1
    fi
}

# Function to check dependencies
check_dependencies() {
    local service_name=$1
    local service_path=$2
    
    echo -e "\n${YELLOW}Checking $service_name dependencies...${NC}"
    cd "$service_path"
    
    if go mod tidy 2>/dev/null; then
        print_status 0 "$service_name dependencies are clean"
        return 0
    else
        print_status 1 "$service_name has dependency issues"
        return 1
    fi
}

# Function to test GraphQL schema
test_graphql_schema() {
    echo -e "\n${YELLOW}Testing GraphQL schema...${NC}"
    
    if [ -f "graphql/schema.graphql" ]; then
        # Basic syntax check (simplified)
        if grep -q "type Query" graphql/schema.graphql && grep -q "type Mutation" graphql/schema.graphql; then
            print_status 0 "GraphQL schema has Query and Mutation types"
        else
            print_status 1 "GraphQL schema missing required types"
        fi
        
        # Check for custom scalars
        if grep -q "scalar DateTime" graphql/schema.graphql && grep -q "scalar BigInt" graphql/schema.graphql; then
            print_status 0 "GraphQL schema has custom scalars"
        else
            print_status 1 "GraphQL schema missing custom scalars"
        fi
    else
        print_status 1 "GraphQL schema file not found"
    fi
}

# Function to test database connection
test_database_connection() {
    echo -e "\n${YELLOW}Testing database connection...${NC}"
    
    if docker-compose ps postgres | grep -q "Up"; then
        print_status 0 "PostgreSQL container is running"
        
        # Test connection
        if docker-compose exec -T postgres pg_isready -U indexer >/dev/null 2>&1; then
            print_status 0 "PostgreSQL is accepting connections"
        else
            print_status 1 "PostgreSQL is not accepting connections"
        fi
    else
        print_status 1 "PostgreSQL container is not running"
        echo -e "${YELLOW}Run 'make dev-up' to start infrastructure${NC}"
    fi
}

# Function to test Redis connection
test_redis_connection() {
    echo -e "\n${YELLOW}Testing Redis connection...${NC}"
    
    if docker-compose ps redis | grep -q "Up"; then
        print_status 0 "Redis container is running"
        
        # Test connection
        if docker-compose exec -T redis redis-cli ping >/dev/null 2>&1; then
            print_status 0 "Redis is accepting connections"
        else
            print_status 1 "Redis is not accepting connections"
        fi
    else
        print_status 1 "Redis container is not running"
        echo -e "${YELLOW}Run 'make dev-up' to start infrastructure${NC}"
    fi
}

# Function to run tests
run_tests() {
    echo -e "\n${YELLOW}Running tests...${NC}"
    
    if go test ./... -v 2>/dev/null; then
        print_status 0 "All tests pass"
    else
        print_status 1 "Some tests failed"
        echo -e "${RED}Test failures:${NC}"
        go test ./... -v 2>&1 | grep -A 5 "FAIL"
    fi
}

# Main execution
main() {
    echo -e "${BLUE}Starting Phase 3 testing...${NC}"
    
    # Check if we're in the right directory
    if [ ! -f "go.work" ]; then
        echo -e "${RED}Error: Not in project root directory${NC}"
        echo "Please run this script from the project root"
        exit 1
    fi
    
    # Create bin directory if it doesn't exist
    mkdir -p bin
    
    # Test infrastructure
    test_database_connection
    test_redis_connection
    
    # Test GraphQL schema
    test_graphql_schema
    
    # Test service builds
    local build_success=0
    
    if test_service_build "API Gateway" "services/api-gateway" "api-gateway"; then
        build_success=$((build_success + 1))
    fi
    
    if test_service_build "Query Service" "services/query-service" "query-service"; then
        build_success=$((build_success + 1))
    fi
    
    if test_service_build "Admin Service" "services/admin-service" "admin-service"; then
        build_success=$((build_success + 1))
    fi
    
    # Test dependencies
    local dep_success=0
    
    if check_dependencies "API Gateway" "services/api-gateway"; then
        dep_success=$((dep_success + 1))
    fi
    
    if check_dependencies "Query Service" "services/query-service"; then
        dep_success=$((dep_success + 1))
    fi
    
    if check_dependencies "Admin Service" "services/admin-service"; then
        dep_success=$((dep_success + 1))
    fi
    
    # Run tests
    run_tests
    
    # Summary
    echo -e "\n${BLUE}Testing Summary${NC}"
    echo "==============="
    echo -e "Services built: $build_success/3"
    echo -e "Dependencies clean: $dep_success/3"
    
    if [ $build_success -eq 3 ]; then
        echo -e "${GREEN}All services build successfully!${NC}"
    else
        echo -e "${YELLOW}Some services have build issues${NC}"
    fi
    
    if [ $dep_success -eq 3 ]; then
        echo -e "${GREEN}All dependencies are clean!${NC}"
    else
        echo -e "${YELLOW}Some services have dependency issues${NC}"
    fi
    
    echo -e "\n${BLUE}Next Steps:${NC}"
    echo "1. Fix any build errors"
    echo "2. Start services: make run-api & make run-query & make run-admin"
    echo "3. Test GraphQL Playground: http://localhost:8000/playground"
    echo "4. Check service logs for any runtime errors"
}

# Run main function
main "$@"
