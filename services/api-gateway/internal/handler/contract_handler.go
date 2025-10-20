package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/api-gateway/internal/config"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// ContractHandler handles contract-related HTTP requests
type ContractHandler struct {
	db          *sql.DB
	redisClient *redis.Client
	logger      utils.Logger
	config      *config.Config
}

// NewContractHandler creates a new ContractHandler
func NewContractHandler(
	db *sql.DB,
	redisClient *redis.Client,
	logger utils.Logger,
	cfg *config.Config,
) *ContractHandler {
	return &ContractHandler{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
		config:      cfg,
	}
}

// AddContractRequest represents the request to add a contract
type AddContractRequest struct {
	Address       string `json:"address" binding:"required"`
	Name          string `json:"name"`
	ABI           string `json:"abi" binding:"required"`
	StartBlock    int64  `json:"start_block" binding:"required"`
	ConfirmBlocks int32  `json:"confirm_blocks"`
}

// GetContracts handles GET /api/v1/contracts
func (h *ContractHandler) GetContracts(c *gin.Context) {
	query := "SELECT id, address, name, abi, start_block, current_block, confirm_blocks, created_at, updated_at FROM contracts"
	args := []interface{}{}
	argIndex := 1

	// Note: is_active column doesn't exist in current schema
	// Filtering by active status is not available in current implementation

	query += " ORDER BY created_at DESC"

	// Add pagination
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")
	limit := 20
	offset := 0

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	// Debug logging
	h.logger.Info("Executing query", "query", query, "args", args)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.logger.Error("Failed to query contracts", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query contracts"})
		return
	}
	defer rows.Close()

	var contracts []models.Contract
	for rows.Next() {
		var contract models.Contract

		err := rows.Scan(
			&contract.ID,
			&contract.Address,
			&contract.Name,
			&contract.ABI,
			&contract.StartBlock,
			&contract.CurrentBlock,
			&contract.ConfirmBlocks,
			&contract.CreatedAt,
			&contract.UpdatedAt,
		)
		if err != nil {
			h.logger.Error("Failed to scan contract", "error", err)
			continue
		}

		contracts = append(contracts, contract)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM contracts"
	countArgs := args[:len(args)-2] // Remove limit and offset

	var totalCount int64
	if err := h.db.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil {
		h.logger.Warn("Failed to get total count", "error", err)
		totalCount = int64(len(contracts))
	}

	c.JSON(http.StatusOK, gin.H{
		"contracts":   contracts,
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
	})
}

// AddContract handles POST /api/v1/contracts
func (h *ContractHandler) AddContract(c *gin.Context) {
	var req AddContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate address
	addr := models.Address(req.Address)
	if err := addr.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract address"})
		return
	}

	// Set default confirm blocks if not provided
	if req.ConfirmBlocks == 0 {
		req.ConfirmBlocks = 6 // Default balanced mode
	}

	// Check if contract already exists
	var existingID int32
	checkQuery := "SELECT id FROM contracts WHERE address = $1"
	err := h.db.QueryRow(checkQuery, req.Address).Scan(&existingID)
	
	if err == nil {
		// Contract exists, return existing contract info
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"contract_id": existingID,
			"is_new":     false,
			"message":    "Contract already exists",
		})
		return
	} else if err != sql.ErrNoRows {
		h.logger.Error("Failed to check existing contract", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing contract"})
		return
	}

	// Validate ABI JSON
	var abiJSON models.JSONB
	if err := json.Unmarshal([]byte(req.ABI), &abiJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ABI JSON"})
		return
	}

	// Insert new contract
	insertQuery := `
		INSERT INTO contracts (address, name, abi, start_block, current_block, confirm_blocks, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var contractID int32
	err = h.db.QueryRow(
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
		h.logger.Error("Failed to insert contract", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contract"})
		return
	}

	h.logger.Info("Contract added", "address", req.Address, "id", contractID)

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"contract_id": contractID,
		"is_new":      true,
		"message":     "Contract added successfully",
	})
}

// GetContract handles GET /api/v1/contracts/:address
func (h *ContractHandler) GetContract(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address is required"})
		return
	}

	query := `
		SELECT id, address, name, abi, start_block, current_block, confirm_blocks, created_at, updated_at
		FROM contracts 
		WHERE address = $1
	`

	var contract models.Contract
	err := h.db.QueryRow(query, address).Scan(
		&contract.ID,
		&contract.Address,
		&contract.Name,
		&contract.ABI,
		&contract.StartBlock,
		&contract.CurrentBlock,
		&contract.ConfirmBlocks,
		&contract.CreatedAt,
		&contract.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
		return
	} else if err != nil {
		h.logger.Error("Failed to query contract", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query contract"})
		return
	}


	c.JSON(http.StatusOK, gin.H{
		"contract": contract,
	})
}

// RemoveContract handles DELETE /api/v1/contracts/:address
func (h *ContractHandler) RemoveContract(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address is required"})
		return
	}

	// Delete contract (no is_active column in current schema)
	query := "DELETE FROM contracts WHERE address = $1"
	result, err := h.db.Exec(query, address)
	if err != nil {
		h.logger.Error("Failed to remove contract", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove contract"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		h.logger.Error("Failed to get rows affected", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove contract"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
		return
	}

	h.logger.Info("Contract removed", "address", address)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Contract removed successfully",
	})
}

// GetContractStats handles GET /api/v1/contracts/:address/stats
func (h *ContractHandler) GetContractStats(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address is required"})
		return
	}

	// Get total events count
	var totalEvents int64
	countQuery := "SELECT COUNT(*) FROM events WHERE contract_address = $1"
	if err := h.db.QueryRow(countQuery, address).Scan(&totalEvents); err != nil {
		h.logger.Error("Failed to get total events", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	// Get latest block
	var latestBlock int64
	latestQuery := "SELECT MAX(block_number) FROM events WHERE contract_address = $1"
	if err := h.db.QueryRow(latestQuery, address).Scan(&latestBlock); err != nil {
		h.logger.Error("Failed to get latest block", "error", err)
		latestBlock = 0
	}

	// Get current block from contract
	var currentBlock int64
	currentQuery := "SELECT current_block FROM contracts WHERE address = $1"
	if err := h.db.QueryRow(currentQuery, address).Scan(&currentBlock); err != nil {
		h.logger.Error("Failed to get current block", "error", err)
		currentBlock = latestBlock
	}

	// Calculate indexer delay (simplified)
	indexerDelay := int64(0) // This would be calculated based on current time vs block timestamp

	c.JSON(http.StatusOK, gin.H{
		"contract_address": address,
		"total_events":     totalEvents,
		"latest_block":     latestBlock,
		"current_block":    currentBlock,
		"indexer_delay":    indexerDelay,
	})
}
