module github.com/smart-contract-event-indexer/api-gateway

go 1.21

require (
	github.com/99designs/gqlgen v0.17.42
	github.com/smart-contract-event-indexer/shared v0.0.0
	github.com/vektah/gqlparser/v2 v2.5.10
)

require (
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/ethereum/go-ethereum v1.13.5 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/sosodev/duration v1.1.0 // indirect
	golang.org/x/crypto v0.15.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
)

replace github.com/smart-contract-event-indexer/shared => ../../shared
