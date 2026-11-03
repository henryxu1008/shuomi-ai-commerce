package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestOfficialUsageMetadataSurvivesCanonicalLedgerAndCannotChangeOnReplay(t *testing.T) {
	db := newInvocationLedgerDB(t)
	ledger := NewGormInvocationRecorder(db)
	ctx := context.Background()
	fact := aicapability.InvocationRecord{InvocationID: "official-metered", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run", InputHash: "input", StartedAt: time.Now().UTC(), Outcome: aicapability.InvocationDispatched, Operation: aicapability.OperationProductAgentDecision}
	acquired, err := ledger.ClaimInvocation(ctx, fact)
	require.NoError(t, err)
	require.True(t, acquired)
	terminal := fact
	terminal.Outcome, terminal.UsageKnown, terminal.FinishedAt = aicapability.InvocationSucceeded, true, time.Now().UTC()
	terminal.PromptTokens, terminal.CompletionTokens, terminal.TotalTokens = 5, 2, 7
	terminal.BillableUsage = &aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true, InputText: 5, CachedInputText: 2, OutputText: 2}
	require.NoError(t, ledger.RecordInvocation(ctx, terminal))
	got, found, err := ledger.FindInvocation(ctx, "org", "member", fact.InvocationID, "input")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, terminal.BillableUsage, got.BillableUsage)
	require.NoError(t, ledger.RecordInvocation(ctx, terminal))
	changedUsage := *terminal.BillableUsage
	changedUsage.CachedInputText = 1
	terminal.BillableUsage = &changedUsage
	require.Error(t, ledger.RecordInvocation(ctx, terminal), "terminal usage is immutable, including cache partitions")

	require.NoError(t, db.Exec("ALTER TABLE ai_invocations DROP COLUMN billable_usage_json").Error)
	require.Error(t, VerifyModelPointInvocationSchema(ctx, db), "startup must reject the missing field without changing the database")
	require.False(t, db.Migrator().HasColumn(&invocationRow{}, "billable_usage_json"))
}

func TestOfficialUsageMetadataIsNotAClaimOrNoSendProof(t *testing.T) {
	db := newInvocationLedgerDB(t)
	ledger := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{InvocationID: "premature-usage", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run", InputHash: "input", StartedAt: time.Now().UTC(), Outcome: aicapability.InvocationDispatched, Operation: aicapability.OperationProductAgentDecision,
		BillableUsage: &aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true}}
	_, err := ledger.ClaimInvocation(context.Background(), fact)
	require.Error(t, err)
	fact.Outcome, fact.UsageKnown, fact.FinishedAt = aicapability.InvocationSucceeded, true, time.Now().UTC()
	fact.PromptTokens, fact.TotalTokens = 5, 5
	require.Error(t, ledger.RecordInvocation(context.Background(), fact), "a complete-looking but inconsistent partition is rejected")
}
