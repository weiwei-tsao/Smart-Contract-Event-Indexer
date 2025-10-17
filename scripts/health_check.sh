#!/bin/bash

# Health check script for all services
# Usage: ./scripts/health_check.sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Symbols
CHECK="${GREEN}✓${NC}"
CROSS="${RED}✗${NC}"
WARN="${YELLOW}⚠${NC}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   Smart Contract Event Indexer${NC}"
echo -e "${BLUE}   Health Check${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check PostgreSQL
echo -n "PostgreSQL...              "
if docker-compose exec -T postgres pg_isready -U indexer -d event_indexer > /dev/null 2>&1; then
    echo -e "${CHECK} Ready"
    POSTGRES_OK=1
else
    echo -e "${CROSS} Not ready"
    POSTGRES_OK=0
fi

# Check Redis
echo -n "Redis...                   "
if docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; then
    echo -e "${CHECK} Ready"
    REDIS_OK=1
else
    echo -e "${CROSS} Not ready"
    REDIS_OK=0
fi

# Check Ganache
echo -n "Ganache (RPC)...           "
if curl -s http://localhost:8545 > /dev/null 2>&1; then
    echo -e "${CHECK} Ready"
    GANACHE_OK=1
else
    echo -e "${CROSS} Not ready"
    GANACHE_OK=0
fi

# Check Adminer
echo -n "Adminer (UI)...            "
if curl -s http://localhost:8080 > /dev/null 2>&1; then
    echo -e "${CHECK} Ready"
    ADMINER_OK=1
else
    echo -e "${CROSS} Not ready"
    ADMINER_OK=0
fi

echo ""
echo -e "${BLUE}========================================${NC}"

# Summary
TOTAL=4
READY=$((POSTGRES_OK + REDIS_OK + GANACHE_OK + ADMINER_OK))

if [ $READY -eq $TOTAL ]; then
    echo -e "${GREEN}All services are ready! ($READY/$TOTAL)${NC}"
    exit 0
elif [ $READY -gt 0 ]; then
    echo -e "${YELLOW}Some services are not ready ($READY/$TOTAL)${NC}"
    exit 1
else
    echo -e "${RED}No services are ready! ($READY/$TOTAL)${NC}"
    exit 1
fi

