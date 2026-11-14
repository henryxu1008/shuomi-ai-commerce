package aiworkbench

import (
	"github.com/stretchr/testify/require"
	"math"
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
	"testing"
	"time"
)

func TestScheduledBudgetUsesOriginalAIPointReservationCeiling(t *testing.T) {
	profile := aicapability.ModelProfile{ModelID: "deepseek-v4.1-flash", Currency: "CNY", MaximumPromptTokens: 32000, MaximumCompletionTokens: 4000, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "reference-price-v1", FXVersion: "cny-identity-v1", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1_000_000, Model: "deepseek-v4.1-flash", Channel: "bailian-beijing", UsageVersion: "reference-text-totals-v1", Rates: aicapability.OfficialModelRates{InputText: 2_000_000, OutputText: 8_000_000}}}
	limits := agent.Limits{Steps: 4, ModelCalls: 2, Tokens: 72000, CostMicros: 1_000_000, Currency: "CNY", Runtime: time.Minute}
	budget, err := BudgetForScheduledExecution(profile, limits, 3)
	require.NoError(t, err)
	require.Equal(t, aicapability.AIPointMicrounitV1, budget.Unit)
	require.EqualValues(t, 19_200_000, budget.PerRunUpperBoundMicros, "retail point upper bound covers both original model-call reservations")
	total, err := budget.TotalUpperBoundMicros()
	require.NoError(t, err)
	require.EqualValues(t, 57_600_000, total)
	_, err = BudgetForScheduledExecution(profile, limits, math.MaxUint64)
	require.Error(t, err)
	profile.OfficialTariff.UnitVersion = "currency-micros"
	_, err = BudgetForScheduledExecution(profile, limits, 3)
	require.Error(t, err, "reference currency must never replace the original point unit")
}
