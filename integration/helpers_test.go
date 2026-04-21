package integration

import (
	"os"
	"testing"
)

const integrationEnv = "RUN_INTEGRATION"

// requireIntegration skips live provider tests unless explicitly enabled.
func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv(integrationEnv) != "1" {
		t.Skip("set RUN_INTEGRATION=1 to run live integration tests")
	}
}

// requireLocalIntegration skips local provider tests unless explicitly enabled.
func requireLocalIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_INTEGRATION_LOCAL") == "" {
		t.Skip("TEST_INTEGRATION_LOCAL not set, skipping local provider tests")
	}
}
