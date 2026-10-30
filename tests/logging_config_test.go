package tests

import (
	"testing"
)

func TestMaintainedConfigFilesDoNotAdvertiseRetiredGlobalInputs(t *testing.T) {
	assertRestoredDebugPathsAbsent(t, "config/config-dev.yaml", "config/config-test.yaml", "config/config-prod.yaml")
}
