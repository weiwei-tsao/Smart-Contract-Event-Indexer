module github.com/smart-contract-event-indexer/indexer-service

go 1.21

require (
	github.com/ethereum/go-ethereum v1.13.5
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.9
	github.com/smart-contract-event-indexer/shared v0.0.0
)

replace github.com/smart-contract-event-indexer/shared => ../../shared

