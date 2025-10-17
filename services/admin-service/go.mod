module github.com/smart-contract-event-indexer/admin-service

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/smart-contract-event-indexer/shared v0.0.0
	google.golang.org/grpc v1.59.0
)

replace github.com/smart-contract-event-indexer/shared => ../../shared

