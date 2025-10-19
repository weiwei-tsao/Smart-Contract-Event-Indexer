module github.com/smart-contract-event-indexer/query-service

go 1.21

require (
	github.com/redis/go-redis/v9 v9.3.0
	github.com/smart-contract-event-indexer/shared v0.0.0
	google.golang.org/grpc v1.59.0
	github.com/prometheus/client_golang v1.17.0
	go.uber.org/zap v1.26.0
	github.com/lib/pq v1.10.9
)

replace github.com/smart-contract-event-indexer/shared => ../../shared

