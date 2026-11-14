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

func TestModelFreeWorkbenchCanDisableDailyReports(t *testing.T) {
	cfg := agentRuntimeConfig()
	database := cfg.ProductAgent.Database
	database.User = "ai_workbench_runtime"
	cfg.ProductAgent = nil
	cfg.CommercialOwnerDatabase = nil
	cfg.Identity.TenantDirectoryToken = "synthetic-service-token"
	cfg.StoreCenter = &StoreCenterConfig{Enabled: true, OperationsCockpit: true}
	cfg.ReportCenter = &ReportCenterConfig{}
	for _, schedules := range []*AIWorkbenchSchedulesConfig{nil, {TemporalAddress: "127.0.0.1:7233", TemporalNamespace: "default", StoreDailyReports: false}} {
		w := AIWorkbenchConfig{Enabled: true, ConversationOnly: true, AllowedOrganizationIDs: []string{"org"}, Database: database, Schedules: schedules}
		require.NoError(t, w.validate(cfg), "storage-only startup must not depend on new report capture being enabled")
		invalid := w
		invalid.ConversationOnly = false
		require.Error(t, invalid.validate(cfg), "disabling reports must not authorize model execution")
		invalid = w
		invalid.Database.User = "unscoped_owner"
		require.Error(t, invalid.validate(cfg), "the original least privilege role is still required")
		invalid = w
		invalid.AllowedOrganizationIDs = nil
		require.Error(t, invalid.validate(cfg), "the organization boundary is still required")
	}
}
