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

// EventHandler handles event-related HTTP requests
type EventHandler struct {
	db          *sql.DB
	redisClient *redis.Client
	logger      utils.Logger
	config      *config.Config
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(
	db *sql.DB,
	redisClient *redis.Client,
	logger utils.Logger,
	cfg *config.Config,
) *EventHandler {
	return &EventHandler{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
		config:      cfg,
	}
}

// GetEvents handles GET /api/v1/events
func (h *EventHandler) GetEvents(c *gin.Context) {
	// Parse query parameters
	contractAddress := c.Query("contract")
	eventName := c.Query("event_name")
	fromBlockStr := c.Query("from_block")
	toBlockStr := c.Query("to_block")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	// Build query
	query := "SELECT id, contract_address, event_name, block_number, block_hash, transaction_hash, transaction_index, log_index, args, timestamp, created_at FROM events WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if contractAddress != "" {
		query += " AND contract_address = $" + strconv.Itoa(argIndex)
		args = append(args, contractAddress)
		argIndex++
	}

	if eventName != "" {
		query += " AND event_name = $" + strconv.Itoa(argIndex)
		args = append(args, eventName)
		argIndex++
	}

	if fromBlockStr != "" {
		if fromBlock, err := strconv.ParseInt(fromBlockStr, 10, 64); err == nil {
			query += " AND block_number >= $" + strconv.Itoa(argIndex)
			args = append(args, fromBlock)
			argIndex++
		}
	}

	if toBlockStr != "" {
		if toBlock, err := strconv.ParseInt(toBlockStr, 10, 64); err == nil {
			query += " AND block_number <= $" + strconv.Itoa(argIndex)
			args = append(args, toBlock)
			argIndex++
		}
	}

	// Add ordering
	query += " ORDER BY block_number DESC, log_index ASC"

	// Add pagination
	limit := h.config.DefaultLimit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= h.config.MaxQueryLimit {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.logger.Error("Failed to query events", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query events"})
		return
	}
	defer rows.Close()

	// Parse results
	var events []models.Event
	for rows.Next() {
		var event models.Event
		var argsJSON string

		err := rows.Scan(
			&event.ID,
			&event.ContractAddress,
			&event.EventName,
			&event.BlockNumber,
			&event.BlockHash,
			&event.TransactionHash,
			&event.TransactionIndex,
			&event.LogIndex,
			&argsJSON,
			&event.Timestamp,
			&event.CreatedAt,
		)
		if err != nil {
			h.logger.Error("Failed to scan event", "error", err)
			continue
		}

		// Parse JSONB args
		if err := json.Unmarshal([]byte(argsJSON), &event.Args); err != nil {
			h.logger.Warn("Failed to parse event args", "error", err)
			event.Args = models.JSONB{}
		}

		events = append(events, event)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM events WHERE 1=1"
	countArgs := args[:len(args)-2] // Remove limit and offset
	if len(countArgs) > 0 {
		countQuery = "SELECT COUNT(*) FROM events WHERE 1=1"
		for i := range countArgs {
			if i == 0 {
				countQuery += " AND contract_address = $1"
			} else if i == 1 {
				countQuery += " AND event_name = $2"
			} else if i == 2 {
				countQuery += " AND block_number >= $3"
			} else if i == 3 {
				countQuery += " AND block_number <= $4"
			}
		}
	}

	var totalCount int64
	if err := h.db.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil {
		h.logger.Warn("Failed to get total count", "error", err)
		totalCount = int64(len(events))
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"events":      events,
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetEventsByTransaction handles GET /api/v1/events/tx/:txHash
func (h *EventHandler) GetEventsByTransaction(c *gin.Context) {
	txHash := c.Param("txHash")
	if txHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction hash is required"})
		return
	}

	query := `
		SELECT id, contract_address, event_name, block_number, block_hash, 
		       transaction_hash, transaction_index, log_index, args, timestamp, created_at 
		FROM events 
		WHERE transaction_hash = $1 
		ORDER BY log_index ASC
	`

	rows, err := h.db.Query(query, txHash)
	if err != nil {
		h.logger.Error("Failed to query events by transaction", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query events"})
		return
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		var argsJSON string

		err := rows.Scan(
			&event.ID,
			&event.ContractAddress,
			&event.EventName,
			&event.BlockNumber,
			&event.BlockHash,
			&event.TransactionHash,
			&event.TransactionIndex,
			&event.LogIndex,
			&argsJSON,
			&event.Timestamp,
			&event.CreatedAt,
		)
		if err != nil {
			h.logger.Error("Failed to scan event", "error", err)
			continue
		}

		// Parse JSONB args
		if err := json.Unmarshal([]byte(argsJSON), &event.Args); err != nil {
			h.logger.Warn("Failed to parse event args", "error", err)
			event.Args = models.JSONB{}
		}

		events = append(events, event)
	}

	c.JSON(http.StatusOK, gin.H{
		"events":      events,
		"total_count": len(events),
	})
}

// GetEventsByAddress handles GET /api/v1/events/address/:address
func (h *EventHandler) GetEventsByAddress(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address is required"})
		return
	}

	// Use JSONB query to find events involving this address
	query := `
		SELECT id, contract_address, event_name, block_number, block_hash, 
		       transaction_hash, transaction_index, log_index, args, timestamp, created_at 
		FROM events 
		WHERE args @> $1 
		ORDER BY block_number DESC, log_index ASC
		LIMIT $2
	`

	limit := h.config.DefaultLimit
	limitStr := c.Query("limit")
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= h.config.MaxQueryLimit {
			limit = parsedLimit
		}
	}

	// Search for address in 'from' field (simplified)
	addressFilter := `{"from": "` + address + `"}`

	rows, err := h.db.Query(query, addressFilter, limit)
	if err != nil {
		h.logger.Error("Failed to query events by address", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query events"})
		return
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		var argsJSON string

		err := rows.Scan(
			&event.ID,
			&event.ContractAddress,
			&event.EventName,
			&event.BlockNumber,
			&event.BlockHash,
			&event.TransactionHash,
			&event.TransactionIndex,
			&event.LogIndex,
			&argsJSON,
			&event.Timestamp,
			&event.CreatedAt,
		)
		if err != nil {
			h.logger.Error("Failed to scan event", "error", err)
			continue
		}

		// Parse JSONB args
		if err := json.Unmarshal([]byte(argsJSON), &event.Args); err != nil {
			h.logger.Warn("Failed to parse event args", "error", err)
			event.Args = models.JSONB{}
		}

		events = append(events, event)
	}

	c.JSON(http.StatusOK, gin.H{
		"events":      events,
		"total_count": len(events),
		"address":     address,
	})
}
