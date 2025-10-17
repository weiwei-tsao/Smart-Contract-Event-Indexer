package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Address represents an Ethereum address
type Address string

// Hash represents an Ethereum transaction or block hash
type Hash string

// BigInt represents a big integer as a string to preserve precision
type BigInt string

// Validate checks if the address is valid
func (a Address) Validate() error {
	if !common.IsHexAddress(string(a)) {
		return fmt.Errorf("invalid ethereum address: %s", a)
	}
	return nil
}

// ToCommonAddress converts Address to go-ethereum common.Address
func (a Address) ToCommonAddress() common.Address {
	return common.HexToAddress(string(a))
}

// Normalize returns the checksummed address
func (a Address) Normalize() Address {
	return Address(common.HexToAddress(string(a)).Hex())
}

// Validate checks if the hash is valid
func (h Hash) Validate() error {
	if !common.IsHexHash(string(h)) {
		return fmt.Errorf("invalid ethereum hash: %s", h)
	}
	return nil
}

// ToCommonHash converts Hash to go-ethereum common.Hash
func (h Hash) ToCommonHash() common.Hash {
	return common.HexToHash(string(h))
}

// JSONB is a custom type for PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSONB: expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, j)
}

// Timestamp wraps time.Time for consistent handling
type Timestamp struct {
	time.Time
}

// NewTimestamp creates a new Timestamp
func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{Time: t}
}

// Now returns current timestamp
func Now() Timestamp {
	return Timestamp{Time: time.Now().UTC()}
}

// ConfirmationStrategy represents the confirmation block strategy
type ConfirmationStrategy string

const (
	ConfirmationRealtime ConfirmationStrategy = "realtime" // 1 block
	ConfirmationBalanced ConfirmationStrategy = "balanced" // 6 blocks (default)
	ConfirmationSafe     ConfirmationStrategy = "safe"     // 12 blocks
)

// ToBlocks returns the number of blocks for the strategy
func (cs ConfirmationStrategy) ToBlocks() int {
	switch cs {
	case ConfirmationRealtime:
		return 1
	case ConfirmationSafe:
		return 12
	default:
		return 6 // balanced
	}
}

