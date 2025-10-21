package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/admin-service/internal/config"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// AdminService handles administrative operations
type AdminService struct {
	db          *sql.DB
	redisClient *redis.Client
	logger      utils.Logger
	config      *config.Config
}

// NewAdminService creates a new AdminService
func NewAdminService(
	db *sql.DB,
	redisClient *redis.Client,
	logger utils.Logger,
	cfg *config.Config,
) *AdminService {
	return &AdminService{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
		config:      cfg,
	}
}

// AddContractRequest represents a request to add a contract
type AddContractRequest struct {
	Address       string `json:"address"`
	Name          string `json:"name"`
	ABI           string `json:"abi"`
	StartBlock    int64  `json:"start_block"`
	ConfirmBlocks int32  `json:"confirm_blocks"`
}

// AddContractResponse represents the response for adding a contract
type AddContractResponse struct {
	Success    bool   `json:"success"`
	ContractID int32  `json:"contract_id"`
	IsNew      bool   `json:"is_new"`
	Message    string `json:"message"`
}

// RemoveContractRequest represents a request to remove a contract
type RemoveContractRequest struct {
	Address string `json:"address"`
}

// RemoveContractResponse represents the response for removing a contract
type RemoveContractResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// BackfillRequest represents a request to trigger backfill
type BackfillRequest struct {
	Address   string `json:"address"`
	FromBlock int64  `json:"from_block"`
	ToBlock   int64  `json:"to_block"`
}

// BackfillResponse represents the response for backfill
type BackfillResponse struct {
	Success     bool   `json:"success"`
	JobID       string `json:"job_id"`
	Message     string `json:"message"`
	EstimatedTime string `json:"estimated_time"`
}

// SystemStatusResponse represents system status
type SystemStatusResponse struct {
	IndexerLag      int64                    `json:"indexer_lag"`
	TotalContracts  int32                    `json:"total_contracts"`
	TotalEvents     int64                    `json:"total_events"`
	CacheHitRate    float64                  `json:"cache_hit_rate"`
	LastIndexedBlock int64                   `json:"last_indexed_block"`
	IsHealthy       bool                     `json:"is_healthy"`
	Uptime          int64                    `json:"uptime"`
	Services        map[string]ServiceStatus `json:"services"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Status  string `json:"status"`
	Latency int64  `json:"latency_ms"`
	Error   string `json:"error,omitempty"`
}

// AddContract adds a new contract for monitoring
func (s *AdminService) AddContract(ctx context.Context, req *AddContractRequest) (*AddContractResponse, error) {
	// Validate address
	addr := models.Address(req.Address)
	if err := addr.Validate(); err != nil {
		return &AddContractResponse{
			Success: false,
			Message: "Invalid contract address",
		}, nil
	}

	// Set default confirm blocks if not provided
	if req.ConfirmBlocks == 0 {
		req.ConfirmBlocks = 6 // Default balanced mode
	}

	// Check if contract already exists
	var existingID int32
	checkQuery := "SELECT id FROM contracts WHERE address = $1"
	err := s.db.QueryRowContext(ctx, checkQuery, req.Address).Scan(&existingID)
	
	if err == nil {
		// Contract exists, return existing contract info
		return &AddContractResponse{
			Success:    true,
			ContractID: existingID,
			IsNew:      false,
			Message:    "Contract already exists",
		}, nil
	} else if err != sql.ErrNoRows {
		s.logger.Error("Failed to check existing contract", "error", err)
		return &AddContractResponse{
			Success: false,
			Message: "Failed to check existing contract",
		}, nil
	}

	// Validate ABI JSON
	var abiJSON models.JSONB
	if err := json.Unmarshal([]byte(req.ABI), &abiJSON); err != nil {
		return &AddContractResponse{
			Success: false,
			Message: "Invalid ABI JSON",
		}, nil
	}

	// Insert new contract
	insertQuery := `
		INSERT INTO contracts (address, name, abi, start_block, current_block, confirm_blocks, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var contractID int32
	err = s.db.QueryRowContext(ctx,
		insertQuery,
		req.Address,
		req.Name,
		req.ABI,
		req.StartBlock,
		req.StartBlock, // current_block starts at start_block
		req.ConfirmBlocks,
		models.Now(),
		models.Now(),
	).Scan(&contractID)

	if err != nil {
		s.logger.Error("Failed to insert contract", "error", err)
		return &AddContractResponse{
			Success: false,
			Message: "Failed to create contract",
		}, nil
	}

	s.logger.Info("Contract added", "address", req.Address, "id", contractID)

	return &AddContractResponse{
		Success:    true,
		ContractID: contractID,
		IsNew:      true,
		Message:    "Contract added successfully",
	}, nil
}

// RemoveContract removes a contract from monitoring
func (s *AdminService) RemoveContract(ctx context.Context, req *RemoveContractRequest) (*RemoveContractResponse, error) {
	// Delete contract (no is_active column in current schema)
	query := "DELETE FROM contracts WHERE address = $1"
	result, err := s.db.ExecContext(ctx, query, req.Address)
	if err != nil {
		s.logger.Error("Failed to remove contract", "error", err)
		return &RemoveContractResponse{
			Success: false,
			Message: "Failed to remove contract",
		}, nil
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to get rows affected", "error", err)
		return &RemoveContractResponse{
			Success: false,
			Message: "Failed to remove contract",
		}, nil
	}

	if rowsAffected == 0 {
		return &RemoveContractResponse{
			Success: false,
			Message: "Contract not found",
		}, nil
	}

	s.logger.Info("Contract removed", "address", req.Address)

	return &RemoveContractResponse{
		Success: true,
		Message: "Contract removed successfully",
	}, nil
}

// TriggerBackfill triggers a historical backfill for a contract
func (s *AdminService) TriggerBackfill(ctx context.Context, req *BackfillRequest) (*BackfillResponse, error) {
	// Validate address
	addr := models.Address(req.Address)
	if err := addr.Validate(); err != nil {
		return &BackfillResponse{
			Success: false,
			Message: "Invalid contract address",
		}, nil
	}

	// Validate block range
	if req.FromBlock >= req.ToBlock {
		return &BackfillResponse{
			Success: false,
			Message: "Invalid block range",
		}, nil
	}

	// Generate job ID
	jobID := fmt.Sprintf("backfill_%s_%d_%d_%d", req.Address, req.FromBlock, req.ToBlock, time.Now().Unix())

	// Store backfill job in Redis
	jobData := map[string]interface{}{
		"address":    req.Address,
		"from_block": req.FromBlock,
		"to_block":   req.ToBlock,
		"status":     "pending",
		"created_at": time.Now().Unix(),
	}

	// Store job metadata
	jobKey := fmt.Sprintf("backfill_job:%s", jobID)
	if err := s.redisClient.HSet(ctx, jobKey, jobData).Err(); err != nil {
		s.logger.Error("Failed to store backfill job", "error", err)
		return &BackfillResponse{
			Success: false,
			Message: "Failed to create backfill job",
		}, nil
	}

	// Set job expiration (24 hours)
	if err := s.redisClient.Expire(ctx, jobKey, 24*time.Hour).Err(); err != nil {
		s.logger.Warn("Failed to set job expiration", "error", err)
	}

	// Calculate estimated time (simplified)
	blockRange := req.ToBlock - req.FromBlock
	estimatedBlocksPerMinute := 100 // Simplified estimate
	estimatedMinutes := int64(blockRange) / int64(estimatedBlocksPerMinute)
	if estimatedMinutes < 1 {
		estimatedMinutes = 1
	}

	s.logger.Info("Backfill job created", "job_id", jobID, "address", req.Address, "from_block", req.FromBlock, "to_block", req.ToBlock)

	return &BackfillResponse{
		Success:       true,
		JobID:         jobID,
		Message:       "Backfill job created successfully",
		EstimatedTime: fmt.Sprintf("%d minutes", estimatedMinutes),
	}, nil
}

// GetSystemStatus returns the current system status
func (s *AdminService) GetSystemStatus(ctx context.Context) (*SystemStatusResponse, error) {
	// Get total contracts count
	var totalContracts int32
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM contracts").Scan(&totalContracts); err != nil {
		s.logger.Error("Failed to get total contracts", "error", err)
		totalContracts = 0
	}

	// Get total events count
	var totalEvents int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&totalEvents); err != nil {
		s.logger.Error("Failed to get total events", "error", err)
		totalEvents = 0
	}

	// Get latest indexed block
	var lastIndexedBlock int64
	if err := s.db.QueryRowContext(ctx, "SELECT MAX(block_number) FROM events").Scan(&lastIndexedBlock); err != nil {
		s.logger.Error("Failed to get last indexed block", "error", err)
		lastIndexedBlock = 0
	}

	// Check service health
	services := make(map[string]ServiceStatus)
	
	// Database health
	dbStart := time.Now()
	if err := s.db.PingContext(ctx); err != nil {
		services["database"] = ServiceStatus{
			Status: "error",
			Error:  err.Error(),
		}
	} else {
		services["database"] = ServiceStatus{
			Status:  "healthy",
			Latency: time.Since(dbStart).Milliseconds(),
		}
	}

	// Redis health
	redisStart := time.Now()
	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		services["redis"] = ServiceStatus{
			Status: "error",
			Error:  err.Error(),
		}
	} else {
		services["redis"] = ServiceStatus{
			Status:  "healthy",
			Latency: time.Since(redisStart).Milliseconds(),
		}
	}

	// Determine overall health
	isHealthy := true
	for _, service := range services {
		if service.Status != "healthy" {
			isHealthy = false
			break
		}
	}

	return &SystemStatusResponse{
		IndexerLag:      0, // Would be calculated based on current time vs block timestamp
		TotalContracts:  totalContracts,
		TotalEvents:     totalEvents,
		CacheHitRate:    0.0, // Would be calculated from Redis metrics
		LastIndexedBlock: lastIndexedBlock,
		IsHealthy:       isHealthy,
		Uptime:          time.Now().Unix(), // Simplified
		Services:        services,
	}, nil
}
