module github.com/smart-contract-event-indexer/query-service

go 1.21

require (
	github.com/redis/go-redis/v9 v9.3.0
	github.com/smart-contract-event-indexer/shared v0.0.0
	google.golang.org/grpc v1.59.0
)

replace github.com/smart-contract-event-indexer/shared => ../../shared

