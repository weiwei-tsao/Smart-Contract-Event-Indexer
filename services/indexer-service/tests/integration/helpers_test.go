package integration

import (
	"os"
	"testing"
)

const integrationEnvVar = "RUN_INDEXER_INTEGRATION_TESTS"

// requireIntegrationEnv skips the test unless the caller explicitly opts in.
func requireIntegrationEnv(t *testing.T) {
	t.Helper()
	if os.Getenv(integrationEnvVar) != "1" {
		t.Skipf("integration tests disabled (set %s=1 to enable)", integrationEnvVar)
	}
}
