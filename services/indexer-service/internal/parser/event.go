package parser

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// EventParser parses blockchain logs into structured events
type EventParser struct {
	abiParser *ABIParser
	logger    *utils.Logger
}

// NewEventParser creates a new event parser
func NewEventParser(abiParser *ABIParser, logger utils.Logger) *EventParser {
	return &EventParser{
		abiParser: abiParser,
		logger:    logger,
	}
}

// ParseLog converts a blockchain log into a structured Event
func (p *EventParser) ParseLog(log types.Log, blockTimestamp time.Time) (*models.Event, error) {
	if len(log.Topics) == 0 {
		return nil, fmt.Errorf("log has no topics")
	}
	
	// Get the event definition from the ABI
	eventID := log.Topics[0].Hex()
	event, exists := p.abiParser.GetEventByID(eventID)
	if !exists {
		return nil, fmt.Errorf("event with ID %s not found in ABI", eventID)
	}
	
	// Parse the event arguments
	args, err := p.parseEventArgs(event, log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event args: %w", err)
	}
	
	// Convert args to JSONB format
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal args to JSON: %w", err)
	}
	
	// Create the Event model
	parsedEvent := &models.Event{
		ContractAddress:  models.Address(log.Address.Hex()),
		EventName:        event.Name,
		BlockNumber:      int64(log.BlockNumber),
		BlockHash:        models.Hash(log.BlockHash.Hex()),
		TransactionHash:  models.Hash(log.TxHash.Hex()),
		TransactionIndex: int(log.TxIndex),
		LogIndex:         int(log.Index),
		Args:             models.JSONB(argsJSON),
		Timestamp:        blockTimestamp,
	}
	
	p.logger.WithFields(map[string]interface{}{
		"event_name":  event.Name,
		"block":       log.BlockNumber,
		"tx":          log.TxHash.Hex(),
		"log_index":   log.Index,
	}).Debug("Parsed event")
	
	return parsedEvent, nil
}

// ParseLogs parses multiple logs
func (p *EventParser) ParseLogs(logs []types.Log, blockTimestamp time.Time) ([]*models.Event, error) {
	events := make([]*models.Event, 0, len(logs))
	
	for _, log := range logs {
		event, err := p.ParseLog(log, blockTimestamp)
		if err != nil {
			p.logger.WithError(err).WithFields(map[string]interface{}{
				"block":     log.BlockNumber,
				"tx":        log.TxHash.Hex(),
				"log_index": log.Index,
			}).Warn("Failed to parse log, skipping")
			continue
		}
		events = append(events, event)
	}
	
	return events, nil
}

// parseEventArgs extracts and parses event arguments from a log
func (p *EventParser) parseEventArgs(event abi.Event, log types.Log) (map[string]interface{}, error) {
	// Unpack the event data
	eventData := make(map[string]interface{})
	
	// Parse indexed and non-indexed arguments
	if len(log.Data) > 0 {
		if err := event.Inputs.UnpackIntoMap(eventData, log.Data); err != nil {
			return nil, fmt.Errorf("failed to unpack event data: %w", err)
		}
	}
	
	// Parse indexed arguments from topics (skip topic[0] which is the event signature)
	indexedIndex := 0
	for i, input := range event.Inputs {
		if input.Indexed {
			topicIndex := indexedIndex + 1 // +1 to skip event signature
			if topicIndex >= len(log.Topics) {
				return nil, fmt.Errorf("not enough topics for indexed parameter %s", input.Name)
			}
			
			// Parse the indexed value
			value, err := p.parseIndexedValue(input, log.Topics[topicIndex])
			if err != nil {
				return nil, fmt.Errorf("failed to parse indexed parameter %s: %w", input.Name, err)
			}
			
			eventData[input.Name] = value
			indexedIndex++
		}
		
		// If the name is empty, use the index as the name
		if input.Name == "" {
			if val, exists := eventData[input.Name]; exists {
				delete(eventData, input.Name)
				eventData[fmt.Sprintf("arg%d", i)] = val
			}
		}
	}
	
	// Convert values to JSON-serializable format
	serializable := make(map[string]interface{})
	for key, value := range eventData {
		serializable[key] = p.convertToSerializable(value)
	}
	
	return serializable, nil
}

// parseIndexedValue parses an indexed parameter from a topic
func (p *EventParser) parseIndexedValue(input abi.Argument, topic common.Hash) (interface{}, error) {
	switch input.Type.T {
	case abi.AddressTy:
		// Address is stored in the last 20 bytes
		return common.BytesToAddress(topic.Bytes()).Hex(), nil
		
	case abi.IntTy, abi.UintTy:
		// Integer types
		return new(big.Int).SetBytes(topic.Bytes()), nil
		
	case abi.BoolTy:
		// Bool is stored as 0 or 1
		return topic.Big().Cmp(big.NewInt(0)) != 0, nil
		
	case abi.BytesTy, abi.FixedBytesTy:
		// Bytes are hashed, so we just return the hash
		return topic.Hex(), nil
		
	case abi.StringTy:
		// Strings are hashed, return the hash
		return topic.Hex(), nil
		
	default:
		// For complex types, return the hash
		return topic.Hex(), nil
	}
}

// convertToSerializable converts ABI values to JSON-serializable types
func (p *EventParser) convertToSerializable(value interface{}) interface{} {
	switch v := value.(type) {
	case *big.Int:
		// Convert big integers to strings to preserve precision
		return v.String()
		
	case common.Address:
		// Convert addresses to checksummed hex strings
		return v.Hex()
		
	case common.Hash:
		return v.Hex()
		
	case []byte:
		// Convert bytes to hex string
		return common.Bytes2Hex(v)
		
	case [32]byte:
		return common.Bytes2Hex(v[:])
		
	case []interface{}:
		// Recursively convert arrays
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = p.convertToSerializable(item)
		}
		return result
		
	case map[string]interface{}:
		// Recursively convert maps (for tuple types)
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = p.convertToSerializable(val)
		}
		return result
		
	default:
		// Return as-is for basic types (string, int, bool, etc.)
		return v
	}
}

// GetEventName returns the event name from a log
func (p *EventParser) GetEventName(log types.Log) (string, error) {
	if len(log.Topics) == 0 {
		return "", fmt.Errorf("log has no topics")
	}
	
	eventID := log.Topics[0].Hex()
	event, exists := p.abiParser.GetEventByID(eventID)
	if !exists {
		return "", fmt.Errorf("event with ID %s not found in ABI", eventID)
	}
	
	return event.Name, nil
}

// IsEventInABI checks if a log's event is defined in the ABI
func (p *EventParser) IsEventInABI(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}
	
	eventID := log.Topics[0].Hex()
	_, exists := p.abiParser.GetEventByID(eventID)
	return exists
}

// GetEventInputNames returns the input names for an event
func (p *EventParser) GetEventInputNames(eventName string) ([]string, error) {
	event, exists := p.abiParser.GetEventByName(eventName)
	if !exists {
		return nil, fmt.Errorf("event %s not found in ABI", eventName)
	}
	
	names := make([]string, len(event.Inputs))
	for i, input := range event.Inputs {
		if input.Name != "" {
			names[i] = input.Name
		} else {
			names[i] = fmt.Sprintf("arg%d", i)
		}
	}
	
	return names, nil
}

