package blockchain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// Client wraps an Ethereum client for blockchain interactions
type Client struct {
	endpoint string
	client   *ethclient.Client
	logger   utils.Logger
}

// NewClient creates a new blockchain client
func NewClient(endpoint string, logger utils.Logger) *Client {
	return &Client{
		endpoint: endpoint,
		logger:   logger,
	}
}

// Connect establishes connection to the Ethereum node
func (c *Client) Connect(ctx context.Context) error {
	c.logger.WithField("endpoint", c.endpoint).Info("Connecting to Ethereum node")
	
	client, err := ethclient.DialContext(ctx, c.endpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}
	
	c.client = client
	
	// Verify connection by getting chain ID
	chainID, err := c.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}
	
	c.logger.WithField("chain_id", chainID.String()).Info("Successfully connected to Ethereum node")
	return nil
}

// Close closes the connection to the Ethereum node
func (c *Client) Close() {
	if c.client != nil {
		c.client.Close()
		c.logger.Info("Disconnected from Ethereum node")
	}
}

// GetLatestBlockNumber returns the latest block number
func (c *Client) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	blockNumber, err := c.client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block number: %w", err)
	}
	return int64(blockNumber), nil
}

// GetBlockByNumber returns a block by its number
func (c *Client) GetBlockByNumber(ctx context.Context, blockNumber int64) (*types.Block, error) {
	block, err := c.client.BlockByNumber(ctx, big.NewInt(blockNumber))
	if err != nil {
		return nil, fmt.Errorf("failed to get block %d: %w", blockNumber, err)
	}
	return block, nil
}

// GetBlockByHash returns a block by its hash
func (c *Client) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, err := c.client.BlockByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get block by hash %s: %w", hash.Hex(), err)
	}
	return block, nil
}

// GetLogs retrieves logs for a given filter query
func (c *Client) GetLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	logs, err := c.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	return logs, nil
}

// GetLogsForContract retrieves logs for a specific contract within a block range
func (c *Client) GetLogsForContract(ctx context.Context, contractAddress common.Address, fromBlock, toBlock int64) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock),
		ToBlock:   big.NewInt(toBlock),
		Addresses: []common.Address{contractAddress},
	}
	
	c.logger.WithFields(map[string]interface{}{
		"contract":   contractAddress.Hex(),
		"from_block": fromBlock,
		"to_block":   toBlock,
	}).Debug("Fetching logs for contract")
	
	return c.GetLogs(ctx, query)
}

// ChainID returns the chain ID
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	return c.client.ChainID(ctx)
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	return c.client != nil
}

// HealthCheck performs a health check on the connection
func (c *Client) HealthCheck(ctx context.Context) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}
	
	// Try to get the latest block number as a health check
	_, err := c.GetLatestBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	
	return nil
}

