package graph

import (
	"fmt"
	"strconv"
	"time"

	"github.com/smart-contract-event-indexer/api-gateway/graph/model"
	"github.com/smart-contract-event-indexer/shared/models"
	protoapi "github.com/smart-contract-event-indexer/shared/proto"
)

func eventConnectionFromProto(resp *protoapi.EventResponse) *models.EventConnection {
	if resp == nil {
		return &models.EventConnection{
			PageInfo: models.PageInfo{},
		}
	}

	events := eventsFromProto(resp.Events)
	edges := make([]*models.EventEdge, 0, len(events))
	for _, evt := range events {
		cursor := fmt.Sprintf("%d", evt.ID)
		edges = append(edges, &models.EventEdge{
			Node:   evt,
			Cursor: cursor,
		})
	}

	return &models.EventConnection{
		Edges:      edges,
		PageInfo:   pageInfoFromProto(resp.PageInfo),
		TotalCount: int(resp.TotalCount),
	}
}

func eventsFromProto(evts []*protoapi.Event) []*models.Event {
	result := make([]*models.Event, 0, len(evts))
	for _, evt := range evts {
		result = append(result, eventFromProto(evt))
	}
	return result
}

func eventFromProto(evt *protoapi.Event) *models.Event {
	if evt == nil {
		return nil
	}

	args := make(models.JSONB)
	for _, arg := range evt.Args {
		if arg == nil {
			continue
		}
		args[arg.Key] = arg.Value
	}

	var timestamp time.Time
	if evt.Timestamp != nil {
		timestamp = evt.Timestamp.AsTime()
	}
	var createdAt time.Time
	if evt.CreatedAt != nil {
		createdAt = evt.CreatedAt.AsTime()
	}

	return &models.Event{
		ID:               evt.Id,
		ContractAddress:  models.Address(evt.ContractAddress),
		EventName:        evt.EventName,
		BlockNumber:      evt.BlockNumber,
		BlockHash:        models.Hash(evt.BlockHash),
		TransactionHash:  models.Hash(evt.TransactionHash),
		TransactionIndex: int(evt.TransactionIndex),
		LogIndex:         int(evt.LogIndex),
		Args:             args,
		Timestamp:        timestamp,
		CreatedAt:        createdAt,
	}
}

func pageInfoFromProto(pi *protoapi.PageInfo) models.PageInfo {
	if pi == nil {
		return models.PageInfo{}
	}
	page := models.PageInfo{
		HasNextPage:     pi.HasNextPage,
		HasPreviousPage: pi.HasPreviousPage,
	}
	if pi.StartCursor != "" {
		page.StartCursor = stringPtr(pi.StartCursor)
	}
	if pi.EndCursor != "" {
		page.EndCursor = stringPtr(pi.EndCursor)
	}
	return page
}

func contractFromProto(p *protoapi.Contract) *models.Contract {
	if p == nil {
		return nil
	}
	contract := &models.Contract{
		ID:            p.Id,
		Address:       models.Address(p.Address),
		ABI:           p.Abi,
		Name:          p.Name,
		StartBlock:    p.StartBlock,
		CurrentBlock:  p.CurrentBlock,
		ConfirmBlocks: int(p.ConfirmBlocks),
	}
	if p.CreatedAt != nil {
		contract.CreatedAt = p.CreatedAt.AsTime()
	}
	if p.UpdatedAt != nil {
		contract.UpdatedAt = p.UpdatedAt.AsTime()
	}
	return contract
}

func contractsFromProto(list []*protoapi.Contract) []*models.Contract {
	result := make([]*models.Contract, 0, len(list))
	for _, c := range list {
		result = append(result, contractFromProto(c))
	}
	return result
}

func statsFromProto(resp *protoapi.StatsResponse) *models.ContractStats {
	if resp == nil {
		return nil
	}
	stats := &models.ContractStats{
		ContractAddress: models.Address(resp.ContractAddress),
		TotalEvents:     resp.TotalEvents,
		LatestBlock:     resp.LatestBlock,
		CurrentBlock:    resp.CurrentBlock,
		IndexerDelay:    resp.IndexerDelay,
	}
	if resp.LastUpdated != nil {
		stats.LastUpdated = resp.LastUpdated.AsTime()
	}
	return stats
}

func backfillPayloadFromProto(resp *protoapi.BackfillResponse) *model.BackfillPayload {
	if resp == nil {
		return &model.BackfillPayload{Success: false, Message: "no response"}
	}
	payload := &model.BackfillPayload{
		Success: resp.Success,
		Message: resp.Message,
	}
	if resp.JobId != "" {
		payload.JobID = stringPtr(resp.JobId)
	}
	if resp.EstimatedTime > 0 {
		val := int(resp.EstimatedTime)
		payload.EstimatedTime = &val
	}
	return payload
}

func serviceStatusFromProto(list []*protoapi.ServiceStatus) []*model.ServiceStatus {
	result := make([]*model.ServiceStatus, 0, len(list))
	for _, svc := range list {
		if svc == nil {
			continue
		}
		var latency *int
		if svc.Latency != 0 {
			val := int(svc.Latency)
			latency = &val
		}
		lastCheck := ""
		if svc.LastCheck != nil {
			lastCheck = svc.LastCheck.AsTime().Format(time.RFC3339)
		}
		result = append(result, &model.ServiceStatus{
			Name:      svc.Name,
			Status:    svc.Status,
			Latency:   latency,
			LastCheck: lastCheck,
		})
	}
	return result
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
}

func parseBigInt(value string) (int64, error) {
	if value == "" {
		return 0, fmt.Errorf("value cannot be empty")
	}
	return strconv.ParseInt(value, 10, 64)
}
