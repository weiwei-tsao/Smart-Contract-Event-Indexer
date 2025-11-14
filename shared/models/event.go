package models

import (
	"time"
)

// Event represents a blockchain event that has been indexed
type Event struct {
	ID               int64     `db:"id" json:"id"`
	ContractAddress  Address   `db:"contract_address" json:"contractAddress"`
	EventName        string    `db:"event_name" json:"eventName"`
	BlockNumber      int64     `db:"block_number" json:"blockNumber"`
	BlockHash        Hash      `db:"block_hash" json:"blockHash"`
	TransactionHash  Hash      `db:"transaction_hash" json:"transactionHash"`
	TransactionIndex int       `db:"transaction_index" json:"transactionIndex"`
	LogIndex         int       `db:"log_index" json:"logIndex"`
	Args             JSONB     `db:"args" json:"args"`
	Timestamp        time.Time `db:"timestamp" json:"timestamp"`
	CreatedAt        time.Time `db:"created_at" json:"createdAt"`
}

// EventArg represents a single argument from an event
type EventArg struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Value   interface{} `json:"value"`
	Indexed bool        `json:"indexed"`
}

// EventFilter represents filters for querying events
type EventFilter struct {
	ContractAddress *Address  `json:"contractAddress,omitempty"`
	EventName       *string   `json:"eventName,omitempty"`
	FromBlock       *int64    `json:"fromBlock,omitempty"`
	ToBlock         *int64    `json:"toBlock,omitempty"`
	TransactionHash *Hash     `json:"transactionHash,omitempty"`
	Addresses       []Address `json:"addresses,omitempty"`
	Address         *Address  `json:"address,omitempty"` // For filtering by address in args
}

// Pagination represents pagination parameters
type Pagination struct {
	First  int     `json:"first"`
	After  *string `json:"after,omitempty"`
	Before *string `json:"before,omitempty"`
}

// EventConnection represents a paginated list of events
type EventConnection struct {
	Edges      []*EventEdge `json:"edges"`
	PageInfo   PageInfo     `json:"pageInfo"`
	TotalCount int          `json:"totalCount"`
}

// EventEdge represents an edge in the event connection
type EventEdge struct {
	Node   *Event `json:"node"`
	Cursor string `json:"cursor"`
}

// PageInfo represents pagination information
type PageInfo struct {
	HasNextPage     bool    `json:"hasNextPage"`
	HasPreviousPage bool    `json:"hasPreviousPage"`
	StartCursor     *string `json:"startCursor,omitempty"`
	EndCursor       *string `json:"endCursor,omitempty"`
}
