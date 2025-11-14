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
)

// EventHandler handles event-related HTTP requests
type EventHandler struct {
	db          *sql.DB
	redisClient *redis.Client
	queryClient protoapi.QueryServiceClient
	logger      utils.Logger
	config      *config.Config
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(
	db *sql.DB,
	redisClient *redis.Client,
	queryClient protoapi.QueryServiceClient,
	logger utils.Logger,
	cfg *config.Config,
) *EventHandler {
	return &EventHandler{
		db:          db,
		redisClient: redisClient,
		queryClient: queryClient,
		logger:      logger,
		config:      cfg,
	}
}

// GetEvents handles GET /api/v1/events
func (h *EventHandler) GetEvents(c *gin.Context) {
	req := &protoapi.EventQuery{}

	if v := c.Query("contract"); v != "" {
		req.ContractAddress = v
	}
	if v := c.Query("event_name"); v != "" {
		req.EventName = v
	}
	if v := c.Query("from_block"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			req.FromBlock = parsed
		}
	}
	if v := c.Query("to_block"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			req.ToBlock = parsed
		}
	}

	limit := h.config.DefaultLimit
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= h.config.MaxQueryLimit {
			limit = parsed
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	req.Limit = int32(limit)
	req.Offset = int32(offset)

	ctx := c.Request.Context()
	resp, err := h.queryClient.GetEvents(ctx, req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch events via query service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":      restEventsFromProto(resp.Events),
		"total_count": resp.TotalCount,
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

	resp, err := h.queryClient.GetEventsByTransaction(c.Request.Context(), &protoapi.TransactionQuery{
		TransactionHash: txHash,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch events by transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query events"})
		return
	}

	events := restEventsFromProto(resp.Events)
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

	limit := h.config.DefaultLimit
	limitStr := c.Query("limit")
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= h.config.MaxQueryLimit {
			limit = parsedLimit
		}
	}

	resp, err := h.queryClient.GetEventsByAddress(c.Request.Context(), &protoapi.AddressQuery{
		Address: address,
		First:   int32(limit),
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch events by address")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query events"})
		return
	}

	events := restEventsFromProto(resp.Events)
	c.JSON(http.StatusOK, gin.H{
		"events":      events,
		"total_count": len(events),
		"address":     address,
	})
}

func restEventsFromProto(evts []*protoapi.Event) []models.Event {
	results := make([]models.Event, 0, len(evts))
	for _, evt := range evts {
		if evt == nil {
			continue
		}
		event := models.Event{
			ID:               evt.Id,
			ContractAddress:  models.Address(evt.ContractAddress),
			EventName:        evt.EventName,
			BlockNumber:      evt.BlockNumber,
			BlockHash:        models.Hash(evt.BlockHash),
			TransactionHash:  models.Hash(evt.TransactionHash),
			TransactionIndex: int(evt.TransactionIndex),
			LogIndex:         int(evt.LogIndex),
			Args:             argsMapFromProto(evt.Args),
		}
		if evt.Timestamp != nil {
			event.Timestamp = evt.Timestamp.AsTime()
		}
		if evt.CreatedAt != nil {
			event.CreatedAt = evt.CreatedAt.AsTime()
		}
		event.RawLog = rawLogFromArgs(event.Args)
		results = append(results, event)
	}
	return results
}

func argsMapFromProto(args []*protoapi.EventArg) models.JSONB {
	if len(args) == 0 {
		return models.JSONB{}
	}
	result := make(models.JSONB)
	for _, arg := range args {
		if arg == nil {
			continue
		}
		result[arg.Key] = arg.Value
	}
	return result
}

func rawLogFromArgs(args models.JSONB) *string {
	if len(args) == 0 {
		return nil
	}
	data, err := json.Marshal(args)
	if err != nil {
		return nil
	}
	raw := string(data)
	return &raw
}
