package httpapi

import (
	"context"
	"math"

	"task-processor/internal/agent"
	"task-processor/internal/aiworkbench"
)

type settledInvocationReader func(context.Context, string, string, string) (int64, error)

// settledRunPoints only totals a complete, non-running original run. A partial
// ledger view is never presented as the whole task's debit.
func settledRunPoints(ctx context.Context, state agent.State, read settledInvocationReader) (int64, error) {
	if read == nil || state.RunID == "" || state.Scope.OrganizationID == "" || state.Phase == agent.Running || state.PendingInvocationID != "" ||
		state.StopReason == agent.StopModelUnknown || state.StopReason == agent.StopUsageUnknown || state.StopReason == agent.StopExecutionOutcomeUnknown {
		return 0, aiworkbench.ErrUnavailable
	}
	seen := make(map[string]struct{})
	var total int64
	for _, observation := range state.History {
		id := observation.InvocationID
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			return 0, aiworkbench.ErrUnavailable
		}
		seen[id] = struct{}{}
		points, err := read(ctx, state.Scope.OrganizationID, id, state.RunID)
		if err != nil || points < 0 || points > math.MaxInt64-total {
			return 0, aiworkbench.ErrUnavailable
		}
		total += points
	}
	if len(seen) == 0 || len(seen) != state.Usage.ModelCalls {
		return 0, aiworkbench.ErrUnavailable
	}
	return total, nil
}
