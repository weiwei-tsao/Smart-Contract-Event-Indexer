package models

import (
	"time"
)

// Contract represents a smart contract being monitored
type Contract struct {
	ID            int64     `db:"id" json:"id"`
	Address       Address   `db:"address" json:"address"`
	ABI           string    `db:"abi" json:"abi"`
	Name          string    `db:"name" json:"name"`
	StartBlock    int64     `db:"start_block" json:"startBlock"`
	CurrentBlock  int64     `db:"current_block" json:"currentBlock"`
	ConfirmBlocks int       `db:"confirm_blocks" json:"confirmBlocks"` // Number of blocks to wait for confirmation
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time `db:"updated_at" json:"updatedAt"`
}

// Validate checks if the contract data is valid
func (c *Contract) Validate() error {
	if err := c.Address.Validate(); err != nil {
		return err
	}
	if c.Name == "" {
		return ErrInvalidContractName
	}
	if c.ABI == "" {
		return ErrInvalidContractABI
	}
	if c.StartBlock < 0 {
		return ErrInvalidBlockNumber
	}
	if c.ConfirmBlocks < 1 || c.ConfirmBlocks > 100 {
		return ErrInvalidConfirmBlocks
	}
	return nil
}

// IsConfirmed checks if a block number is confirmed based on the latest block
func (c *Contract) IsConfirmed(blockNumber, latestBlock int64) bool {
	return latestBlock-blockNumber >= int64(c.ConfirmBlocks)
}

// AddContractInput represents input for adding a new contract
type AddContractInput struct {
	Address       Address              `json:"address"`
	ABI           string               `json:"abi"`
	Name          string               `json:"name"`
	StartBlock    int64                `json:"startBlock"`
	ConfirmBlocks *int                 `json:"confirmBlocks,omitempty"` // Optional, defaults to 6
	Strategy      ConfirmationStrategy `json:"strategy,omitempty"`      // Optional, overrides confirmBlocks
}

// GetConfirmBlocks returns the confirmation blocks based on strategy or explicit value
func (i *AddContractInput) GetConfirmBlocks() int {
	if i.Strategy != "" {
		return i.Strategy.ToBlocks()
	}
	if i.ConfirmBlocks != nil {
		return *i.ConfirmBlocks
	}
	return 6 // default balanced strategy
}

