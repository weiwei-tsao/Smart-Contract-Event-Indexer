#!/bin/bash

# Wait for all services to be ready
# Usage: ./scripts/wait_for_services.sh [timeout_seconds]

set -e

TIMEOUT=${1:-60}
ELAPSED=0
INTERVAL=2

echo "Waiting for services to be ready (timeout: ${TIMEOUT}s)..."

while [ $ELAPSED -lt $TIMEOUT ]; do
    if ./scripts/health_check.sh > /dev/null 2>&1; then
        echo "All services are ready!"
        exit 0
    fi
    
    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
    echo "Waiting... ${ELAPSED}s elapsed"
done

echo "Timeout reached! Services are not ready."
./scripts/health_check.sh
exit 1

