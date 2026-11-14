package setup

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/app/runtime/currentapplication"
	"testing"
)

func TestCandidatePermissionRejectionBlocksDisableWithoutPromotingUnrelatedUnknown(t *testing.T) {
	modules := []currentapplication.ModuleInspection{{InstallModule: currentapplication.InstallModule{ID: "productCollections"}, CheckStatus: "rejected"}, {InstallModule: currentapplication.InstallModule{ID: "marketResearch"}, CheckStatus: "unavailable"}}
	require.NotEmpty(t, candidatePermissionBlocks(modules, map[string]bool{"acquisitionPlans": false}))
	require.Empty(t, candidatePermissionBlocks(modules, map[string]bool{"notificationCenterDatabase": false}))
	modules[0].CheckStatus = "passed"
	require.Empty(t, candidatePermissionBlocks(modules, map[string]bool{"acquisitionPlans": false}))
}
