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
	protoapi "github.com/smart-contract-event-indexer/shared/proto"
	"github.com/smart-contract-event-indexer/shared/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ContractHandler handles contract-related HTTP requests
type ContractHandler struct {
	db          *sql.DB
	redisClient *redis.Client
	adminClient protoapi.AdminServiceClient
	queryClient protoapi.QueryServiceClient
	logger      utils.Logger
	config      *config.Config
}

// NewContractHandler creates a new ContractHandler
func NewContractHandler(
	db *sql.DB,
	redisClient *redis.Client,
	adminClient protoapi.AdminServiceClient,
	queryClient protoapi.QueryServiceClient,
	logger utils.Logger,
	cfg *config.Config,
) *ContractHandler {
	return &ContractHandler{
		db:          db,
		redisClient: redisClient,
		adminClient: adminClient,
		queryClient: queryClient,
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
	limit := h.config.DefaultLimit
	offset := 0

	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := c.Query("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	resp, err := h.adminClient.ListContracts(c.Request.Context(), &protoapi.ListContractsRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to list contracts via admin service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query contracts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contracts":   restContractsFromProto(resp.Contracts),
		"total_count": resp.TotalCount,
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
			"success":     true,
			"contract_id": existingID,
			"is_new":      false,
			"message":     "Contract already exists",
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

	resp, err := h.adminClient.AddContract(c.Request.Context(), &protoapi.AddContractRequest{
		Address:       req.Address,
		Abi:           req.ABI,
		Name:          req.Name,
		StartBlock:    req.StartBlock,
		ConfirmBlocks: req.ConfirmBlocks,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to add contract via admin service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contract"})
		return
	}

	payload := gin.H{
		"success": resp.Success,
		"is_new":  resp.IsNew,
		"message": resp.Message,
	}
	if resp.Contract != nil {
		payload["contract"] = restContractFromProto(resp.Contract)
	}

	c.JSON(http.StatusCreated, payload)
}

// GetContract handles GET /api/v1/contracts/:address
func (h *ContractHandler) GetContract(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address is required"})
		return
	}

	resp, err := h.adminClient.GetContract(c.Request.Context(), &protoapi.GetContractRequest{
		Address: address,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to fetch contract")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query contract"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contract": restContractFromProto(resp),
	})
}

// RemoveContract handles DELETE /api/v1/contracts/:address
func (h *ContractHandler) RemoveContract(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address is required"})
		return
	}

	resp, err := h.adminClient.RemoveContract(c.Request.Context(), &protoapi.RemoveContractRequest{
		Address: address,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to remove contract")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove contract"})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusNotFound, gin.H{"error": resp.Message})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": resp.Success,
		"message": resp.Message,
	})
}

// GetContractStats handles GET /api/v1/contracts/:address/stats
func (h *ContractHandler) GetContractStats(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address is required"})
		return
	}

	stats, err := h.queryClient.GetContractStats(c.Request.Context(), &protoapi.StatsQuery{
		ContractAddress: address,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch contract stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contract_address": stats.ContractAddress,
		"total_events":     stats.TotalEvents,
		"latest_block":     stats.LatestBlock,
		"current_block":    stats.CurrentBlock,
		"indexer_delay":    stats.IndexerDelay,
	})
}

func restContractFromProto(contract *protoapi.Contract) models.Contract {
	if contract == nil {
		return models.Contract{}
	}
	result := models.Contract{
		ID:            contract.Id,
		Address:       models.Address(contract.Address),
		ABI:           contract.Abi,
		Name:          contract.Name,
		StartBlock:    contract.StartBlock,
		CurrentBlock:  contract.CurrentBlock,
		ConfirmBlocks: int(contract.ConfirmBlocks),
	}
	if contract.CreatedAt != nil {
		result.CreatedAt = contract.CreatedAt.AsTime()
	}
	if contract.UpdatedAt != nil {
		result.UpdatedAt = contract.UpdatedAt.AsTime()
	}

	return result
}

func restContractsFromProto(list []*protoapi.Contract) []models.Contract {
	results := make([]models.Contract, 0, len(list))
	for _, contract := range list {
		results = append(results, restContractFromProto(contract))
	}
	return results
}
