package aiworkbench

import (
	"math"
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
)

// BudgetForScheduledExecution consumes an already qualified frozen profile.
// The existing resource owner's per-call point reservation is the bound;
// reference currency costs are never treated as customer point charges.
func BudgetForScheduledExecution(profile aicapability.ModelProfile, limits agent.Limits, count uint64) (ScheduleBudget, error) {
	if !limits.Valid() || profile.Currency != limits.Currency || profile.MaximumPromptTokens <= 0 || profile.MaximumCompletionTokens <= 0 || !profile.OfficialTariff.Valid() || !profile.OfficialTariff.MatchesWireModel(profile.ModelID) {
		return ScheduleBudget{}, ErrInvalidSchedule
	}
	perCall, err := profile.OfficialTariff.ReserveTextMicropoints(profile.MaximumPromptTokens, profile.MaximumCompletionTokens)
	if err != nil || perCall <= 0 || uint64(limits.ModelCalls) > math.MaxInt64/uint64(perCall) {
		return ScheduleBudget{}, ErrInvalidSchedule
	}
	budget := ScheduleBudget{Unit: aicapability.AIPointMicrounitV1, MaxOccurrences: count, PerRunUpperBoundMicros: uint64(perCall) * uint64(limits.ModelCalls)}
	total, err := budget.TotalUpperBoundMicros()
	if err != nil || total > math.MaxInt64 {
		return ScheduleBudget{}, ErrInvalidSchedule
	}
	return budget, nil
}
