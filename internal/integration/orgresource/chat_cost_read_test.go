package orgresourceadapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestReadSettledModelInvocationPointsRequiresOriginalRunAndReceipt(t *testing.T) {
	db, recorder, repo, auth, fact := modelPointFixture(t, aicapability.OperationProductContentDecision)
	ctx := context.Background()
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, fact.TenantID, fact.MemberID, fact.InvocationID, 20, fact.StartedAt))
	_, err := repo.ReadSettledModelInvocationPoints(ctx, fact.TenantID, fact.InvocationID, fact.AgentRunID)
	require.Error(t, err, "a reservation is not a settlement")
	fact.Outcome, fact.UsageKnown, fact.FinishedAt = aicapability.InvocationSucceeded, true, fact.StartedAt.Add(time.Second)
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 3, 2, 5
	fact.BillableUsage = syntheticUncachedUsage(3, 2)
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	points, err := repo.ReadSettledModelInvocationPoints(ctx, fact.TenantID, fact.InvocationID, fact.AgentRunID)
	require.NoError(t, err)
	require.EqualValues(t, 7, points)
	_, err = repo.ReadSettledModelInvocationPoints(ctx, fact.TenantID, fact.InvocationID, "other-run")
	require.Error(t, err)
	_, err = repo.ReadSettledModelInvocationPoints(ctx, "other-org", fact.InvocationID, fact.AgentRunID)
	require.Error(t, err)
	auth.err = errors.New("denied")
	_, err = repo.ReadSettledModelInvocationPoints(ctx, fact.TenantID, fact.InvocationID, fact.AgentRunID)
	require.Error(t, err)
	auth.err = nil
	require.NoError(t, db.Model(&organizationResourceOperationRow{}).Where("operation_id = ?", modelPointKey("finalize", fact.InvocationID)).Update("immutable_result_snapshot", "{}").Error)
	_, err = repo.ReadSettledModelInvocationPoints(ctx, fact.TenantID, fact.InvocationID, fact.AgentRunID)
	require.Error(t, err, "a corrupted receipt must not yield an estimated charge")
}
