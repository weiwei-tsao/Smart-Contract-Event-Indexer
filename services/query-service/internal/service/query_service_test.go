package service

import (
	"testing"

	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"github.com/smart-contract-event-indexer/query-service/internal/types"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

func TestDetermineEventQueryPath(t *testing.T) {
	svc := &QueryService{config: &config.Config{}, logger: utils.NewTestLogger()}

	addr := "0xabc"
	evt := "Transfer"
	simple := &types.EventQuery{ContractAddress: &addr, EventName: &evt}
	if path := svc.determineEventQueryPath(simple); path != queryPathSimple {
		t.Fatalf("expected simple path, got %s", path)
	}

	complex := &types.EventQuery{Addresses: []string{"0x123"}}
	if path := svc.determineEventQueryPath(complex); path != queryPathComplex {
		t.Fatalf("expected complex path, got %s", path)
	}
}

func TestBuildPageInfo(t *testing.T) {
	svc := &QueryService{config: &config.Config{}, logger: utils.NewTestLogger()}

	events := []*models.Event{{ID: 101}, {ID: 102}, {ID: 103}}
	first := int32(3)
	info := svc.buildPageInfo(events, &types.EventQuery{First: &first})

	if !info.HasNextPage {
		t.Fatalf("expected HasNextPage true when results hit requested limit")
	}
	if info.StartCursor == nil || *info.StartCursor != 101 {
		t.Fatalf("unexpected start cursor: %+v", info.StartCursor)
	}
	if info.EndCursor == nil || *info.EndCursor != 103 {
		t.Fatalf("unexpected end cursor: %+v", info.EndCursor)
	}
}
