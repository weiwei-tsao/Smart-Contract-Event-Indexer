package models

import (
	"time"
)

// IndexerState represents the current state of the indexer for a contract
type IndexerState struct {
	ContractAddress  Address   `db:"contract_address" json:"contractAddress"`
	LastIndexedBlock int64     `db:"last_indexed_block" json:"lastIndexedBlock"`
	UpdatedAt        time.Time `db:"updated_at" json:"updatedAt"`
}

// BlockCache represents cached block information for reorg detection
type BlockCache struct {
	BlockNumber int64     `db:"block_number" json:"blockNumber"`
	BlockHash   Hash      `db:"block_hash" json:"blockHash"`
	ParentHash  Hash      `db:"parent_hash" json:"parentHash"`
	Timestamp   time.Time `db:"timestamp" json:"timestamp"`
	CachedAt    time.Time `db:"cached_at" json:"cachedAt"`
}

// ContractStats represents statistics for a contract
type ContractStats struct {
	ContractAddress Address   `json:"contractAddress"`
	TotalEvents     int64     `json:"totalEvents"`
	LatestBlock     int64     `json:"latestBlock"`
	CurrentBlock    int64     `json:"currentBlock"`
	IndexerDelay    int64     `json:"indexerDelay"` // blocks behind
	LastUpdated     time.Time `json:"lastUpdated"`
}

// BackfillJob represents a historical data backfill job
type BackfillJob struct {
	ID              string    `db:"id" json:"id"`
	ContractAddress Address   `db:"contract_address" json:"contractAddress"`
	FromBlock       int64     `db:"from_block" json:"fromBlock"`
	ToBlock         int64     `db:"to_block" json:"toBlock"`
	CurrentBlock    int64     `db:"current_block" json:"currentBlock"`
	Status          string    `db:"status" json:"status"` // pending, running, completed, failed
	ErrorMessage    *string   `db:"error_message" json:"errorMessage,omitempty"`
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
	CompletedAt     *time.Time `db:"completed_at" json:"completedAt,omitempty"`
}

// Progress returns the progress percentage of the backfill job
func (b *BackfillJob) Progress() float64 {
	if b.ToBlock <= b.FromBlock {
		return 0
	}
	total := float64(b.ToBlock - b.FromBlock)
	current := float64(b.CurrentBlock - b.FromBlock)
	return (current / total) * 100
}

