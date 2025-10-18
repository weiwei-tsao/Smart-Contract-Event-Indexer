package testutil

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Sample ERC20 ABI for testing
const ERC20ABI = `[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "from",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "to",
				"type": "address"
			},
			{
				"indexed": false,
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "Transfer",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "owner",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "spender",
				"type": "address"
			},
			{
				"indexed": false,
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "Approval",
		"type": "event"
	}
]`

// Invalid ABI for error testing
const InvalidABI = `[{"invalid": "json structure"`

// CreateMockTransferLog creates a mock ERC20 Transfer event log
func CreateMockTransferLog() types.Log {
	return types.Log{
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Topics: []common.Hash{
			// Transfer event signature
			common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			// from address (indexed)
			common.HexToHash("0x000000000000000000000000a0b0c0d0e0f0a0b0c0d0e0f0a0b0c0d0e0f0a0b0"),
			// to address (indexed)
			common.HexToHash("0x000000000000000000000000b1c1d1e1f1a1b1c1d1e1f1a1b1c1d1e1f1a1b1c1"),
		},
		// value (non-indexed) - 1000000000000000000 (1 ETH in wei)
		Data:        common.Hex2Bytes("0000000000000000000000000000000000000000000000000de0b6b3a7640000"),
		BlockNumber: 12345,
		TxHash:      common.HexToHash("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"),
		TxIndex:     5,
		BlockHash:   common.HexToHash("0x1111222233334444555566667777888899990000aaaabbbbccccddddeeeeffff"),
		Index:       10,
		Removed:     false,
	}
}

// CreateMockApprovalLog creates a mock ERC20 Approval event log
func CreateMockApprovalLog() types.Log {
	return types.Log{
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Topics: []common.Hash{
			// Approval event signature
			common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"),
			// owner address (indexed)
			common.HexToHash("0x000000000000000000000000a0b0c0d0e0f0a0b0c0d0e0f0a0b0c0d0e0f0a0b0"),
			// spender address (indexed)
			common.HexToHash("0x000000000000000000000000c2d2e2f2a2b2c2d2e2f2a2b2c2d2e2f2a2b2c2d2"),
		},
		// value (non-indexed) - 5000000000000000000 (5 ETH in wei)
		Data:        common.Hex2Bytes("0000000000000000000000000000000000000000000000004563918244f40000"),
		BlockNumber: 12346,
		TxHash:      common.HexToHash("0xfedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"),
		TxIndex:     3,
		BlockHash:   common.HexToHash("0xfffeeddccbbaa998877665544332211000998877665544332211ffeeddccbb"),
		Index:       8,
		Removed:     false,
	}
}

// CreateLogWithInvalidTopics creates a log with missing topics (for error testing)
func CreateLogWithInvalidTopics() types.Log {
	return types.Log{
		Address:     common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Topics:      []common.Hash{}, // Empty topics - invalid
		Data:        []byte{},
		BlockNumber: 100,
	}
}

// BigIntValue returns a sample big.Int for testing
func BigIntValue(value int64) *big.Int {
	return big.NewInt(value)
}

// TestAddresses returns common test addresses
var TestAddresses = struct {
	Contract common.Address
	Alice    common.Address
	Bob      common.Address
	Charlie  common.Address
}{
	Contract: common.HexToAddress("0x1234567890123456789012345678901234567890"),
	Alice:    common.HexToAddress("0xA0B0C0D0E0F0A0B0C0D0E0F0A0B0C0D0E0F0A0B0"),
	Bob:      common.HexToAddress("0xB1C1D1E1F1A1B1C1D1E1F1A1B1C1D1E1F1A1B1C1"),
	Charlie:  common.HexToAddress("0xC2D2E2F2A2B2C2D2E2F2A2B2C2D2E2F2A2B2C2D2"),
}

