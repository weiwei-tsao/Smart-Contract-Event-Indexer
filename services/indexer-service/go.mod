module github.com/smart-contract-event-indexer/indexer-service

go 1.21

require (
	github.com/ethereum/go-ethereum v1.13.5
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.9
	github.com/smart-contract-event-indexer/shared v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/smart-contract-event-indexer/shared => ../../shared
