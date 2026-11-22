package setup

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/app/runtime/currentapplication"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCandidatePermissionRejectionBlocksDisableWithoutPromotingUnrelatedUnknown(t *testing.T) {
	modules := []currentapplication.ModuleInspection{{InstallModule: currentapplication.InstallModule{ID: "productCollections"}, CheckStatus: "rejected"}, {InstallModule: currentapplication.InstallModule{ID: "marketResearch"}, CheckStatus: "unavailable"}}
	require.NotEmpty(t, candidatePermissionBlocks(modules, map[string]bool{"acquisitionPlans": false}))
	require.Empty(t, candidatePermissionBlocks(modules, map[string]bool{"notificationCenterDatabase": false}))
	modules[0].CheckStatus = "passed"
	require.Empty(t, candidatePermissionBlocks(modules, map[string]bool{"acquisitionPlans": false}))
}

func TestCandidatePermissionRejectionBlocksEnablingEcoservices(t *testing.T) {
	module := currentapplication.ModuleInspection{
		InstallModule: currentapplication.InstallModule{ID: "ecoservices", Label: "生态服务"},
		CheckStatus:   "rejected",
	}
	selected := map[string]bool{"ecoservices": true}
	require.NotEmpty(t, candidatePermissionBlocks([]currentapplication.ModuleInspection{module}, selected))

	module.CheckStatus = "passed"
	require.Empty(t, candidatePermissionBlocks([]currentapplication.ModuleInspection{module}, selected))
}
