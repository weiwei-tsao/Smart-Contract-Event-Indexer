# Smart Contract Event Indexer - API Documentation

## Overview

The Smart Contract Event Indexer provides a comprehensive API for querying blockchain events and managing contract monitoring. The API is built with a microservices architecture and exposes both REST and GraphQL endpoints.

## Architecture

```
Client → API Gateway (Port 8000) → Query Service (Port 8081) → Database
                ↓                            ↓
           Admin Service (Port 8082)    Redis Cache
```

## Quick Start

### 1. Start the Services

```bash
# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f api-gateway
```

### 2. Run Integration Tests

```bash
# Run the API integration test
./test_api.sh
```

### 3. Access the API

- **API Base URL**: `http://localhost:8000/api/v1`
- **Health Check**: `http://localhost:8000/api/v1/health`
- **GraphQL Playground**: `http://localhost:8000/playground` (Coming Soon)

## API Endpoints

### Health Check

#### GET /api/v1/health

Check the health status of the API Gateway and its dependencies.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-20T10:30:00Z",
  "services": {
    "database": {
      "status": "healthy",
      "latency": 5
    },
    "redis": {
      "status": "healthy",
      "latency": 2
    }
  }
}
```

### Contract Management

#### GET /api/v1/contracts

Get a list of monitored contracts.

**Query Parameters:**
- `is_active` (boolean): Filter by active status
- `limit` (int): Number of contracts to return (default: 20)
- `offset` (int): Number of contracts to skip (default: 0)

**Response:**
```json
{
  "contracts": [
    {
      "id": 1,
      "address": "0x1234567890123456789012345678901234567890",
      "name": "Uniswap V3 Pool",
      "abi": "[{\"type\":\"function\",\"name\":\"swap\",\"inputs\":[]}]",
      "start_block": 1000000,
      "current_block": 1000100,
      "confirm_blocks": 6,
      "is_active": true,
      "created_at": "2025-01-20T10:00:00Z",
      "updated_at": "2025-01-20T10:00:00Z"
    }
  ],
  "total_count": 1,
  "limit": 20,
  "offset": 0
}
```

#### POST /api/v1/contracts

Add a new contract for monitoring.

**Request Body:**
```json
{
  "address": "0x1234567890123456789012345678901234567890",
  "name": "Uniswap V3 Pool",
  "abi": "[{\"type\":\"function\",\"name\":\"swap\",\"inputs\":[]}]",
  "start_block": 1000000,
  "confirm_blocks": 6
}
```

**Response:**
```json
{
  "success": true,
  "contract_id": 1,
  "is_new": true,
  "message": "Contract added successfully"
}
```

#### GET /api/v1/contracts/{address}

Get details of a specific contract.

**Response:**
```json
{
  "contract": {
    "id": 1,
    "address": "0x1234567890123456789012345678901234567890",
    "name": "Uniswap V3 Pool",
    "abi": "[{\"type\":\"function\",\"name\":\"swap\",\"inputs\":[]}]",
    "start_block": 1000000,
    "current_block": 1000100,
    "confirm_blocks": 6,
    "is_active": true,
    "created_at": "2025-01-20T10:00:00Z",
    "updated_at": "2025-01-20T10:00:00Z"
  }
}
```

#### DELETE /api/v1/contracts/{address}

Remove a contract from monitoring.

**Response:**
```json
{
  "success": true,
  "message": "Contract removed successfully"
}
```

#### GET /api/v1/contracts/{address}/stats

Get statistics for a specific contract.

**Response:**
```json
{
  "contract_address": "0x1234567890123456789012345678901234567890",
  "total_events": 1500,
  "latest_block": 1000100,
  "current_block": 1000100,
  "indexer_delay": 0
}
```

### Event Queries

#### GET /api/v1/events

Get blockchain events with filtering and pagination.

**Query Parameters:**
- `contract` (string): Filter by contract address
- `event_name` (string): Filter by event name
- `from_block` (int): Start block number
- `to_block` (int): End block number
- `limit` (int): Number of events to return (default: 20)
- `offset` (int): Number of events to skip (default: 0)

**Response:**
```json
{
  "events": [
    {
      "id": 1,
      "contract_id": 1,
      "contract_address": "0x1234567890123456789012345678901234567890",
      "event_name": "Swap",
      "block_number": 1000100,
      "block_timestamp": "2025-01-20T10:30:00Z",
      "transaction_hash": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
      "transaction_index": 0,
      "log_index": 0,
      "args": {
        "sender": "0x1111111111111111111111111111111111111111",
        "recipient": "0x2222222222222222222222222222222222222222",
        "amount0": "1000000000000000000",
        "amount1": "2000000000000000000"
      },
      "created_at": "2025-01-20T10:30:00Z"
    }
  ],
  "total_count": 1500,
  "limit": 20,
  "offset": 0
}
```

#### GET /api/v1/events/tx/{txHash}

Get all events for a specific transaction.

**Response:**
```json
{
  "events": [
    {
      "id": 1,
      "contract_id": 1,
      "contract_address": "0x1234567890123456789012345678901234567890",
      "event_name": "Swap",
      "block_number": 1000100,
      "block_timestamp": "2025-01-20T10:30:00Z",
      "transaction_hash": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
      "transaction_index": 0,
      "log_index": 0,
      "args": {
        "sender": "0x1111111111111111111111111111111111111111",
        "recipient": "0x2222222222222222222222222222222222222222",
        "amount0": "1000000000000000000",
        "amount1": "2000000000000000000"
      },
      "created_at": "2025-01-20T10:30:00Z"
    }
  ],
  "total_count": 1
}
```

#### GET /api/v1/events/address/{address}

Get all events involving a specific address.

**Query Parameters:**
- `limit` (int): Number of events to return (default: 20)

**Response:**
```json
{
  "events": [
    {
      "id": 1,
      "contract_id": 1,
      "contract_address": "0x1234567890123456789012345678901234567890",
      "event_name": "Swap",
      "block_number": 1000100,
      "block_timestamp": "2025-01-20T10:30:00Z",
      "transaction_hash": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
      "transaction_index": 0,
      "log_index": 0,
      "args": {
        "sender": "0x1111111111111111111111111111111111111111",
        "recipient": "0x2222222222222222222222222222222222222222",
        "amount0": "1000000000000000000",
        "amount1": "2000000000000000000"
      },
      "created_at": "2025-01-20T10:30:00Z"
    }
  ],
  "total_count": 1,
  "address": "0x1111111111111111111111111111111111111111"
}
```

## Error Handling

The API returns standard HTTP status codes and structured error responses:

### Error Response Format

```json
{
  "error": "Error message describing what went wrong"
}
```

### Common Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

## Rate Limiting

The API implements rate limiting to prevent abuse:

- **Free Tier**: 100 requests per minute
- **Pro Tier**: 1000 requests per minute

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Request limit per minute
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Time when the rate limit resets

## CORS

The API supports Cross-Origin Resource Sharing (CORS) for web applications. By default, the following origins are allowed:

- `http://localhost:3000` (development)
- `http://localhost:8080` (development)

## Performance

### Response Times

- **P50**: < 50ms
- **P95**: < 200ms
- **P99**: < 500ms

### Caching

The API uses Redis for caching to improve performance:

- **Hot queries**: 30 seconds TTL
- **Statistics**: 5 minutes TTL
- **Historical data**: 1 hour TTL

## Monitoring

### Health Checks

- **API Gateway**: `http://localhost:8000/api/v1/health`
- **Query Service**: `http://localhost:8081/health` (internal)
- **Admin Service**: `http://localhost:8082/health` (internal)

### Metrics

Prometheus metrics are available at:
- **API Gateway**: `http://localhost:8000/metrics`
- **Query Service**: `http://localhost:8081/metrics` (internal)
- **Admin Service**: `http://localhost:8082/metrics` (internal)

## Development

### Running Locally

```bash
# Start infrastructure services
docker-compose up -d postgres redis ganache

# Run migrations
docker-compose run --rm migrate

# Start API services
docker-compose up -d api-gateway query-service admin-service

# Check logs
docker-compose logs -f api-gateway
```

### Testing

```bash
# Run integration tests
./test_api.sh

# Run unit tests
make test

# Run with coverage
make test-coverage
```

### Adding New Endpoints

1. Add the endpoint to the appropriate handler
2. Update the API documentation
3. Add tests for the new endpoint
4. Update the integration test script

## GraphQL (Coming Soon)

The API will also support GraphQL queries for more flexible data fetching:

```graphql
query GetEvents($filter: EventFilter, $pagination: PaginationInput) {
  events(filter: $filter, pagination: $pagination) {
    edges {
      node {
        id
        contractAddress
        eventName
        blockNumber
        blockTimestamp
        transactionHash
        args
      }
    }
    pageInfo {
      hasNextPage
      hasPreviousPage
      startCursor
      endCursor
    }
  }
}
```

## Support

For questions or issues:

1. Check the logs: `docker-compose logs -f api-gateway`
2. Run the health check: `curl http://localhost:8000/api/v1/health`
3. Check service status: `docker-compose ps`
4. Review the documentation in `/docs/`

## License

This project is licensed under the MIT License - see the LICENSE file for details.
