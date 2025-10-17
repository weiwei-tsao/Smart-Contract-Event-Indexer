package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// ABIParser handles ABI parsing and event definition extraction
type ABIParser struct {
	contractABI abi.ABI
	eventsByID  map[string]abi.Event // Map of event ID (topic0) to event
	eventsByName map[string]abi.Event // Map of event name to event
	logger      utils.Logger
}

// NewABIParser creates a new ABI parser
func NewABIParser(abiJSON string, logger utils.Logger) (*ABIParser, error) {
	// Parse the ABI
	contractABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	
	parser := &ABIParser{
		contractABI:  contractABI,
		eventsByID:   make(map[string]abi.Event),
		eventsByName: make(map[string]abi.Event),
		logger:       logger,
	}
	
	// Build event maps
	for _, event := range contractABI.Events {
		eventID := event.ID.Hex()
		parser.eventsByID[eventID] = event
		parser.eventsByName[event.Name] = event
		
		logger.WithFields(map[string]interface{}{
			"event_name": event.Name,
			"event_id":   eventID,
		}).Debug("Registered event")
	}
	
	logger.WithField("event_count", len(contractABI.Events)).Info("ABI parsed successfully")
	
	return parser, nil
}

// GetEventByID returns an event by its ID (topic0)
func (p *ABIParser) GetEventByID(eventID string) (abi.Event, bool) {
	event, exists := p.eventsByID[eventID]
	return event, exists
}

// GetEventByName returns an event by its name
func (p *ABIParser) GetEventByName(name string) (abi.Event, bool) {
	event, exists := p.eventsByName[name]
	return event, exists
}

// GetAllEvents returns all events in the ABI
func (p *ABIParser) GetAllEvents() []abi.Event {
	events := make([]abi.Event, 0, len(p.eventsByName))
	for _, event := range p.eventsByName {
		events = append(events, event)
	}
	return events
}

// GetEventSignature returns the signature hash for an event
func (p *ABIParser) GetEventSignature(eventName string) (string, error) {
	event, exists := p.eventsByName[eventName]
	if !exists {
		return "", fmt.Errorf("event %s not found in ABI", eventName)
	}
	return event.ID.Hex(), nil
}

// ValidateABI checks if the ABI is valid and contains events
func ValidateABI(abiJSON string) error {
	var abiArray []map[string]interface{}
	if err := json.Unmarshal([]byte(abiJSON), &abiArray); err != nil {
		return fmt.Errorf("invalid ABI JSON: %w", err)
	}
	
	// Check if there's at least one event
	hasEvent := false
	for _, item := range abiArray {
		if itemType, ok := item["type"].(string); ok && itemType == "event" {
			hasEvent = true
			break
		}
	}
	
	if !hasEvent {
		return fmt.Errorf("ABI does not contain any events")
	}
	
	// Try to parse it with go-ethereum
	_, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return fmt.Errorf("ABI parsing failed: %w", err)
	}
	
	return nil
}

// EventSignatureToID converts an event signature string to its topic0 hash
// Example: "Transfer(address,address,uint256)" -> 0xddf252ad...
func EventSignatureToID(signature string) string {
	hash := crypto.Keccak256Hash([]byte(signature))
	return hash.Hex()
}

// GetABI returns the parsed ABI object
func (p *ABIParser) GetABI() abi.ABI {
	return p.contractABI
}

