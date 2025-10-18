package parser

import (
	"testing"

	"github.com/smart-contract-event-indexer/indexer-service/internal/testutil"
)

func TestNewABIParser_ValidABI(t *testing.T) {
	logger := testutil.NewTestLogger()
	
	parser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if parser == nil {
		t.Fatal("Expected non-nil parser")
	}
	
	// Verify events were extracted
	if len(parser.eventsByName) == 0 {
		t.Error("Expected events to be extracted from ABI")
	}
}

func TestNewABIParser_InvalidABI(t *testing.T) {
	logger := testutil.NewTestLogger()
	
	_, err := NewABIParser(testutil.InvalidABI, logger)
	if err == nil {
		t.Fatal("Expected error for invalid ABI, got nil")
	}
}

func TestNewABIParser_EmptyABI(t *testing.T) {
	logger := testutil.NewTestLogger()
	
	_, err := NewABIParser("", logger)
	if err == nil {
		t.Fatal("Expected error for empty ABI, got nil")
	}
}

func TestABIParser_GetEventByID(t *testing.T) {
	logger := testutil.NewTestLogger()
	parser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	// Transfer event ID (topic0)
	transferID := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	
	event, found := parser.GetEventByID(transferID)
	if !found {
		t.Fatalf("Expected to find Transfer event")
	}
	
	if event.Name != "Transfer" {
		t.Errorf("Expected event name 'Transfer', got: %s", event.Name)
	}
	
	// Verify event has expected inputs
	if len(event.Inputs) != 3 {
		t.Errorf("Expected 3 inputs for Transfer event, got: %d", len(event.Inputs))
	}
}

func TestABIParser_GetEventByID_NotFound(t *testing.T) {
	logger := testutil.NewTestLogger()
	parser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	// Non-existent event ID
	invalidID := "0x0000000000000000000000000000000000000000000000000000000000000000"
	
	_, found := parser.GetEventByID(invalidID)
	if found {
		t.Fatal("Expected event not to be found")
	}
}

func TestABIParser_GetEventByName(t *testing.T) {
	logger := testutil.NewTestLogger()
	parser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	event, found := parser.GetEventByName("Transfer")
	if !found {
		t.Fatalf("Expected to find Transfer event")
	}
	
	if event.Name != "Transfer" {
		t.Errorf("Expected event name 'Transfer', got: %s", event.Name)
	}
	
	// Verify Transfer event structure
	expectedInputNames := []string{"from", "to", "value"}
	if len(event.Inputs) != len(expectedInputNames) {
		t.Fatalf("Expected %d inputs, got %d", len(expectedInputNames), len(event.Inputs))
	}
	
	for i, input := range event.Inputs {
		if input.Name != expectedInputNames[i] {
			t.Errorf("Expected input name '%s' at index %d, got: %s", 
				expectedInputNames[i], i, input.Name)
		}
	}
}

func TestABIParser_GetEventByName_NotFound(t *testing.T) {
	logger := testutil.NewTestLogger()
	parser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	_, found := parser.GetEventByName("NonExistentEvent")
	if found {
		t.Fatal("Expected event not to be found")
	}
}

func TestABIParser_MultipleEvents(t *testing.T) {
	logger := testutil.NewTestLogger()
	parser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	// ERC20 ABI should have Transfer and Approval events
	expectedEvents := []string{"Transfer", "Approval"}
	
	for _, eventName := range expectedEvents {
		event, found := parser.GetEventByName(eventName)
		if !found {
			t.Errorf("Expected to find %s event", eventName)
			continue
		}
		
		if event.Name != eventName {
			t.Errorf("Expected event name '%s', got: %s", eventName, event.Name)
		}
	}
	
	// Verify both events are in the maps
	if len(parser.eventsByName) != len(expectedEvents) {
		t.Errorf("Expected %d events in eventsByName, got: %d", 
			len(expectedEvents), len(parser.eventsByName))
	}
	
	if len(parser.eventsByID) != len(expectedEvents) {
		t.Errorf("Expected %d events in eventsByID, got: %d", 
			len(expectedEvents), len(parser.eventsByID))
	}
}

func TestABIParser_EventInputTypes(t *testing.T) {
	logger := testutil.NewTestLogger()
	parser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	event, found := parser.GetEventByName("Transfer")
	if !found {
		t.Fatalf("Failed to find Transfer event")
	}
	
	// Verify input types
	expectedTypes := []string{"address", "address", "uint256"}
	for i, input := range event.Inputs {
		if input.Type.String() != expectedTypes[i] {
			t.Errorf("Expected input type '%s' at index %d, got: %s",
				expectedTypes[i], i, input.Type.String())
		}
	}
	
	// Verify indexed status
	expectedIndexed := []bool{true, true, false}
	for i, input := range event.Inputs {
		if input.Indexed != expectedIndexed[i] {
			t.Errorf("Expected input indexed=%v at index %d, got: %v",
				expectedIndexed[i], i, input.Indexed)
		}
	}
}

