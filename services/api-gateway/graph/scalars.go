package graph

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/smart-contract-event-indexer/shared/models"
)

// MarshalDateTime marshals a Timestamp to GraphQL DateTime
func MarshalDateTime(t models.Timestamp) string {
	return t.Time.Format(time.RFC3339)
}

// UnmarshalDateTime unmarshals a GraphQL DateTime to Timestamp
func UnmarshalDateTime(v interface{}) (models.Timestamp, error) {
	switch v := v.(type) {
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return models.Timestamp{}, err
		}
		return models.Timestamp{Time: t}, nil
	case int64:
		return models.Timestamp{Time: time.Unix(v, 0)}, nil
	case float64:
		return models.Timestamp{Time: time.Unix(int64(v), 0)}, nil
	default:
		return models.Timestamp{}, fmt.Errorf("invalid DateTime: %v", v)
	}
}

// MarshalBigInt marshals a BigInt to GraphQL BigInt
func MarshalBigInt(bi models.BigInt) string {
	return string(bi)
}

// UnmarshalBigInt unmarshals a GraphQL BigInt to BigInt
func UnmarshalBigInt(v interface{}) (models.BigInt, error) {
	switch v := v.(type) {
	case string:
		return models.BigInt(v), nil
	case int64:
		return models.BigInt(strconv.FormatInt(v, 10)), nil
	case int:
		return models.BigInt(strconv.Itoa(v)), nil
	case float64:
		return models.BigInt(strconv.FormatInt(int64(v), 10)), nil
	default:
		return models.BigInt(""), fmt.Errorf("invalid BigInt: %v", v)
	}
}

// MarshalAddress marshals an Address to GraphQL Address
func MarshalAddress(a models.Address) string {
	return string(a)
}

// UnmarshalAddress unmarshals a GraphQL Address to Address
func UnmarshalAddress(v interface{}) (models.Address, error) {
	switch v := v.(type) {
	case string:
		addr := models.Address(v)
		if err := addr.Validate(); err != nil {
			return models.Address(""), err
		}
		return addr.Normalize(), nil
	default:
		return models.Address(""), fmt.Errorf("invalid Address: %v", v)
	}
}

// MarshalJSONB marshals a JSONB to JSON string
func MarshalJSONB(jb models.JSONB) string {
	data, _ := json.Marshal(jb)
	return string(data)
}

// UnmarshalJSONB unmarshals a JSON string to JSONB
func UnmarshalJSONB(v interface{}) (models.JSONB, error) {
	switch v := v.(type) {
	case string:
		var jb models.JSONB
		err := json.Unmarshal([]byte(v), &jb)
		return jb, err
	case map[string]interface{}:
		return models.JSONB(v), nil
	default:
		return models.JSONB{}, fmt.Errorf("invalid JSONB: %v", v)
	}
}
