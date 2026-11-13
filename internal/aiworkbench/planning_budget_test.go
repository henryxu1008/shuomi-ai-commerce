package aiworkbench

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPlanningBudgetAcceptsOnlyMatchingSnapshotAndNoLargerCharge(t *testing.T) {
	visible := PlanningBudget{ProfileDigest: "profile", Sequence: 3, MaximumCostMicros: 10, MaximumPointsMicros: "20", Currency: "CNY"}
	require.NoError(t, visible.Check(visible))
	changed := visible
	changed.Sequence++
	require.Error(t, visible.Check(changed))
	changed = visible
	changed.MaximumPointsMicros = "21"
	require.Error(t, visible.Check(changed))
	changed = visible
	changed.ProfileDigest = "other"
	require.Error(t, visible.Check(changed))
}
