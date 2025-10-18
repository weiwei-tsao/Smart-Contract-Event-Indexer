#!/bin/bash

# Integration Test Runner
# This script sets up the test environment and runs integration tests

set -e

echo "🧪 Smart Contract Event Indexer - Integration Tests"
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "go.work" ]; then
    print_error "Please run this script from the project root directory"
    exit 1
fi

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

print_status "Starting test environment setup..."

# Start required services
print_status "Starting Docker services (PostgreSQL, Redis, Ganache)..."
docker-compose up -d postgres redis ganache

# Wait for services to be ready
print_status "Waiting for services to be ready..."
sleep 10

# Check if services are running
print_status "Checking service health..."

# Check PostgreSQL
if docker-compose exec -T postgres pg_isready -U indexer >/dev/null 2>&1; then
    print_success "PostgreSQL is ready"
else
    print_error "PostgreSQL is not ready"
    exit 1
fi

# Check Redis
if docker-compose exec -T redis redis-cli ping >/dev/null 2>&1; then
    print_success "Redis is ready"
else
    print_error "Redis is not ready"
    exit 1
fi

# Check Ganache
if curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' http://localhost:8545 >/dev/null 2>&1; then
    print_success "Ganache is ready"
else
    print_error "Ganache is not ready"
    exit 1
fi

# Run database migrations
print_status "Running database migrations..."
docker-compose run --rm migrate

# Install test dependencies
print_status "Installing test dependencies..."
cd services/indexer-service
go mod tidy

# Install testify for testing
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require

# Run integration tests
print_status "Running integration tests..."
echo ""

# Run tests with verbose output
CGO_ENABLED=0 go test -v ./tests/integration/... -timeout 5m

# Check test results
if [ $? -eq 0 ]; then
    print_success "All integration tests passed! 🎉"
    echo ""
    print_status "Test Summary:"
    echo "  ✅ Service connectivity tests"
    echo "  ✅ Database schema verification"
    echo "  ✅ Data operations (CRUD)"
    echo "  ✅ Binary build and execution"
    echo ""
    print_status "Integration tests completed successfully!"
else
    print_error "Some integration tests failed"
    exit 1
fi

# Optional: Keep services running for manual testing
if [ "$1" = "--keep-running" ]; then
    print_status "Keeping services running for manual testing..."
    print_status "To stop services: docker-compose down"
else
    print_status "Stopping test services..."
    docker-compose down
fi

print_success "Integration test run completed!"
