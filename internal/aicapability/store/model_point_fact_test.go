package store

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"testing"
	"time"
)

func TestModelPointSchemaAndScopedCanonicalFact(t *testing.T) {
	db := newInvocationLedgerDB(t)
	require.NoError(t, VerifyModelPointInvocationSchema(context.Background(), db))
	r := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{InvocationID: "canonical", TenantID: "org", UserID: "user", MemberID: "member", AgentRunID: "run", InputHash: "input", Operation: aicapability.OperationProductAgentDecision, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 100, MaximumCompletionTokens: 50, PointTariff: aicapability.ModelPointTariff{PriceVersion: "synthetic", InputPointsPerMillionTokens: 1, OutputPointsPerMillionTokens: 1}}
	_, err := r.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	got, err := r.ReadModelInvocation(context.Background(), "org", "canonical")
	require.NoError(t, err)
	require.Equal(t, fact.PointTariff, got.PointTariff)
	_, err = r.ReadModelInvocation(context.Background(), "other", "canonical")
	require.Error(t, err)
	// Synthetic obsolete schema is rejected without changing it.
	require.NoError(t, db.Exec("ALTER TABLE ai_invocations DROP COLUMN point_price_version").Error)
	require.Error(t, VerifyModelPointInvocationSchema(context.Background(), db))
	require.False(t, db.Migrator().HasColumn(&invocationRow{}, "point_price_version"))
}

func TestContentPricedInvocationKeepsCanonicalIdentityAndReplay(t *testing.T) {
	db := newInvocationLedgerDB(t)
	ledger := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{InvocationID: "content-invocation", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "content-run", InputHash: "frozen-content-selection", Operation: aicapability.OperationProductContentDecision, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 100, MaximumCompletionTokens: 50, PointTariff: aicapability.ModelPointTariff{PriceVersion: "synthetic", InputPointsPerMillionTokens: 1, OutputPointsPerMillionTokens: 1}}
	acquired, err := ledger.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.True(t, acquired)
	acquired, err = ledger.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.False(t, acquired)
	read, err := ledger.ReadModelInvocation(context.Background(), fact.TenantID, fact.InvocationID)
	require.NoError(t, err)
	require.Equal(t, fact.Operation, read.Operation)
	require.Equal(t, fact.AgentRunID, read.AgentRunID)
	foreign := fact
	foreign.Operation = aicapability.OperationProductAgentDecision
	_, err = ledger.ClaimInvocation(context.Background(), foreign)
	require.Error(t, err, "a title dispatch cannot reuse the content ledger identity")
	_, err = ledger.ReadModelInvocation(context.Background(), "foreign", fact.InvocationID)
	require.Error(t, err)
}

func TestAdvisoryAgentsPricedInvocationRetainsExistingScopedLedger(t *testing.T) {
	for _, op := range []aicapability.Operation{aicapability.OperationListingMaterialsDecision, aicapability.OperationProductSelectionDecision} {
		t.Run(string(op), func(t *testing.T) {
			db := newInvocationLedgerDB(t)
			ledger := NewGormInvocationRecorder(db)
			fact := aicapability.InvocationRecord{InvocationID: "materials-invocation", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "materials-run", InputHash: "exact-material-disclosure", Operation: op, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 100, MaximumCompletionTokens: 50, PointTariff: aicapability.ModelPointTariff{PriceVersion: "synthetic", InputPointsPerMillionTokens: 1, OutputPointsPerMillionTokens: 1}}
			acquired, err := ledger.ClaimInvocation(context.Background(), fact)
			require.NoError(t, err)
			require.True(t, acquired)
			acquired, err = ledger.ClaimInvocation(context.Background(), fact)
			require.NoError(t, err)
			require.False(t, acquired)
			read, err := ledger.ReadModelInvocation(context.Background(), "org", fact.InvocationID)
			require.NoError(t, err)
			require.Equal(t, fact.AgentRunID, read.AgentRunID)
			require.Equal(t, fact.PointTariff, read.PointTariff)
			foreign := fact
			foreign.Operation = aicapability.OperationProductContentDecision
			_, err = ledger.ClaimInvocation(context.Background(), foreign)
			require.Error(t, err)
			_, err = ledger.ReadModelInvocation(context.Background(), "other-org", fact.InvocationID)
			require.Error(t, err)

		})
	}
}

func TestPlannerPricedInvocationUsesSameScopedOwnerWithoutFakeAgentRun(t *testing.T) {
	db := newInvocationLedgerDB(t)
	r := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{
		InvocationID: "chat-plan-invocation", TenantID: "org", UserID: "actor", MemberID: "member",
		InputHash: "frozen-profile-and-message", Operation: aicapability.OperationAIWorkbenchChatPlan,
		Capability: aicapability.CapabilityAIWorkbenchChatPlanning,
		Outcome:    aicapability.InvocationDispatched, StartedAt: time.Now().UTC(),
		MaximumPromptTokens: 100, MaximumCompletionTokens: 50,
		PointTariff: aicapability.ModelPointTariff{PriceVersion: "synthetic", InputPointsPerMillionTokens: 1, OutputPointsPerMillionTokens: 1},
	}
	acquired, err := r.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.True(t, acquired)
	acquired, err = r.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.False(t, acquired)
	got, err := r.ReadModelInvocation(context.Background(), "org", fact.InvocationID)
	require.NoError(t, err)
	require.Empty(t, got.AgentRunID)
	require.Empty(t, got.BusinessTaskID)
	require.Equal(t, fact.PointTariff, got.PointTariff)
	foreign := fact
	foreign.AgentRunID = "invented-run"
	_, err = r.ClaimInvocation(context.Background(), foreign)
	require.Error(t, err)
	foreign = fact
	foreign.BusinessTaskID = "invented-task"
	_, err = r.ClaimInvocation(context.Background(), foreign)
	require.Error(t, err)
}
