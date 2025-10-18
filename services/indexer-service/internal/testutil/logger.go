package testutil

import (
	"github.com/smart-contract-event-indexer/shared/utils"
)

// NewTestLogger creates a logger for testing (discards output)
func NewTestLogger() utils.Logger {
	return utils.NewTestLogger()
}

// NewDebugLogger creates a logger for debugging tests (shows all output)
func NewDebugLogger() utils.Logger {
	return utils.NewLogger("test", "debug", "text")
}

