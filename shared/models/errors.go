package models

import "errors"

// Common errors
var (
	ErrContractNotFound       = errors.New("contract not found")
	ErrContractAlreadyExists  = errors.New("contract already exists")
	ErrInvalidContractAddress = errors.New("invalid contract address")
	ErrInvalidContractName    = errors.New("invalid contract name")
	ErrInvalidContractABI     = errors.New("invalid contract ABI")
	ErrInvalidBlockNumber     = errors.New("invalid block number")
	ErrInvalidConfirmBlocks   = errors.New("invalid confirm blocks: must be between 1 and 100")
	ErrInvalidTransactionHash = errors.New("invalid transaction hash")
	ErrEventNotFound          = errors.New("event not found")
	ErrInvalidEventFilter     = errors.New("invalid event filter")
	ErrInvalidPagination      = errors.New("invalid pagination parameters")
	ErrDatabaseConnection     = errors.New("database connection error")
	ErrRedisConnection        = errors.New("redis connection error")
	ErrRPCConnection          = errors.New("rpc connection error")
)

