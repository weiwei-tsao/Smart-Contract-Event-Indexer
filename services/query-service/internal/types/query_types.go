package types

import (
	"time"

	"github.com/smart-contract-event-indexer/shared/models"
)

// EventQuery represents a query for events
type EventQuery struct {
	ContractAddress *string    `json:"contractAddress,omitempty"`
	EventName       *string    `json:"eventName,omitempty"`
	FromBlock       *int64     `json:"fromBlock,omitempty"`
	ToBlock         *int64     `json:"toBlock,omitempty"`
	FromDate        *time.Time `json:"fromDate,omitempty"`
	ToDate          *time.Time `json:"toDate,omitempty"`
	Addresses       []string   `json:"addresses,omitempty"`
	TransactionHash *string    `json:"transactionHash,omitempty"`
	First           *int32     `json:"first,omitempty"`
	After           *string    `json:"after,omitempty"`
	Before          *string    `json:"before,omitempty"`
	Last            *int32     `json:"last,omitempty"`
	Limit           int32      `json:"limit"`
	Offset          int32      `json:"offset"`
	OrderBy         string     `json:"orderBy"`
	OrderDirection  string     `json:"orderDirection"`
}

// AddressQuery represents a query for events by address
type AddressQuery struct {
	Address         string     `json:"address"`
	ContractAddress *string    `json:"contractAddress,omitempty"`
	EventName       *string    `json:"eventName,omitempty"`
	FromBlock       *int64     `json:"fromBlock,omitempty"`
	ToBlock         *int64     `json:"toBlock,omitempty"`
	FromDate        *time.Time `json:"fromDate,omitempty"`
	ToDate          *time.Time `json:"toDate,omitempty"`
	First           *int32     `json:"first,omitempty"`
	After           *string    `json:"after,omitempty"`
	Before          *string    `json:"before,omitempty"`
	Last            *int32     `json:"last,omitempty"`
	Limit           int32      `json:"limit"`
	Offset          int32      `json:"offset"`
	OrderBy         string     `json:"orderBy"`
	OrderDirection  string     `json:"orderDirection"`
}

// TransactionQuery represents a query for events by transaction
type TransactionQuery struct {
	TransactionHash string `json:"transactionHash"`
}

// StatsQuery represents a query for contract statistics
type StatsQuery struct {
	ContractAddress string `json:"contractAddress"`
}

// EventResponse represents the response for event queries
type EventResponse struct {
	Events     []*models.Event `json:"events"`
	TotalCount int32           `json:"totalCount"`
	PageInfo   *PageInfo       `json:"pageInfo"`
}

// PageInfo represents pagination information
type PageInfo struct {
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	StartCursor     *int64 `json:"startCursor,omitempty"`
	EndCursor       *int64 `json:"endCursor,omitempty"`
}

// StatsResponse represents the response for statistics queries
type StatsResponse struct {
	ContractAddress string    `json:"contractAddress"`
	TotalEvents     int64     `json:"totalEvents"`
	LatestBlock     int64     `json:"latestBlock"`
	CurrentBlock    int64     `json:"currentBlock"`
	IndexerDelay    int64     `json:"indexerDelay"`
	LastUpdated     time.Time `json:"lastUpdated"`
	UniqueAddresses *int64    `json:"uniqueAddresses,omitempty"`
}

// TimeRangeQuery represents aggregation requests for a contract.
type TimeRangeQuery struct {
	ContractAddress string    `json:"contractAddress"`
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	Interval        string    `json:"interval"`
	EventName       *string   `json:"eventName,omitempty"`
}

// TimeBucketStat represents a bucketed aggregation response.
type TimeBucketStat struct {
	BucketStart time.Time `json:"bucketStart"`
	BucketEnd   time.Time `json:"bucketEnd"`
	EventCount  int64     `json:"eventCount"`
}

// TopNQuery represents a request for top addresses.
type TopNQuery struct {
	ContractAddress string        `json:"contractAddress"`
	EventName       *string       `json:"eventName,omitempty"`
	Limit           int           `json:"limit"`
	Window          time.Duration `json:"window"`
}

// TopAddressStat represents aggregated address activity.
type TopAddressStat struct {
	Address    string `json:"address"`
	EventCount int64  `json:"eventCount"`
}
