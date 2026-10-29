package imageagentworker

import (
	"context"
	"task-processor/internal/imageagent"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageSetExecutorDoesNotDispatchUnconfirmedOrSingleImageInputs(t *testing.T) {
	executor := imageSetSlotExecutor{}
	input := imageagent.SlotExecutionInput{}
	_, err := executor.QuoteSlot(context.Background(), input, imageagent.BudgetPolicy{})
	require.ErrorIs(t, err, imageagent.ErrCommandBlocked)
	_, err = executor.GenerateQuotedSlot(context.Background(), input, imageagent.SlotUsageQuote{})
	require.ErrorIs(t, err, imageagent.ErrCommandBlocked)
	_, err = executor.BuildSlotResult(context.Background(), input, imageagent.PublishedSlotOutput{})
	require.ErrorIs(t, err, imageagent.ErrCommandBlocked)
	_, err = NewManagedImageSetTemporalDependencies(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.ErrorIs(t, err, imageagent.ErrValidation)
}
