module github.com/smart-contract-event-indexer/query-service

go 1.24.0

toolchain go1.24.9

require (
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.17.0
	github.com/redis/go-redis/v9 v9.3.0
	github.com/smart-contract-event-indexer/shared v0.0.0
	go.uber.org/zap v1.26.0
	google.golang.org/grpc v1.76.0
)

require (
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace github.com/smart-contract-event-indexer/shared => ../../shared
