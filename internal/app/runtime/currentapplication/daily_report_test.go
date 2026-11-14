package currentapplication

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScheduledDailyReportDoesNotRequireProductAgent(t *testing.T) {
	cfg := agentRuntimeConfig()
	database := cfg.ProductAgent.Database
	database.User = "ai_workbench_runtime"
	cfg.ProductAgent = nil
	cfg.CommercialOwnerDatabase = nil
	cfg.Identity.TenantDirectoryToken = "synthetic-service-token"
	cfg.StoreCenter = &StoreCenterConfig{Enabled: true, OperationsCockpit: true}
	cfg.ReportCenter = &ReportCenterConfig{}
	var w AIWorkbenchConfig
	require.NoError(t, json.Unmarshal([]byte(`{"enabled":true,"conversationOnly":true,"allowedOrganizationIds":["org"],"schedules":{"temporalAddress":"127.0.0.1:7233","temporalNamespace":"default","storeDailyReports":true}}`), &w))
	w.Database = database
	require.NoError(t, w.validate(cfg), "deterministic reports must not require a product agent or model route")
	cfg.StoreCenter.OperationsCockpit = false
	require.Error(t, w.validate(cfg), "the real financial owner remains required")
}
