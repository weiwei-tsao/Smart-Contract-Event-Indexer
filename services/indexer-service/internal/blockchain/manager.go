package blockchain

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// RPCManager manages multiple RPC endpoints with fallback support
type RPCManager struct {
	primaryClient   *Client
	fallbackClients []*Client
	currentClient   *Client
	mu              sync.RWMutex
	logger          utils.Logger
	
	// Health check settings
	healthCheckInterval time.Duration
	maxRetries          int
	retryDelay          time.Duration
}

// RPCConfig holds configuration for RPC manager
type RPCConfig struct {
	PrimaryEndpoint    string
	FallbackEndpoints  []string
	HealthCheckInterval time.Duration
	MaxRetries         int
	RetryDelay         time.Duration
}

// NewRPCManager creates a new RPC manager with fallback support
func NewRPCManager(config *RPCConfig, logger utils.Logger) *RPCManager {
	primaryClient := NewClient(config.PrimaryEndpoint, logger)
	
	fallbackClients := make([]*Client, 0, len(config.FallbackEndpoints))
	for _, endpoint := range config.FallbackEndpoints {
		fallbackClients = append(fallbackClients, NewClient(endpoint, logger))
	}
	
	return &RPCManager{
		primaryClient:       primaryClient,
		fallbackClients:     fallbackClients,
		currentClient:       primaryClient,
		logger:              logger,
		healthCheckInterval: config.HealthCheckInterval,
		maxRetries:          config.MaxRetries,
		retryDelay:          config.RetryDelay,
	}
}

// Connect establishes connection to the primary RPC endpoint
func (m *RPCManager) Connect(ctx context.Context) error {
	m.logger.Info("Connecting to primary RPC endpoint")
	
	// Try primary endpoint first
	if err := m.primaryClient.Connect(ctx); err != nil {
		m.logger.WithError(err).Warn("Failed to connect to primary RPC, trying fallbacks")
		
		// Try fallback endpoints
		for i, client := range m.fallbackClients {
			m.logger.WithField("fallback_index", i).Info("Trying fallback RPC endpoint")
			
			if err := client.Connect(ctx); err != nil {
				m.logger.WithError(err).WithField("fallback_index", i).Warn("Failed to connect to fallback RPC")
				continue
			}
			
			// Successfully connected to fallback
			m.mu.Lock()
			m.currentClient = client
			m.mu.Unlock()
			
			m.logger.WithField("fallback_index", i).Info("Connected to fallback RPC endpoint")
			return nil
		}
		
		return fmt.Errorf("failed to connect to any RPC endpoint")
	}
	
	m.logger.Info("Successfully connected to primary RPC endpoint")
	return nil
}

// GetCurrentClient returns the currently active client
func (m *RPCManager) GetCurrentClient() *Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentClient
}

// GetLatestBlockNumber returns the latest block number with fallback
func (m *RPCManager) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	result, err := m.executeWithFallback(ctx, func(client *Client) (interface{}, error) {
		blockNum, err := client.GetLatestBlockNumber(ctx)
		return blockNum, err
	})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// GetLogs retrieves logs with fallback support
func (m *RPCManager) GetLogs(ctx context.Context, contractAddress string, fromBlock, toBlock int64) (interface{}, error) {
	return m.executeWithFallback(ctx, func(client *Client) (interface{}, error) {
		return client.GetLogsForContract(ctx, 
			common.HexToAddress(contractAddress),
			fromBlock,
			toBlock,
		)
	})
}

// executeWithFallback executes a function with automatic fallback on failure
func (m *RPCManager) executeWithFallback(ctx context.Context, fn func(*Client) (interface{}, error)) (interface{}, error) {
	var lastErr error
	
	for attempt := 0; attempt < m.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(m.retryDelay):
			}
		}
		
		client := m.GetCurrentClient()
		result, err := fn(client)
		
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		m.logger.WithError(err).WithField("attempt", attempt+1).Warn("RPC call failed")
		
		// Check if we should fallback
		if m.shouldFallback(err) {
			if err := m.switchToFallback(ctx); err != nil {
				m.logger.WithError(err).Warn("Failed to switch to fallback")
			}
		}
	}
	
	return nil, fmt.Errorf("all RPC endpoints failed after %d attempts: %w", m.maxRetries, lastErr)
}

// shouldFallback determines if an error warrants switching to a fallback
func (m *RPCManager) shouldFallback(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for common connection/timeout errors
	errStr := err.Error()
	return contains(errStr, []string{
		"connection refused",
		"connection reset",
		"context deadline exceeded",
		"timeout",
		"EOF",
		"broken pipe",
	})
}

// switchToFallback switches to the next available fallback endpoint
func (m *RPCManager) switchToFallback(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	currentClient := m.currentClient
	
	// If we're on primary, try first fallback
	if currentClient == m.primaryClient && len(m.fallbackClients) > 0 {
		if err := m.fallbackClients[0].Connect(ctx); err == nil {
			m.currentClient = m.fallbackClients[0]
			m.logger.Info("Switched to first fallback RPC")
			return nil
		}
	}
	
	// Try other fallbacks
	for i, client := range m.fallbackClients {
		if client == currentClient {
			continue
		}
		
		if err := client.Connect(ctx); err == nil {
			m.currentClient = client
			m.logger.WithField("fallback_index", i).Info("Switched to fallback RPC")
			return nil
		}
	}
	
	// Try to reconnect to primary
	if err := m.primaryClient.Connect(ctx); err == nil {
		m.currentClient = m.primaryClient
		m.logger.Info("Reconnected to primary RPC")
		return nil
	}
	
	return fmt.Errorf("no healthy RPC endpoints available")
}

// StartHealthCheck starts periodic health checks on all endpoints
func (m *RPCManager) StartHealthCheck(ctx context.Context) {
	if m.healthCheckInterval == 0 {
		return
	}
	
	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()
	
	m.logger.WithField("interval", m.healthCheckInterval).Info("Starting RPC health checks")
	
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Stopping RPC health checks")
			return
			
		case <-ticker.C:
			m.checkHealth(ctx)
		}
	}
}

// checkHealth performs health checks on all endpoints
func (m *RPCManager) checkHealth(ctx context.Context) {
	// Check primary
	if err := m.primaryClient.HealthCheck(ctx); err != nil {
		m.logger.WithError(err).Warn("Primary RPC health check failed")
	} else {
		// If primary is healthy and we're on fallback, switch back
		m.mu.RLock()
		isOnFallback := m.currentClient != m.primaryClient
		m.mu.RUnlock()
		
		if isOnFallback {
			m.mu.Lock()
			m.currentClient = m.primaryClient
			m.mu.Unlock()
			m.logger.Info("Primary RPC is healthy, switched back from fallback")
		}
	}
	
	// Check fallbacks
	for i, client := range m.fallbackClients {
		if err := client.HealthCheck(ctx); err != nil {
			m.logger.WithError(err).WithField("fallback_index", i).Debug("Fallback RPC health check failed")
		}
	}
}

// Close closes all RPC connections
func (m *RPCManager) Close() {
	m.primaryClient.Close()
	for _, client := range m.fallbackClients {
		client.Close()
	}
	m.logger.Info("All RPC connections closed")
}

// GetStats returns statistics about RPC endpoints
func (m *RPCManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	currentEndpoint := "unknown"
	if m.currentClient == m.primaryClient {
		currentEndpoint = "primary"
	} else {
		for i, client := range m.fallbackClients {
			if client == m.currentClient {
				currentEndpoint = fmt.Sprintf("fallback-%d", i)
				break
			}
		}
	}
	
	return map[string]interface{}{
		"current_endpoint":  currentEndpoint,
		"fallback_count":    len(m.fallbackClients),
		"health_check_interval": m.healthCheckInterval.String(),
	}
}

// Helper function to check if string contains any substring
func contains(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

